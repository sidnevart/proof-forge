package analytics

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sidnevart/proof-forge/backend/internal/users"
)

// Handler exposes the analytics REST surface.
type Handler struct {
	recorder Recorder
}

// NewHandler constructs a Handler over the given Recorder.
func NewHandler(recorder Recorder) *Handler {
	return &Handler{recorder: recorder}
}

// RegisterRoutes installs the analytics endpoint.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/analytics/event", h.handleEvent)
}

// DTOs.

type eventInput struct {
	EventName  string         `json:"event_name"`
	TeamID     *int64         `json:"team_id,omitempty"`
	Properties map[string]any `json:"properties,omitempty"`
}

func (h *Handler) handleEvent(w http.ResponseWriter, r *http.Request) {
	user, ok := users.CurrentUser(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "user.not_authenticated", "Authentication required")
		return
	}

	var in eventInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "validation.field_invalid", "Invalid JSON")
		return
	}

	name := EventName(in.EventName)
	if !AllowedFrontendEvents[name] {
		writeError(w, http.StatusBadRequest, "analytics.event_not_allowed", "Event not allowed from frontend")
		return
	}

	if err := ValidateProperties(in.Properties); err != nil {
		writeError(w, http.StatusBadRequest, "analytics.invalid_properties", "Properties contain forbidden keys")
		return
	}

	if err := h.recorder.Record(r.Context(), name, SourceWeb, &user.ID, in.TeamID, nil, nil, in.Properties); err != nil {
		writeError(w, http.StatusInternalServerError, "internal.unexpected", "Failed to record event")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{"code": code, "message": message},
	})
}
