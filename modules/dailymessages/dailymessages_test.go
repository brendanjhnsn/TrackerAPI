package dailymessages_test

import (
	"net/http"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/brendanjhnsn/go-api/modules/dailymessages"
)

var _ interface {
	Register(*discordgo.Session)
	RegisterRoutes(*http.ServeMux)
} = (*dailymessages.Module)(nil)

func TestNew(t *testing.T) {
	if dailymessages.New(nil, nil) == nil {
		t.Fatal("New returned nil")
	}
}
