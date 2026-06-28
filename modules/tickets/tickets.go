package tickets

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
	s.AddHandler(m.onChannelCreate)
	s.AddHandler(m.onMessageCreate)
}

func (m *Module) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/tickets", m.handleTickets)
}

func (m *Module) onChannelCreate(s *discordgo.Session, cc *discordgo.ChannelCreate) {
	if m.cfg.TicketCategoryID == "" || cc.ParentID != m.cfg.TicketCategoryID {
		return
	}
	if err := m.db.Create(&database.Ticket{
		GuildID:   cc.GuildID,
		ChannelID: cc.ID,
	}).Error; err != nil {
		log.Printf("[TICKETS] Failed to create ticket for channel %s: %v", cc.ID, err)
	}
}

func (m *Module) onMessageCreate(s *discordgo.Session, msg *discordgo.MessageCreate) {
	if msg.Author.Bot {
		return
	}
	member, err := s.GuildMember(msg.GuildID, msg.Author.ID)
	if err != nil {
		return
	}
	modRoleID, err := m.getModRoleID(s, msg.GuildID)
	if err != nil || !memberHasModRole(member, modRoleID) {
		return
	}
	m.trackFirstModResponse(msg.GuildID, msg.ChannelID, msg.Author.ID)
}

func (m *Module) trackFirstModResponse(guildID, channelID, modID string) {
	var ticket database.Ticket
	result := m.db.Where("guild_id = ? AND channel_id = ? AND first_resp_id IS NULL", guildID, channelID).First(&ticket)
	if result.Error == nil {
		now := time.Now().UTC()
		dateOnly := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		if err := m.db.Model(&ticket).Updates(map[string]interface{}{
			"first_resp_id":   modID,
			"first_resp_at":   now,
			"first_resp_date": &dateOnly,
		}).Error; err != nil {
			log.Printf("[TICKETS] Failed to update first responder for channel %s: %v", channelID, err)
		}
	}
}

func (m *Module) handleTickets(w http.ResponseWriter, r *http.Request) {
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

	type DailyTicketRow struct {
		Date     string `json:"date"`
		MemberID string `json:"member_id"`
		Tickets  int64  `json:"tickets"`
	}

	query := m.db.Model(&database.Ticket{}).
		Where("first_resp_id IS NOT NULL AND first_resp_at IS NOT NULL AND first_resp_date IS NOT NULL")

	if memberID != "" {
		query = query.Where("first_resp_id = ?", memberID)
	}
	if dateStr != "" {
		d, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid date format, use YYYY-MM-DD"})
			return
		}
		query = query.Where("first_resp_date::date = ?", d)
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
		query = query.Where("first_resp_date::date >= ? AND first_resp_date::date <= ?", start, end)
	}

	var rows []DailyTicketRow
	if err := query.Select("to_char(first_resp_date, 'YYYY-MM-DD') as date, first_resp_id as member_id, count(*) as tickets").
		Group("to_char(first_resp_date, 'YYYY-MM-DD'), first_resp_id").
		Order("to_char(first_resp_date, 'YYYY-MM-DD') DESC").
		Scan(&rows).Error; err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "database error"})
		return
	}
	if rows == nil {
		rows = []DailyTicketRow{}
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
