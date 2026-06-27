package auth

import (
	"context"
	"net/http"

	"github.com/brendanjhnsn/TrackerAPI/core/database"
)

type Role int

const (
	RoleMod      Role = iota
	RoleManager
	RoleDirector
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
func (m *Module) resolveRole(_ []string) (Role, bool) { return 0, false }
