package gameleads

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/brendanjhnsn/TrackerAPI/core/config"
	"github.com/brendanjhnsn/TrackerAPI/core/database"
	"github.com/brendanjhnsn/TrackerAPI/core/discordapi"
	"github.com/brendanjhnsn/TrackerAPI/modules/auth"
	"github.com/brendanjhnsn/TrackerAPI/modules/permissions"
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
	s.AddHandler(m.onVoiceStateUpdate)
}

func (m *Module) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/game-leads/channels", m.handleChannels)
	mux.HandleFunc("/api/game-leads", m.handleGameLeads)
	mux.HandleFunc("/api/game-lead-assignments", m.handleAssignments)
	mux.HandleFunc("/api/game-lead-messages", m.handleMessages)
	mux.HandleFunc("/api/game-lead-voice", m.handleVoice)
	mux.HandleFunc("/api/game-lead-notes", m.handleNotes)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (m *Module) requireSection(w http.ResponseWriter, r *http.Request, section string) (auth.Role, bool) {
	role, ok := auth.RoleFromContext(r.Context())
	if !ok || (role != auth.RoleManager && role != auth.RoleDirector) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "manager or director role required"})
		return 0, false
	}
	userID, _ := auth.UserIDFromContext(r.Context())
	if !permissions.CanAccess(m.db, role == auth.RoleDirector, userID, section) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden: missing " + section + " permission"})
		return 0, false
	}
	return role, true
}

func (m *Module) memberHasRole(s *discordgo.Session, guildID, userID, roleID string) bool {
	if roleID == "" {
		return false
	}
	member, err := s.GuildMember(guildID, userID)
	if err != nil {
		return false
	}
	for _, r := range member.Roles {
		if r == roleID {
			return true
		}
	}
	return false
}

// --- Discord event handlers (stubs — implemented in Task 5) ---

func (m *Module) onMessageCreate(s *discordgo.Session, msg *discordgo.MessageCreate) {}

func (m *Module) onVoiceStateUpdate(s *discordgo.Session, vs *discordgo.VoiceStateUpdate) {}

// --- Types for channel endpoints ---

type channelInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type discordChannel struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     int    `json:"type"`
	ParentID string `json:"parent_id"`
}

type assignmentRequest struct {
	UserID    string `json:"user_id"`
	ChannelID string `json:"channel_id"`
}

// --- HTTP handlers (stubs — implemented in Tasks 3, 4, 6) ---

func (m *Module) handleGameLeads(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if _, ok := m.requireSection(w, r, "game_leads"); !ok {
		return
	}
	if m.cfg.GameLeadRoleID == "" {
		writeJSON(w, http.StatusOK, []string{})
		return
	}
	ids, err := discordapi.ListMembersWithRole(
		r.Context(),
		&http.Client{Timeout: 10 * time.Second},
		discordapi.DefaultBaseURL,
		m.cfg.DiscordToken,
		m.cfg.DiscordGuildID,
		m.cfg.GameLeadRoleID,
		1000,
	)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "could not fetch game leads from Discord"})
		return
	}
	if ids == nil {
		ids = []string{}
	}
	writeJSON(w, http.StatusOK, ids)
}

func (m *Module) fetchChannels(ctx context.Context) ([]channelInfo, error) {
	url := fmt.Sprintf("%s/guilds/%s/channels", discordapi.DefaultBaseURL, m.cfg.DiscordGuildID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bot "+m.cfg.DiscordToken)
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discord API returned status %d", resp.StatusCode)
	}
	var channels []discordChannel
	if err := json.NewDecoder(resp.Body).Decode(&channels); err != nil {
		return nil, err
	}
	categorySet := map[string]bool{}
	for _, catID := range strings.Split(m.cfg.GameLeadCategoryID, ",") {
		catID = strings.TrimSpace(catID)
		if catID != "" {
			categorySet[catID] = true
		}
	}
	var result []channelInfo
	for _, ch := range channels {
		if ch.Type != 0 {
			continue
		}
		if len(categorySet) > 0 && !categorySet[ch.ParentID] {
			continue
		}
		result = append(result, channelInfo{ID: ch.ID, Name: ch.Name})
	}
	return result, nil
}

func (m *Module) handleChannels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if _, ok := m.requireSection(w, r, "game_leads"); !ok {
		return
	}
	channels, err := m.fetchChannels(r.Context())
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "could not fetch channels from Discord"})
		return
	}
	if channels == nil {
		channels = []channelInfo{}
	}
	writeJSON(w, http.StatusOK, channels)
}

func (m *Module) handleAssignments(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		m.getAssignments(w, r)
	case http.MethodPost:
		m.postAssignment(w, r)
	case http.MethodDelete:
		m.deleteAssignment(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (m *Module) getAssignments(w http.ResponseWriter, r *http.Request) {
	if _, ok := m.requireSection(w, r, "game_leads"); !ok {
		return
	}
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "user_id is required"})
		return
	}
	var assignments []database.GameLeadAssignment
	if err := m.db.Where("user_id = ?", userID).Find(&assignments).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "database error"})
		return
	}
	if assignments == nil {
		assignments = []database.GameLeadAssignment{}
	}
	writeJSON(w, http.StatusOK, assignments)
}

func (m *Module) postAssignment(w http.ResponseWriter, r *http.Request) {
	if _, ok := m.requireSection(w, r, "game_leads"); !ok {
		return
	}
	var req assignmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.UserID == "" || req.ChannelID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "user_id and channel_id are required"})
		return
	}
	assignment := database.GameLeadAssignment{UserID: req.UserID, ChannelID: req.ChannelID}
	if err := m.db.Create(&assignment).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create assignment"})
		return
	}
	writeJSON(w, http.StatusCreated, assignment)
}

func (m *Module) deleteAssignment(w http.ResponseWriter, r *http.Request) {
	if _, ok := m.requireSection(w, r, "game_leads"); !ok {
		return
	}
	userID := r.URL.Query().Get("user_id")
	channelID := r.URL.Query().Get("channel_id")
	if userID == "" || channelID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "user_id and channel_id are required"})
		return
	}
	result := m.db.Where("user_id = ? AND channel_id = ?", userID, channelID).Delete(&database.GameLeadAssignment{})
	if result.Error != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "database error"})
		return
	}
	if result.RowsAffected == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "assignment not found"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (m *Module) handleMessages(w http.ResponseWriter, r *http.Request) {}
func (m *Module) handleVoice(w http.ResponseWriter, r *http.Request)    {}
func (m *Module) handleNotes(w http.ResponseWriter, r *http.Request)    {}

// suppress unused import errors until handlers are implemented
var (
	_ = errors.New
	_ = log.Printf
)
