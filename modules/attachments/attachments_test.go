package attachments_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/brendanjhnsn/TrackerAPI/modules/attachments"
	"github.com/brendanjhnsn/TrackerAPI/modules/auth"
)

func managerCtx() context.Context {
	return auth.NewContext(context.Background(), auth.RoleManager, "mgr1")
}

func directorCtx() context.Context {
	return auth.NewContext(context.Background(), auth.RoleDirector, "dir1")
}

func modCtx() context.Context {
	return auth.NewContext(context.Background(), auth.RoleMod, "mod1")
}

func newMux() *http.ServeMux {
	m := attachments.New(nil, nil)
	mux := http.NewServeMux()
	m.RegisterRoutes(mux)
	return mux
}

// --- /api/attachments (list) ---

func TestList_NoAuth_Returns403(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/attachments?owner_type=note&owner_ids=1", nil)
	rec := httptest.NewRecorder()
	newMux().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("want 403, got %d", rec.Code)
	}
}

func TestList_ModRole_Returns403(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/attachments?owner_type=note&owner_ids=1", nil)
	req = req.WithContext(modCtx())
	rec := httptest.NewRecorder()
	newMux().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("want 403, got %d", rec.Code)
	}
}

func TestList_MissingOwnerType_Returns400(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/attachments?owner_ids=1", nil)
	req = req.WithContext(managerCtx())
	rec := httptest.NewRecorder()
	newMux().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestList_InvalidOwnerType_Returns400(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/attachments?owner_type=comment&owner_ids=1", nil)
	req = req.WithContext(managerCtx())
	rec := httptest.NewRecorder()
	newMux().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestList_MissingOwnerIDs_Returns400(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/attachments?owner_type=note", nil)
	req = req.WithContext(managerCtx())
	rec := httptest.NewRecorder()
	newMux().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

// --- /api/attachments (upload) ---

func TestUpload_NoAuth_Returns403(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/attachments?owner_type=note&owner_id=1", nil)
	rec := httptest.NewRecorder()
	newMux().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("want 403, got %d", rec.Code)
	}
}

func TestUpload_MissingOwnerType_Returns400(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/attachments?owner_id=1", nil)
	req = req.WithContext(managerCtx())
	rec := httptest.NewRecorder()
	newMux().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestUpload_InvalidOwnerType_Returns400(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/attachments?owner_type=bad&owner_id=1", nil)
	req = req.WithContext(managerCtx())
	rec := httptest.NewRecorder()
	newMux().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestUpload_MissingOwnerID_Returns400(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/attachments?owner_type=note", nil)
	req = req.WithContext(managerCtx())
	rec := httptest.NewRecorder()
	newMux().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestUpload_InvalidOwnerID_Returns400(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/attachments?owner_type=note&owner_id=abc", nil)
	req = req.WithContext(managerCtx())
	rec := httptest.NewRecorder()
	newMux().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

// --- /api/attachments (delete) ---

func TestDelete_NoAuth_Returns403(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/api/attachments?id=1", nil)
	rec := httptest.NewRecorder()
	newMux().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("want 403, got %d", rec.Code)
	}
}

func TestDelete_ModRole_Returns403(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/api/attachments?id=1", nil)
	req = req.WithContext(modCtx())
	rec := httptest.NewRecorder()
	newMux().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("want 403, got %d", rec.Code)
	}
}

func TestDelete_MissingID_Returns400(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/api/attachments", nil)
	req = req.WithContext(managerCtx())
	rec := httptest.NewRecorder()
	newMux().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestDelete_InvalidID_Returns400(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/api/attachments?id=xyz", nil)
	req = req.WithContext(directorCtx())
	rec := httptest.NewRecorder()
	newMux().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

// --- /api/attachments (method not allowed) ---

func TestAttachments_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPatch, "/api/attachments", nil)
	req = req.WithContext(managerCtx())
	rec := httptest.NewRecorder()
	newMux().ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("want 405, got %d", rec.Code)
	}
}

// --- /api/attachments/file ---

func TestServe_NoAuth_Returns403(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/attachments/file?id=1", nil)
	rec := httptest.NewRecorder()
	newMux().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("want 403, got %d", rec.Code)
	}
}

func TestServe_MissingID_Returns400(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/attachments/file", nil)
	req = req.WithContext(managerCtx())
	rec := httptest.NewRecorder()
	newMux().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestServe_InvalidID_Returns400(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/attachments/file?id=bad", nil)
	req = req.WithContext(managerCtx())
	rec := httptest.NewRecorder()
	newMux().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}
