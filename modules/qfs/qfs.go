package qfs

import (
	"encoding/json"
	"errors"
	"fmt"
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
	s.AddHandler(m.onReactionRemove)
}

func (m *Module) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/qfs", m.handleChecks)
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
	isCheckmark := em == "✅" || em == "✔️" || em == "✔" || em == "☑️" || em == "☑"
	if !isCheckmark {
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
	isStaff := false
	for _, rid := range member.Roles {
		if rid == modRoleID || rid == m.cfg.ManagerRoleID || rid == m.cfg.DirectorRoleID {
			isStaff = true
			break
		}
	}
	if !isStaff {
		log.Printf("[CHECKMARK] Reactor is not staff (mod/manager/director): roles=%v", member.Roles)
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

func (m *Module) onReactionRemove(s *discordgo.Session, r *discordgo.MessageReactionRemove) {
	if m.cfg.QFSChannelID == "" || r.ChannelID != m.cfg.QFSChannelID {
		return
	}
	em := r.Emoji.Name
	isCheckmark := em == "✅" || em == "✔️" || em == "✔" || em == "☑️" || em == "☑"
	if !isCheckmark {
		return
	}

	var q database.Question
	err := m.db.Where("message_id = ?", r.MessageID).First(&q).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return
	}
	if err != nil {
		log.Printf("[CHECKMARK] Failed to query question for reaction remove: %v", err)
		return
	}

	now := time.Now().UTC()
	result := m.db.Model(&database.QuestionCheck{}).
		Where("question_id = ? AND member_id = ? AND removed_at IS NULL", q.ID, r.UserID).
		Update("removed_at", now)
	if result.Error != nil {
		log.Printf("[CHECKMARK] Failed to soft-delete question check: %v", result.Error)
	} else if result.RowsAffected > 0 {
		log.Printf("[CHECKMARK] Removed checkmark credit for %s on question %s", r.UserID, r.MessageID)
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

	// Build outer WHERE filters (applied after deduplication).
	// Each question is credited to the first mod who checked it (DISTINCT ON ordered by checked_at).
	// This prevents double-counting when two mods accidentally react to the same question.
	var whereParts []string
	var args []interface{}
	if memberID != "" {
		whereParts = append(whereParts, "fc.member_id = ?")
		args = append(args, memberID)
	}
	if dateStr != "" {
		d, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid date format, use YYYY-MM-DD"})
			return
		}
		whereParts = append(whereParts, "fc.date::date = ?")
		args = append(args, d)
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
		whereParts = append(whereParts, "fc.date::date >= ? AND fc.date::date <= ?")
		args = append(args, start, end)
	}

	outerWhere := "1=1"
	if len(whereParts) > 0 {
		outerWhere = strings.Join(whereParts, " AND ")
	}

	rawSQL := fmt.Sprintf(`
		WITH first_checks AS (
			SELECT DISTINCT ON (question_id) question_id, member_id, date
			FROM question_checks
			WHERE removed_at IS NULL AND date IS NOT NULL
			ORDER BY question_id, checked_at ASC
		)
		SELECT to_char(fc.date, 'YYYY-MM-DD') AS date, fc.member_id, count(*) AS count
		FROM first_checks fc
		WHERE %s
		GROUP BY to_char(fc.date, 'YYYY-MM-DD'), fc.member_id
		ORDER BY to_char(fc.date, 'YYYY-MM-DD') DESC
	`, outerWhere)

	var rows []DailyCheckRow
	if err := m.db.Raw(rawSQL, args...).Scan(&rows).Error; err != nil {
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
