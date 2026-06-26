package tickets_test

import (
	"net/http"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/brendanjhnsn/go-api/modules/tickets"
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
