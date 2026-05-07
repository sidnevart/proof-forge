package dailylog

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/sidnevart/proof-forge/backend/internal/users"
)

// Handler exposes the REST surface for daily-log.
type Handler struct {
	service *Service
}

// NewHandler constructs a Handler over the given Service.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes installs all daily-log endpoints.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/teams/{teamID}/daily-log", h.handleSubmit)
	r.Post("/teams/{teamID}/daily-log/freeze", h.handleFreeze)
	r.Get("/teams/{teamID}/daily-log", h.handleList)
	r.Get("/teams/{teamID}/daily-log/streak", h.handleGetStreak)
}

// DTOs.

type submitInput struct {
	LogDate     string  `json:"log_date"`
	TextContent string  `json:"text_content"`
	ExternalURL *string `json:"external_url,omitempty"`
	ClientSource string `json:"client_source,omitempty"`
}

type freezeInput struct {
	LogDate string `json:"log_date"`
	Reason  string `json:"reason,omitempty"`
}

type entryDTO struct {
	ID          int64      `json:"id"`
	LogDate     string     `json:"log_date"`
	Status      string     `json:"status"`
	TextContent string     `json:"text_content,omitempty"`
	HasArtifact bool       `json:"has_artifact"`
	ExternalURL *string    `json:"external_url,omitempty"`
	SubmittedAt time.Time  `json:"submitted_at"`
}

type streakDTO struct {
	Current              int    `json:"current"`
	Best                 int    `json:"best"`
	IsNewRecord          bool   `json:"is_new_record"`
	FreezesUsedThisMonth int    `json:"freezes_used_this_month"`
}

func toEntryDTO(e Entry, withContent bool) entryDTO {
	d := entryDTO{
		ID:          e.ID,
		LogDate:     e.LogDate.Format("2006-01-02"),
		Status:      string(e.Status),
		HasArtifact: e.HasArtifact,
		SubmittedAt: e.SubmittedAt,
	}
	if withContent {
		d.TextContent = e.TextContent
		d.ExternalURL = e.ExternalURL
	}
	return d
}

// Handlers.

func (h *Handler) handleSubmit(w http.ResponseWriter, r *http.Request) {
	user, ok := requireAuth(w, r)
	if !ok {
		return
	}
	teamID, ok := pathInt64(w, r, "teamID")
	if !ok {
		return
	}
	var in submitInput
	if err := decodeJSON(w, r, &in); !err {
		return
	}

	logDate, parseErr := time.Parse("2006-01-02", in.LogDate)
	if parseErr != nil {
		writeErrorCode(w, http.StatusBadRequest, "validation.field_invalid", "log_date must be YYYY-MM-DD")
		return
	}

	entry, su, err := h.service.SubmitLog(r.Context(), SubmitInput{
		UserID:       user.ID,
		TeamID:       teamID,
		LogDate:      logDate,
		TextContent:  in.TextContent,
		ExternalURL:    in.ExternalURL,
		ClientSource: in.ClientSource,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"entry_id": entry.ID,
			"status":   string(entry.Status),
			"streak": streakDTO{
				Current:              su.NewCurrent,
				Best:                 su.NewBest,
				IsNewRecord:          su.IsNewRecord,
				FreezesUsedThisMonth: 0, // populated separately if needed
			},
		},
	})
}

func (h *Handler) handleFreeze(w http.ResponseWriter, r *http.Request) {
	user, ok := requireAuth(w, r)
	if !ok {
		return
	}
	teamID, ok := pathInt64(w, r, "teamID")
	if !ok {
		return
	}
	var in freezeInput
	if err := decodeJSON(w, r, &in); !err {
		return
	}

	logDate, parseErr := time.Parse("2006-01-02", in.LogDate)
	if parseErr != nil {
		writeErrorCode(w, http.StatusBadRequest, "validation.field_invalid", "log_date must be YYYY-MM-DD")
		return
	}

	entry, err := h.service.FreezeDay(r.Context(), FreezeInput{
		UserID:  user.ID,
		TeamID:  teamID,
		LogDate: logDate,
		Reason:  in.Reason,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"entry_id": entry.ID,
			"status":   string(entry.Status),
		},
	})
}

func (h *Handler) handleList(w http.ResponseWriter, r *http.Request) {
	user, ok := requireAuth(w, r)
	if !ok {
		return
	}
	teamID, ok := pathInt64(w, r, "teamID")
	if !ok {
		return
	}

	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	if from == "" || to == "" {
		writeErrorCode(w, http.StatusBadRequest, "validation.field_invalid", "from and to query params required")
		return
	}
	fromDate, err1 := time.Parse("2006-01-02", from)
	toDate, err2 := time.Parse("2006-01-02", to)
	if err1 != nil || err2 != nil {
		writeErrorCode(w, http.StatusBadRequest, "validation.field_invalid", "from/to must be YYYY-MM-DD")
		return
	}

	entries, err := h.service.ListMyEntries(r.Context(), user.ID, teamID, fromDate, toDate)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	out := make([]entryDTO, 0, len(entries))
	for _, e := range entries {
		out = append(out, toEntryDTO(e, true))
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"entries": out}})
}

func (h *Handler) handleGetStreak(w http.ResponseWriter, r *http.Request) {
	user, ok := requireAuth(w, r)
	if !ok {
		return
	}
	teamID, ok := pathInt64(w, r, "teamID")
	if !ok {
		return
	}

	streak, err := h.service.GetMyStreak(r.Context(), user.ID, teamID)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"streak": streakDTO{
				Current:              streak.Current,
				Best:                 streak.Best,
				IsNewRecord:          false,
				FreezesUsedThisMonth: streak.FreezesUsedThisMonth,
			},
		},
	})
}

// ──────────────────────────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────────────────────────

func requireAuth(w http.ResponseWriter, r *http.Request) (users.User, bool) {
	user, ok := users.CurrentUser(r.Context())
	if !ok {
		writeErrorCode(w, http.StatusUnauthorized, "user.not_authenticated", "Authentication required")
		return users.User{}, false
	}
	return user, true
}

func decodeJSON(w http.ResponseWriter, r *http.Request, out any) bool {
	if err := json.NewDecoder(r.Body).Decode(out); err != nil {
		writeErrorCode(w, http.StatusBadRequest, "validation.field_invalid", "Request body must be valid JSON")
		return false
	}
	return true
}

func pathInt64(w http.ResponseWriter, r *http.Request, key string) (int64, bool) {
	raw := chi.URLParam(r, key)
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		writeErrorCode(w, http.StatusBadRequest, "validation.field_invalid", "Path parameter must be a positive integer")
		return 0, false
	}
	return value, true
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeErrorCode(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}

var errMap = map[error]struct {
	status int
	code   string
}{
	ErrEntryNotFound:      {http.StatusNotFound, "daily_log.not_found"},
	ErrInvalidLogDate:     {http.StatusBadRequest, "validation.field_invalid"},
	ErrTextTooShort:       {http.StatusBadRequest, "validation.field_invalid"},
	ErrFutureDate:         {http.StatusBadRequest, "validation.field_invalid"},
	ErrAlreadyLogged:      {http.StatusConflict, "daily_log.already_logged"},
	ErrFreezeLimitReached: {http.StatusConflict, "daily_log.freeze_limit_reached"},
	ErrNotMember:          {http.StatusForbidden, "team.not_member"},
	ErrCannotReadContent:  {http.StatusForbidden, "daily_log.cannot_read"},
	ErrInvalidTimezone:    {http.StatusBadRequest, "validation.field_invalid"},
}

func writeDomainError(w http.ResponseWriter, err error) {
	for sentinel, mapping := range errMap {
		if errors.Is(err, sentinel) {
			writeErrorCode(w, mapping.status, mapping.code, sentinel.Error())
			return
		}
	}
	writeErrorCode(w, http.StatusInternalServerError, "internal.unexpected", "An unexpected error occurred")
}
