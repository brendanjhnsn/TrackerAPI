package auth

import (
	"net/http"

	"github.com/brendanjhnsn/TrackerAPI/core/config"
	"gorm.io/gorm"
)

type Module struct {
	db  *gorm.DB
	cfg *config.Config
}

func New(db *gorm.DB, cfg *config.Config) *Module {
	return &Module{db: db, cfg: cfg}
}

func (m *Module) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/auth/discord/redirect", m.handleRedirect)
	mux.HandleFunc("/auth/discord/callback", m.handleCallback)
}

func (m *Module) handleRedirect(w http.ResponseWriter, r *http.Request) {}
func (m *Module) handleCallback(w http.ResponseWriter, r *http.Request) {}
