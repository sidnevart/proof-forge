package stakes

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/sidnevart/proof-forge/backend/internal/users"
)

type Handler struct {
	log     *slog.Logger
	service *Service
}

func NewHandler(log *slog.Logger, service *Service) *Handler {
	return &Handler{log: log, service: service}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/goals/{goalID}/stakes", h.handleCreate)
	r.Get("/goals/{goalID}/stakes", h.handleList)
	r.Delete("/stakes/{stakeID}", h.handleCancel)
	r.Post("/stakes/{stakeID}/forfeit", h.handleForfeit)
}

func (h *Handler) handleCreate(w http.ResponseWriter, r *http.Request) {
	actor, ok := currentUser(w, r)
	if !ok {
		return
	}
	goalID, ok := pathInt64(w, r, "goalID")
	if !ok {
		return
	}

	var input CreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON")
		return
	}

	view, err := h.service.Create(r.Context(), actor, goalID, input)
	if err != nil {
		h.writeServiceError(w, "create stake", err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"stake": view})
}

func (h *Handler) handleList(w http.ResponseWriter, r *http.Request) {
	actor, ok := currentUser(w, r)
	if !ok {
		return
	}
	goalID, ok := pathInt64(w, r, "goalID")
	if !ok {
		return
	}

	views, err := h.service.ListForGoal(r.Context(), actor, goalID)
	if err != nil {
		h.writeServiceError(w, "list stakes", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"stakes": views})
}

func (h *Handler) handleCancel(w http.ResponseWriter, r *http.Request) {
	actor, ok := currentUser(w, r)
	if !ok {
		return
	}
	stakeID, ok := pathInt64(w, r, "stakeID")
	if !ok {
		return
	}

	if err := h.service.Cancel(r.Context(), actor, stakeID); err != nil {
		h.writeServiceError(w, "cancel stake", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleForfeit(w http.ResponseWriter, r *http.Request) {
	actor, ok := currentUser(w, r)
	if !ok {
		return
	}
	stakeID, ok := pathInt64(w, r, "stakeID")
	if !ok {
		return
	}

	var input ForfeitInput
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON")
			return
		}
	}

	view, err := h.service.Forfeit(r.Context(), actor, stakeID, input)
	if err != nil {
		h.writeServiceError(w, "forfeit stake", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"stake": view})
}

func (h *Handler) writeServiceError(w http.ResponseWriter, op string, err error) {
	switch {
	case errors.Is(err, ErrStakeNotFound):
		writeError(w, http.StatusNotFound, "not_found", "Stake not found")
	case errors.Is(err, ErrNotGoalOwner):
		writeError(w, http.StatusForbidden, "forbidden", "Only the goal owner can manage stakes")
	case errors.Is(err, ErrNotBuddy):
		writeError(w, http.StatusForbidden, "forbidden", "Only the goal buddy can forfeit a stake")
	case errors.Is(err, ErrStakeNotActive):
		writeError(w, http.StatusConflict, "stake_not_active", "Stake is not active")
	case errors.Is(err, ErrGoalNotActive):
		writeError(w, http.StatusConflict, "goal_not_active", "Goal is not active")
	case errors.Is(err, ErrInvalidInput):
		writeError(w, http.StatusBadRequest, "invalid_input", err.Error())
	default:
		if h.log != nil {
			h.log.Error(op, "err", err)
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Internal server error")
	}
}

func currentUser(w http.ResponseWriter, r *http.Request) (users.User, bool) {
	actor, ok := users.CurrentUser(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth_required", "Authentication required")
	}
	return actor, ok
}

func pathInt64(w http.ResponseWriter, r *http.Request, param string) (int64, bool) {
	raw := chi.URLParam(r, param)
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_param", param+" must be a positive integer")
		return 0, false
	}
	return id, true
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]any{"code": code, "message": message},
	})
}
