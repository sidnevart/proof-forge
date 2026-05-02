package milestones

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
	r.Post("/goals/{goalID}/milestones", h.handleCreate)
	r.Get("/goals/{goalID}/milestones", h.handleList)
	r.Patch("/milestones/{milestoneID}", h.handleUpdate)
	r.Delete("/milestones/{milestoneID}", h.handleDelete)
	r.Post("/milestones/{milestoneID}/complete", h.handleComplete)
	r.Post("/milestones/{milestoneID}/reopen", h.handleReopen)
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

	m, err := h.service.Create(r.Context(), actor, goalID, input)
	if err != nil {
		h.writeServiceError(w, "create milestone", err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"milestone": m})
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

	list, err := h.service.ListForGoal(r.Context(), actor, goalID)
	if err != nil {
		h.writeServiceError(w, "list milestones", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"milestones": list})
}

func (h *Handler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	actor, ok := currentUser(w, r)
	if !ok {
		return
	}
	milestoneID, ok := pathInt64(w, r, "milestoneID")
	if !ok {
		return
	}

	var input UpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON")
		return
	}

	m, err := h.service.Update(r.Context(), actor, milestoneID, input)
	if err != nil {
		h.writeServiceError(w, "update milestone", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"milestone": m})
}

func (h *Handler) handleDelete(w http.ResponseWriter, r *http.Request) {
	actor, ok := currentUser(w, r)
	if !ok {
		return
	}
	milestoneID, ok := pathInt64(w, r, "milestoneID")
	if !ok {
		return
	}

	if err := h.service.Delete(r.Context(), actor, milestoneID); err != nil {
		h.writeServiceError(w, "delete milestone", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleComplete(w http.ResponseWriter, r *http.Request) {
	actor, ok := currentUser(w, r)
	if !ok {
		return
	}
	milestoneID, ok := pathInt64(w, r, "milestoneID")
	if !ok {
		return
	}

	m, err := h.service.Complete(r.Context(), actor, milestoneID)
	if err != nil {
		h.writeServiceError(w, "complete milestone", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"milestone": m})
}

func (h *Handler) handleReopen(w http.ResponseWriter, r *http.Request) {
	actor, ok := currentUser(w, r)
	if !ok {
		return
	}
	milestoneID, ok := pathInt64(w, r, "milestoneID")
	if !ok {
		return
	}

	m, err := h.service.Reopen(r.Context(), actor, milestoneID)
	if err != nil {
		h.writeServiceError(w, "reopen milestone", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"milestone": m})
}

func (h *Handler) writeServiceError(w http.ResponseWriter, op string, err error) {
	switch {
	case errors.Is(err, ErrMilestoneNotFound):
		writeError(w, http.StatusNotFound, "not_found", "Milestone not found")
	case errors.Is(err, ErrNotGoalOwner):
		writeError(w, http.StatusForbidden, "forbidden", "Only the goal owner can manage milestones")
	case errors.Is(err, ErrNotBuddy):
		writeError(w, http.StatusForbidden, "forbidden", "Only the goal buddy can complete milestones")
	case errors.Is(err, ErrMilestoneNotPending):
		writeError(w, http.StatusConflict, "milestone_not_pending", "Milestone is not pending")
	case errors.Is(err, ErrMilestoneNotComplete):
		writeError(w, http.StatusConflict, "milestone_not_complete", "Milestone is not completed")
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
