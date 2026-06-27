package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/brendanjhnsn/TrackerAPI/core/config"
	"github.com/brendanjhnsn/TrackerAPI/core/database"
)

type fakeStore struct {
	sess *database.Session
	err  error
}

func (f *fakeStore) FindValid(_ string) (*database.Session, error) {
	return f.sess, f.err
}

type fakeFetcher struct {
	roles []string
	err   error
}

func (f *fakeFetcher) GetMemberRoles(_ context.Context, _ string) ([]string, error) {
	return f.roles, f.err
}

func newTestModule(modID, managerID, directorID string) *Module {
	return New(nil, &config.Config{
		ModRoleID:      modID,
		ManagerRoleID:  managerID,
		DirectorRoleID: directorID,
	})
}

func validSession() *database.Session {
	return &database.Session{
		Token:         "tok",
		DiscordUserID: "user123",
		ExpiresAt:     time.Now().Add(time.Hour),
	}
}

var okHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
})

func TestMiddleware_NoCookie_Returns401(t *testing.T) {
	m := newTestModule("mod", "mgr", "dir")
	h := m.middleware(okHandler, &fakeStore{}, &fakeFetcher{})
	req := httptest.NewRequest(http.MethodGet, "/api/tickets", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", rec.Code)
	}
}

func TestMiddleware_InvalidSession_Returns401(t *testing.T) {
	m := newTestModule("mod", "mgr", "dir")
	store := &fakeStore{err: errors.New("not found")}
	h := m.middleware(okHandler, store, &fakeFetcher{})
	req := httptest.NewRequest(http.MethodGet, "/api/tickets", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "badtoken"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", rec.Code)
	}
}

func TestMiddleware_UserNotInGuild_Returns403(t *testing.T) {
	m := newTestModule("mod", "mgr", "dir")
	fetcher := &fakeFetcher{err: ErrUserNotInGuild}
	h := m.middleware(okHandler, &fakeStore{sess: validSession()}, fetcher)
	req := httptest.NewRequest(http.MethodGet, "/api/tickets", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("want 403, got %d", rec.Code)
	}
}

func TestMiddleware_NoMatchingRole_Returns403(t *testing.T) {
	m := newTestModule("mod", "mgr", "dir")
	fetcher := &fakeFetcher{roles: []string{"some-other-role"}}
	h := m.middleware(okHandler, &fakeStore{sess: validSession()}, fetcher)
	req := httptest.NewRequest(http.MethodGet, "/api/tickets", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("want 403, got %d", rec.Code)
	}
}

func TestMiddleware_ModRole_PassesWithRoleInContext(t *testing.T) {
	m := newTestModule("mod", "mgr", "dir")
	fetcher := &fakeFetcher{roles: []string{"mod"}}
	var gotRole Role
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRole, _ = RoleFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})
	h := m.middleware(next, &fakeStore{sess: validSession()}, fetcher)
	req := httptest.NewRequest(http.MethodGet, "/api/tickets", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
	if gotRole != RoleMod {
		t.Errorf("want RoleMod, got %d", gotRole)
	}
}

func TestMiddleware_ManagerRole_PassesWithRoleInContext(t *testing.T) {
	m := newTestModule("mod", "mgr", "dir")
	fetcher := &fakeFetcher{roles: []string{"mgr"}}
	var gotRole Role
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRole, _ = RoleFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})
	h := m.middleware(next, &fakeStore{sess: validSession()}, fetcher)
	req := httptest.NewRequest(http.MethodGet, "/api/tickets", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
	if gotRole != RoleManager {
		t.Errorf("want RoleManager, got %d", gotRole)
	}
}

func TestMiddleware_DirectorRole_PassesWithRoleInContext(t *testing.T) {
	m := newTestModule("mod", "mgr", "dir")
	fetcher := &fakeFetcher{roles: []string{"dir"}}
	var gotRole Role
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRole, _ = RoleFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})
	h := m.middleware(next, &fakeStore{sess: validSession()}, fetcher)
	req := httptest.NewRequest(http.MethodGet, "/api/tickets", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
	if gotRole != RoleDirector {
		t.Errorf("want RoleDirector, got %d", gotRole)
	}
}

func TestMiddleware_DirectorTakesPriorityOverManager(t *testing.T) {
	m := newTestModule("mod", "mgr", "dir")
	fetcher := &fakeFetcher{roles: []string{"mgr", "dir"}}
	var gotRole Role
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRole, _ = RoleFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})
	h := m.middleware(next, &fakeStore{sess: validSession()}, fetcher)
	req := httptest.NewRequest(http.MethodGet, "/api/tickets", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if gotRole != RoleDirector {
		t.Errorf("want RoleDirector, got %d", gotRole)
	}
}

func TestMiddleware_LOAPassesWithoutAuth(t *testing.T) {
	m := newTestModule("mod", "mgr", "dir")
	store := &fakeStore{err: errors.New("not found")}
	h := m.middleware(okHandler, store, &fakeFetcher{})
	req := httptest.NewRequest(http.MethodGet, "/api/loa", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("want 200 (pass-through), got %d", rec.Code)
	}
}

func TestMiddleware_LOAPostRequiresAuth(t *testing.T) {
	m := newTestModule("mod", "mgr", "dir")
	store := &fakeStore{err: errors.New("not found")}
	h := m.middleware(okHandler, store, &fakeFetcher{})
	req := httptest.NewRequest(http.MethodPost, "/api/loa", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401 for unauthenticated POST /api/loa, got %d", rec.Code)
	}
}

func TestMiddleware_AuthRoutesPassWithoutAuth(t *testing.T) {
	m := newTestModule("mod", "mgr", "dir")
	store := &fakeStore{err: errors.New("not found")}
	h := m.middleware(okHandler, store, &fakeFetcher{})
	req := httptest.NewRequest(http.MethodGet, "/auth/discord/redirect", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("want 200 (pass-through), got %d", rec.Code)
	}
}
