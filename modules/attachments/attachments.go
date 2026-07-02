package attachments

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/brendanjhnsn/TrackerAPI/core/config"
	"github.com/brendanjhnsn/TrackerAPI/core/database"
	"github.com/brendanjhnsn/TrackerAPI/modules/auth"
	"gorm.io/gorm"
)

const maxUploadSize = 10 << 20 // 10 MB

var allowedMIME = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
	"application/pdf":    true,
	"application/msword": true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
	"text/plain":   true,
	"text/csv":     true,
	"application/vnd.ms-excel": true,
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": true,
}

// extMIME maps file extensions to MIME types for formats that http.DetectContentType
// cannot distinguish (e.g. .docx/.xlsx are ZIP archives internally).
var extMIME = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",
	".webp": "image/webp",
	".pdf":  "application/pdf",
	".doc":  "application/msword",
	".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	".xls":  "application/vnd.ms-excel",
	".csv":  "text/csv",
	".txt":  "text/plain",
}

type Module struct {
	db         *gorm.DB
	cfg        *config.Config
	uploadsDir string
}

func New(db *gorm.DB, cfg *config.Config) *Module {
	dir := "uploads"
	if cfg != nil && cfg.UploadsDir != "" {
		dir = cfg.UploadsDir
	}
	_ = os.MkdirAll(dir, 0755)
	return &Module{db: db, cfg: cfg, uploadsDir: dir}
}

func (m *Module) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/attachments/file", m.handleServe)
	mux.HandleFunc("/api/attachments", m.handleAttachments)
}

func requireManager(w http.ResponseWriter, r *http.Request) (auth.Role, string, bool) {
	role, ok := auth.RoleFromContext(r.Context())
	if !ok || role < auth.RoleManager {
		http.Error(w, "forbidden", http.StatusForbidden)
		return 0, "", false
	}
	userID, _ := auth.UserIDFromContext(r.Context())
	return role, userID, true
}

func (m *Module) handleAttachments(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		m.handleList(w, r)
	case http.MethodPost:
		m.handleUpload(w, r)
	case http.MethodDelete:
		m.handleDelete(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (m *Module) handleList(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := requireManager(w, r); !ok {
		return
	}
	ownerType := r.URL.Query().Get("owner_type")
	if ownerType != "note" && ownerType != "action" {
		http.Error(w, "owner_type must be 'note' or 'action'", http.StatusBadRequest)
		return
	}
	ownerIDsRaw := r.URL.Query().Get("owner_ids")
	if ownerIDsRaw == "" {
		http.Error(w, "owner_ids is required", http.StatusBadRequest)
		return
	}
	var ownerIDs []uint64
	for _, s := range strings.Split(ownerIDsRaw, ",") {
		id, err := strconv.ParseUint(strings.TrimSpace(s), 10, 64)
		if err != nil {
			continue
		}
		ownerIDs = append(ownerIDs, id)
	}
	if len(ownerIDs) == 0 {
		writeJSON(w, http.StatusOK, []database.Attachment{})
		return
	}
	var atts []database.Attachment
	if err := m.db.Where("owner_type = ? AND owner_id IN ?", ownerType, ownerIDs).Order("created_at asc").Find(&atts).Error; err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if atts == nil {
		atts = []database.Attachment{}
	}
	writeJSON(w, http.StatusOK, atts)
}

func (m *Module) handleUpload(w http.ResponseWriter, r *http.Request) {
	_, userID, ok := requireManager(w, r)
	if !ok {
		return
	}
	ownerType := r.URL.Query().Get("owner_type")
	if ownerType != "note" && ownerType != "action" {
		http.Error(w, "owner_type must be 'note' or 'action'", http.StatusBadRequest)
		return
	}
	ownerIDStr := r.URL.Query().Get("owner_id")
	if ownerIDStr == "" {
		http.Error(w, "owner_id is required", http.StatusBadRequest)
		return
	}
	ownerID, err := strconv.ParseUint(ownerIDStr, 10, 64)
	if err != nil || ownerID == 0 {
		http.Error(w, "invalid owner_id", http.StatusBadRequest)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		http.Error(w, "file too large (max 10 MB)", http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file field is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Detect MIME from first 512 bytes, then seek back.
	buf := make([]byte, 512)
	n, _ := file.Read(buf)
	mimeType := http.DetectContentType(buf[:n])
	// Strip parameters — DetectContentType can return "text/plain; charset=utf-8".
	if idx := strings.Index(mimeType, ";"); idx != -1 {
		mimeType = strings.TrimSpace(mimeType[:idx])
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	// DetectContentType returns "application/zip" for .docx/.xlsx (ZIP-based formats)
	// and "application/octet-stream" for .doc. Use the hardcoded extension map as a
	// fallback so uploads aren't rejected for types the sniffer can't distinguish.
	if !allowedMIME[mimeType] {
		if mapped, ok := extMIME[strings.ToLower(filepath.Ext(header.Filename))]; ok {
			mimeType = mapped
		}
	}
	if !allowedMIME[mimeType] {
		http.Error(w, "file type not allowed: "+mimeType, http.StatusUnprocessableEntity)
		return
	}

	ext := filepath.Ext(header.Filename)
	b := make([]byte, 16)
	rand.Read(b)
	storedName := hex.EncodeToString(b) + ext

	dst, err := os.Create(filepath.Join(m.uploadsDir, storedName))
	if err != nil {
		http.Error(w, "failed to save file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()
	size, err := io.Copy(dst, file)
	if err != nil {
		os.Remove(filepath.Join(m.uploadsDir, storedName))
		http.Error(w, "failed to write file", http.StatusInternalServerError)
		return
	}

	att := database.Attachment{
		OwnerType:  ownerType,
		OwnerID:    uint(ownerID),
		FileName:   header.Filename,
		StoredName: storedName,
		MimeType:   mimeType,
		Size:       size,
		UploadedBy: userID,
	}
	if err := m.db.Create(&att).Error; err != nil {
		os.Remove(filepath.Join(m.uploadsDir, storedName))
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, att)
}

func (m *Module) handleServe(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := requireManager(w, r); !ok {
		return
	}
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id == 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var att database.Attachment
	if err := m.db.First(&att, id).Error; err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	filePath := filepath.Join(m.uploadsDir, att.StoredName)
	f, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", att.MimeType)
	if strings.HasPrefix(att.MimeType, "image/") {
		w.Header().Set("Content-Disposition", "inline; filename=\""+att.FileName+"\"")
	} else {
		w.Header().Set("Content-Disposition", "attachment; filename=\""+att.FileName+"\"")
	}
	io.Copy(w, f)
}

func (m *Module) handleDelete(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := requireManager(w, r); !ok {
		return
	}
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id == 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var att database.Attachment
	if err := m.db.First(&att, id).Error; err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	_ = os.Remove(filepath.Join(m.uploadsDir, att.StoredName))
	if err := m.db.Delete(&att).Error; err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
