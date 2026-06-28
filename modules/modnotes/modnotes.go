package modnotes

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/brendanjhnsn/TrackerAPI/core/config"
	"github.com/brendanjhnsn/TrackerAPI/core/database"
	"github.com/brendanjhnsn/TrackerAPI/modules/auth"
	"gorm.io/gorm"
)

type Module struct {
	db  *gorm.DB
	cfg *config.Config
}

func New(db *gorm.DB, cfg *config.Config) *Module {
	return &Module{db: db, cfg: cfg}
}

func (m *Module) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/notes", m.handleNotes)
	mux.HandleFunc("/api/training", m.handleTraining)
	mux.HandleFunc("/api/removed-mods", m.handleRemovedMods)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func requireManagement(w http.ResponseWriter, r *http.Request) (auth.Role, bool) {
	role, ok := auth.RoleFromContext(r.Context())
	if !ok || (role != auth.RoleManager && role != auth.RoleDirector) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "manager or director role required"})
		return 0, false
	}
	return role, true
}

// ----- Notes -----

func (m *Module) handleNotes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		m.getNotes(w, r)
	case http.MethodPost:
		m.createNote(w, r)
	case http.MethodDelete:
		m.deleteNote(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (m *Module) getNotes(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireManagement(w, r); !ok {
		return
	}
	modID := r.URL.Query().Get("mod_id")
	if modID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "mod_id is required"})
		return
	}
	var notes []database.ModNote
	if err := m.db.Where("mod_member_id = ?", modID).Order("created_at desc").Find(&notes).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "database error"})
		return
	}
	if notes == nil {
		notes = []database.ModNote{}
	}
	writeJSON(w, http.StatusOK, notes)
}

type createNoteRequest struct {
	ModMemberID string `json:"mod_member_id"`
	Content     string `json:"content"`
}

func (m *Module) createNote(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireManagement(w, r); !ok {
		return
	}
	var req createNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.ModMemberID == "" || req.Content == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "mod_member_id and content are required"})
		return
	}
	authorID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	note := database.ModNote{
		ModMemberID:    req.ModMemberID,
		AuthorMemberID: authorID,
		Content:        req.Content,
	}
	if err := m.db.Create(&note).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create note"})
		return
	}
	writeJSON(w, http.StatusCreated, note)
}

func (m *Module) deleteNote(w http.ResponseWriter, r *http.Request) {
	role, ok := auth.RoleFromContext(r.Context())
	if !ok || role != auth.RoleDirector {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "only directors can delete notes"})
		return
	}
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id is required"})
		return
	}
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	result := m.db.Delete(&database.ModNote{}, id)
	if result.Error != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete note"})
		return
	}
	if result.RowsAffected == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "note not found"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ----- Training -----

func (m *Module) handleTraining(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		m.getTraining(w, r)
	case http.MethodPut:
		m.updateTraining(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (m *Module) getTraining(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireManagement(w, r); !ok {
		return
	}
	modID := r.URL.Query().Get("mod_id")
	if modID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "mod_id is required"})
		return
	}
	var training database.ModTraining
	err := m.db.Where("mod_member_id = ?", modID).First(&training).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeJSON(w, http.StatusOK, database.ModTraining{ModMemberID: modID})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "database error"})
		return
	}
	writeJSON(w, http.StatusOK, training)
}

type updateTrainingRequest struct {
	ModMemberID   string  `json:"mod_member_id"`
	InTraining    bool    `json:"in_training"`
	TrainingStart *string `json:"training_start"`
	TrainingEnd   *string `json:"training_end"`
}

func (m *Module) updateTraining(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireManagement(w, r); !ok {
		return
	}
	var req updateTrainingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.ModMemberID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "mod_member_id is required"})
		return
	}

	var startDate, endDate *time.Time
	if req.TrainingStart != nil && *req.TrainingStart != "" {
		t, err := time.Parse("2006-01-02", *req.TrainingStart)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid training_start format, use YYYY-MM-DD"})
			return
		}
		startDate = &t
	}
	if req.TrainingEnd != nil && *req.TrainingEnd != "" {
		t, err := time.Parse("2006-01-02", *req.TrainingEnd)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid training_end format, use YYYY-MM-DD"})
			return
		}
		endDate = &t
	}

	var training database.ModTraining
	m.db.Where("mod_member_id = ?", req.ModMemberID).FirstOrInit(&training)
	training.ModMemberID = req.ModMemberID
	training.InTraining = req.InTraining
	training.TrainingStart = startDate
	training.TrainingEnd = endDate

	if err := m.db.Save(&training).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update training"})
		return
	}
	writeJSON(w, http.StatusOK, training)
}

// ----- Removed Mods -----

func (m *Module) handleRemovedMods(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		m.getRemovedMods(w, r)
	case http.MethodPost:
		m.addRemovedMod(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (m *Module) getRemovedMods(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireManagement(w, r); !ok {
		return
	}
	var removed []database.RemovedMod
	if err := m.db.Find(&removed).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "database error"})
		return
	}
	if removed == nil {
		removed = []database.RemovedMod{}
	}
	writeJSON(w, http.StatusOK, removed)
}

type removeModRequest struct {
	MemberID string `json:"member_id"`
}

func (m *Module) addRemovedMod(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireManagement(w, r); !ok {
		return
	}
	var req removeModRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.MemberID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "member_id is required"})
		return
	}
	removed := database.RemovedMod{MemberID: req.MemberID}
	if err := m.db.Where("member_id = ?", req.MemberID).FirstOrCreate(&removed).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to remove mod"})
		return
	}
	writeJSON(w, http.StatusOK, removed)
}
