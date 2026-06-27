# Discord Auth + Middleware Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add Discord OAuth2 login and per-request role-based middleware to the Tracker API.

**Architecture:** A new `modules/auth` Go module provides OAuth handlers and HTTP middleware. Sessions are stored in Postgres. On every protected request, the middleware calls Discord's guild member API (using the existing bot token) to resolve the user's role in real time — Mod (read-only), Manager (read+write), or Director (read+write). All `/api/*` routes except `/api/loa` require a valid session and a recognized role.

**Tech Stack:** Go 1.22, `net/http` stdlib, GORM, Postgres, Discord OAuth2 + bot API

---

## File Map

| Action | File | Responsibility |
|---|---|---|
| Modify | `core/config/config.go` | Add 7 new env var fields |
| Modify | `core/database/models.go` | Add `Session` GORM model |
| Modify | `core/database/database.go` | Add `Session` to AutoMigrate (both Postgres + MySQL paths) |
| Create | `modules/auth/go.mod` | Auth sub-module definition |
| Create | `modules/auth/auth.go` | `Module` struct, `RegisterRoutes`, OAuth redirect/callback handlers |
| Create | `modules/auth/auth_test.go` | Tests for OAuth handlers |
| Create | `modules/auth/middleware.go` | `Role` type, `SessionStore`/`RoleFetcher` interfaces, `Middleware()` |
| Create | `modules/auth/middleware_test.go` | Tests for middleware |
| Modify | `go.mod` (root) | Add auth module to `require` and `replace` |
| Modify | `API/main.go` | Wire auth module, wrap mux with `Middleware()` |

> **GOCACHE note:** This system has a stale GOCACHE path. Prefix all `go` commands run in PowerShell with `$env:GOCACHE="C:\Users\brend\AppData\Local\go-build";`

---

## Task 1: Config fields + Session model

**Files:**
- Modify: `core/config/config.go`
- Modify: `core/database/models.go`
- Modify: `core/database/database.go`

- [ ] **Step 1: Add new fields to Config struct and Load()**

In `core/config/config.go`, add to the `Config` struct after `AdminCategoryID`:
```go
DiscordClientID     string
DiscordClientSecret string
DiscordRedirectURI  string
DiscordGuildID      string
ManagerRoleID       string
DirectorRoleID      string
FrontendURL         string
```

In `Load()`, add after the `AdminCategoryID` entry:
```go
DiscordClientID:     getEnv("DISCORD_CLIENT_ID", ""),
DiscordClientSecret: getEnv("DISCORD_CLIENT_SECRET", ""),
DiscordRedirectURI:  getEnv("DISCORD_REDIRECT_URI", ""),
DiscordGuildID:      getEnv("DISCORD_GUILD_ID", ""),
ManagerRoleID:       getEnv("MANAGER_ROLE_ID", ""),
DirectorRoleID:      getEnv("DIRECTOR_ROLE_ID", ""),
FrontendURL:         getEnv("FRONTEND_URL", "http://localhost:3000"),
```

- [ ] **Step 2: Add Session model to `core/database/models.go`**

Append after the `LOA` struct:
```go
type Session struct {
	Token         string    `gorm:"primaryKey"`
	DiscordUserID string    `gorm:"not null;index"`
	ExpiresAt     time.Time `gorm:"not null"`
	CreatedAt     time.Time
}
```

- [ ] **Step 3: Add Session to AutoMigrate in `core/database/database.go`**

Line 43 (Postgres path) — change:
```go
if err := db.AutoMigrate(&Question{}, &QuestionCheck{}, &Ticket{}, &DailyMessage{}, &VoiceTime{}, &LOA{}); err != nil {
```
To:
```go
if err := db.AutoMigrate(&Question{}, &QuestionCheck{}, &Ticket{}, &DailyMessage{}, &VoiceTime{}, &LOA{}, &Session{}); err != nil {
```

Line 89 (MySQL path) — make the same change.

- [ ] **Step 4: Verify core compiles**

```powershell
$env:GOCACHE="C:\Users\brend\AppData\Local\go-build"; go build ./...
```
Run from: `c:\Users\brend\Desktop\Tracker\core`
Expected: no output (success)

- [ ] **Step 5: Commit**

```
git add core/config/config.go core/database/models.go core/database/database.go
git commit -m "feat: add Session model and auth config fields"
```

---

## Task 2: Create auth module scaffold

**Files:**
- Create: `modules/auth/go.mod`
- Create: `modules/auth/auth.go` (type stubs only — no implementation yet)
- Create: `modules/auth/middleware.go` (type stubs only — no implementation yet)
- Modify: `go.mod` (root)

The stubs let `go mod tidy` run and let tests be written before implementation.

- [ ] **Step 1: Create `modules/auth/go.mod`**

```
module github.com/brendanjhnsn/TrackerAPI/modules/auth

go 1.22

require (
	github.com/brendanjhnsn/TrackerAPI/core v0.0.0
	gorm.io/gorm v1.31.1
)

require (
	github.com/go-sql-driver/mysql v1.7.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.6.0 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	golang.org/x/crypto v0.31.0 // indirect
	golang.org/x/sync v0.10.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	gorm.io/driver/mysql v1.5.7 // indirect
	gorm.io/driver/postgres v1.6.0 // indirect
)

replace github.com/brendanjhnsn/TrackerAPI/core => ../../core
```

- [ ] **Step 2: Create stub `modules/auth/auth.go`**

```go
package auth

import (
	"net/http"

	"github.com/brendanjhnsn/TrackerAPI/core/config"
	"gorm.io/gorm"
)

type Module struct {
	db  *gorm.DB
	cfg *config.Config
}

func New(db *gorm.DB, cfg *config.Config) *Module {
	return &Module{db: db, cfg: cfg}
}

func (m *Module) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/auth/discord/redirect", m.handleRedirect)
	mux.HandleFunc("/auth/discord/callback", m.handleCallback)
}

func (m *Module) handleRedirect(w http.ResponseWriter, r *http.Request) {}
func (m *Module) handleCallback(w http.ResponseWriter, r *http.Request) {}
```

- [ ] **Step 3: Create stub `modules/auth/middleware.go`**

```go
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

func (m *Module) Middleware(next http.Handler) http.Handler               { return next }
func (m *Module) middleware(next http.Handler, _ SessionStore, _ RoleFetcher) http.Handler {
	return next
}
func (m *Module) resolveRole(_ []string) (Role, bool) { return 0, false }
```

- [ ] **Step 4: Update root `go.mod`**

In `go.mod` (root), add to the `require` block:
```
github.com/brendanjhnsn/TrackerAPI/modules/auth v0.0.0
```

Add to the `replace` block:
```
github.com/brendanjhnsn/TrackerAPI/modules/auth => ./modules/auth
```

- [ ] **Step 5: Run `go mod tidy` in the auth module**

```powershell
cd "c:\Users\brend\Desktop\Tracker\modules\auth"
$env:GOCACHE="C:\Users\brend\AppData\Local\go-build"; go mod tidy
```
Expected: creates `modules/auth/go.sum`, no errors.

- [ ] **Step 6: Run `go mod tidy` in root**

```powershell
cd "c:\Users\brend\Desktop\Tracker"
$env:GOCACHE="C:\Users\brend\AppData\Local\go-build"; go mod tidy
```
Expected: updates root `go.sum`, no errors.

- [ ] **Step 7: Commit**

```
git add modules/auth/ go.mod go.sum
git commit -m "feat: scaffold auth module with type stubs"
```

---

## Task 3: Implement middleware (TDD)

**Files:**
- Create: `modules/auth/middleware_test.go`
- Modify: `modules/auth/middleware.go`

Tests live in `package auth` (same package) to access the unexported `middleware()` helper.

- [ ] **Step 1: Create `modules/auth/middleware_test.go`**

```go
package auth

import (
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

func (f *fakeFetcher) GetMemberRoles(_ string) ([]string, error) {
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
	fetcher := &fakeFetcher{err: errors.New("user not in guild")}
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
		t.Errorf("want RoleMod(%d), got %d", RoleMod, gotRole)
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
		t.Errorf("want RoleManager(%d), got %d", RoleManager, gotRole)
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
		t.Errorf("want RoleDirector(%d), got %d", RoleDirector, gotRole)
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
		t.Errorf("want RoleDirector(%d), got %d", RoleDirector, gotRole)
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
```

- [ ] **Step 2: Run tests — verify they fail**

```powershell
cd "c:\Users\brend\Desktop\Tracker\modules\auth"
$env:GOCACHE="C:\Users\brend\AppData\Local\go-build"; go test ./... -v -run TestMiddleware
```
Expected: most tests FAIL (middleware stubs return `next` unguarded)

- [ ] **Step 3: Replace `modules/auth/middleware.go` with full implementation**

```go
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
	RoleMod      Role = iota
	RoleManager
	RoleDirector
)

type contextKey struct{}

func RoleFromContext(ctx context.Context) (Role, bool) {
	r, ok := ctx.Value(contextKey{}).(Role)
	return r, ok
}

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

		role, ok := m.resolveRole(roles)
		if !ok {
			writeJSON(w, http.StatusForbidden, "forbidden")
			return
		}

		ctx := context.WithValue(r.Context(), contextKey{}, role)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *Module) resolveRole(discordRoles []string) (Role, bool) {
	roleSet := make(map[string]bool, len(discordRoles))
	for _, r := range discordRoles {
		roleSet[r] = true
	}
	switch {
	case roleSet[m.cfg.DirectorRoleID]:
		return RoleDirector, true
	case roleSet[m.cfg.ManagerRoleID]:
		return RoleManager, true
	case roleSet[m.cfg.ModRoleID]:
		return RoleMod, true
	default:
		return 0, false
	}
}

func writeJSON(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
```

- [ ] **Step 4: Run tests — verify they pass**

```powershell
cd "c:\Users\brend\Desktop\Tracker\modules\auth"
$env:GOCACHE="C:\Users\brend\AppData\Local\go-build"; go test ./... -v -run TestMiddleware
```
Expected: all `TestMiddleware_*` tests PASS

- [ ] **Step 5: Commit**

```
git add modules/auth/middleware.go modules/auth/middleware_test.go
git commit -m "feat: implement auth middleware with role resolution"
```

---

## Task 4: Implement OAuth handlers (TDD)

**Files:**
- Create: `modules/auth/auth_test.go`
- Modify: `modules/auth/auth.go`

- [ ] **Step 1: Create `modules/auth/auth_test.go`**

```go
package auth

import (
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
	// no oauth_state cookie
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
```

- [ ] **Step 2: Run tests — verify they fail**

```powershell
cd "c:\Users\brend\Desktop\Tracker\modules\auth"
$env:GOCACHE="C:\Users\brend\AppData\Local\go-build"; go test ./... -v -run "TestNew|TestHandleRedirect|TestHandleCallback"
```
Expected: tests FAIL (stub handlers do nothing, return 200 instead of 307/400)

- [ ] **Step 3: Replace `modules/auth/auth.go` with full implementation**

```go
package auth

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/brendanjhnsn/TrackerAPI/core/config"
	"github.com/brendanjhnsn/TrackerAPI/core/database"
	"gorm.io/gorm"
)

type Module struct {
	db  *gorm.DB
	cfg *config.Config
}

func New(db *gorm.DB, cfg *config.Config) *Module {
	return &Module{db: db, cfg: cfg}
}

func (m *Module) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/auth/discord/redirect", m.handleRedirect)
	mux.HandleFunc("/auth/discord/callback", m.handleCallback)
}

func (m *Module) handleRedirect(w http.ResponseWriter, r *http.Request) {
	state, err := randomHex(16)
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		MaxAge:   300,
		HttpOnly: true,
		Secure:   m.cfg.Environment == "production",
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	})
	authURL := fmt.Sprintf(
		"https://discord.com/oauth2/authorize?client_id=%s&redirect_uri=%s&response_type=code&scope=identify&state=%s",
		m.cfg.DiscordClientID,
		url.QueryEscape(m.cfg.DiscordRedirectURI),
		state,
	)
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

func (m *Module) handleCallback(w http.ResponseWriter, r *http.Request) {
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil || stateCookie.Value != r.URL.Query().Get("state") {
		http.Error(w, `{"error":"invalid state"}`, http.StatusBadRequest)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:   "oauth_state",
		Value:  "",
		MaxAge: -1,
		Path:   "/",
	})

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, `{"error":"missing code"}`, http.StatusBadRequest)
		return
	}

	accessToken, err := m.exchangeCode(code)
	if err != nil {
		http.Error(w, `{"error":"token exchange failed"}`, http.StatusInternalServerError)
		return
	}

	userID, err := m.getDiscordUserID(accessToken)
	if err != nil {
		http.Error(w, `{"error":"failed to get user info"}`, http.StatusInternalServerError)
		return
	}

	sessionToken, err := randomHex(32)
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	sess := database.Session{
		Token:         sessionToken,
		DiscordUserID: userID,
		ExpiresAt:     time.Now().UTC().Add(24 * time.Hour),
	}
	if err := m.db.Create(&sess).Error; err != nil {
		http.Error(w, `{"error":"failed to create session"}`, http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    sessionToken,
		MaxAge:   86400,
		HttpOnly: true,
		Secure:   m.cfg.Environment == "production",
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	})
	http.Redirect(w, r, m.cfg.FrontendURL, http.StatusTemporaryRedirect)
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
}

func (m *Module) exchangeCode(code string) (string, error) {
	resp, err := http.PostForm("https://discord.com/api/oauth2/token", url.Values{
		"client_id":     {m.cfg.DiscordClientID},
		"client_secret": {m.cfg.DiscordClientSecret},
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {m.cfg.DiscordRedirectURI},
	})
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var tok tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return "", err
	}
	if tok.AccessToken == "" {
		return "", fmt.Errorf("empty access token from Discord")
	}
	return tok.AccessToken, nil
}

type discordUser struct {
	ID string `json:"id"`
}

func (m *Module) getDiscordUserID(accessToken string) (string, error) {
	req, _ := http.NewRequest(http.MethodGet, "https://discord.com/api/users/@me", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var user discordUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return "", err
	}
	if user.ID == "" {
		return "", fmt.Errorf("empty user ID from Discord")
	}
	return user.ID, nil
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
```

- [ ] **Step 4: Run all auth module tests — verify they pass**

```powershell
cd "c:\Users\brend\Desktop\Tracker\modules\auth"
$env:GOCACHE="C:\Users\brend\AppData\Local\go-build"; go test ./... -v
```
Expected: all tests PASS

- [ ] **Step 5: Commit**

```
git add modules/auth/auth.go modules/auth/auth_test.go
git commit -m "feat: implement Discord OAuth handlers"
```

---

## Task 5: Wire up in main.go

**Files:**
- Modify: `API/main.go`

- [ ] **Step 1: Update `API/main.go`**

Replace the entire file with:

```go
package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/brendanjhnsn/TrackerAPI/core/config"
	"github.com/brendanjhnsn/TrackerAPI/core/database"
	"github.com/brendanjhnsn/TrackerAPI/modules/auth"
	"github.com/brendanjhnsn/TrackerAPI/modules/dailymessages"
	"github.com/brendanjhnsn/TrackerAPI/modules/loa"
	"github.com/brendanjhnsn/TrackerAPI/modules/qfs"
	"github.com/brendanjhnsn/TrackerAPI/modules/tickets"
	"github.com/brendanjhnsn/TrackerAPI/modules/voicetime"
	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	cfg := config.Load()
	log.Printf("Environment: %s", cfg.Environment)

	db, sqlDB, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
	defer database.Close(sqlDB)

	loaMod      := loa.New(db, cfg)
	ticketsMod  := tickets.New(db, cfg)
	voiceMod    := voicetime.New(db, cfg)
	qfsMod      := qfs.New(db, cfg)
	dailyMod    := dailymessages.New(db, cfg)
	authMod     := auth.New(db, cfg)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	loaMod.RegisterRoutes(mux)
	ticketsMod.RegisterRoutes(mux)
	voiceMod.RegisterRoutes(mux)
	qfsMod.RegisterRoutes(mux)
	dailyMod.RegisterRoutes(mux)
	authMod.RegisterRoutes(mux)

	go func() {
		log.Printf("Starting API server on port %s", cfg.ServerPort)
		if err := http.ListenAndServe(":"+cfg.ServerPort, authMod.Middleware(mux)); err != nil {
			log.Fatalf("Error starting API server: %v", err)
		}
	}()

	discord, err := discordgo.New("Bot " + cfg.DiscordToken)
	if err != nil {
		log.Fatalf("Error creating Discord session: %v", err)
	}
	discord.Identify.Intents = discordgo.IntentsGuilds |
		discordgo.IntentsGuildMessages |
		discordgo.IntentsGuildMessageReactions |
		discordgo.IntentsGuildVoiceStates |
		discordgo.IntentsGuildMembers
	defer discord.Close()

	ticketsMod.Register(discord)
	voiceMod.Register(discord)
	qfsMod.Register(discord)
	dailyMod.Register(discord)

	if err := discord.Open(); err != nil {
		log.Fatalf("Error opening Discord connection: %v", err)
	}

	log.Println("Bot is now running. Press Ctrl+C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM)
	<-sc
}
```

- [ ] **Step 2: Run `go mod tidy` in root**

```powershell
cd "c:\Users\brend\Desktop\Tracker"
$env:GOCACHE="C:\Users\brend\AppData\Local\go-build"; go mod tidy
```

- [ ] **Step 3: Build the full project**

```powershell
cd "c:\Users\brend\Desktop\Tracker"
$env:GOCACHE="C:\Users\brend\AppData\Local\go-build"; go build ./...
```
Expected: no errors, produces `API.exe`

- [ ] **Step 4: Run all tests across all modules**

```powershell
cd "c:\Users\brend\Desktop\Tracker"
$env:GOCACHE="C:\Users\brend\AppData\Local\go-build"; go test ./...
```
Expected: all tests pass

- [ ] **Step 5: Commit**

```
git add API/main.go go.mod go.sum
git commit -m "feat: wire auth module into main — all routes now protected by middleware"
```

---

## After Implementation: .env additions

Add these to your `.env` file before running the server. Get the values from the [Discord Developer Portal](https://discord.com/developers/applications):

```
DISCORD_CLIENT_ID=your_app_client_id
DISCORD_CLIENT_SECRET=your_app_client_secret
DISCORD_REDIRECT_URI=http://localhost:8080/auth/discord/callback
DISCORD_GUILD_ID=your_guild_id
MANAGER_ROLE_ID=discord_role_id_for_managers
DIRECTOR_ROLE_ID=discord_role_id_for_directors
FRONTEND_URL=http://localhost:3000
```

`MOD_ROLE_ID` is already in your `.env`.

In the Discord Developer Portal, under OAuth2 → Redirects, add `DISCORD_REDIRECT_URI` exactly as set above.
