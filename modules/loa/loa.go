package loa

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/brendanjhnsn/TrackerAPI/core/config"
	"github.com/brendanjhnsn/TrackerAPI/core/database"
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
	mux.HandleFunc("/api/loa", m.handleLOA)
}

func IsMemberOnLOA(db *gorm.DB, guildID, memberID string, today time.Time) bool {
	var loa database.LOA
	result := db.Where("guild_id = ? AND member_id = ? AND start_date <= ? AND end_date >= ?",
		guildID, memberID, today, today).First(&loa)
	return result.Error == nil
}

func (m *Module) handleLOA(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case http.MethodGet:
		m.getLOAs(w, r)
	case http.MethodPost:
		m.createLOA(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (m *Module) getLOAs(w http.ResponseWriter, r *http.Request) {
	var loas []database.LOA
	if r.URL.Query().Get("all") == "true" {
		if err := m.db.Find(&loas).Error; err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "database error"})
			return
		}
	} else {
		now := time.Now().UTC()
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		if err := m.db.Where("start_date <= ? AND end_date >= ?", today, today).Find(&loas).Error; err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "database error"})
			return
		}
	}
	if loas == nil {
		loas = []database.LOA{}
	}
	_ = json.NewEncoder(w).Encode(loas)
}

type createLOARequest struct {
	GuildID   string `json:"guild_id"`
	MemberID  string `json:"member_id"`
	Reason    string `json:"reason"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

func (m *Module) createLOA(w http.ResponseWriter, r *http.Request) {
	var req createLOARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}
	if req.GuildID == "" || req.MemberID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "guild_id and member_id are required"})
		return
	}
	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid start_date format, use YYYY-MM-DD"})
		return
	}
	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid end_date format, use YYYY-MM-DD"})
		return
	}
	if endDate.Before(startDate) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "end_date must be on or after start_date"})
		return
	}
	loa := database.LOA{
		GuildID:   req.GuildID,
		MemberID:  req.MemberID,
		Reason:    req.Reason,
		StartDate: &startDate,
		EndDate:   &endDate,
	}
	if err := m.db.Create(&loa).Error; err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to create LOA"})
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(loa)
}
