package modnotes

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/brendanjhnsn/TrackerAPI/core/config"
	"github.com/brendanjhnsn/TrackerAPI/core/database"
	"github.com/brendanjhnsn/TrackerAPI/modules/auth"
	"github.com/brendanjhnsn/TrackerAPI/modules/permissions"
	"github.com/bwmarrin/discordgo"
	"gorm.io/gorm"
)

// Official YAGPDB application ID. Change this if using a self-hosted YAGPDB instance.
const yagpdbAppID = "204255221017214977"

var snowflakeRe    = regexp.MustCompile(`\d{17,19}`)
var targetInDescRe = regexp.MustCompile(`\*\(ID (\d{17,19})\)\*`)

type Module struct {
	db  *gorm.DB
	cfg *config.Config
}

func New(db *gorm.DB, cfg *config.Config) *Module {
	return &Module{db: db, cfg: cfg}
}

// Register adds Discord event handlers to the session.
func (m *Module) Register(s *discordgo.Session) {
	s.AddHandler(m.onModLogMessage)
	s.AddHandler(m.onAuditLogEntry)
}

// onModLogMessage fires on every message in the configured modlog channel.
// It records:
//   - ModIssuedAction: always, when a moderator issues any action (tracks their activity stats)
//   - ModAction: only when the *target* is a staff member (tracks discipline received)
func (m *Module) onModLogMessage(s *discordgo.Session, msg *discordgo.MessageCreate) {
	if m.cfg.ModLogChannelID == "" {
		return
	}
	inChannel := false
	for _, id := range strings.Split(m.cfg.ModLogChannelID, ",") {
		if strings.TrimSpace(id) == msg.ChannelID {
			inChannel = true
			break
		}
	}
	if !inChannel {
		return
	}
	// Accept messages from the YAGPDB bot OR from webhooks in the channel
	// (some YAGPDB setups post via webhook with a different author ID).
	if msg.Author == nil {
		return
	}
	isYAGPDB := msg.Author.ID == yagpdbAppID || msg.WebhookID != ""
	if !isYAGPDB || len(msg.Embeds) == 0 {
		return
	}

	embed := msg.Embeds[0]

	// YAGPDB (default config) puts everything in the description:
	//   "**⚠Warned** username *(ID TARGET_ID)*\n📄**Reason:** ..."
	// and the moderator in the embed author:
	//   "moderatorname (ID MOD_ID)"
	// Some self-hosted configs use embed fields instead — we try fields first, then fall back.

	// Action type: try title → description (author name is the mod, not the action)
	titleToCheck := embed.Title
	if titleToCheck == "" {
		titleToCheck = embed.Description
	}
	actionType := parseModActionType(titleToCheck)
	if actionType == "" {
		log.Printf("[MODLOG] Could not detect action type. title=%q desc=%q", embed.Title, embed.Description)
		return
	}

	var targetID, moderatorID, reason string

	// Try embed fields first (some YAGPDB configs use them)
	for _, f := range embed.Fields {
		lower := strings.ToLower(f.Name)
		switch {
		case strings.Contains(lower, "moderator") || strings.Contains(lower, "responsible") || strings.Contains(lower, "staff"):
			moderatorID = extractSnowflake(f.Value)
		case lower == "user" || lower == "member" || lower == "target":
			targetID = extractSnowflake(f.Value)
		case lower == "reason":
			reason = f.Value
		}
	}

	// Fall back to description parsing: "**Action** name *(ID SNOWFLAKE)*\n...Reason:**..."
	if targetID == "" && embed.Description != "" {
		targetID = extractTargetFromDesc(embed.Description)
	}
	if reason == "" && embed.Description != "" {
		reason = extractReasonFromDesc(embed.Description)
	}

	// Moderator is the embed author: "username (ID SNOWFLAKE)"
	if moderatorID == "" && embed.Author != nil {
		moderatorID = extractSnowflake(embed.Author.Name)
	}

	log.Printf("[MODLOG] %s by=%s target=%s reason=%q", actionType, moderatorID, targetID, reason)

	// Always record issued action so we can show how many actions each mod has done.
	if moderatorID != "" {
		issued := database.ModIssuedAction{
			ModMemberID: moderatorID,
			ActionType:  actionType,
			Reason:      reason,
			IssuedAt:    time.Now().UTC(),
		}
		if err := m.db.Create(&issued).Error; err != nil {
			log.Printf("[MODLOG] Failed to save issued action: %v", err)
		}
	}

	// Record discipline only when the target holds a staff role.
	if targetID == "" {
		return
	}
	member, err := s.GuildMember(msg.GuildID, targetID)
	if err != nil {
		log.Printf("[MODLOG] GuildMember(%s) error: %v", targetID, err)
		return
	}
	isStaff := false
	for _, roleID := range member.Roles {
		if roleID == m.cfg.ModRoleID || roleID == m.cfg.ManagerRoleID || roleID == m.cfg.DirectorRoleID {
			isStaff = true
			break
		}
	}
	if !isStaff {
		return
	}
	action := database.ModAction{
		ModMemberID:    targetID,
		AuthorMemberID: moderatorID,
		ActionType:     actionType,
		Reason:         reason,
		IssuedAt:       time.Now().UTC(),
	}
	if err := m.db.Create(&action).Error; err != nil {
		log.Printf("[MODLOG] Failed to save mod action: %v", err)
	} else {
		log.Printf("[MODLOG] Discipline recorded: %s against staff %s", actionType, targetID)
	}
}

func parseModActionType(title string) string {
	lower := strings.ToLower(title)
	switch {
	case strings.Contains(lower, "warn"):
		return "warning"
	case strings.Contains(lower, "timeout"), strings.Contains(lower, "timed out"), strings.Contains(lower, "muted"):
		return "timeout"
	case strings.Contains(lower, "kick"):
		return "kick"
	case strings.Contains(lower, "ban"):
		return "ban"
	default:
		return ""
	}
}

// extractSnowflake pulls the first Discord snowflake ID (17–19 digits) from s.
func extractSnowflake(s string) string {
	return snowflakeRe.FindString(s)
}

// extractTargetFromDesc finds the target user ID from YAGPDB description format:
// "**⚠Warned** username *(ID SNOWFLAKE)*\n📄**Reason:** ..."
func extractTargetFromDesc(desc string) string {
	if m := targetInDescRe.FindStringSubmatch(desc); len(m) > 1 {
		return m[1]
	}
	return extractSnowflake(desc)
}

// extractReasonFromDesc pulls the reason text from YAGPDB description format.
// Strips the trailing ([Logs](...)) link if present.
func extractReasonFromDesc(desc string) string {
	idx := strings.Index(desc, "Reason:**")
	if idx == -1 {
		return ""
	}
	rest := strings.TrimSpace(desc[idx+len("Reason:**"):])
	if i := strings.LastIndex(rest, " (["); i != -1 {
		rest = strings.TrimSpace(rest[:i])
	}
	return rest
}

// onAuditLogEntry captures kicks, bans, and timeouts issued directly through
// Discord (right-click context menu) rather than through YAGPDB commands.
// YAGPDB-initiated actions are skipped here because the modlog channel listener
// handles those and correctly attributes them to the human moderator.
func (m *Module) onAuditLogEntry(s *discordgo.Session, entry *discordgo.GuildAuditLogEntryCreate) {
	// GuildAuditLogEntryCreate has no GuildID field in discordgo v0.28.1;
	// the bot only connects to one guild so no guild filter is needed.

	// YAGPDB is the actor when it executes slash commands — let the modlog listener handle those.
	if entry.UserID == yagpdbAppID {
		return
	}
	if entry.ActionType == nil {
		return
	}

	var actionType string
	switch *entry.ActionType {
	case discordgo.AuditLogActionMemberKick:
		actionType = "kick"
	case discordgo.AuditLogActionMemberBanAdd:
		actionType = "ban"
	case discordgo.AuditLogActionMemberUpdate:
		// Timeout is a member update that sets communication_disabled_until to a future time.
		for _, change := range entry.Changes {
			if change.Key != nil &&
				*change.Key == discordgo.AuditLogChangeKey("communication_disabled_until") &&
				change.NewValue != nil {
				actionType = "timeout"
				break
			}
		}
	}
	if actionType == "" {
		return
	}

	moderatorID := entry.UserID
	targetID := entry.TargetID
	reason := entry.Reason

	log.Printf("[AUDIT] %s by=%s target=%s reason=%q", actionType, moderatorID, targetID, reason)

	if moderatorID != "" {
		issued := database.ModIssuedAction{
			ModMemberID: moderatorID,
			ActionType:  actionType,
			Reason:      reason,
			IssuedAt:    time.Now().UTC(),
		}
		if err := m.db.Create(&issued).Error; err != nil {
			log.Printf("[AUDIT] Failed to save issued action: %v", err)
		}
	}

	// Record discipline if the target holds a staff role.
	if targetID == "" {
		return
	}
	member, err := s.GuildMember(m.cfg.DiscordGuildID, targetID)
	if err != nil {
		return
	}
	isStaff := false
	for _, roleID := range member.Roles {
		if roleID == m.cfg.ModRoleID || roleID == m.cfg.ManagerRoleID || roleID == m.cfg.DirectorRoleID {
			isStaff = true
			break
		}
	}
	if !isStaff {
		return
	}
	action := database.ModAction{
		ModMemberID:    targetID,
		AuthorMemberID: moderatorID,
		ActionType:     actionType,
		Reason:         reason,
		IssuedAt:       time.Now().UTC(),
	}
	if err := m.db.Create(&action).Error; err != nil {
		log.Printf("[AUDIT] Failed to save discipline record: %v", err)
	}
}

func (m *Module) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/notes", m.handleNotes)
	mux.HandleFunc("/api/training", m.handleTraining)
	mux.HandleFunc("/api/removed-mods", m.handleRemovedMods)
	mux.HandleFunc("/api/mod-actions", m.handleModActions)
	mux.HandleFunc("/api/mod-issued-actions", m.handleModIssuedActions)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (m *Module) requireSection(w http.ResponseWriter, r *http.Request, section string) (auth.Role, bool) {
	role, ok := auth.RoleFromContext(r.Context())
	if !ok || (role != auth.RoleManager && role != auth.RoleDirector) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "manager or director role required"})
		return 0, false
	}
	userID, _ := auth.UserIDFromContext(r.Context())
	if !permissions.CanAccess(m.db, role == auth.RoleDirector, userID, section) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden: missing " + section + " permission"})
		return 0, false
	}
	return role, true
}

// ----- Notes -----

func (m *Module) handleNotes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		m.getNotes(w, r)
	case http.MethodPost:
		m.createNote(w, r)
	case http.MethodDelete:
		m.deleteNote(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (m *Module) getNotes(w http.ResponseWriter, r *http.Request) {
	if _, ok := m.requireSection(w, r, "moderators"); !ok {
		return
	}
	modID := r.URL.Query().Get("mod_id")
	if modID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "mod_id is required"})
		return
	}
	var notes []database.ModNote
	if err := m.db.Where("mod_member_id = ?", modID).Order("created_at desc").Find(&notes).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "database error"})
		return
	}
	if notes == nil {
		notes = []database.ModNote{}
	}
	writeJSON(w, http.StatusOK, notes)
}

type createNoteRequest struct {
	ModMemberID string `json:"mod_member_id"`
	Content     string `json:"content"`
}

func (m *Module) createNote(w http.ResponseWriter, r *http.Request) {
	if _, ok := m.requireSection(w, r, "moderators"); !ok {
		return
	}
	var req createNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.ModMemberID == "" || req.Content == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "mod_member_id and content are required"})
		return
	}
	authorID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	note := database.ModNote{
		ModMemberID:    req.ModMemberID,
		AuthorMemberID: authorID,
		Content:        req.Content,
	}
	if err := m.db.Create(&note).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create note"})
		return
	}
	writeJSON(w, http.StatusCreated, note)
}

func (m *Module) deleteNote(w http.ResponseWriter, r *http.Request) {
	role, ok := auth.RoleFromContext(r.Context())
	if !ok || role != auth.RoleDirector {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "only directors can delete notes"})
		return
	}
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id is required"})
		return
	}
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	result := m.db.Delete(&database.ModNote{}, id)
	if result.Error != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete note"})
		return
	}
	if result.RowsAffected == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "note not found"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ----- Training -----

func (m *Module) handleTraining(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		m.getTraining(w, r)
	case http.MethodPut:
		m.updateTraining(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (m *Module) getTraining(w http.ResponseWriter, r *http.Request) {
	if _, ok := m.requireSection(w, r, "moderators"); !ok {
		return
	}
	modID := r.URL.Query().Get("mod_id")
	if modID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "mod_id is required"})
		return
	}
	var training database.ModTraining
	err := m.db.Where("mod_member_id = ?", modID).First(&training).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeJSON(w, http.StatusOK, database.ModTraining{ModMemberID: modID})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "database error"})
		return
	}
	writeJSON(w, http.StatusOK, training)
}

type updateTrainingRequest struct {
	ModMemberID   string  `json:"mod_member_id"`
	InTraining    bool    `json:"in_training"`
	TrainingStart *string `json:"training_start"`
	TrainingEnd   *string `json:"training_end"`
}

func (m *Module) updateTraining(w http.ResponseWriter, r *http.Request) {
	if _, ok := m.requireSection(w, r, "moderators"); !ok {
		return
	}
	var req updateTrainingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.ModMemberID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "mod_member_id is required"})
		return
	}

	var startDate, endDate *time.Time
	if req.TrainingStart != nil && *req.TrainingStart != "" {
		t, err := time.Parse("2006-01-02", *req.TrainingStart)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid training_start format, use YYYY-MM-DD"})
			return
		}
		startDate = &t
	}
	if req.TrainingEnd != nil && *req.TrainingEnd != "" {
		t, err := time.Parse("2006-01-02", *req.TrainingEnd)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid training_end format, use YYYY-MM-DD"})
			return
		}
		endDate = &t
	}

	var training database.ModTraining
	m.db.Where("mod_member_id = ?", req.ModMemberID).FirstOrInit(&training)
	training.ModMemberID = req.ModMemberID
	training.InTraining = req.InTraining
	training.TrainingStart = startDate
	training.TrainingEnd = endDate

	if err := m.db.Save(&training).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update training"})
		return
	}
	writeJSON(w, http.StatusOK, training)
}

// ----- Removed Mods -----

func (m *Module) handleRemovedMods(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		m.getRemovedMods(w, r)
	case http.MethodPost:
		m.addRemovedMod(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (m *Module) getRemovedMods(w http.ResponseWriter, r *http.Request) {
	if _, ok := m.requireSection(w, r, "moderators"); !ok {
		return
	}
	var removed []database.RemovedMod
	if err := m.db.Find(&removed).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "database error"})
		return
	}
	if removed == nil {
		removed = []database.RemovedMod{}
	}
	writeJSON(w, http.StatusOK, removed)
}

type removeModRequest struct {
	MemberID string `json:"member_id"`
}

func (m *Module) addRemovedMod(w http.ResponseWriter, r *http.Request) {
	if _, ok := m.requireSection(w, r, "moderators"); !ok {
		return
	}
	var req removeModRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.MemberID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "member_id is required"})
		return
	}
	removed := database.RemovedMod{MemberID: req.MemberID}
	if err := m.db.Where("member_id = ?", req.MemberID).FirstOrCreate(&removed).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to remove mod"})
		return
	}
	writeJSON(w, http.StatusOK, removed)
}

// ----- Mod Actions (warnings, timeouts, bans) -----

func (m *Module) handleModActions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		m.getModActions(w, r)
	case http.MethodPost:
		m.createModAction(w, r)
	case http.MethodDelete:
		m.deleteModAction(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (m *Module) getModActions(w http.ResponseWriter, r *http.Request) {
	if _, ok := m.requireSection(w, r, "moderators"); !ok {
		return
	}
	modID := r.URL.Query().Get("mod_id")
	if modID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "mod_id is required"})
		return
	}
	var actions []database.ModAction
	if err := m.db.Where("mod_member_id = ?", modID).Order("issued_at desc").Find(&actions).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "database error"})
		return
	}
	if actions == nil {
		actions = []database.ModAction{}
	}
	writeJSON(w, http.StatusOK, actions)
}

type createModActionRequest struct {
	ModMemberID string `json:"mod_member_id"`
	ActionType  string `json:"action_type"`
	Reason      string `json:"reason"`
	IssuedAt    string `json:"issued_at"` // YYYY-MM-DD
}

func (m *Module) createModAction(w http.ResponseWriter, r *http.Request) {
	if _, ok := m.requireSection(w, r, "moderators"); !ok {
		return
	}
	var req createModActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.ModMemberID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "mod_member_id is required"})
		return
	}
	switch req.ActionType {
	case "warning", "timeout", "ban", "kick":
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "action_type must be warning, timeout, ban, or kick"})
		return
	}
	authorID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	issuedAt := time.Now().UTC()
	if req.IssuedAt != "" {
		t, err := time.Parse("2006-01-02", req.IssuedAt)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid issued_at format, use YYYY-MM-DD"})
			return
		}
		issuedAt = t
	}
	action := database.ModAction{
		ModMemberID:    req.ModMemberID,
		AuthorMemberID: authorID,
		ActionType:     req.ActionType,
		Reason:         req.Reason,
		IssuedAt:       issuedAt,
	}
	if err := m.db.Create(&action).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create action"})
		return
	}
	writeJSON(w, http.StatusCreated, action)
}

func (m *Module) deleteModAction(w http.ResponseWriter, r *http.Request) {
	role, ok := auth.RoleFromContext(r.Context())
	if !ok || role != auth.RoleDirector {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "only directors can delete actions"})
		return
	}
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id is required"})
		return
	}
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	result := m.db.Delete(&database.ModAction{}, id)
	if result.Error != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete action"})
		return
	}
	if result.RowsAffected == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "action not found"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ----- Mod Issued Actions (how many warnings/timeouts/kicks/bans a mod has done) -----

func (m *Module) handleModIssuedActions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	m.getModIssuedActions(w, r)
}

// getModIssuedActions returns counts of each action type issued by a mod within an optional date range.
// Response: {"warning": N, "timeout": N, "kick": N, "ban": N}
func (m *Module) getModIssuedActions(w http.ResponseWriter, r *http.Request) {
	if _, ok := m.requireSection(w, r, "moderators"); !ok {
		return
	}
	modID := r.URL.Query().Get("mod_id")
	if modID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "mod_id is required"})
		return
	}

	query := m.db.Model(&database.ModIssuedAction{}).Where("mod_member_id = ?", modID)
	if start := r.URL.Query().Get("start_date"); start != "" {
		if t, err := time.Parse("2006-01-02", start); err == nil {
			query = query.Where("issued_at >= ?", t)
		}
	}
	if end := r.URL.Query().Get("end_date"); end != "" {
		if t, err := time.Parse("2006-01-02", end); err == nil {
			query = query.Where("issued_at < ?", t.AddDate(0, 0, 1))
		}
	}

	type countRow struct {
		ActionType string
		Count      int
	}
	var rows []countRow
	if err := query.Select("action_type, count(*) as count").Group("action_type").Scan(&rows).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "database error"})
		return
	}

	counts := map[string]int{"warning": 0, "timeout": 0, "kick": 0, "ban": 0}
	for _, row := range rows {
		counts[row.ActionType] = row.Count
	}
	writeJSON(w, http.StatusOK, counts)
}
