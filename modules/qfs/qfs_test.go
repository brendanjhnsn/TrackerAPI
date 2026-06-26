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
