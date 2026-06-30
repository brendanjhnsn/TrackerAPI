package permissions_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/brendanjhnsn/TrackerAPI/core/config"
	"github.com/brendanjhnsn/TrackerAPI/modules/permissions"
)

func TestCanAccess_Director_AlwaysTrue(t *testing.T) {
	// isDirector=true short-circuits before touching db — db=nil is safe
	if !permissions.CanAccess(nil, true, "any-manager", "moderators") {
		t.Error("director should always have access")
	}
}

func TestCanAccess_NilDB_ReturnsFalse(t *testing.T) {
	// non-director with nil db (no permission row found) → false
	if permissions.CanAccess(nil, false, "manager123", "moderators") {
		t.Error("nil db should return false")
	}
}

func TestHandleManagerPermissions_NoRole_Returns403(t *testing.T) {
	m := permissions.New(nil, &config.Config{})
	mux := http.NewServeMux()
	m.RegisterRoutes(mux)
	req := httptest.NewRequest(http.MethodGet, "/api/manager-permissions", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("want 403, got %d", rec.Code)
	}
}

func TestHandleManagerPermissions_PUT_NoRole_Returns403(t *testing.T) {
	m := permissions.New(nil, &config.Config{})
	mux := http.NewServeMux()
	m.RegisterRoutes(mux)
	req := httptest.NewRequest(http.MethodPut, "/api/manager-permissions", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("want 403, got %d", rec.Code)
	}
}

func TestHandleManagerPermissions_DELETE_Returns405(t *testing.T) {
	// unsupported method is rejected before role check
	m := permissions.New(nil, &config.Config{})
	mux := http.NewServeMux()
	m.RegisterRoutes(mux)
	req := httptest.NewRequest(http.MethodDelete, "/api/manager-permissions", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("want 405, got %d", rec.Code)
	}
}
