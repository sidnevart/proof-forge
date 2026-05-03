package goals

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
	return &Handler{
		log:     log,
		service: service,
	}
}

// RegisterPublicRoutes mounts routes that use the invite token as credential
// and do not require a session cookie.
func (h *Handler) RegisterPublicRoutes(r chi.Router) {
	r.Get("/invites/{token}", h.handleGetInvite)
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/goals", h.handleCreateGoal)
	r.Get("/goals", h.handleListGoals)
	r.Get("/goals/{goalID}", h.handleGetGoal)
	r.Delete("/goals/{goalID}", h.handleDeleteGoal)
	r.Put("/goals/{goalID}/deadline", h.handleSetGoalDeadline)
	r.Post("/goals/{goalID}/accept-invite", h.handleAcceptInviteForGoal)
	r.Get("/dashboard", h.handleDashboard)
	r.Post("/invites/{token}/accept", h.handleAcceptInvite)
}

func (h *Handler) handleCreateGoal(w http.ResponseWriter, r *http.Request) {
	actor, ok := users.CurrentUser(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth_required", "Authentication required")
		return
	}

	var input CreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON")
		return
	}

	goal, err := h.service.CreateGoal(r.Context(), actor, input)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidGoalInput):
			writeError(w, http.StatusBadRequest, "invalid_input", err.Error())
		default:
			if h.log != nil {
				h.log.Error("create goal", "err", err)
			}
			writeError(w, http.StatusInternalServerError, "internal_error", "Could not create goal")
		}
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"goal": goal,
	})
}

func (h *Handler) handleListGoals(w http.ResponseWriter, r *http.Request) {
	actor, ok := users.CurrentUser(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth_required", "Authentication required")
		return
	}

	dashboard, err := h.service.Dashboard(r.Context(), actor)
	if err != nil {
		if h.log != nil {
			h.log.Error("list goals", "err", err)
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Could not load goals")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"goals": dashboard.Goals,
	})
}

func (h *Handler) handleDashboard(w http.ResponseWriter, r *http.Request) {
	actor, ok := users.CurrentUser(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth_required", "Authentication required")
		return
	}

	dashboard, err := h.service.Dashboard(r.Context(), actor)
	if err != nil {
		if h.log != nil {
			h.log.Error("load dashboard", "err", err)
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Could not load dashboard")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"user":    actor,
		"summary": dashboard.Summary,
		"goals":   dashboard.Goals,
	})
}

func (h *Handler) handleGetGoal(w http.ResponseWriter, r *http.Request) {
	actor, ok := users.CurrentUser(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth_required", "Authentication required")
		return
	}

	rawID := chi.URLParam(r, "goalID")
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_param", "goalID must be a positive integer")
		return
	}

	view, err := h.service.GetGoal(r.Context(), actor, id)
	if err != nil {
		switch {
		case errors.Is(err, ErrGoalNotFound):
			writeError(w, http.StatusNotFound, "not_found", "Goal not found")
		default:
			if h.log != nil {
				h.log.Error("get goal", "err", err)
			}
			writeError(w, http.StatusInternalServerError, "internal_error", "Could not load goal")
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"goal": view})
}

func (h *Handler) handleGetInvite(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	if token == "" {
		writeError(w, http.StatusBadRequest, "invalid_token", "Token is required")
		return
	}

	record, err := h.service.GetInvitePreview(r.Context(), token)
	if err != nil {
		switch {
		case errors.Is(err, ErrInviteNotFound):
			writeError(w, http.StatusNotFound, "invite_not_found", "Invite not found")
		default:
			if h.log != nil {
				h.log.Error("get invite preview", "err", err)
			}
			writeError(w, http.StatusInternalServerError, "internal_error", "Could not load invite")
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"invite": map[string]any{
			"goal_title":    record.GoalTitle,
			"owner_name":    record.OwnerName,
			"invitee_email": record.InviteeEmail,
			"status":        record.InviteStatus,
			"expires_at":    record.ExpiresAt,
		},
	})
}

func (h *Handler) handleSetGoalDeadline(w http.ResponseWriter, r *http.Request) {
	actor, ok := users.CurrentUser(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth_required", "Authentication required")
		return
	}

	rawID := chi.URLParam(r, "goalID")
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_param", "goalID must be a positive integer")
		return
	}

	var body struct {
		DeadlineAt *string `json:"deadline_at"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON")
		return
	}

	if err := h.service.SetGoalDeadline(r.Context(), actor, id, body.DeadlineAt); err != nil {
		switch {
		case errors.Is(err, ErrGoalNotFound):
			writeError(w, http.StatusNotFound, "not_found", "Goal not found or you are not the owner")
		case errors.Is(err, ErrInvalidGoalInput):
			writeError(w, http.StatusBadRequest, "invalid_input", err.Error())
		default:
			if h.log != nil {
				h.log.Error("set goal deadline", "err", err)
			}
			writeError(w, http.StatusInternalServerError, "internal_error", "Could not set deadline")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleDeleteGoal(w http.ResponseWriter, r *http.Request) {
	actor, ok := users.CurrentUser(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth_required", "Authentication required")
		return
	}

	rawID := chi.URLParam(r, "goalID")
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_param", "goalID must be a positive integer")
		return
	}

	if err := h.service.DeleteGoal(r.Context(), actor, id); err != nil {
		switch {
		case errors.Is(err, ErrGoalNotFound):
			writeError(w, http.StatusNotFound, "not_found", "Goal not found or you are not the owner")
		default:
			if h.log != nil {
				h.log.Error("delete goal", "err", err)
			}
			writeError(w, http.StatusInternalServerError, "internal_error", "Could not delete goal")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleAcceptInviteForGoal(w http.ResponseWriter, r *http.Request) {
	actor, ok := users.CurrentUser(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth_required", "Authentication required")
		return
	}

	rawID := chi.URLParam(r, "goalID")
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_param", "goalID must be a positive integer")
		return
	}

	if err := h.service.AcceptInviteForGoal(r.Context(), actor, id); err != nil {
		switch {
		case errors.Is(err, ErrInviteNotFound):
			writeError(w, http.StatusNotFound, "invite_not_found", "Invite not found")
		case errors.Is(err, ErrInviteExpired):
			writeError(w, http.StatusGone, "invite_expired", "This invite has expired")
		case errors.Is(err, ErrInviteAlreadyAccepted):
			writeError(w, http.StatusConflict, "invite_already_accepted", "This invite has already been accepted")
		case errors.Is(err, ErrUnauthorizedAcceptance):
			writeError(w, http.StatusForbidden, "forbidden", "Only the invited buddy can accept this invite")
		default:
			if h.log != nil {
				h.log.Error("accept invite for goal", "err", err)
			}
			writeError(w, http.StatusInternalServerError, "internal_error", "Could not accept invite")
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"accepted": true})
}

func (h *Handler) handleAcceptInvite(w http.ResponseWriter, r *http.Request) {
	actor, ok := users.CurrentUser(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth_required", "Authentication required")
		return
	}

	token := chi.URLParam(r, "token")
	if token == "" {
		writeError(w, http.StatusBadRequest, "invalid_token", "Token is required")
		return
	}

	if err := h.service.AcceptInvite(r.Context(), actor, token); err != nil {
		switch {
		case errors.Is(err, ErrInviteNotFound):
			writeError(w, http.StatusNotFound, "invite_not_found", "Invite not found")
		case errors.Is(err, ErrInviteExpired):
			writeError(w, http.StatusGone, "invite_expired", "This invite has expired")
		case errors.Is(err, ErrInviteAlreadyAccepted):
			writeError(w, http.StatusConflict, "invite_already_accepted", "This invite has already been accepted")
		case errors.Is(err, ErrUnauthorizedAcceptance):
			writeError(w, http.StatusForbidden, "forbidden", "Only the invited buddy can accept this invite")
		default:
			if h.log != nil {
				h.log.Error("accept invite", "err", err)
			}
			writeError(w, http.StatusInternalServerError, "internal_error", "Could not accept invite")
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"accepted": true})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}
