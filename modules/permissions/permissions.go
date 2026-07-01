package permissions

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/brendanjhnsn/TrackerAPI/core/config"
	"github.com/brendanjhnsn/TrackerAPI/core/database"
	"github.com/brendanjhnsn/TrackerAPI/core/discordapi"
	"github.com/brendanjhnsn/TrackerAPI/modules/auth"
	"gorm.io/gorm"
)

var validSections = map[string]bool{
	"moderators":       true,
	"management_panel": true,
	"game_leads":       true,
}

var validModSections = map[string]bool{
	"moderators": true,
	"game_leads": true,
}

type Module struct {
	db  *gorm.DB
	cfg *config.Config
}

func New(db *gorm.DB, cfg *config.Config) *Module {
	return &Module{db: db, cfg: cfg}
}

func (m *Module) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/manager-permissions", m.handleManagerPermissions)
	mux.HandleFunc("/api/mod-permissions", m.handleModPermissions)
	mux.HandleFunc("/api/all-role-permissions", m.handleAllRolePermissions)
}

// CanAccess reports whether the requesting user may access the given section.
// Pass isDirector=true when the caller's role is director — directors always have access.
// Pass isDirector=false for managers, which triggers a DB lookup (missing row → false).
// db=nil always returns false (treats as no permission row found).
func CanAccess(db *gorm.DB, isDirector bool, managerID, section string) bool {
	if isDirector {
		return true
	}
	if db == nil {
		return false
	}
	var perm database.ManagerPermission
	err := db.Where("manager_id = ? AND section = ?", managerID, section).First(&perm).Error
	if err != nil {
		return false
	}
	return perm.Enabled
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (m *Module) isDirector(r *http.Request) bool {
	return auth.IsDirectorContext(r)
}

func (m *Module) handleManagerPermissions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		m.getManagerPermissions(w, r)
	case http.MethodPut:
		m.putManagerPermission(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

type permissionRow struct {
	ManagerID   string          `json:"manager_id"`
	Permissions map[string]bool `json:"permissions"`
}

func (m *Module) getManagerPermissions(w http.ResponseWriter, r *http.Request) {
	if !m.isDirector(r) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "director role required"})
		return
	}
	managerIDs, err := discordapi.ListMembersWithRole(
		r.Context(),
		&http.Client{Timeout: 10 * time.Second},
		discordapi.DefaultBaseURL,
		m.cfg.DiscordToken,
		m.cfg.DiscordGuildID,
		m.cfg.ManagerRoleID,
		1000,
	)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "could not fetch manager list from Discord"})
		return
	}
	if len(managerIDs) == 0 {
		writeJSON(w, http.StatusOK, []permissionRow{})
		return
	}

	var perms []database.ManagerPermission
	if err := m.db.Where("manager_id IN ?", managerIDs).Find(&perms).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "database error"})
		return
	}

	permMap := map[string]map[string]bool{}
	for _, id := range managerIDs {
		permMap[id] = map[string]bool{"moderators": false, "management_panel": false, "game_leads": false}
	}
	for _, p := range perms {
		if _, ok := permMap[p.ManagerID]; ok {
			permMap[p.ManagerID][p.Section] = p.Enabled
		}
	}

	result := make([]permissionRow, 0, len(managerIDs))
	for _, id := range managerIDs {
		result = append(result, permissionRow{ManagerID: id, Permissions: permMap[id]})
	}
	writeJSON(w, http.StatusOK, result)
}

type putPermissionRequest struct {
	ManagerID string `json:"manager_id"`
	Section   string `json:"section"`
	Enabled   bool   `json:"enabled"`
}

type directorRow struct {
	MemberID string `json:"member_id"`
}

type modPermissionRow struct {
	MemberID    string          `json:"member_id"`
	Permissions map[string]bool `json:"permissions"`
}

type allRolePermissionsResponse struct {
	Directors []directorRow      `json:"directors"`
	Managers  []permissionRow    `json:"managers"`
	Mods      []modPermissionRow `json:"mods"`
}

func (m *Module) putManagerPermission(w http.ResponseWriter, r *http.Request) {
	if !m.isDirector(r) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "director role required"})
		return
	}
	var req putPermissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.ManagerID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "manager_id is required"})
		return
	}
	if !validSections[req.Section] {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "section must be one of: moderators, management_panel, game_leads"})
		return
	}

	var perm database.ManagerPermission
	err := m.db.Where("manager_id = ? AND section = ?", req.ManagerID, req.Section).First(&perm).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		perm = database.ManagerPermission{ManagerID: req.ManagerID, Section: req.Section, Enabled: req.Enabled}
		if err := m.db.Create(&perm).Error; err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create permission"})
			return
		}
	} else if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "database error"})
		return
	} else {
		perm.Enabled = req.Enabled
		if err := m.db.Save(&perm).Error; err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update permission"})
			return
		}
	}
	writeJSON(w, http.StatusOK, perm)
}

// --- GET /api/all-role-permissions ---

func (m *Module) handleAllRolePermissions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	m.getAllRolePermissions(w, r)
}

func (m *Module) getAllRolePermissions(w http.ResponseWriter, r *http.Request) {
	if !m.isDirector(r) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "director role required"})
		return
	}

	client := &http.Client{Timeout: 10 * time.Second}

	type roleResult struct {
		ids []string
		err error
	}
	dirCh := make(chan roleResult, 1)
	mgrCh := make(chan roleResult, 1)
	modCh := make(chan roleResult, 1)

	fetch := func(roleID string, ch chan<- roleResult) {
		ids, err := discordapi.ListMembersWithRole(
			r.Context(), client, discordapi.DefaultBaseURL,
			m.cfg.DiscordToken, m.cfg.DiscordGuildID, roleID, 1000,
		)
		ch <- roleResult{ids, err}
	}
	go fetch(m.cfg.DirectorRoleID, dirCh)
	go fetch(m.cfg.ManagerRoleID, mgrCh)
	go fetch(m.cfg.ModRoleID, modCh)

	dirRes := <-dirCh
	mgrRes := <-mgrCh
	modRes := <-modCh

	if dirRes.err != nil || mgrRes.err != nil || modRes.err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "could not fetch role lists from Discord"})
		return
	}

	directorIDs := dirRes.ids
	managerIDs := mgrRes.ids
	modIDs := modRes.ids

	// Query manager permissions
	var managerPerms []database.ManagerPermission
	if len(managerIDs) > 0 {
		if err := m.db.Where("manager_id IN ?", managerIDs).Find(&managerPerms).Error; err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "database error"})
			return
		}
	}

	// Query mod permissions
	var modPerms []database.ModPermission
	if len(modIDs) > 0 {
		if err := m.db.Where("member_id IN ?", modIDs).Find(&modPerms).Error; err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "database error"})
			return
		}
	}

	// Build directors
	directors := make([]directorRow, 0, len(directorIDs))
	for _, id := range directorIDs {
		directors = append(directors, directorRow{MemberID: id})
	}

	// Build managers (default all false, overlay DB values)
	mgrPermMap := make(map[string]map[string]bool, len(managerIDs))
	for _, id := range managerIDs {
		mgrPermMap[id] = map[string]bool{"moderators": false, "management_panel": false, "game_leads": false}
	}
	for _, p := range managerPerms {
		if row, ok := mgrPermMap[p.ManagerID]; ok {
			row[p.Section] = p.Enabled
		}
	}
	managers := make([]permissionRow, 0, len(managerIDs))
	for _, id := range managerIDs {
		managers = append(managers, permissionRow{ManagerID: id, Permissions: mgrPermMap[id]})
	}

	// Build mods (default all false, overlay DB values)
	modPermMap := make(map[string]map[string]bool, len(modIDs))
	for _, id := range modIDs {
		modPermMap[id] = map[string]bool{"moderators": false, "game_leads": false}
	}
	for _, p := range modPerms {
		if row, ok := modPermMap[p.MemberID]; ok {
			row[p.Section] = p.Enabled
		}
	}
	mods := make([]modPermissionRow, 0, len(modIDs))
	for _, id := range modIDs {
		mods = append(mods, modPermissionRow{MemberID: id, Permissions: modPermMap[id]})
	}

	writeJSON(w, http.StatusOK, allRolePermissionsResponse{
		Directors: directors,
		Managers:  managers,
		Mods:      mods,
	})
}

// --- PUT /api/mod-permissions ---

func (m *Module) handleModPermissions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	m.putModPermission(w, r)
}

type putModPermissionRequest struct {
	MemberID string `json:"member_id"`
	Section  string `json:"section"`
	Enabled  bool   `json:"enabled"`
}

func (m *Module) putModPermission(w http.ResponseWriter, r *http.Request) {
	if !m.isDirector(r) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "director role required"})
		return
	}
	var req putModPermissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.MemberID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "member_id is required"})
		return
	}
	if !validModSections[req.Section] {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "section must be one of: moderators, game_leads"})
		return
	}

	var perm database.ModPermission
	err := m.db.Where("member_id = ? AND section = ?", req.MemberID, req.Section).First(&perm).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		perm = database.ModPermission{MemberID: req.MemberID, Section: req.Section, Enabled: req.Enabled}
		if err := m.db.Create(&perm).Error; err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create permission"})
			return
		}
	} else if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "database error"})
		return
	} else {
		perm.Enabled = req.Enabled
		if err := m.db.Save(&perm).Error; err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update permission"})
			return
		}
	}
	writeJSON(w, http.StatusOK, perm)
}
