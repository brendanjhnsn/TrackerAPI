package loa_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/brendanjhnsn/TrackerAPI/modules/loa"
)

var _ interface{ RegisterRoutes(*http.ServeMux) } = (*loa.Module)(nil)

func TestNew(t *testing.T) {
	if loa.New(nil, nil) == nil {
		t.Fatal("New returned nil")
	}
}

func TestCreateLOA_MissingGuildID_Returns400(t *testing.T) {
	m := loa.New(nil, nil)
	mux := http.NewServeMux()
	m.RegisterRoutes(mux)
	body, _ := json.Marshal(map[string]string{
		"member_id":  "user123",
		"start_date": "2026-07-01",
		"end_date":   "2026-07-07",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/loa", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestCreateLOA_MissingMemberID_Returns400(t *testing.T) {
	m := loa.New(nil, nil)
	mux := http.NewServeMux()
	m.RegisterRoutes(mux)
	body, _ := json.Marshal(map[string]string{
		"guild_id":   "guild123",
		"start_date": "2026-07-01",
		"end_date":   "2026-07-07",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/loa", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestCreateLOA_InvalidStartDate_Returns400(t *testing.T) {
	m := loa.New(nil, nil)
	mux := http.NewServeMux()
	m.RegisterRoutes(mux)
	body, _ := json.Marshal(map[string]string{
		"guild_id":   "guild123",
		"member_id":  "user123",
		"start_date": "not-a-date",
		"end_date":   "2026-07-07",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/loa", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestCreateLOA_EndBeforeStart_Returns400(t *testing.T) {
	m := loa.New(nil, nil)
	mux := http.NewServeMux()
	m.RegisterRoutes(mux)
	body, _ := json.Marshal(map[string]string{
		"guild_id":   "guild123",
		"member_id":  "user123",
		"start_date": "2026-07-07",
		"end_date":   "2026-07-01",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/loa", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}
