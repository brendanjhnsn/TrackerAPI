package gameleads_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/brendanjhnsn/TrackerAPI/core/config"
	"github.com/brendanjhnsn/TrackerAPI/modules/gameleads"
)

func newModule() *gameleads.Module {
	return gameleads.New(nil, &config.Config{})
}

func TestHandleGameLeads_NoRole_Returns403(t *testing.T) {
	m := newModule()
	mux := http.NewServeMux()
	m.RegisterRoutes(mux)
	req := httptest.NewRequest(http.MethodGet, "/api/game-leads", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("want 403, got %d", rec.Code)
	}
}

func TestHandleGameLeads_WrongMethod_Returns405(t *testing.T) {
	m := newModule()
	mux := http.NewServeMux()
	m.RegisterRoutes(mux)
	req := httptest.NewRequest(http.MethodPost, "/api/game-leads", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("want 405, got %d", rec.Code)
	}
}

func TestHandleChannels_NoRole_Returns403(t *testing.T) {
	m := newModule()
	mux := http.NewServeMux()
	m.RegisterRoutes(mux)
	req := httptest.NewRequest(http.MethodGet, "/api/game-leads/channels", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("want 403, got %d", rec.Code)
	}
}

func TestHandleChannels_WrongMethod_Returns405(t *testing.T) {
	m := newModule()
	mux := http.NewServeMux()
	m.RegisterRoutes(mux)
	req := httptest.NewRequest(http.MethodPost, "/api/game-leads/channels", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("want 405, got %d", rec.Code)
	}
}
