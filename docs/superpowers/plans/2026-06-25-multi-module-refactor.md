# Multi-Module Refactor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Split the monolithic Go API into a multi-module monorepo with a shared `core` module and five independent feature modules (loa, tickets, voicetime, qfs, dailymessages), each with its own `go.mod`.

**Architecture:** A `core` module holds config, database connection, and all models. Five feature library modules each import `core` via a local `replace` directive and expose `New(db, cfg)`, `Register(session)`, and `RegisterRoutes(mux)`. The main binary imports all five and wires them together.

**Tech Stack:** Go 1.22, GORM v1.31.1, discordgo v0.28.1, PostgreSQL/MySQL, net/http

---

## File Map

**Create:**
- `core/go.mod`
- `core/config/config.go`
- `core/database/database.go`
- `core/database/models.go`
- `modules/loa/go.mod`, `modules/loa/loa.go`, `modules/loa/loa_test.go`
- `modules/tickets/go.mod`, `modules/tickets/tickets.go`, `modules/tickets/tickets_test.go`
- `modules/voicetime/go.mod`, `modules/voicetime/voicetime.go`, `modules/voicetime/voicetime_test.go`
- `modules/qfs/go.mod`, `modules/qfs/qfs.go`, `modules/qfs/qfs_test.go`
- `modules/dailymessages/go.mod`, `modules/dailymessages/dailymessages.go`, `modules/dailymessages/dailymessages_test.go`

**Modify:**
- `go.mod` — add replace directives for all local modules
- `API/main.go` — rewrite as thin orchestrator

**Delete:**
- `config/config.go`
- `database/database.go`
- `database/models.go`
- `API/server/server.go`

---

## Task 1: Create core module

**Files:**
- Create: `core/go.mod`
- Create: `core/config/config.go`
- Create: `core/database/models.go`
- Create: `core/database/database.go`

- [ ] **Step 1: Create `core/go.mod`**

```
module github.com/brendanjhnsn/go-api/core

go 1.22

require (
	github.com/lib/pq v1.12.3
	gorm.io/driver/mysql v1.5.7
	gorm.io/driver/postgres v1.6.0
	gorm.io/gorm v1.31.1
)
```

- [ ] **Step 2: Create `core/config/config.go`**

Exact copy of the current `config/config.go` — only the package declaration changes (it stays `package config`):

```go
package config

import (
	"os"
	"strconv"
)

type Config struct {
	Environment      string
	ServerPort       string
	DiscordToken     string
	DBDriver         string
	DBHost           string
	DBUser           string
	DBPassword       string
	DBName           string
	DBPort           int
	QFSChannelID     string
	ModRoleID        string
	ModRoleName      string
	TicketCategoryID string
	QFSChannelName   string
	AdminCategoryID  string
}

func Load() *Config {
	return &Config{
		Environment:      getEnv("ENVIRONMENT", "development"),
		ServerPort:       getEnv("SERVER_PORT", "8080"),
		DiscordToken:     getEnv("DISCORD_TOKEN", ""),
		DBDriver:         getEnv("DB_DRIVER", "postgres"),
		DBHost:           getEnv("DB_HOST", "127.0.0.1"),
		DBUser:           getEnv("DB_USER", "postgres"),
		DBPassword:       getEnv("DB_PASSWORD", ""),
		DBName:           getEnv("DB_NAME", "community_tracker"),
		DBPort:           getEnvAsInt("DB_PORT", 5432),
		QFSChannelID:     getEnv("QFS_CHANNEL_ID", ""),
		ModRoleID:        getEnv("MOD_ROLE_ID", ""),
		ModRoleName:      getEnv("MOD_ROLE_NAME", ""),
		TicketCategoryID: getEnv("TICKET_CATEGORY_ID", ""),
		QFSChannelName:   getEnv("QFS_CHANNEL_NAME", ""),
		AdminCategoryID:  getEnv("ADMIN_CATEGORY_ID", ""),
	}
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getEnvAsInt(key string, fallback int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return fallback
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return fallback
	}
	return value
}
```

- [ ] **Step 3: Create `core/database/models.go`**

Same as current `database/models.go`:

```go
package database

import "time"

type Question struct {
	ID        uint      `gorm:"primaryKey"`
	GuildID   string    `gorm:"index;not null"`
	ChannelID string    `gorm:"index;not null"`
	MessageID string    `gorm:"index;not null;unique"`
	Title     string    `gorm:"type:varchar(255)"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type QuestionCheck struct {
	ID         uint       `gorm:"primaryKey"`
	QuestionID uint       `gorm:"index;not null"`
	Question   Question   `gorm:"constraint:OnDelete:CASCADE"`
	RoleID     string     `gorm:"index;not null"`
	MemberID   string     `gorm:"index;not null"`
	Date       *time.Time `gorm:"index"`
	CheckedAt  time.Time  `gorm:"autoCreateTime"`
	RemovedAt  *time.Time
}

type Ticket struct {
	ID            uint       `gorm:"primaryKey"`
	GuildID       string     `gorm:"index;not null"`
	ChannelID     string     `gorm:"index;not null;unique"`
	FirstRespID   string     `gorm:"index"`
	FirstRespDate *time.Time `gorm:"index"`
	FirstRespAt   *time.Time
	ClosedAt      *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type DailyMessage struct {
	Date      time.Time `gorm:"index;not null,primaryKey"`
	GuildID   string    `gorm:"index;not null"`
	MemberID  string    `gorm:"index;not null"`
	Count     int       `gorm:"default:0"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type VoiceTime struct {
	ID        uint       `gorm:"primaryKey"`
	GuildID   string     `gorm:"index;not null"`
	MemberID  string     `gorm:"index;not null"`
	ChannelID string
	Date      *time.Time `gorm:"index"`
	JoinedAt  time.Time  `gorm:"index;not null"`
	LeftAt    *time.Time
	Duration  int64      `gorm:"default:0"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type LOA struct {
	ID        uint       `gorm:"primaryKey"`
	GuildID   string     `gorm:"index;not null"`
	MemberID  string     `gorm:"index;not null"`
	StartDate *time.Time `gorm:"index"`
	EndDate   *time.Time `gorm:"index"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
```

- [ ] **Step 4: Create `core/database/database.go`**

Same logic as current `database/database.go`, but import path updated to `core/config` and `LOA` added to AutoMigrate:

```go
package database

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/brendanjhnsn/go-api/core/config"
	_ "github.com/lib/pq"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(cfg *config.Config) (*gorm.DB, *sql.DB, error) {
	switch strings.ToLower(cfg.DBDriver) {
	case "postgres", "postgresql":
		return connectPostgres(cfg)
	case "mysql":
		return connectMySQL(cfg)
	default:
		return nil, nil, fmt.Errorf("unsupported DB driver: %s", cfg.DBDriver)
	}
}

func connectPostgres(cfg *config.Config) (*gorm.DB, *sql.DB, error) {
	if err := ensurePostgresDatabase(cfg); err != nil {
		return nil, nil, fmt.Errorf("failed to ensure postgres database: %w", err)
	}
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable TimeZone=UTC",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}
	if err := db.AutoMigrate(&Question{}, &QuestionCheck{}, &Ticket{}, &DailyMessage{}, &VoiceTime{}, &LOA{}); err != nil {
		return nil, nil, fmt.Errorf("failed to migrate postgres schema: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get postgres sql db instance: %w", err)
	}
	return db, sqlDB, nil
}

func ensurePostgresDatabase(cfg *config.Config) error {
	adminDSN := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=postgres sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword)
	adminDB, err := sql.Open("postgres", adminDSN)
	if err != nil {
		return err
	}
	defer adminDB.Close()
	if err := adminDB.Ping(); err != nil {
		return err
	}
	var exists bool
	if err := adminDB.QueryRow("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", cfg.DBName).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return nil
	}
	quoted := fmt.Sprintf("\"%s\"", strings.ReplaceAll(cfg.DBName, "\"", "\"\""))
	_, err = adminDB.Exec("CREATE DATABASE " + quoted)
	return err
}

func connectMySQL(cfg *config.Config) (*gorm.DB, *sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=UTC",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to mysql: %w", err)
	}
	if err := db.AutoMigrate(&Question{}, &QuestionCheck{}, &Ticket{}, &DailyMessage{}, &VoiceTime{}, &LOA{}); err != nil {
		return nil, nil, fmt.Errorf("failed to migrate mysql schema: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get mysql sql db instance: %w", err)
	}
	return db, sqlDB, nil
}

func Close(sqlDB *sql.DB) {
	if sqlDB != nil {
		_ = sqlDB.Close()
	}
}
```

- [ ] **Step 5: Verify core compiles**

```
cd core
go mod tidy
go build ./...
```

Expected: no errors. `go mod tidy` will populate indirect dependencies in `go.mod` and create `go.sum`.

- [ ] **Step 6: Commit**

```
git add core/
git commit -m "feat: add core module with config, database, and models"
```

---

## Task 2: Create LOA module

**Files:**
- Create: `modules/loa/go.mod`
- Create: `modules/loa/loa_test.go`
- Create: `modules/loa/loa.go`

- [ ] **Step 1: Create `modules/loa/go.mod`**

```
module github.com/brendanjhnsn/go-api/modules/loa

go 1.22

require (
	github.com/brendanjhnsn/go-api/core v0.0.0
	gorm.io/gorm v1.31.1
)

replace github.com/brendanjhnsn/go-api/core => ../../core
```

- [ ] **Step 2: Write failing test `modules/loa/loa_test.go`**

```go
package loa_test

import (
	"net/http"
	"testing"

	"github.com/brendanjhnsn/go-api/modules/loa"
)

var _ interface{ RegisterRoutes(*http.ServeMux) } = (*loa.Module)(nil)

func TestNew(t *testing.T) {
	if loa.New(nil, nil) == nil {
		t.Fatal("New returned nil")
	}
}
```

- [ ] **Step 3: Verify test fails to compile (loa.go doesn't exist yet)**

```
cd modules/loa
go mod tidy
go test ./...
```

Expected: compile error — `undefined: loa.Module`

- [ ] **Step 4: Create `modules/loa/loa.go`**

```go
package loa

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/brendanjhnsn/go-api/core/config"
	"github.com/brendanjhnsn/go-api/core/database"
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
	mux.HandleFunc("/api/loa", m.handleLOA)
}

func IsMemberOnLOA(db *gorm.DB, guildID, memberID string, today time.Time) bool {
	var loa database.LOA
	result := db.Where("guild_id = ? AND member_id = ? AND start_date <= ? AND end_date >= ?",
		guildID, memberID, today, today).First(&loa)
	return result.Error == nil
}

func (m *Module) handleLOA(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	switch r.Method {
	case http.MethodGet:
		m.getLOAs(w, r)
	case http.MethodPost:
		m.createLOA(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (m *Module) getLOAs(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	var loas []database.LOA
	m.db.Where("start_date <= ? AND end_date >= ?", today, today).Find(&loas)
	if loas == nil {
		loas = []database.LOA{}
	}
	_ = json.NewEncoder(w).Encode(loas)
}

type createLOARequest struct {
	GuildID   string `json:"guild_id"`
	MemberID  string `json:"member_id"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

func (m *Module) createLOA(w http.ResponseWriter, r *http.Request) {
	var req createLOARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}
	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid start_date format, use YYYY-MM-DD"})
		return
	}
	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid end_date format, use YYYY-MM-DD"})
		return
	}
	loa := database.LOA{
		GuildID:   req.GuildID,
		MemberID:  req.MemberID,
		StartDate: &startDate,
		EndDate:   &endDate,
	}
	if err := m.db.Create(&loa).Error; err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to create LOA"})
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(loa)
}
```

- [ ] **Step 5: Run tests**

```
go test ./...
```

Expected: `ok  github.com/brendanjhnsn/go-api/modules/loa`

- [ ] **Step 6: Commit**

```
git add modules/loa/
git commit -m "feat: add loa module with /api/loa GET and POST endpoints"
```

---

## Task 3: Create tickets module

**Files:**
- Create: `modules/tickets/go.mod`
- Create: `modules/tickets/tickets_test.go`
- Create: `modules/tickets/tickets.go`

- [ ] **Step 1: Create `modules/tickets/go.mod`**

```
module github.com/brendanjhnsn/go-api/modules/tickets

go 1.22

require (
	github.com/brendanjhnsn/go-api/core v0.0.0
	github.com/bwmarrin/discordgo v0.28.1
	gorm.io/gorm v1.31.1
)

replace github.com/brendanjhnsn/go-api/core => ../../core
```

- [ ] **Step 2: Write failing test `modules/tickets/tickets_test.go`**

```go
package tickets_test

import (
	"net/http"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/brendanjhnsn/go-api/modules/tickets"
)

var _ interface {
	Register(*discordgo.Session)
	RegisterRoutes(*http.ServeMux)
} = (*tickets.Module)(nil)

func TestNew(t *testing.T) {
	if tickets.New(nil, nil) == nil {
		t.Fatal("New returned nil")
	}
}
```

- [ ] **Step 3: Verify test fails to compile**

```
cd modules/tickets
go mod tidy
go test ./...
```

Expected: compile error — `undefined: tickets.Module`

- [ ] **Step 4: Create `modules/tickets/tickets.go`**

```go
package tickets

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/brendanjhnsn/go-api/core/config"
	"github.com/brendanjhnsn/go-api/core/database"
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
	s.AddHandler(m.onChannelCreate)
	s.AddHandler(m.onMessageCreate)
}

func (m *Module) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/tickets", m.handleTickets)
}

func (m *Module) onChannelCreate(s *discordgo.Session, cc *discordgo.ChannelCreate) {
	if m.cfg.TicketCategoryID == "" || cc.ParentID != m.cfg.TicketCategoryID {
		return
	}
	_ = m.db.Create(&database.Ticket{
		GuildID:   cc.GuildID,
		ChannelID: cc.ID,
	}).Error
}

func (m *Module) onMessageCreate(s *discordgo.Session, msg *discordgo.MessageCreate) {
	if msg.Author.Bot {
		return
	}
	member, err := s.GuildMember(msg.GuildID, msg.Author.ID)
	if err != nil {
		return
	}
	modRoleID, err := m.getModRoleID(s, msg.GuildID)
	if err != nil || !memberHasModRole(member, modRoleID) {
		return
	}
	m.trackFirstModResponse(msg.GuildID, msg.ChannelID, msg.Author.ID)
}

func (m *Module) trackFirstModResponse(guildID, channelID, modID string) {
	var ticket database.Ticket
	result := m.db.Where("guild_id = ? AND channel_id = ? AND first_resp_id IS NULL", guildID, channelID).First(&ticket)
	if result.Error == nil {
		now := time.Now().UTC()
		dateOnly := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		_ = m.db.Model(&ticket).Updates(map[string]interface{}{
			"first_resp_id":   modID,
			"first_resp_at":   now,
			"first_resp_date": &dateOnly,
		}).Error
	}
}

func (m *Module) handleTickets(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	q := r.URL.Query()
	memberID := q.Get("member_id")
	dateStr := q.Get("date")
	startDateStr := q.Get("start_date")
	endDateStr := q.Get("end_date")

	type DailyTicketRow struct {
		Date     string `json:"date"`
		MemberID string `json:"member_id"`
		Tickets  int64  `json:"tickets"`
	}

	query := m.db.Model(&database.Ticket{}).
		Where("first_resp_id IS NOT NULL AND first_resp_at IS NOT NULL AND first_resp_date IS NOT NULL")

	if memberID != "" {
		query = query.Where("first_resp_id = ?", memberID)
	}
	if dateStr != "" {
		d, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid date format, use YYYY-MM-DD"})
			return
		}
		query = query.Where("first_resp_date::date = ?", d)
	} else if startDateStr != "" && endDateStr != "" {
		start, err := time.Parse("2006-01-02", startDateStr)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid start_date format, use YYYY-MM-DD"})
			return
		}
		end, err := time.Parse("2006-01-02", endDateStr)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid end_date format, use YYYY-MM-DD"})
			return
		}
		query = query.Where("first_resp_date::date >= ? AND first_resp_date::date <= ?", start, end)
	}

	var rows []DailyTicketRow
	query.Select("to_char(first_resp_date, 'YYYY-MM-DD') as date, first_resp_id as member_id, count(*) as tickets").
		Group("first_resp_date::date, first_resp_id").
		Order("first_resp_date DESC").
		Scan(&rows)
	if rows == nil {
		rows = []DailyTicketRow{}
	}
	_ = json.NewEncoder(w).Encode(rows)
}

func (m *Module) getModRoleID(s *discordgo.Session, guildID string) (string, error) {
	if m.cfg.ModRoleID != "" {
		return m.cfg.ModRoleID, nil
	}
	if m.cfg.ModRoleName == "" {
		return "", errors.New("mod role ID or name is not configured")
	}
	return resolveRoleIDByName(s, guildID, m.cfg.ModRoleName)
}

func memberHasModRole(member *discordgo.Member, roleID string) bool {
	for _, rid := range member.Roles {
		if rid == roleID {
			return true
		}
	}
	return false
}

func resolveRoleIDByName(s *discordgo.Session, guildID, roleName string) (string, error) {
	roles, err := s.GuildRoles(guildID)
	if err != nil {
		return "", err
	}
	for _, role := range roles {
		if strings.EqualFold(role.Name, roleName) {
			return role.ID, nil
		}
	}
	return "", errors.New("role not found by name")
}
```

- [ ] **Step 5: Run tests**

```
go test ./...
```

Expected: `ok  github.com/brendanjhnsn/go-api/modules/tickets`

- [ ] **Step 6: Commit**

```
git add modules/tickets/
git commit -m "feat: add tickets module with channel tracking, first-responder wiring, and /api/tickets"
```

---

## Task 4: Create voicetime module

**Files:**
- Create: `modules/voicetime/go.mod`
- Create: `modules/voicetime/voicetime_test.go`
- Create: `modules/voicetime/voicetime.go`

- [ ] **Step 1: Create `modules/voicetime/go.mod`**

```
module github.com/brendanjhnsn/go-api/modules/voicetime

go 1.22

require (
	github.com/brendanjhnsn/go-api/core v0.0.0
	github.com/bwmarrin/discordgo v0.28.1
	gorm.io/gorm v1.31.1
)

replace github.com/brendanjhnsn/go-api/core => ../../core
```

- [ ] **Step 2: Write failing test `modules/voicetime/voicetime_test.go`**

```go
package voicetime_test

import (
	"net/http"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/brendanjhnsn/go-api/modules/voicetime"
)

var _ interface {
	Register(*discordgo.Session)
	RegisterRoutes(*http.ServeMux)
} = (*voicetime.Module)(nil)

func TestNew(t *testing.T) {
	if voicetime.New(nil, nil) == nil {
		t.Fatal("New returned nil")
	}
}
```

- [ ] **Step 3: Verify test fails to compile**

```
cd modules/voicetime
go mod tidy
go test ./...
```

Expected: compile error — `undefined: voicetime.Module`

- [ ] **Step 4: Create `modules/voicetime/voicetime.go`**

```go
package voicetime

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/brendanjhnsn/go-api/core/config"
	"github.com/brendanjhnsn/go-api/core/database"
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
	s.AddHandler(m.onVoiceStateUpdate)
}

func (m *Module) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/voicetime", m.handleVoiceTime)
}

func (m *Module) onVoiceStateUpdate(s *discordgo.Session, vs *discordgo.VoiceStateUpdate) {
	if m.cfg.ModRoleID == "" && m.cfg.ModRoleName == "" {
		return
	}
	member, err := s.GuildMember(vs.GuildID, vs.UserID)
	if err != nil {
		return
	}
	modRoleID, err := m.getModRoleID(s, vs.GuildID)
	if err != nil || !memberHasModRole(member, modRoleID) {
		return
	}
	if vs.ChannelID == "" {
		var vt database.VoiceTime
		result := m.db.Where("member_id = ? AND guild_id = ? AND left_at IS NULL", vs.UserID, vs.GuildID).First(&vt)
		if result.Error == nil {
			now := time.Now().UTC()
			_ = m.db.Model(&vt).Updates(map[string]interface{}{
				"left_at":  now,
				"duration": int64(now.Sub(vt.JoinedAt).Seconds()),
			}).Error
		}
	} else {
		now := time.Now().UTC()
		dateOnly := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		_ = m.db.Create(&database.VoiceTime{
			GuildID:   vs.GuildID,
			MemberID:  vs.UserID,
			ChannelID: vs.ChannelID,
			Date:      &dateOnly,
			JoinedAt:  now,
		}).Error
	}
}

func (m *Module) handleVoiceTime(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	q := r.URL.Query()
	memberID := q.Get("member_id")
	dateStr := q.Get("date")
	startDateStr := q.Get("start_date")
	endDateStr := q.Get("end_date")

	type DailyVoiceRow struct {
		Date         string `json:"date"`
		MemberID     string `json:"member_id"`
		TotalSeconds int64  `json:"total_seconds"`
		Hours        int64  `json:"hours"`
		Minutes      int64  `json:"minutes"`
	}

	query := m.db.Model(&database.VoiceTime{}).Where("date IS NOT NULL")
	if memberID != "" {
		query = query.Where("member_id = ?", memberID)
	}
	if dateStr != "" {
		d, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid date format, use YYYY-MM-DD"})
			return
		}
		query = query.Where("date::date = ?", d)
	} else if startDateStr != "" && endDateStr != "" {
		start, err := time.Parse("2006-01-02", startDateStr)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid start_date format, use YYYY-MM-DD"})
			return
		}
		end, err := time.Parse("2006-01-02", endDateStr)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid end_date format, use YYYY-MM-DD"})
			return
		}
		query = query.Where("date::date >= ? AND date::date <= ?", start, end)
	}

	var rows []DailyVoiceRow
	query.Select(`to_char(date, 'YYYY-MM-DD') as date, member_id,
		COALESCE(SUM(duration), 0) as total_seconds,
		COALESCE(SUM(duration), 0) / 3600 as hours,
		(COALESCE(SUM(duration), 0) % 3600) / 60 as minutes`).
		Group("date::date, member_id").
		Order("date DESC").
		Scan(&rows)
	if rows == nil {
		rows = []DailyVoiceRow{}
	}
	_ = json.NewEncoder(w).Encode(rows)
}

func (m *Module) getModRoleID(s *discordgo.Session, guildID string) (string, error) {
	if m.cfg.ModRoleID != "" {
		return m.cfg.ModRoleID, nil
	}
	if m.cfg.ModRoleName == "" {
		return "", errors.New("mod role ID or name is not configured")
	}
	return resolveRoleIDByName(s, guildID, m.cfg.ModRoleName)
}

func memberHasModRole(member *discordgo.Member, roleID string) bool {
	for _, rid := range member.Roles {
		if rid == roleID {
			return true
		}
	}
	return false
}

func resolveRoleIDByName(s *discordgo.Session, guildID, roleName string) (string, error) {
	roles, err := s.GuildRoles(guildID)
	if err != nil {
		return "", err
	}
	for _, role := range roles {
		if strings.EqualFold(role.Name, roleName) {
			return role.ID, nil
		}
	}
	return "", errors.New("role not found by name")
}
```

- [ ] **Step 5: Run tests**

```
go test ./...
```

Expected: `ok  github.com/brendanjhnsn/go-api/modules/voicetime`

- [ ] **Step 6: Commit**

```
git add modules/voicetime/
git commit -m "feat: add voicetime module with voice tracking and /api/voicetime"
```

---

## Task 5: Create QFS module

**Files:**
- Create: `modules/qfs/go.mod`
- Create: `modules/qfs/qfs_test.go`
- Create: `modules/qfs/qfs.go`

- [ ] **Step 1: Create `modules/qfs/go.mod`**

```
module github.com/brendanjhnsn/go-api/modules/qfs

go 1.22

require (
	github.com/brendanjhnsn/go-api/core v0.0.0
	github.com/bwmarrin/discordgo v0.28.1
	gorm.io/gorm v1.31.1
)

replace github.com/brendanjhnsn/go-api/core => ../../core
```

- [ ] **Step 2: Write failing test `modules/qfs/qfs_test.go`**

```go
package qfs_test

import (
	"net/http"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/brendanjhnsn/go-api/modules/qfs"
)

var _ interface {
	Register(*discordgo.Session)
	RegisterRoutes(*http.ServeMux)
} = (*qfs.Module)(nil)

func TestNew(t *testing.T) {
	if qfs.New(nil, nil) == nil {
		t.Fatal("New returned nil")
	}
}
```

- [ ] **Step 3: Verify test fails to compile**

```
cd modules/qfs
go mod tidy
go test ./...
```

Expected: compile error — `undefined: qfs.Module`

- [ ] **Step 4: Create `modules/qfs/qfs.go`**

```go
package qfs

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/brendanjhnsn/go-api/core/config"
	"github.com/brendanjhnsn/go-api/core/database"
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
	s.AddHandler(m.onReactionAdd)
}

func (m *Module) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/checks", m.handleChecks)
}

func (m *Module) onMessageCreate(s *discordgo.Session, msg *discordgo.MessageCreate) {
	if msg.Author.Bot {
		return
	}
	if m.cfg.QFSChannelID == "" || msg.ChannelID != m.cfg.QFSChannelID {
		return
	}
	if msg.MessageReference != nil && msg.MessageReference.MessageID != "" {
		return
	}
	member, err := s.GuildMember(msg.GuildID, msg.Author.ID)
	if err != nil {
		return
	}
	modRoleID, err := m.getModRoleID(s, msg.GuildID)
	if err == nil && memberHasModRole(member, modRoleID) {
		return
	}

	log.Printf("[QFS] New message in questions channel: %s (Author: %s)", msg.ID, msg.Author.Username)

	var q database.Question
	err = m.db.Where("message_id = ?", msg.ID).First(&q).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		q = database.Question{
			GuildID:   msg.GuildID,
			ChannelID: msg.ChannelID,
			MessageID: msg.ID,
			Title:     truncate(msg.Content, 255),
		}
		if createErr := m.db.Create(&q).Error; createErr != nil {
			log.Printf("[ERROR] Failed to create question: %v", createErr)
		} else {
			log.Printf("[QFS] Created question record for message %s", msg.ID)
		}
	}
}

func (m *Module) onReactionAdd(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	log.Printf("[CHECKMARK] Reaction event: message=%s channel=%s user=%s emoji=%s",
		r.MessageID, r.ChannelID, r.UserID, r.Emoji.Name)

	if m.cfg.QFSChannelID == "" || r.ChannelID != m.cfg.QFSChannelID {
		log.Printf("[CHECKMARK] Ignored because not in QFS channel: %s", r.ChannelID)
		return
	}
	em := r.Emoji.Name
	if em != "✅" && em != "☑️" {
		log.Printf("[CHECKMARK] Ignored because emoji is not a checkmark: %s", em)
		return
	}

	member, err := s.GuildMember(r.GuildID, r.UserID)
	if err != nil {
		log.Printf("[CHECKMARK] Failed to fetch guild member: %v", err)
		return
	}
	modRoleID := m.cfg.ModRoleID
	if modRoleID == "" && m.cfg.ModRoleName != "" {
		resolved, resolveErr := resolveRoleIDByName(s, r.GuildID, m.cfg.ModRoleName)
		if resolveErr != nil {
			log.Printf("[CHECKMARK] Failed to resolve mod role by name: %v", resolveErr)
		} else {
			modRoleID = resolved
			log.Printf("[CHECKMARK] Resolved mod role name %s to ID %s", m.cfg.ModRoleName, modRoleID)
		}
	}
	isMod := false
	for _, rid := range member.Roles {
		if rid == modRoleID {
			isMod = true
			break
		}
	}
	if !isMod {
		log.Printf("[CHECKMARK] Reactor is not mod (role %s missing): roles=%v", modRoleID, member.Roles)
		return
	}

	var q database.Question
	err = m.db.Where("message_id = ?", r.MessageID).First(&q).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		log.Printf("[CHECKMARK] Question not found, creating: %s", r.MessageID)
		fetchedMsg, _ := s.ChannelMessage(r.ChannelID, r.MessageID)
		title := ""
		if fetchedMsg != nil {
			title = truncate(fetchedMsg.Content, 255)
		}
		q = database.Question{
			GuildID:   r.GuildID,
			ChannelID: r.ChannelID,
			MessageID: r.MessageID,
			Title:     title,
		}
		if createErr := m.db.Create(&q).Error; createErr != nil {
			log.Printf("[ERROR] Failed to create question in reaction handler: %v", createErr)
			return
		}
		log.Printf("[CHECKMARK] Created question %s in reaction handler", r.MessageID)
	}

	var existing database.QuestionCheck
	err = m.db.Where("question_id = ? AND member_id = ? AND removed_at IS NULL", q.ID, r.UserID).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		now := time.Now().UTC()
		dateOnly := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		qc := database.QuestionCheck{
			QuestionID: q.ID,
			RoleID:     modRoleID,
			MemberID:   r.UserID,
			Date:       &dateOnly,
			CheckedAt:  now,
		}
		if createErr := m.db.Create(&qc).Error; createErr != nil {
			log.Printf("[ERROR] Failed to create question check: %v", createErr)
		} else {
			log.Printf("[CHECKMARK] Recorded checkmark from %s on question %s", r.UserID, r.MessageID)
		}
	} else if err != nil {
		log.Printf("[ERROR] Failed to query existing question check: %v", err)
	} else {
		log.Printf("[CHECKMARK] Duplicate check prevented for member %s on question %s", r.UserID, r.MessageID)
	}
}

func (m *Module) handleChecks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	q := r.URL.Query()
	memberID := q.Get("member_id")
	dateStr := q.Get("date")
	startDateStr := q.Get("start_date")
	endDateStr := q.Get("end_date")

	type DailyCheckRow struct {
		Date     string `json:"date"`
		MemberID string `json:"member_id"`
		Count    int64  `json:"count"`
	}

	query := m.db.Model(&database.QuestionCheck{}).Where("date IS NOT NULL")
	if memberID != "" {
		query = query.Where("member_id = ?", memberID)
	}
	if dateStr != "" {
		d, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid date format, use YYYY-MM-DD"})
			return
		}
		query = query.Where("date::date = ?", d)
	} else if startDateStr != "" && endDateStr != "" {
		start, err := time.Parse("2006-01-02", startDateStr)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid start_date format, use YYYY-MM-DD"})
			return
		}
		end, err := time.Parse("2006-01-02", endDateStr)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid end_date format, use YYYY-MM-DD"})
			return
		}
		query = query.Where("date::date >= ? AND date::date <= ?", start, end)
	}

	var rows []DailyCheckRow
	query.Select("to_char(date, 'YYYY-MM-DD') as date, member_id, count(*) as count").
		Group("date::date, member_id").
		Order("date DESC").
		Scan(&rows)
	if rows == nil {
		rows = []DailyCheckRow{}
	}
	_ = json.NewEncoder(w).Encode(rows)
}

func (m *Module) getModRoleID(s *discordgo.Session, guildID string) (string, error) {
	if m.cfg.ModRoleID != "" {
		return m.cfg.ModRoleID, nil
	}
	if m.cfg.ModRoleName == "" {
		return "", errors.New("mod role ID or name is not configured")
	}
	return resolveRoleIDByName(s, guildID, m.cfg.ModRoleName)
}

func memberHasModRole(member *discordgo.Member, roleID string) bool {
	for _, rid := range member.Roles {
		if rid == roleID {
			return true
		}
	}
	return false
}

func resolveRoleIDByName(s *discordgo.Session, guildID, roleName string) (string, error) {
	roles, err := s.GuildRoles(guildID)
	if err != nil {
		return "", err
	}
	for _, role := range roles {
		if strings.EqualFold(role.Name, roleName) {
			return role.ID, nil
		}
	}
	return "", errors.New("role not found by name")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
```

- [ ] **Step 5: Run tests**

```
go test ./...
```

Expected: `ok  github.com/brendanjhnsn/go-api/modules/qfs`

- [ ] **Step 6: Commit**

```
git add modules/qfs/
git commit -m "feat: add qfs module with question tracking, checkmark reactions, and /api/checks"
```

---

## Task 6: Create dailymessages module

**Files:**
- Create: `modules/dailymessages/go.mod`
- Create: `modules/dailymessages/dailymessages_test.go`
- Create: `modules/dailymessages/dailymessages.go`

- [ ] **Step 1: Create `modules/dailymessages/go.mod`**

```
module github.com/brendanjhnsn/go-api/modules/dailymessages

go 1.22

require (
	github.com/brendanjhnsn/go-api/core v0.0.0
	github.com/bwmarrin/discordgo v0.28.1
	gorm.io/gorm v1.31.1
)

replace github.com/brendanjhnsn/go-api/core => ../../core
```

- [ ] **Step 2: Write failing test `modules/dailymessages/dailymessages_test.go`**

```go
package dailymessages_test

import (
	"net/http"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/brendanjhnsn/go-api/modules/dailymessages"
)

var _ interface {
	Register(*discordgo.Session)
	RegisterRoutes(*http.ServeMux)
} = (*dailymessages.Module)(nil)

func TestNew(t *testing.T) {
	if dailymessages.New(nil, nil) == nil {
		t.Fatal("New returned nil")
	}
}
```

- [ ] **Step 3: Verify test fails to compile**

```
cd modules/dailymessages
go mod tidy
go test ./...
```

Expected: compile error — `undefined: dailymessages.Module`

- [ ] **Step 4: Create `modules/dailymessages/dailymessages.go`**

```go
package dailymessages

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/brendanjhnsn/go-api/core/config"
	"github.com/brendanjhnsn/go-api/core/database"
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
}

func (m *Module) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/messages", m.handleMessages)
}

func (m *Module) onMessageCreate(s *discordgo.Session, msg *discordgo.MessageCreate) {
	if msg.Author.Bot {
		return
	}
	member, err := s.GuildMember(msg.GuildID, msg.Author.ID)
	if err != nil {
		return
	}
	modRoleID, roleErr := m.getModRoleID(s, msg.GuildID)
	if roleErr != nil || !memberHasModRole(member, modRoleID) {
		return
	}
	if m.cfg.TicketCategoryID != "" && msg.ChannelID == m.cfg.TicketCategoryID {
		return
	}
	if m.cfg.AdminCategoryID != "" && msg.ChannelID == m.cfg.AdminCategoryID {
		return
	}

	today := time.Now().UTC()
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	var dm database.DailyMessage
	result := m.db.Where("guild_id = ? AND member_id = ? AND date = ?", msg.GuildID, msg.Author.ID, today).First(&dm)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		_ = m.db.Create(&database.DailyMessage{
			GuildID:  msg.GuildID,
			MemberID: msg.Author.ID,
			Date:     today,
			Count:    1,
		}).Error
	} else if result.Error == nil {
		_ = m.db.Model(&dm).Update("count", dm.Count+1).Error
	}
}

func (m *Module) handleMessages(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	q := r.URL.Query()
	memberID := q.Get("member_id")
	dateStr := q.Get("date")
	startDateStr := q.Get("start_date")
	endDateStr := q.Get("end_date")

	type DailyMessageRow struct {
		Date     string `json:"date"`
		MemberID string `json:"member_id"`
		Count    int    `json:"count"`
	}

	query := m.db.Model(&database.DailyMessage{})
	if memberID != "" {
		query = query.Where("member_id = ?", memberID)
	}
	if dateStr != "" {
		d, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid date format, use YYYY-MM-DD"})
			return
		}
		query = query.Where("date = ?", d)
	} else if startDateStr != "" && endDateStr != "" {
		start, err := time.Parse("2006-01-02", startDateStr)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid start_date format, use YYYY-MM-DD"})
			return
		}
		end, err := time.Parse("2006-01-02", endDateStr)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid end_date format, use YYYY-MM-DD"})
			return
		}
		query = query.Where("date >= ? AND date <= ?", start, end)
	}

	var rows []DailyMessageRow
	query.Select("date, member_id, count as count").
		Order("date DESC").
		Scan(&rows)
	if rows == nil {
		rows = []DailyMessageRow{}
	}
	_ = json.NewEncoder(w).Encode(rows)
}

func (m *Module) getModRoleID(s *discordgo.Session, guildID string) (string, error) {
	if m.cfg.ModRoleID != "" {
		return m.cfg.ModRoleID, nil
	}
	if m.cfg.ModRoleName == "" {
		return "", errors.New("mod role ID or name is not configured")
	}
	return resolveRoleIDByName(s, guildID, m.cfg.ModRoleName)
}

func memberHasModRole(member *discordgo.Member, roleID string) bool {
	for _, rid := range member.Roles {
		if rid == roleID {
			return true
		}
	}
	return false
}

func resolveRoleIDByName(s *discordgo.Session, guildID, roleName string) (string, error) {
	roles, err := s.GuildRoles(guildID)
	if err != nil {
		return "", err
	}
	for _, role := range roles {
		if strings.EqualFold(role.Name, roleName) {
			return role.ID, nil
		}
	}
	return "", errors.New("role not found by name")
}
```

- [ ] **Step 5: Run tests**

```
go test ./...
```

Expected: `ok  github.com/brendanjhnsn/go-api/modules/dailymessages`

- [ ] **Step 6: Commit**

```
git add modules/dailymessages/
git commit -m "feat: add dailymessages module with message count tracking and /api/messages"
```

---

## Task 7: Rewrite main module and delete old code

**Files:**
- Modify: `go.mod`
- Modify: `API/main.go`
- Delete: `config/config.go`, `database/database.go`, `database/models.go`, `API/server/server.go`

- [ ] **Step 1: Update `go.mod`**

Replace the entire content of `go.mod` with:

```
module github.com/brendanjhnsn/go-api

go 1.22

require (
	github.com/brendanjhnsn/go-api/core v0.0.0
	github.com/brendanjhnsn/go-api/modules/dailymessages v0.0.0
	github.com/brendanjhnsn/go-api/modules/loa v0.0.0
	github.com/brendanjhnsn/go-api/modules/qfs v0.0.0
	github.com/brendanjhnsn/go-api/modules/tickets v0.0.0
	github.com/brendanjhnsn/go-api/modules/voicetime v0.0.0
	github.com/bwmarrin/discordgo v0.28.1
	github.com/joho/godotenv v1.5.1
	gorm.io/gorm v1.31.1
)

replace (
	github.com/brendanjhnsn/go-api/core => ./core
	github.com/brendanjhnsn/go-api/modules/dailymessages => ./modules/dailymessages
	github.com/brendanjhnsn/go-api/modules/loa => ./modules/loa
	github.com/brendanjhnsn/go-api/modules/qfs => ./modules/qfs
	github.com/brendanjhnsn/go-api/modules/tickets => ./modules/tickets
	github.com/brendanjhnsn/go-api/modules/voicetime => ./modules/voicetime
)
```

- [ ] **Step 2: Rewrite `API/main.go`**

```go
package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/brendanjhnsn/go-api/core/config"
	"github.com/brendanjhnsn/go-api/core/database"
	"github.com/brendanjhnsn/go-api/modules/dailymessages"
	"github.com/brendanjhnsn/go-api/modules/loa"
	"github.com/brendanjhnsn/go-api/modules/qfs"
	"github.com/brendanjhnsn/go-api/modules/tickets"
	"github.com/brendanjhnsn/go-api/modules/voicetime"
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

	loaMod     := loa.New(db, cfg)
	ticketsMod := tickets.New(db, cfg)
	voiceMod   := voicetime.New(db, cfg)
	qfsMod     := qfs.New(db, cfg)
	dailyMod   := dailymessages.New(db, cfg)

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

	go func() {
		log.Printf("Starting API server on port %s", cfg.ServerPort)
		if err := http.ListenAndServe(":"+cfg.ServerPort, mux); err != nil {
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

- [ ] **Step 3: Run go mod tidy and verify the root builds**

```
go mod tidy
go build ./...
```

Expected: no errors. `go.sum` will be updated to include transitive deps from all local modules.

- [ ] **Step 4: Delete old files**

```
Remove-Item -Force config\config.go
Remove-Item -Force database\database.go
Remove-Item -Force database\models.go
Remove-Item -Force API\server\server.go
```

Then remove the now-empty directories:

```
Remove-Item -Recurse -Force config
Remove-Item -Recurse -Force database
Remove-Item -Recurse -Force API\server
```

- [ ] **Step 5: Verify build still passes after deletions**

```
go build ./...
```

Expected: no errors.

- [ ] **Step 6: Commit**

```
git add -A
git commit -m "refactor: rewrite main as thin orchestrator, remove old config/database/server packages"
```
