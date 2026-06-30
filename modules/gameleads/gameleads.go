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

// --- HTTP handlers (stubs — implemented in Tasks 3, 4, 6) ---

func (m *Module) handleGameLeads(w http.ResponseWriter, r *http.Request)   {}
func (m *Module) handleChannels(w http.ResponseWriter, r *http.Request)    {}
func (m *Module) handleAssignments(w http.ResponseWriter, r *http.Request) {}
func (m *Module) handleMessages(w http.ResponseWriter, r *http.Request)    {}
func (m *Module) handleVoice(w http.ResponseWriter, r *http.Request)       {}
func (m *Module) handleNotes(w http.ResponseWriter, r *http.Request)       {}

// suppress unused import errors until handlers are implemented
var (
	_ = context.Background
	_ = errors.New
	_ = fmt.Sprintf
	_ = log.Printf
	_ = strings.Split
	_ = time.Now
	_ = discordapi.DefaultBaseURL
	_ = database.GameLeadAssignment{}
)
