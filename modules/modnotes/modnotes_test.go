package modnotes_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/brendanjhnsn/TrackerAPI/modules/auth"
	"github.com/brendanjhnsn/TrackerAPI/modules/modnotes"
)

func directorCtx() context.Context {
	return auth.NewContext(context.Background(), auth.RoleDirector, "dir123")
}

func managerCtx() context.Context {
	return auth.NewContext(context.Background(), auth.RoleManager, "mgr123")
}

func newMux() *http.ServeMux {
	m := modnotes.New(nil, nil)
	mux := http.NewServeMux()
	m.RegisterRoutes(mux)
	return mux
}

// --- deleteNote ---

func TestDeleteNote_NoAuth_Returns403(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/api/notes?id=1", nil)
	rec := httptest.NewRecorder()
	newMux().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("want 403, got %d", rec.Code)
	}
}

func TestDeleteNote_ManagerAllowed_MissingID_Returns400(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/api/notes", nil)
	req = req.WithContext(managerCtx())
	rec := httptest.NewRecorder()
	newMux().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestDeleteNote_InvalidID_Returns400(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/api/notes?id=abc", nil)
	req = req.WithContext(directorCtx())
	rec := httptest.NewRecorder()
	newMux().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

// --- deleteModAction ---

func TestDeleteAction_NoAuth_Returns403(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/api/mod-actions?id=1", nil)
	rec := httptest.NewRecorder()
	newMux().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("want 403, got %d", rec.Code)
	}
}

func TestDeleteAction_ManagerAllowed_MissingID_Returns400(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/api/mod-actions", nil)
	req = req.WithContext(managerCtx())
	rec := httptest.NewRecorder()
	newMux().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestDeleteAction_InvalidID_Returns400(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/api/mod-actions?id=xyz", nil)
	req = req.WithContext(directorCtx())
	rec := httptest.NewRecorder()
	newMux().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

// --- handleAuditLog ---

func TestAuditLog_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/audit-log", nil)
	req = req.WithContext(directorCtx())
	rec := httptest.NewRecorder()
	newMux().ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("want 405, got %d", rec.Code)
	}
}

func TestAuditLog_NonDirector_Returns403(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/audit-log?type=notes", nil)
	req = req.WithContext(managerCtx())
	rec := httptest.NewRecorder()
	newMux().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("want 403, got %d", rec.Code)
	}
}

func TestAuditLog_MissingType_Returns400(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/audit-log", nil)
	req = req.WithContext(directorCtx())
	rec := httptest.NewRecorder()
	newMux().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestAuditLog_InvalidType_Returns400(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/audit-log?type=invalid", nil)
	req = req.WithContext(directorCtx())
	rec := httptest.NewRecorder()
	newMux().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

// --- handleModRestore ---

func TestModRestore_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/mod-restore", nil)
	req = req.WithContext(directorCtx())
	rec := httptest.NewRecorder()
	newMux().ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("want 405, got %d", rec.Code)
	}
}

func TestModRestore_NonDirector_Returns403(t *testing.T) {
	body, _ := json.Marshal(map[string]string{"member_id": "123"})
	req := httptest.NewRequest(http.MethodPost, "/api/mod-restore", bytes.NewReader(body))
	req = req.WithContext(managerCtx())
	rec := httptest.NewRecorder()
	newMux().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("want 403, got %d", rec.Code)
	}
}

func TestModRestore_MissingMemberID_Returns400(t *testing.T) {
	body, _ := json.Marshal(map[string]string{})
	req := httptest.NewRequest(http.MethodPost, "/api/mod-restore", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(directorCtx())
	rec := httptest.NewRecorder()
	newMux().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}
