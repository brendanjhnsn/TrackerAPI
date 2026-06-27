package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

// ErrUserNotInGuild is returned by RoleFetcher when the user is not a member of the configured guild.
var ErrUserNotInGuild = errors.New("user not in guild")

type contextKey struct{}

// RoleFromContext retrieves the Role attached by the auth middleware.
// Returns (RoleNone, false) if no role is present.
func RoleFromContext(ctx context.Context) (Role, bool) {
	r, ok := ctx.Value(contextKey{}).(Role)
	return r, ok
}

type userIDKey struct{}

// UserIDFromContext retrieves the Discord user ID attached by the auth middleware.
func UserIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(userIDKey{}).(string)
	return id, ok
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.db.Where("token = ?", token).Delete(&database.Session{})
		}
		return nil, err
	}
	return &sess, nil
}

// RoleFetcher retrieves a guild member's role IDs from Discord.
type RoleFetcher interface {
	GetMemberRoles(ctx context.Context, userID string) ([]string, error)
}

type discordRoleFetcher struct {
	botToken string
	guildID  string
	client   *http.Client
}

func newDiscordRoleFetcher(botToken, guildID string) RoleFetcher {
	return &discordRoleFetcher{
		botToken: botToken,
		guildID:  guildID,
		client:   &http.Client{Timeout: 5 * time.Second},
	}
}

type guildMember struct {
	Roles []string `json:"roles"`
}

func (d *discordRoleFetcher) GetMemberRoles(ctx context.Context, userID string) ([]string, error) {
	url := fmt.Sprintf("https://discord.com/api/guilds/%s/members/%s", d.guildID, userID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("building discord request: %w", err)
	}
	req.Header.Set("Authorization", "Bot "+d.botToken)
	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, ErrUserNotInGuild
	}
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
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
		isPublicLOA := path == "/api/loa" && r.Method == http.MethodGet && r.URL.Query().Get("all") == ""
		if !strings.HasPrefix(path, "/api/") || isPublicLOA {
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

		roles, err := fetcher.GetMemberRoles(r.Context(), sess.DiscordUserID)
		if err != nil {
			if errors.Is(err, ErrUserNotInGuild) {
				writeJSON(w, http.StatusForbidden, "forbidden")
			} else {
				writeJSON(w, http.StatusServiceUnavailable, "service unavailable")
			}
			return
		}

		role, ok := resolveRole(roles, m.cfg.ModRoleID, m.cfg.ManagerRoleID, m.cfg.DirectorRoleID)
		if !ok {
			writeJSON(w, http.StatusForbidden, "forbidden")
			return
		}

		if path == "/api/loa" && r.Method == http.MethodGet && r.URL.Query().Get("all") == "true" {
			if role != RoleManager && role != RoleDirector {
				writeJSON(w, http.StatusForbidden, "forbidden")
				return
			}
		}
		ctx := context.WithValue(r.Context(), contextKey{}, role)
		ctx = context.WithValue(ctx, userIDKey{}, sess.DiscordUserID)
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
