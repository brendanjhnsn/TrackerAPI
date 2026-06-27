package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/brendanjhnsn/TrackerAPI/core/database"
	"gorm.io/gorm"
)

type Role int

const (
	RoleNone     Role = iota // zero value — context key absent
	RoleMod                  // read-only
	RoleManager              // read + write
	RoleDirector             // read + write
)

type contextKey struct{}

// RoleFromContext retrieves the Role attached by the auth middleware.
// Returns (RoleNone, false) if no role is present.
func RoleFromContext(ctx context.Context) (Role, bool) {
	r, ok := ctx.Value(contextKey{}).(Role)
	return r, ok
}

// SessionStore abstracts session lookup for testability.
type SessionStore interface {
	FindValid(token string) (*database.Session, error)
}

type gormSessionStore struct {
	db *gorm.DB
}

func (s *gormSessionStore) FindValid(token string) (*database.Session, error) {
	var sess database.Session
	err := s.db.Where("token = ? AND expires_at > ?", token, time.Now().UTC()).First(&sess).Error
	if err != nil {
		s.db.Where("token = ?", token).Delete(&database.Session{})
		return nil, err
	}
	return &sess, nil
}

// RoleFetcher retrieves a guild member's role IDs from Discord.
type RoleFetcher interface {
	GetMemberRoles(userID string) ([]string, error)
}

type discordRoleFetcher struct {
	botToken string
	guildID  string
}

func newDiscordRoleFetcher(botToken, guildID string) RoleFetcher {
	return &discordRoleFetcher{botToken: botToken, guildID: guildID}
}

type guildMember struct {
	Roles []string `json:"roles"`
}

func (d *discordRoleFetcher) GetMemberRoles(userID string) ([]string, error) {
	url := fmt.Sprintf("https://discord.com/api/guilds/%s/members/%s", d.guildID, userID)
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("Authorization", "Bot "+d.botToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("user not in guild")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discord API error: %d", resp.StatusCode)
	}
	var member guildMember
	if err := json.NewDecoder(resp.Body).Decode(&member); err != nil {
		return nil, err
	}
	return member.Roles, nil
}

// Middleware returns an http.Handler that enforces auth on all /api/* routes except /api/loa.
func (m *Module) Middleware(next http.Handler) http.Handler {
	return m.middleware(
		next,
		&gormSessionStore{m.db},
		newDiscordRoleFetcher(m.cfg.DiscordToken, m.cfg.DiscordGuildID),
	)
}

func (m *Module) middleware(next http.Handler, store SessionStore, fetcher RoleFetcher) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasPrefix(path, "/auth/") || path == "/api/loa" {
			next.ServeHTTP(w, r)
			return
		}
		if !strings.HasPrefix(path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}

		cookie, err := r.Cookie("session")
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		sess, err := store.FindValid(cookie.Value)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		roles, err := fetcher.GetMemberRoles(sess.DiscordUserID)
		if err != nil {
			writeJSON(w, http.StatusForbidden, "forbidden")
			return
		}

		role, ok := resolveRole(roles, m.cfg.ModRoleID, m.cfg.ManagerRoleID, m.cfg.DirectorRoleID)
		if !ok {
			writeJSON(w, http.StatusForbidden, "forbidden")
			return
		}

		ctx := context.WithValue(r.Context(), contextKey{}, role)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func resolveRole(discordRoles []string, modID, managerID, directorID string) (Role, bool) {
	roleSet := make(map[string]bool, len(discordRoles))
	for _, r := range discordRoles {
		roleSet[r] = true
	}
	switch {
	case roleSet[directorID]:
		return RoleDirector, true
	case roleSet[managerID]:
		return RoleManager, true
	case roleSet[modID]:
		return RoleMod, true
	default:
		return RoleNone, false
	}
}

func writeJSON(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
