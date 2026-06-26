package voicetime_test

import (
	"net/http"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/brendanjhnsn/go-api/modules/voicetime"
)

var _ interface {
	Register(*discordgo.Session)
	RegisterRoutes(*http.ServeMux)
} = (*voicetime.Module)(nil)

func TestNew(t *testing.T) {
	if voicetime.New(nil, nil) == nil {
		t.Fatal("New returned nil")
	}
}
