package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/brendanjhnsn/TrackerAPI/core/config"
	"github.com/brendanjhnsn/TrackerAPI/core/database"
	"gorm.io/gorm"
)

var discordClient = &http.Client{Timeout: 10 * time.Second}

type discordProfile struct {
	Username  string
	AvatarURL string
}

type profileCacheEntry struct {
	profile   discordProfile
	expiresAt time.Time
}

type Module struct {
	db           *gorm.DB
	cfg          *config.Config
	profileMu    sync.RWMutex
	profileCache map[string]profileCacheEntry
	roleFetcher  *cachedRoleFetcher
}

func New(db *gorm.DB, cfg *config.Config) *Module {
	return &Module{
		db:           db,
		cfg:          cfg,
		profileCache: make(map[string]profileCacheEntry),
		roleFetcher:  newCachedRoleFetcher(newDiscordRoleFetcher(cfg.DiscordToken, cfg.DiscordGuildID), 5*time.Minute),
	}
}

func (m *Module) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/auth/discord/redirect", m.handleRedirect)
	mux.HandleFunc("/auth/discord/callback", m.handleCallback)
	mux.HandleFunc("/auth/discord/logout", m.handleLogout)
	mux.HandleFunc("/api/me", m.handleMe)
	mux.HandleFunc("/api/profiles", m.handleProfiles)
}

func (m *Module) handleProfiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	idsParam := r.URL.Query().Get("ids")
	type profileResult struct {
		ID        string `json:"id"`
		Username  string `json:"username"`
		AvatarURL string `json:"avatar_url"`
	}
	if idsParam == "" {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]profileResult{})
		return
	}
	rawIDs := strings.Split(idsParam, ",")
	results := make([]profileResult, 0, len(rawIDs))
	for _, id := range rawIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		p := m.fetchDiscordProfile(r.Context(), id)
		results = append(results, profileResult{ID: id, Username: p.Username, AvatarURL: p.AvatarURL})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(results)
}

func (m *Module) fetchDiscordProfile(ctx context.Context, userID string) discordProfile {
	m.profileMu.RLock()
	if e, ok := m.profileCache[userID]; ok && time.Now().Before(e.expiresAt) {
		p := e.profile
		m.profileMu.RUnlock()
		return p
	}
	m.profileMu.RUnlock()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://discord.com/api/users/"+userID, nil)
	if err != nil {
		return discordProfile{}
	}
	req.Header.Set("Authorization", "Bot "+m.cfg.DiscordToken)
	resp, err := discordClient.Do(req)
	if err != nil {
		return discordProfile{}
	}
	defer resp.Body.Close()

	var data struct {
		Username   string  `json:"username"`
		GlobalName *string `json:"global_name"`
		Avatar     string  `json:"avatar"`
	}
	if resp.StatusCode == http.StatusOK {
		_ = json.NewDecoder(resp.Body).Decode(&data)
	} else {
		_, _ = io.Copy(io.Discard, resp.Body)
	}

	name := data.Username
	if data.GlobalName != nil && *data.GlobalName != "" {
		name = *data.GlobalName
	}

	avatarURL := ""
	if data.Avatar != "" {
		avatarURL = fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png?size=64", userID, data.Avatar)
	}

	profile := discordProfile{Username: name, AvatarURL: avatarURL}

	m.profileMu.Lock()
	m.profileCache[userID] = profileCacheEntry{profile: profile, expiresAt: time.Now().Add(10 * time.Minute)}
	m.profileMu.Unlock()

	return profile
}

func (m *Module) handleMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	role, ok := RoleFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	userID, _ := UserIDFromContext(r.Context())
	roleStr := "mod"
	switch role {
	case RoleManager:
		roleStr = "manager"
	case RoleDirector:
		roleStr = "director"
	default:
		roleStr = "mod"
	}

	profile := m.fetchDiscordProfile(r.Context(), userID)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"discord_user_id": userID,
		"role":            roleStr,
		"username":        profile.Username,
		"avatar_url":      profile.AvatarURL,
	})
}

func (m *Module) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeJSON(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if cookie, err := r.Cookie("session"); err == nil {
		var sess database.Session
		if m.db.Where("token = ?", cookie.Value).First(&sess).Error == nil {
			m.roleFetcher.invalidate(sess.DiscordUserID)
		}
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
