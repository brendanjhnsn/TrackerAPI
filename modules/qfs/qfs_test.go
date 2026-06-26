package qfs_test

import (
	"net/http"
	"testing"

	"github.com/brendanjhnsn/TrackerAPI/modules/qfs"
	"github.com/bwmarrin/discordgo"
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
