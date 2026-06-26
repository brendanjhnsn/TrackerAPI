package dailymessages

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
	s.AddHandler(m.onMessageCreate)
}

func (m *Module) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/messages", m.handleMessages)
}

func (m *Module) onMessageCreate(s *discordgo.Session, msg *discordgo.MessageCreate) {
	if msg.Author.Bot {
		return
	}
	member, err := s.GuildMember(msg.GuildID, msg.Author.ID)
	if err != nil {
		return
	}
	modRoleID, roleErr := m.getModRoleID(s, msg.GuildID)
	if roleErr != nil || !memberHasModRole(member, modRoleID) {
		return
	}
	if m.cfg.TicketCategoryID != "" && msg.ChannelID == m.cfg.TicketCategoryID {
		return
	}
	if m.cfg.AdminCategoryID != "" && msg.ChannelID == m.cfg.AdminCategoryID {
		return
	}

	today := time.Now().UTC()
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	var dm database.DailyMessage
	result := m.db.Where("guild_id = ? AND member_id = ? AND date = ?", msg.GuildID, msg.Author.ID, today).First(&dm)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		if err := m.db.Create(&database.DailyMessage{
			GuildID:  msg.GuildID,
			MemberID: msg.Author.ID,
			Date:     today,
			Count:    1,
		}).Error; err != nil {
			log.Printf("[DAILYMESSAGES] Failed to create daily message record: %v", err)
		}
	} else if result.Error == nil {
		if err := m.db.Model(&dm).Update("count", dm.Count+1).Error; err != nil {
			log.Printf("[DAILYMESSAGES] Failed to update daily message count: %v", err)
		}
	}
}

func (m *Module) handleMessages(w http.ResponseWriter, r *http.Request) {
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

	type DailyMessageRow struct {
		Date     string `json:"date"`
		MemberID string `json:"member_id"`
		Count    int    `json:"count"`
	}

	query := m.db.Model(&database.DailyMessage{})
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
		query = query.Where("date = ?", d)
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
		query = query.Where("date >= ? AND date <= ?", start, end)
	}

	var rows []DailyMessageRow
	if err := query.Select("date, member_id, count as count").
		Order("date DESC").
		Scan(&rows).Error; err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "database error"})
		return
	}
	if rows == nil {
		rows = []DailyMessageRow{}
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
