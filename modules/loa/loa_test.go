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
