package tickets_test

import (
	"net/http"
	"testing"

	"github.com/brendanjhnsn/TrackerAPI/modules/tickets"
	"github.com/bwmarrin/discordgo"
)

var _ interface {
	Register(*discordgo.Session)
	RegisterRoutes(*http.ServeMux)
} = (*tickets.Module)(nil)

func TestNew(t *testing.T) {
	if tickets.New(nil, nil) == nil {
		t.Fatal("New returned nil")
	}
}
