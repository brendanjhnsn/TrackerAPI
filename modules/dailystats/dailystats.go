package dailystats

import (
	"encoding/json"
	"net/http"
	"sort"
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
	mux.HandleFunc("/api/daily-stats", m.handleDailyStats)
}

// dateExpr returns a SQL expression that extracts the calendar date from col,
// applying the configured timezone so late-evening activity lands on the correct day.
// Only safe timezone name characters are allowed; falls back to plain DATE() on bad input.
func (m *Module) dateExpr(col string) string {
	tz := m.cfg.Timezone
	if tz == "" || tz == "UTC" {
		return "DATE(" + col + ")"
	}
	for _, c := range tz {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == '/' || c == '_' || c == '-' || c == '+') {
			return "DATE(" + col + ")"
		}
	}
	return "DATE(" + col + " AT TIME ZONE '" + tz + "')"
}

type DayStats struct {
	Date       string  `json:"date"`
	Messages   int     `json:"messages"`
	Tickets    int     `json:"tickets"`
	QA         int     `json:"qa"`
	VoiceHours float64 `json:"voice_hours"`
	Warning    int     `json:"warning"`
	Timeout    int     `json:"timeout"`
	Kick       int     `json:"kick"`
	Ban        int     `json:"ban"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (m *Module) handleDailyStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query()
	memberID := q.Get("member_id")
	startStr := q.Get("start_date")
	endStr := q.Get("end_date")

	var startDate, endDate time.Time
	var err error

	if startStr != "" {
		startDate, err = time.Parse("2006-01-02", startStr)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid start_date"})
			return
		}
	} else {
		startDate = time.Now().UTC().AddDate(0, 0, -30)
	}

	if endStr != "" {
		endDate, err = time.Parse("2006-01-02", endStr)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid end_date"})
			return
		}
	} else {
		endDate = time.Now().UTC()
	}

	startDate = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, time.UTC)
	endDate = time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 23, 59, 59, 999999999, time.UTC)

	days := map[string]*DayStats{}
	ensure := func(date string) *DayStats {
		if days[date] == nil {
			days[date] = &DayStats{Date: date}
		}
		return days[date]
	}

	// Messages
	{
		type row struct {
			Date  time.Time `gorm:"column:date"`
			Total int       `gorm:"column:total"`
		}
		var rows []row
		tx := m.db.Model(&database.DailyMessage{}).
			Select("date, SUM(count) as total").
			Where("date >= ? AND date <= ?", startDate, endDate).
			Group("date")
		if memberID != "" {
			tx = tx.Where("member_id = ?", memberID)
		}
		if tx.Scan(&rows).Error == nil {
			for _, r := range rows {
				ensure(r.Date.UTC().Format("2006-01-02")).Messages += r.Total
			}
		}
	}

	// Tickets (first responses)
	{
		type row struct {
			Date  time.Time `gorm:"column:date"`
			Total int       `gorm:"column:total"`
		}
		var rows []row
		dateE := m.dateExpr("first_resp_date")
		tx := m.db.Model(&database.Ticket{}).
			Select(dateE+" as date, COUNT(*) as total").
			Where("first_resp_date >= ? AND first_resp_date <= ?", startDate, endDate).
			Where("first_resp_id IS NOT NULL").
			Group(dateE)
		if memberID != "" {
			tx = tx.Where("first_resp_id = ?", memberID)
		}
		if tx.Scan(&rows).Error == nil {
			for _, r := range rows {
				ensure(r.Date.UTC().Format("2006-01-02")).Tickets += r.Total
			}
		}
	}

	// Q&A (QuestionCheck)
	{
		type row struct {
			Date  time.Time `gorm:"column:date"`
			Total int       `gorm:"column:total"`
		}
		var rows []row
		tx := m.db.Model(&database.QuestionCheck{}).
			Select("DATE(date) as date, COUNT(*) as total").
			Where("date >= ? AND date <= ? AND removed_at IS NULL", startDate, endDate).
			Group("DATE(date)")
		if memberID != "" {
			tx = tx.Where("member_id = ?", memberID)
		}
		if tx.Scan(&rows).Error == nil {
			for _, r := range rows {
				ensure(r.Date.UTC().Format("2006-01-02")).QA += r.Total
			}
		}
	}

	// Voice (sum duration in seconds → hours)
	{
		type row struct {
			Date         time.Time `gorm:"column:date"`
			TotalSeconds int64     `gorm:"column:total_seconds"`
		}
		var rows []row
		dateE := m.dateExpr("joined_at")
		tx := m.db.Model(&database.VoiceTime{}).
			Select(dateE+" as date, SUM(duration) as total_seconds").
			Where("joined_at >= ? AND joined_at <= ?", startDate, endDate).
			Where("duration > 0").
			Group(dateE)
		if memberID != "" {
			tx = tx.Where("member_id = ?", memberID)
		}
		if tx.Scan(&rows).Error == nil {
			for _, r := range rows {
				ensure(r.Date.UTC().Format("2006-01-02")).VoiceHours += float64(r.TotalSeconds) / 3600
			}
		}
	}

	// Issued actions (warning / timeout / kick / ban)
	{
		type row struct {
			Date       time.Time `gorm:"column:date"`
			ActionType string    `gorm:"column:action_type"`
			Total      int       `gorm:"column:total"`
		}
		var rows []row
		dateE := m.dateExpr("issued_at")
		tx := m.db.Model(&database.ModIssuedAction{}).
			Select(dateE+" as date, action_type, COUNT(*) as total").
			Where("issued_at >= ? AND issued_at <= ?", startDate, endDate).
			Group(dateE + ", action_type")
		if memberID != "" {
			tx = tx.Where("mod_member_id = ?", memberID)
		}
		if tx.Scan(&rows).Error == nil {
			for _, r := range rows {
				day := ensure(r.Date.UTC().Format("2006-01-02"))
				switch r.ActionType {
				case "warning":
					day.Warning += r.Total
				case "timeout":
					day.Timeout += r.Total
				case "kick":
					day.Kick += r.Total
				case "ban":
					day.Ban += r.Total
				}
			}
		}
	}

	result := make([]DayStats, 0, len(days))
	for _, d := range days {
		result = append(result, *d)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Date < result[j].Date
	})

	writeJSON(w, http.StatusOK, result)
}
