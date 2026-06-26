# Multi-Module Refactor Design

**Date:** 2026-06-25  
**Status:** Approved

## Overview

Refactor the Go API monolith into a multi-module monorepo. Each of the five feature domains (LOA, Ticket Tracking, Voice Time, QFS, Daily Messages) becomes its own Go module with its own `go.mod`. A shared `core` module holds config, database connection, and all models. A single main binary imports and wires everything together.

## Directory Layout

```
Go API/
├── go.mod                              (github.com/brendanjhnsn/go-api — main binary)
├── go.sum
├── .env
├── API/
│   └── main.go                         (thin orchestrator)
├── core/
│   ├── go.mod                          (github.com/brendanjhnsn/go-api/core)
│   ├── config/
│   │   └── config.go
│   └── database/
│       ├── database.go
│       └── models.go
└── modules/
    ├── loa/
    │   ├── go.mod                      (github.com/brendanjhnsn/go-api/modules/loa)
    │   └── loa.go
    ├── tickets/
    │   ├── go.mod                      (github.com/brendanjhnsn/go-api/modules/tickets)
    │   └── tickets.go
    ├── voicetime/
    │   ├── go.mod                      (github.com/brendanjhnsn/go-api/modules/voicetime)
    │   └── voicetime.go
    ├── qfs/
    │   ├── go.mod                      (github.com/brendanjhnsn/go-api/modules/qfs)
    │   └── qfs.go
    └── dailymessages/
        ├── go.mod                      (github.com/brendanjhnsn/go-api/modules/dailymessages)
        └── dailymessages.go
```

The existing `API/server/server.go` is deleted. Each feature module owns its HTTP endpoints directly.

## Core Module

**Path:** `github.com/brendanjhnsn/go-api/core`  
**Location:** `core/`

Contains the shared foundation that every feature module and the main binary depend on.

### core/config/config.go
Unchanged from current `config/config.go`. Exports `Config` struct and `Load()`.

### core/database/database.go
Unchanged from current `database/database.go`. Exports `Connect()` and `Close()`.

### core/database/models.go
Unchanged from current `database/models.go`. Exports all six models:
- `LOA`
- `Ticket`
- `DailyMessage`
- `VoiceTime`
- `Question`
- `QuestionCheck`

### core/go.mod dependencies
- `gorm.io/gorm`
- `gorm.io/driver/postgres`
- `gorm.io/driver/mysql`
- `github.com/lib/pq`
- Does **not** import `discordgo`.

## Feature Module Pattern

Every feature module exposes exactly three public symbols:

```go
func New(db *gorm.DB, cfg *config.Config) *Module
func (m *Module) Register(s *discordgo.Session)   // only on modules with Discord handlers
func (m *Module) RegisterRoutes(mux *http.ServeMux)
```

All handler methods are unexported. Each module's `go.mod` declares `core` as a dependency with a `replace` directive:

```
require github.com/brendanjhnsn/go-api/core v0.0.0
replace github.com/brendanjhnsn/go-api/core => ../../core
```

## Feature Modules

| Module | Go module path | Discord events | HTTP endpoint |
|---|---|---|---|
| `loa` | `github.com/brendanjhnsn/go-api/modules/loa` | none | `/api/loa` (GET, POST) |
| `tickets` | `github.com/brendanjhnsn/go-api/modules/tickets` | `ChannelCreate`, `MessageCreate` | `/api/tickets` |
| `voicetime` | `github.com/brendanjhnsn/go-api/modules/voicetime` | `VoiceStateUpdate` | `/api/voicetime` |
| `qfs` | `github.com/brendanjhnsn/go-api/modules/qfs` | `MessageCreate`, `MessageReactionAdd` | `/api/checks` |
| `dailymessages` | `github.com/brendanjhnsn/go-api/modules/dailymessages` | `MessageCreate` | `/api/messages` |

### loa
- No Discord handler. `loaHandler` is removed from `main.go` — it was never implemented and the LOA workflow is managed entirely through the website frontend.
- `GET /api/loa` — returns all active LOA records.
- `POST /api/loa` — creates a new LOA record.
- `IsMemberOnLOA(db, guildID, memberID string, today time.Time) bool` is exported for potential use by other modules.

### tickets
- `onChannelCreate` — creates a `Ticket` record when a channel is created inside the configured ticket category.
- `onMessageCreate` — calls `trackFirstModResponse` when a mod sends a message in a ticket channel. This function was defined but never called in the original code; it is wired up here.
- `GET /api/tickets` — returns first-responder ticket counts, filterable by `member_id`, `date`, `start_date`/`end_date`.

### voicetime
- `onVoiceStateUpdate` — records join/leave events for mods, computes duration on leave.
- `GET /api/voicetime` — returns aggregated voice time per member per day.

### qfs
- `onMessageCreate` — tracks new questions posted in the QFS channel (non-mod, non-reply messages).
- `onReactionAdd` — records a mod checkmark (✅ or ☑️) on a question in the QFS channel.
- `GET /api/checks` — returns question check counts per member per day.

### dailymessages
- `onMessageCreate` — increments or creates the daily message count for mod members. Ignores ticket and admin category channels.
- `GET /api/messages` — returns daily message counts per member per day.

## Shared Helper Functions

Three small Discord utilities are used by multiple modules (`resolveRoleIDByName`, `memberHasModRole`, `getModRoleID`). Because `core` does not import `discordgo`, these are duplicated into each module that needs them (`qfs`, `dailymessages`, `voicetime`, `tickets`). They are small (under 25 lines total) and stable, so duplication is acceptable over adding a sixth module.

## Main Module Wiring

`API/main.go` becomes a thin orchestrator:

```go
func main() {
    godotenv.Load()
    cfg := config.Load()
    db, sqlDB, err := database.Connect(cfg)
    defer database.Close(sqlDB)

    loaMod     := loa.New(db, cfg)
    ticketsMod := tickets.New(db, cfg)
    voiceMod   := voicetime.New(db, cfg)
    qfsMod     := qfs.New(db, cfg)
    dailyMod   := dailymessages.New(db, cfg)

    mux := http.NewServeMux()
    mux.HandleFunc("/", healthHandler)
    loaMod.RegisterRoutes(mux)
    ticketsMod.RegisterRoutes(mux)
    voiceMod.RegisterRoutes(mux)
    qfsMod.RegisterRoutes(mux)
    dailyMod.RegisterRoutes(mux)
    go http.ListenAndServe(":"+cfg.ServerPort, mux)

    discord, _ := discordgo.New("Bot " + cfg.DiscordToken)
    discord.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages |
        discordgo.IntentsGuildMessageReactions | discordgo.IntentsGuildVoiceStates | discordgo.IntentsGuildMembers
    ticketsMod.Register(discord)
    voiceMod.Register(discord)
    qfsMod.Register(discord)
    dailyMod.Register(discord)
    discord.Open()
    defer discord.Close()

    sc := make(chan os.Signal, 1)
    signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM)
    <-sc
}
```

The main `go.mod` lists all five feature modules and `core` with `replace` directives pointing to local paths.

## Code Migration Summary

| Symbol | Current location | New location |
|---|---|---|
| `config.Config`, `config.Load` | `config/config.go` | `core/config/config.go` |
| `database.Connect`, `database.Close` | `database/database.go` | `core/database/database.go` |
| All models | `database/models.go` | `core/database/models.go` |
| `messageCreateHandler` (daily msg part) | `API/main.go` | `modules/dailymessages/dailymessages.go` |
| `messageCreateHandler` (QFS part) | `API/main.go` | `modules/qfs/qfs.go` |
| `messageCreateHandler` (ticket first-resp part) | `API/main.go` (unused) | `modules/tickets/tickets.go` |
| `reactionAddHandler` | `API/main.go` | `modules/qfs/qfs.go` |
| `voiceStateUpdateHandler` | `API/main.go` | `modules/voicetime/voicetime.go` |
| `channelCreateHandler` | `API/main.go` | `modules/tickets/tickets.go` |
| `isMemberOnLOA` | `API/main.go` (unused) | `modules/loa/loa.go` (exported) |
| `trackFirstModResponse` | `API/main.go` (never called) | `modules/tickets/tickets.go` (now called) |
| `truncate` | `API/main.go` | `modules/qfs/qfs.go` |
| `resolveRoleIDByName`, `memberHasModRole`, `getModRoleID` | `API/main.go` | duplicated in each module that needs them |
| `loaHandler` | `API/main.go` (never defined) | **removed** |
| All HTTP handlers | `API/server/server.go` | split into respective feature modules |
