package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/brendanjhnsn/TrackerAPI/core/config"
)

func TestNew_ReturnsNonNil(t *testing.T) {
	if New(nil, nil) == nil {
		t.Fatal("New returned nil")
	}
}

func TestHandleRedirect_SetsStateCookieAndRedirects(t *testing.T) {
	m := New(nil, &config.Config{
		DiscordClientID:    "client123",
		DiscordRedirectURI: "http://localhost:8080/auth/discord/callback",
		Environment:        "development",
	})
	mux := http.NewServeMux()
	m.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/auth/discord/redirect", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusTemporaryRedirect {
		t.Errorf("want 307, got %d", rec.Code)
	}
	var stateCookie *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == "oauth_state" {
			stateCookie = c
		}
	}
	if stateCookie == nil {
		t.Fatal("oauth_state cookie not set")
	}
	if len(stateCookie.Value) != 32 {
		t.Errorf("want 32-char state (16 random bytes hex-encoded), got len %d", len(stateCookie.Value))
	}
	if !stateCookie.HttpOnly {
		t.Error("oauth_state cookie must be HttpOnly")
	}
	if rec.Header().Get("Location") == "" {
		t.Error("no Location header in redirect")
	}
}

func TestHandleCallback_StateMismatch_Returns400(t *testing.T) {
	m := New(nil, &config.Config{})
	mux := http.NewServeMux()
	m.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/auth/discord/callback?state=wrong&code=abc", nil)
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: "correct"})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestHandleCallback_MissingStateCookie_Returns400(t *testing.T) {
	m := New(nil, &config.Config{})
	mux := http.NewServeMux()
	m.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/auth/discord/callback?state=abc&code=xyz", nil)
	// no oauth_state cookie set
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestHandleCallback_MissingCode_Returns400(t *testing.T) {
	m := New(nil, &config.Config{})
	mux := http.NewServeMux()
	m.RegisterRoutes(mux)

	// state matches but no code param
	req := httptest.NewRequest(http.MethodGet, "/auth/discord/callback?state=abc", nil)
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: "abc"})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestHandleMe_NoRoleInContext_Returns401(t *testing.T) {
	m := newTestModule("mod", "mgr", "dir")
	mux := http.NewServeMux()
	m.RegisterRoutes(mux)
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", rec.Code)
	}
}

func TestHandleMe_WithManagerRole_ReturnsManagerAndUserID(t *testing.T) {
	m := newTestModule("mod", "mgr", "dir")
	mux := http.NewServeMux()
	m.RegisterRoutes(mux)
	ctx := context.WithValue(context.Background(), contextKey{}, RoleManager)
	ctx = context.WithValue(ctx, userIDKey{}, "user456")
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["role"] != "manager" {
		t.Errorf("want role=manager, got %q", body["role"])
	}
	if body["discord_user_id"] != "user456" {
		t.Errorf("want discord_user_id=user456, got %q", body["discord_user_id"])
	}
}

func TestHandleMe_WithDirectorRole_ReturnsDirector(t *testing.T) {
	m := newTestModule("mod", "mgr", "dir")
	mux := http.NewServeMux()
	m.RegisterRoutes(mux)
	ctx := context.WithValue(context.Background(), contextKey{}, RoleDirector)
	ctx = context.WithValue(ctx, userIDKey{}, "user789")
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["role"] != "director" {
		t.Errorf("want role=director, got %q", body["role"])
	}
}

func TestHandleMe_WithModRole_ReturnsMod(t *testing.T) {
	m := newTestModule("mod", "mgr", "dir")
	mux := http.NewServeMux()
	m.RegisterRoutes(mux)
	ctx := context.WithValue(context.Background(), contextKey{}, RoleMod)
	ctx = context.WithValue(ctx, userIDKey{}, "user101")
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["role"] != "mod" {
		t.Errorf("want role=mod, got %q", body["role"])
	}
}
