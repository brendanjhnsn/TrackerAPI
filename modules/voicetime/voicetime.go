package voicetime

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/brendanjhnsn/TrackerAPI/core/config"
	"github.com/brendanjhnsn/TrackerAPI/core/database"
	"github.com/bwmarrin/discordgo"
	"gorm.io/gorm"
)

type Module struct {
	db  *gorm.DB
	cfg *config.Config
}

func New(db *gorm.DB, cfg *config.Config) *Module {
	return &Module{db: db, cfg: cfg}
}

func (m *Module) Register(s *discordgo.Session) {
	s.AddHandler(m.onVoiceStateUpdate)
}

func (m *Module) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/voicetime", m.handleVoiceTime)
}

func (m *Module) onVoiceStateUpdate(s *discordgo.Session, vs *discordgo.VoiceStateUpdate) {
	if m.cfg.ModRoleID == "" && m.cfg.ModRoleName == "" {
		return
	}
	member, err := s.GuildMember(vs.GuildID, vs.UserID)
	if err != nil {
		return
	}
	modRoleID, err := m.getModRoleID(s, vs.GuildID)
	if err != nil || !memberHasModRole(member, modRoleID) {
		return
	}
	if vs.ChannelID == "" {
		var vt database.VoiceTime
		result := m.db.Where("member_id = ? AND guild_id = ? AND left_at IS NULL", vs.UserID, vs.GuildID).First(&vt)
		if result.Error == nil {
			now := time.Now().UTC()
			if err := m.db.Model(&vt).Updates(map[string]interface{}{
				"left_at":  now,
				"duration": int64(now.Sub(vt.JoinedAt).Seconds()),
			}).Error; err != nil {
				log.Printf("[VOICETIME] Failed to update voice leave for member %s: %v", vs.UserID, err)
			}
		} else if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			log.Printf("[VOICETIME] Failed to query open voice session for member %s: %v", vs.UserID, result.Error)
		}
	} else {
		now := time.Now().UTC()
		dateOnly := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		if err := m.db.Create(&database.VoiceTime{
			GuildID:   vs.GuildID,
			MemberID:  vs.UserID,
			ChannelID: vs.ChannelID,
			Date:      &dateOnly,
			JoinedAt:  now,
		}).Error; err != nil {
			log.Printf("[VOICETIME] Failed to create voice join for member %s: %v", vs.UserID, err)
		}
	}
}

func (m *Module) handleVoiceTime(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	q := r.URL.Query()
	memberID := q.Get("member_id")
	dateStr := q.Get("date")
	startDateStr := q.Get("start_date")
	endDateStr := q.Get("end_date")

	type DailyVoiceRow struct {
		Date         string `json:"date"`
		MemberID     string `json:"member_id"`
		TotalSeconds int64  `json:"total_seconds"`
		Hours        int64  `json:"hours"`
		Minutes      int64  `json:"minutes"`
	}

	query := m.db.Model(&database.VoiceTime{}).Where("date IS NOT NULL")
	if memberID != "" {
		query = query.Where("member_id = ?", memberID)
	}
	if dateStr != "" {
		d, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid date format, use YYYY-MM-DD"})
			return
		}
		query = query.Where("date::date = ?", d)
	} else if startDateStr != "" && endDateStr != "" {
		start, err := time.Parse("2006-01-02", startDateStr)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid start_date format, use YYYY-MM-DD"})
			return
		}
		end, err := time.Parse("2006-01-02", endDateStr)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid end_date format, use YYYY-MM-DD"})
			return
		}
		query = query.Where("date::date >= ? AND date::date <= ?", start, end)
	}

	var rows []DailyVoiceRow
	if err := query.Select(`to_char(date, 'YYYY-MM-DD') as date, member_id,
		COALESCE(SUM(duration), 0) as total_seconds,
		COALESCE(SUM(duration), 0) / 3600 as hours,
		(COALESCE(SUM(duration), 0) % 3600) / 60 as minutes`).
		Group("date::date, member_id").
		Order("date DESC").
		Scan(&rows).Error; err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "database error"})
		return
	}
	if rows == nil {
		rows = []DailyVoiceRow{}
	}
	_ = json.NewEncoder(w).Encode(rows)
}

func (m *Module) getModRoleID(s *discordgo.Session, guildID string) (string, error) {
	if m.cfg.ModRoleID != "" {
		return m.cfg.ModRoleID, nil
	}
	if m.cfg.ModRoleName == "" {
		return "", errors.New("mod role ID or name is not configured")
	}
	return resolveRoleIDByName(s, guildID, m.cfg.ModRoleName)
}

func memberHasModRole(member *discordgo.Member, roleID string) bool {
	for _, rid := range member.Roles {
		if rid == roleID {
			return true
		}
	}
	return false
}

func resolveRoleIDByName(s *discordgo.Session, guildID, roleName string) (string, error) {
	roles, err := s.GuildRoles(guildID)
	if err != nil {
		return "", err
	}
	for _, role := range roles {
		if strings.EqualFold(role.Name, roleName) {
			return role.ID, nil
		}
	}
	return "", errors.New("role not found by name")
}
