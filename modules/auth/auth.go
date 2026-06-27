package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/brendanjhnsn/TrackerAPI/core/config"
	"github.com/brendanjhnsn/TrackerAPI/core/database"
	"gorm.io/gorm"
)

var discordClient = &http.Client{Timeout: 10 * time.Second}

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
	mux.HandleFunc("/auth/discord/logout", m.handleLogout)
}

func (m *Module) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeJSON(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if cookie, err := r.Cookie("session"); err == nil {
		m.db.Where("token = ?", cookie.Value).Delete(&database.Session{})
	}
	http.SetCookie(w, &http.Cookie{
		Name:   "session",
		Value:  "",
		MaxAge: -1,
		Path:   "/",
	})
	writeJSON(w, http.StatusOK, "logged out")
}

func (m *Module) handleRedirect(w http.ResponseWriter, r *http.Request) {
	state, err := randomHex(16)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, "internal error")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		MaxAge:   300,
		HttpOnly: true,
		Secure:   m.cfg.Environment == "production",
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	})
	authURL := fmt.Sprintf(
		"https://discord.com/oauth2/authorize?client_id=%s&redirect_uri=%s&response_type=code&scope=identify&state=%s",
		m.cfg.DiscordClientID,
		url.QueryEscape(m.cfg.DiscordRedirectURI),
		url.QueryEscape(state),
	)
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

func (m *Module) handleCallback(w http.ResponseWriter, r *http.Request) {
	stateCookie, err := r.Cookie("oauth_state")
	queryState := r.URL.Query().Get("state")
	if err != nil || subtle.ConstantTimeCompare([]byte(stateCookie.Value), []byte(queryState)) != 1 {
		writeJSON(w, http.StatusBadRequest, "invalid state")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:   "oauth_state",
		Value:  "",
		MaxAge: -1,
		Path:   "/",
	})

	code := r.URL.Query().Get("code")
	if code == "" {
		writeJSON(w, http.StatusBadRequest, "missing code")
		return
	}

	accessToken, err := m.exchangeCode(r, code)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, "token exchange failed")
		return
	}

	userID, err := m.getDiscordUserID(r, accessToken)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, "failed to get user info")
		return
	}

	sessionToken, err := randomHex(32)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, "internal error")
		return
	}

	sess := database.Session{
		Token:         sessionToken,
		DiscordUserID: userID,
		ExpiresAt:     time.Now().UTC().Add(24 * time.Hour),
	}
	if err := m.db.Create(&sess).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, "failed to create session")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    sessionToken,
		MaxAge:   86400,
		HttpOnly: true,
		Secure:   m.cfg.Environment == "production",
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	})
	http.Redirect(w, r, m.cfg.FrontendURL, http.StatusTemporaryRedirect)
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
}

func (m *Module) exchangeCode(r *http.Request, code string) (string, error) {
	body := url.Values{
		"client_id":     {m.cfg.DiscordClientID},
		"client_secret": {m.cfg.DiscordClientSecret},
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {m.cfg.DiscordRedirectURI},
	}.Encode()
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, "https://discord.com/api/oauth2/token", strings.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("building token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := discordClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		return "", fmt.Errorf("discord token endpoint returned %d", resp.StatusCode)
	}
	var tok tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return "", err
	}
	if tok.AccessToken == "" {
		return "", fmt.Errorf("empty access token from Discord")
	}
	return tok.AccessToken, nil
}

type discordUser struct {
	ID string `json:"id"`
}

func (m *Module) getDiscordUserID(r *http.Request, accessToken string) (string, error) {
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, "https://discord.com/api/users/@me", nil)
	if err != nil {
		return "", fmt.Errorf("building discord user request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := discordClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		return "", fmt.Errorf("discord users/@me returned %d", resp.StatusCode)
	}
	var user discordUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return "", err
	}
	if user.ID == "" {
		return "", fmt.Errorf("empty user ID from Discord")
	}
	return user.ID, nil
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
