package qfs

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
	s.AddHandler(m.onReactionAdd)
}

func (m *Module) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/checks", m.handleChecks)
}

func (m *Module) onMessageCreate(s *discordgo.Session, msg *discordgo.MessageCreate) {
	if msg.Author.Bot {
		return
	}
	if m.cfg.QFSChannelID == "" || msg.ChannelID != m.cfg.QFSChannelID {
		return
	}
	if msg.MessageReference != nil && msg.MessageReference.MessageID != "" {
		return
	}
	member, err := s.GuildMember(msg.GuildID, msg.Author.ID)
	if err != nil {
		return
	}
	modRoleID, err := m.getModRoleID(s, msg.GuildID)
	if err == nil && memberHasModRole(member, modRoleID) {
		return
	}

	log.Printf("[QFS] New message in questions channel: %s (Author: %s)", msg.ID, msg.Author.Username)

	var q database.Question
	err = m.db.Where("message_id = ?", msg.ID).First(&q).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		q = database.Question{
			GuildID:   msg.GuildID,
			ChannelID: msg.ChannelID,
			MessageID: msg.ID,
			Title:     truncate(msg.Content, 255),
		}
		if createErr := m.db.Create(&q).Error; createErr != nil {
			log.Printf("[ERROR] Failed to create question: %v", createErr)
		} else {
			log.Printf("[QFS] Created question record for message %s", msg.ID)
		}
	} else if err != nil {
		log.Printf("[QFS] Failed to query question for message %s: %v", msg.ID, err)
	}
}

func (m *Module) onReactionAdd(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	log.Printf("[CHECKMARK] Reaction event: message=%s channel=%s user=%s emoji=%s",
		r.MessageID, r.ChannelID, r.UserID, r.Emoji.Name)

	if m.cfg.QFSChannelID == "" || r.ChannelID != m.cfg.QFSChannelID {
		log.Printf("[CHECKMARK] Ignored because not in QFS channel: %s", r.ChannelID)
		return
	}
	em := r.Emoji.Name
	if em != "✅" && em != "☑️" {
		log.Printf("[CHECKMARK] Ignored because emoji is not a checkmark: %s", em)
		return
	}

	member, err := s.GuildMember(r.GuildID, r.UserID)
	if err != nil {
		log.Printf("[CHECKMARK] Failed to fetch guild member: %v", err)
		return
	}
	modRoleID := m.cfg.ModRoleID
	if modRoleID == "" && m.cfg.ModRoleName != "" {
		resolved, resolveErr := resolveRoleIDByName(s, r.GuildID, m.cfg.ModRoleName)
		if resolveErr != nil {
			log.Printf("[CHECKMARK] Failed to resolve mod role by name: %v", resolveErr)
		} else {
			modRoleID = resolved
			log.Printf("[CHECKMARK] Resolved mod role name %s to ID %s", m.cfg.ModRoleName, modRoleID)
		}
	}
	isMod := false
	for _, rid := range member.Roles {
		if rid == modRoleID {
			isMod = true
			break
		}
	}
	if !isMod {
		log.Printf("[CHECKMARK] Reactor is not mod (role %s missing): roles=%v", modRoleID, member.Roles)
		return
	}

	var q database.Question
	err = m.db.Where("message_id = ?", r.MessageID).First(&q).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		log.Printf("[CHECKMARK] Question not found, creating: %s", r.MessageID)
		fetchedMsg, _ := s.ChannelMessage(r.ChannelID, r.MessageID)
		title := ""
		if fetchedMsg != nil {
			title = truncate(fetchedMsg.Content, 255)
		}
		q = database.Question{
			GuildID:   r.GuildID,
			ChannelID: r.ChannelID,
			MessageID: r.MessageID,
			Title:     title,
		}
		if createErr := m.db.Create(&q).Error; createErr != nil {
			log.Printf("[ERROR] Failed to create question in reaction handler: %v", createErr)
			return
		}
		log.Printf("[CHECKMARK] Created question %s in reaction handler", r.MessageID)
	}

	var existing database.QuestionCheck
	err = m.db.Where("question_id = ? AND member_id = ? AND removed_at IS NULL", q.ID, r.UserID).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		now := time.Now().UTC()
		dateOnly := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		qc := database.QuestionCheck{
			QuestionID: q.ID,
			RoleID:     modRoleID,
			MemberID:   r.UserID,
			Date:       &dateOnly,
			CheckedAt:  now,
		}
		if createErr := m.db.Create(&qc).Error; createErr != nil {
			log.Printf("[ERROR] Failed to create question check: %v", createErr)
		} else {
			log.Printf("[CHECKMARK] Recorded checkmark from %s on question %s", r.UserID, r.MessageID)
		}
	} else if err != nil {
		log.Printf("[ERROR] Failed to query existing question check: %v", err)
	} else {
		log.Printf("[CHECKMARK] Duplicate check prevented for member %s on question %s", r.UserID, r.MessageID)
	}
}

func (m *Module) handleChecks(w http.ResponseWriter, r *http.Request) {
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

	type DailyCheckRow struct {
		Date     string `json:"date"`
		MemberID string `json:"member_id"`
		Count    int64  `json:"count"`
	}

	query := m.db.Model(&database.QuestionCheck{}).Where("date IS NOT NULL")
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

	var rows []DailyCheckRow
	if err := query.Select("to_char(date, 'YYYY-MM-DD') as date, member_id, count(*) as count").
		Group("date::date, member_id").
		Order("date DESC").
		Scan(&rows).Error; err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "database error"})
		return
	}
	if rows == nil {
		rows = []DailyCheckRow{}
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

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
