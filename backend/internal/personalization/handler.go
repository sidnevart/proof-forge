package personalization

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"log/slog"

	"github.com/sidnevart/proof-forge/backend/internal/users"
)

// Handler exposes personalization endpoints.
type Handler struct {
	service *Service
	pool    *pgxpool.Pool
	log     *slog.Logger
}

// NewHandler constructs a Handler.
func NewHandler(service *Service, pool *pgxpool.Pool, log *slog.Logger) *Handler {
	return &Handler{service: service, pool: pool, log: log}
}

// RegisterRoutes installs personalization endpoints.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/teams/{teamID}/personalization/assemble-proof", h.handleAssembleProof)
	r.Get("/teams/{teamID}/members/{userID}/briefing", h.handleBriefing)
}

// DTOs.

type assembleInput struct {
	GoalID int64 `json:"goal_id,omitempty"`
}

type candidateDTO struct {
	GoalID      int64   `json:"goal_id"`
	NoteIDs     []int64 `json:"note_ids"`
	Rationale   string  `json:"rationale"`
	Confidence  string  `json:"confidence"`
}

type assembleResultDTO struct {
	Candidates []candidateDTO `json:"candidates"`
	Provider   string         `json:"provider"`
}

type briefingResultDTO struct {
	Text     string `json:"text"`
	Provider string `json:"provider"`
}

// Handlers.

func (h *Handler) handleAssembleProof(w http.ResponseWriter, r *http.Request) {
	user, ok := requireAuth(w, r)
	if !ok {
		return
	}
	teamID, ok := pathInt64(w, r, "teamID")
	if !ok {
		return
	}

	// Verify active membership.
	mem, err := h.getMembership(r.Context(), teamID, user.ID)
	if err != nil {
		writeError(w, http.StatusForbidden, "team.not_member", "Not an active member")
		return
	}

	// Resolve team AI mode and user consent.
	aiMode, consent, err := h.getTeamAISettings(r.Context(), teamID, user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal.unexpected", "Failed to load AI settings")
		return
	}

	mode := resolveMode(aiMode, consent)
	if mode == ModeOff {
		writeJSON(w, http.StatusOK, map[string]any{
			"data": assembleResultDTO{Candidates: nil, Provider: string(ProviderTemplate)},
		})
		return
	}

	// Read unconsumed daily-log entries for the user in this team.
	entries, err := h.getUnconsumedEntries(r.Context(), user.ID, teamID)
	if err != nil {
		h.log.Warn("assemble-proof: failed to load entries", "err", err)
		writeError(w, http.StatusInternalServerError, "internal.unexpected", "Failed to load entries")
		return
	}

	// Read active team goals.
	goals, err := h.getTeamGoals(r.Context(), teamID)
	if err != nil {
		h.log.Warn("assemble-proof: failed to load goals", "err", err)
		writeError(w, http.StatusInternalServerError, "internal.unexpected", "Failed to load goals")
		return
	}

	// If no entries or no goals, return empty candidates via template.
	if len(entries) == 0 || len(goals) == 0 {
		writeJSON(w, http.StatusOK, map[string]any{
			"data": assembleResultDTO{Candidates: []candidateDTO{}, Provider: string(ProviderTemplate)},
		})
		return
	}

	var in assembleInput
	_ = json.NewDecoder(r.Body).Decode(&in)

	systemPrompt := buildAssembleProofSystemPrompt()
	userPrompt := buildAssembleProofUserPrompt(entries, goals, in.GoalID, mem.Alias, mode)

	res, _, err := h.service.Prompt(r.Context(), FeatureAssembleProof, mode, 8*time.Second, systemPrompt, userPrompt, 512)
	if err != nil {
		h.log.Warn("assemble-proof: prompt failed", "err", err)
		writeError(w, http.StatusInternalServerError, "internal.unexpected", "AI assembly failed")
		return
	}

	candidates := parseAssembleCandidates(res.Text)
	validated := validateAndFilterCandidates(candidates, entries, goals)

	writeJSON(w, http.StatusOK, map[string]any{
		"data": assembleResultDTO{
			Candidates: validated,
			Provider:   string(res.Provider),
		},
	})
}

func (h *Handler) handleBriefing(w http.ResponseWriter, r *http.Request) {
	user, ok := requireAuth(w, r)
	if !ok {
		return
	}
	teamID, ok := pathInt64(w, r, "teamID")
	if !ok {
		return
	}
	targetUserID, ok := pathInt64(w, r, "userID")
	if !ok {
		return
	}

	// Requester must be lead or trusted.
	requester, err := h.getMembership(r.Context(), teamID, user.ID)
	if err != nil || requester.Role != "lead" && requester.Role != "trusted_approver" {
		writeError(w, http.StatusForbidden, "team.not_lead", "Only lead or trusted approver can view briefing")
		return
	}

	// Target must be active member.
	target, err := h.getMembership(r.Context(), teamID, targetUserID)
	if err != nil {
		writeError(w, http.StatusNotFound, "team.member_not_found", "Member not found")
		return
	}

	aiMode, consent, err := h.getTeamAISettings(r.Context(), teamID, targetUserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal.unexpected", "Failed to load AI settings")
		return
	}

	mode := resolveMode(aiMode, consent)
	if mode == ModeOff {
		writeJSON(w, http.StatusOK, map[string]any{
			"data": briefingResultDTO{Text: "AI выключен для этого участника.", Provider: string(ProviderTemplate)},
		})
		return
	}

	stats, err := h.getMemberProofStats(r.Context(), targetUserID, teamID)
	if err != nil {
		h.log.Warn("briefing: failed to load stats", "err", err)
		writeError(w, http.StatusInternalServerError, "internal.unexpected", "Failed to load stats")
		return
	}

	systemPrompt := buildBriefingSystemPrompt()
	userPrompt := buildBriefingUserPrompt(target.Alias, stats, mode)

	res, _, err := h.service.Prompt(r.Context(), FeatureLeadBriefing, mode, 4*time.Second, systemPrompt, userPrompt, 256)
	if err != nil {
		h.log.Warn("briefing: prompt failed", "err", err)
		writeError(w, http.StatusInternalServerError, "internal.unexpected", "AI briefing failed")
		return
	}

	// Simple output validation: 30-80 words.
	words := len(splitWords(res.Text))
	if words < 30 || words > 80 {
		// Fall back to statistical summary.
		res.Text = fmt.Sprintf("За последние 30 дней: %d proof отправлено, %d одобрено, %d отклонено. Стрик: %d дней.",
			stats.SubmittedCount, stats.ApprovedCount, stats.RejectedCount, stats.CurrentStreak)
		res.Provider = ProviderTemplate
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": briefingResultDTO{Text: res.Text, Provider: string(res.Provider)},
	})
}

// Data models.

type membershipRow struct {
	Role   string
	Status string
	Alias  string
}

type entryRow struct {
	ID          int64
	LogDate     time.Time
	TextContent *string
	HasArtifact bool
	GoalID      *int64
}

type goalRow struct {
	ID    int64
	Title string
}

type memberStats struct {
	SubmittedCount int
	ApprovedCount  int
	RejectedCount  int
	CurrentStreak  int
}

// Query helpers.

func (h *Handler) getMembership(ctx context.Context, teamID, userID int64) (membershipRow, error) {
	const sql = `
		SELECT role, status, u.display_name
		FROM team_memberships m
		JOIN users u ON u.id = m.user_id
		WHERE m.team_id = $1 AND m.user_id = $2 AND m.status = 'active'
	`
	var m membershipRow
	err := h.pool.QueryRow(ctx, sql, teamID, userID).Scan(&m.Role, &m.Status, &m.Alias)
	return m, err
}

func (h *Handler) getTeamAISettings(ctx context.Context, teamID, userID int64) (string, bool, error) {
	const sql = `
		SELECT t.ai_mode, m.ai_consent
		FROM teams t
		JOIN team_memberships m ON m.team_id = t.id AND m.user_id = $2
		WHERE t.id = $1
	`
	var mode string
	var consent bool
	err := h.pool.QueryRow(ctx, sql, teamID, userID).Scan(&mode, &consent)
	return mode, consent, err
}

func (h *Handler) getUnconsumedEntries(ctx context.Context, userID, teamID int64) ([]entryRow, error) {
	const sql = `
		SELECT id, log_date, text_content, has_artifact, goal_id
		FROM daily_log_entries
		WHERE user_id = $1 AND team_id = $2 AND status = 'logged' AND consumed_in_check_in_id IS NULL
		ORDER BY log_date DESC
		LIMIT 50
	`
	rows, err := h.pool.Query(ctx, sql, userID, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []entryRow
	for rows.Next() {
		var e entryRow
		var text *string
		var goalID *int64
		err := rows.Scan(&e.ID, &e.LogDate, &text, &e.HasArtifact, &goalID)
		if err != nil {
			return nil, err
		}
		e.TextContent = text
		e.GoalID = goalID
		out = append(out, e)
	}
	return out, rows.Err()
}

func (h *Handler) getTeamGoals(ctx context.Context, teamID int64) ([]goalRow, error) {
	const sql = `
		SELECT id, title FROM goals WHERE team_id = $1 AND status = 'active'
	`
	rows, err := h.pool.Query(ctx, sql, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []goalRow
	for rows.Next() {
		var g goalRow
		if err := rows.Scan(&g.ID, &g.Title); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

func (h *Handler) getMemberProofStats(ctx context.Context, userID, teamID int64) (memberStats, error) {
	const sql = `
		SELECT
			COALESCE(SUM(CASE WHEN status = 'submitted' THEN 1 ELSE 0 END), 0) AS submitted,
			COALESCE(SUM(CASE WHEN status = 'approved' THEN 1 ELSE 0 END), 0) AS approved,
			COALESCE(SUM(CASE WHEN status = 'rejected' THEN 1 ELSE 0 END), 0) AS rejected
		FROM check_ins
		WHERE owner_user_id = $1 AND team_id = $2 AND submitted_at > NOW() - INTERVAL '30 days'
	`
	var s memberStats
	err := h.pool.QueryRow(ctx, sql, userID, teamID).Scan(&s.SubmittedCount, &s.ApprovedCount, &s.RejectedCount)
	if err != nil {
		return s, err
	}

	// Also get current streak from user_streak.
	const streakSQL = `SELECT COALESCE(current_streak, 0) FROM user_streak WHERE user_id = $1 AND team_id = $2`
	_ = h.pool.QueryRow(ctx, streakSQL, userID, teamID).Scan(&s.CurrentStreak)
	return s, nil
}

// Prompt builders.

func buildAssembleProofSystemPrompt() string {
	return `Ты помощник, который помогает пользователю собрать weekly proof из daily log entries.
Верни ТОЛЬКО валидный JSON в точно таком формате:
{"candidates":[{"goal_id":42,"note_ids":[101,103],"rationale":"...","confidence":"high|medium|low"}]}
Правила:
- Каждый кандидат группирует связанные записи
- Высокая уверенность требует хотя бы один артефакт
- Максимум 3 кандидата
- Обоснование максимум 240 символов, без оценочных суждений
- note_ids должны быть из предоставленных записей
- goal_id должен быть из предоставленных целей`
}

func buildAssembleProofUserPrompt(entries []entryRow, goals []goalRow, preferredGoalID int64, alias string, mode Mode) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Пользователь: %s\nРежим приватности: %s\n\nЦели:\n", alias, mode))
	for _, g := range goals {
		b.WriteString(fmt.Sprintf("- ID %d: %s\n", g.ID, g.Title))
	}
	if preferredGoalID > 0 {
		b.WriteString(fmt.Sprintf("\nПредпочтительная цель: ID %d\n", preferredGoalID))
	}
	b.WriteString("\nЗаписи:\n")
	for _, e := range entries {
		artifact := ""
		if e.HasArtifact {
			artifact = " [артефакт]"
		}
		var text string
		if e.TextContent != nil {
			text = *e.TextContent
		}
		if len(text) > 120 {
			text = text[:120] + "..."
		}
		b.WriteString(fmt.Sprintf("- ID %d | %s | %s%s\n", e.ID, e.LogDate.Format("2006-01-02"), text, artifact))
	}
	b.WriteString("\nСформируй кандидатов.")
	return b.String()
}

func buildBriefingSystemPrompt() string {
	return `Ты AI-ассистент для тимлидов. Суммируй активность участника за последние 30 дней в 30-80 словах.
Фокусируйся на наблюдаемых фактах (количество, даты), а не на личностных качествах.
Включи один раздел рисков или явно скажи "Без рисков".
Не используй ярлыки вроде слабый/сильный/плохой performer.
Ответ на русском языке, обычным текстом.`
}

func buildBriefingUserPrompt(alias string, stats memberStats, mode Mode) string {
	return fmt.Sprintf("Участник: %s\nРежим: %s\n\nСтатистика за 30 дней:\n- Proof отправлено: %d\n- Одобрено: %d\n- Отклонено: %d\n- Текущий стрик: %d дней\n\nНапиши краткий брифинг.",
		alias, mode, stats.SubmittedCount, stats.ApprovedCount, stats.RejectedCount, stats.CurrentStreak)
}

// Parsing and validation.

func parseAssembleCandidates(text string) []candidateRow {
	// Try to extract JSON from the response.
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start < 0 || end < 0 || end <= start {
		return nil
	}
	var wrapper struct {
		Candidates []candidateRow `json:"candidates"`
	}
	if err := json.Unmarshal([]byte(text[start:end+1]), &wrapper); err != nil {
		// Try top-level array fallback.
		var arr []candidateRow
		if err2 := json.Unmarshal([]byte(text[start:end+1]), &arr); err2 == nil {
			return arr
		}
		return nil
	}
	return wrapper.Candidates
}

func validateAndFilterCandidates(candidates []candidateRow, entries []entryRow, goals []goalRow) []candidateDTO {
	validEntryIDs := make(map[int64]bool)
	for _, e := range entries {
		validEntryIDs[e.ID] = true
	}
	validGoalIDs := make(map[int64]bool)
	for _, g := range goals {
		validGoalIDs[g.ID] = true
	}

	var out []candidateDTO
	for _, c := range candidates {
		if !validGoalIDs[c.GoalID] {
			continue
		}
		var validNotes []int64
		hasArtifact := false
		for _, nid := range c.NoteIDs {
			if validEntryIDs[nid] {
				validNotes = append(validNotes, nid)
				for _, e := range entries {
					if e.ID == nid && e.HasArtifact {
						hasArtifact = true
						break
					}
				}
			}
		}
		if len(validNotes) == 0 {
			continue
		}
		if c.Confidence == "high" && !hasArtifact {
			c.Confidence = "medium"
		}
		if len(c.Rationale) > 240 {
			c.Rationale = c.Rationale[:240]
		}
		out = append(out, candidateDTO{
			GoalID:     c.GoalID,
			NoteIDs:    validNotes,
			Rationale:  c.Rationale,
			Confidence: c.Confidence,
		})
		if len(out) >= 3 {
			break
		}
	}
	return out
}

type candidateRow struct {
	GoalID      int64   `json:"goal_id"`
	NoteIDs     []int64 `json:"note_ids"`
	Rationale   string  `json:"rationale"`
	Confidence  string  `json:"confidence"`
}

func resolveMode(aiMode string, consent bool) Mode {
	switch aiMode {
	case "off":
		return ModeOff
	case "metadata-only":
		return ModeMetadataOnly
	case "full":
		if consent {
			return ModeFull
		}
		return ModeMetadataOnly
	}
	return ModeOff
}

// HTTP helpers (duplicated from teamproof for package isolation; could be shared later).

func requireAuth(w http.ResponseWriter, r *http.Request) (users.User, bool) {
	user, ok := users.CurrentUser(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "user.not_authenticated", "Authentication required")
		return users.User{}, false
	}
	return user, true
}

func pathInt64(w http.ResponseWriter, r *http.Request, key string) (int64, bool) {
	raw := chi.URLParam(r, key)
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		writeError(w, http.StatusBadRequest, "validation.field_invalid", "Path parameter must be a positive integer")
		return 0, false
	}
	return value, true
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}
