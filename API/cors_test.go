package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

var passHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
})

func TestCORSMiddleware_Production_SetsHeaders(t *testing.T) {
	h := corsMiddleware("https://frontend.example.com", "production", passHandler)
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://frontend.example.com" {
		t.Errorf("want ACAO=https://frontend.example.com, got %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("want ACAC=true, got %q", got)
	}
}

func TestCORSMiddleware_Development_NoHeaders(t *testing.T) {
	h := corsMiddleware("https://frontend.example.com", "development", passHandler)
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("want no ACAO header in development, got %q", got)
	}
}

func TestCORSMiddleware_Options_Returns204InProduction(t *testing.T) {
	h := corsMiddleware("https://frontend.example.com", "production", passHandler)
	req := httptest.NewRequest(http.MethodOptions, "/api/me", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Errorf("want 204 for OPTIONS in production, got %d", rec.Code)
	}
}
