# Discord OAuth + Auth Middleware Design

**Date:** 2026-06-26
**Status:** Approved

## Overview

Add Discord OAuth2 login and per-request role-based access control to the Tracker API. A new `modules/auth` module handles the OAuth flow and HTTP middleware. All `/api/*` routes except `/api/loa` require a valid session and a recognized Discord guild role.

---

## Architecture

### New files
- `modules/auth/auth.go` — `Module` struct, `RegisterRoutes()`, OAuth redirect/callback handlers
- `modules/auth/middleware.go` — `Middleware()` wrapper, `Role` type, context helpers

### Config additions (`core/config/config.go`)
| Env var | Purpose |
|---|---|
| `DISCORD_CLIENT_ID` | OAuth2 application client ID |
| `DISCORD_CLIENT_SECRET` | OAuth2 application client secret |
| `DISCORD_REDIRECT_URI` | Callback URL registered in Discord developer portal |
| `DISCORD_GUILD_ID` | Guild to check membership and roles against |
| `MANAGER_ROLE_ID` | Discord role ID for Managers |
| `DIRECTOR_ROLE_ID` | Discord role ID for Directors |
| `FRONTEND_URL` | URL to redirect the browser to after successful login |

`MOD_ROLE_ID` already exists in config.

### Route protection
| Pattern | Auth required |
|---|---|
| `/auth/*` | Public — OAuth flow |
| `/api/loa` | Public — managed by community tracker website |
| All other `/api/*` | Session + Discord role required |

---

## Data Model

New GORM model added to `core/database/models.go`:

```go
type Session struct {
    Token         string    `gorm:"primaryKey"`
    DiscordUserID string    `gorm:"not null;index"`
    ExpiresAt     time.Time `gorm:"not null"`
    CreatedAt     time.Time
}
```

- Tokens: 32-byte cryptographically random values, hex-encoded (64-char strings)
- Expiry: 24 hours from creation
- Cleanup: expired sessions deleted on lookup (no background job)
- GORM auto-migrates this table at startup alongside existing models

---

## Data Flow

### Login (one-time)

1. Frontend navigates to `GET /auth/discord/redirect`
2. Backend generates a random `state` value, stores it in a 5-minute HTTP-only `oauth_state` cookie, and redirects to Discord's authorization URL
3. User authorizes → Discord redirects to `GET /auth/discord/callback?code=...&state=...`
4. Backend validates `state` against the cookie (CSRF protection)
5. Backend POSTs to Discord token endpoint to exchange `code` for an access token
6. Backend calls `GET /users/@me` with that access token to retrieve the Discord user ID
7. Backend creates a `Session` record in Postgres and sets an HTTP-only `session` cookie
8. Backend redirects browser to the frontend dashboard

### Per-request middleware

1. Read `session` cookie → look up in `sessions` table → **401** if missing or expired
2. Use bot token (`cfg.DiscordToken`) to call `GET /guilds/{guild_id}/members/{user_id}` → **403** if user not in guild
3. Map returned role IDs against `DIRECTOR_ROLE_ID`, `MANAGER_ROLE_ID`, `MOD_ROLE_ID` → **403** if none match
4. Attach resolved `Role` to `context.WithValue` and pass to next handler

---

## Role Model

```go
type Role int

const (
    RoleMod      Role = iota // read-only
    RoleManager              // read + write
    RoleDirector             // read + write
)
```

Role resolution priority (highest wins): Director > Manager > Mod. A user with multiple matching roles gets the highest one.

### Handler-level enforcement

Middleware handles authentication and role resolution. Individual write handlers check the role from context:

```go
role := auth.RoleFromContext(r.Context())
if role == auth.RoleMod {
    http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
    return
}
```

GET requests are permitted for any authenticated role. POST/PUT/DELETE are blocked for `RoleMod`.

---

## Integration in main.go

```go
authMod := auth.New(db, cfg)
authMod.RegisterRoutes(mux)                          // /auth/discord/redirect, /auth/discord/callback
http.ListenAndServe(":"+cfg.ServerPort, authMod.Middleware(mux))
```

---

## Out of Scope

- Token refresh (Discord access token is used once at login; bot token is used for role checks)
- Logout endpoint (can be added later; delete session row)
- Per-module role granularity beyond read/write split
