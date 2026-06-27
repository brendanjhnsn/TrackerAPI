package auth

import (
	"context"
	"net/http"

	"github.com/brendanjhnsn/TrackerAPI/core/database"
)

type Role int

const (
	RoleNone     Role = iota // zero value — context key absent
	RoleMod                  // read-only
	RoleManager              // read + write
	RoleDirector             // read + write
)

type contextKey struct{}

func RoleFromContext(ctx context.Context) (Role, bool) { return 0, false }

type SessionStore interface {
	FindValid(token string) (*database.Session, error)
}

type RoleFetcher interface {
	GetMemberRoles(userID string) ([]string, error)
}

func (m *Module) Middleware(next http.Handler) http.Handler { return next }
func (m *Module) middleware(next http.Handler, _ SessionStore, _ RoleFetcher) http.Handler {
	return next
}
func resolveRole(_ []string, _, _, _ string) (Role, bool) { return 0, false }
