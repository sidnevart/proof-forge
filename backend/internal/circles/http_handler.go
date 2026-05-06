package circles

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/sidnevart/proof-forge/backend/internal/users"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/circles", h.handleCreateCircle)
	r.Get("/circles", h.handleListCircles)
	r.Get("/circles/{circleID}", h.handleGetCircle)
	r.Post("/circles/join", h.handleJoinCircle)
	r.Get("/circles/{circleID}/standings", h.handleGetStandings)
	r.Get("/circles/{circleID}/weekly-assembly", h.handleGetWeeklyAssembly)
	r.Post("/circles/{id}/invitations", h.handleInviteToCircle)
	r.Get("/me/invitations", h.handleListMyInvitations)
	r.Post("/me/invitations/{id}/accept", h.handleAcceptCircleInvitation)
	r.Post("/me/invitations/{id}/decline", h.handleDeclineCircleInvitation)
	r.Post("/circles/{id}/seasons/{sid}/end", h.handleEndSeason)
}

func (h *Handler) handleCreateCircle(w http.ResponseWriter, r *http.Request) {
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

	item, err := h.service.CreateCircle(r.Context(), actor, input)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidCircleInput):
			writeError(w, http.StatusBadRequest, "invalid_input", err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "internal_error", "Could not create circle")
		}
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"circle": item.Circle, "members": item.Members, "active_season": item.ActiveSeason})
}

func (h *Handler) handleListCircles(w http.ResponseWriter, r *http.Request) {
	actor, ok := users.CurrentUser(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth_required", "Authentication required")
		return
	}

	items, err := h.service.ListCircles(r.Context(), actor)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Could not load circles")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"circles": items})
}

func (h *Handler) handleGetCircle(w http.ResponseWriter, r *http.Request) {
	actor, ok := users.CurrentUser(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth_required", "Authentication required")
		return
	}
	circleID, ok := pathInt64(w, r, "circleID")
	if !ok {
		return
	}
	item, err := h.service.GetCircle(r.Context(), actor, circleID)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotCircleMember):
			writeError(w, http.StatusForbidden, "forbidden", "You are not a member of this circle")
		case errors.Is(err, ErrCircleNotFound):
			writeError(w, http.StatusNotFound, "circle_not_found", "Circle not found")
		default:
			writeError(w, http.StatusInternalServerError, "internal_error", "Could not load circle")
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"circle": item.Circle, "members": item.Members, "active_season": item.ActiveSeason})
}

func (h *Handler) handleJoinCircle(w http.ResponseWriter, r *http.Request) {
	actor, ok := users.CurrentUser(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth_required", "Authentication required")
		return
	}

	var input JoinInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON")
		return
	}

	item, err := h.service.JoinCircle(r.Context(), actor, input)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidCircleInput):
			writeError(w, http.StatusBadRequest, "invalid_input", err.Error())
		case errors.Is(err, ErrCircleNotFound):
			writeError(w, http.StatusNotFound, "circle_not_found", "Circle not found")
		case errors.Is(err, ErrAlreadyCircleMember):
			writeError(w, http.StatusConflict, "already_member", "User is already in the circle")
		case errors.Is(err, ErrCircleIsFull):
			writeError(w, http.StatusConflict, "circle_full", "Circle already reached the member limit")
		default:
			writeError(w, http.StatusInternalServerError, "internal_error", "Could not join circle")
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"circle": item.Circle, "members": item.Members, "active_season": item.ActiveSeason})
}

func (h *Handler) handleGetStandings(w http.ResponseWriter, r *http.Request) {
	actor, ok := users.CurrentUser(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth_required", "Authentication required")
		return
	}
	circleID, ok := pathInt64(w, r, "circleID")
	if !ok {
		return
	}

	standings, err := h.service.GetStandings(r.Context(), actor, circleID)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotCircleMember):
			writeError(w, http.StatusForbidden, "forbidden", "You are not a member of this circle")
		default:
			writeError(w, http.StatusInternalServerError, "internal_error", "Could not load standings")
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"standings": standings})
}

func (h *Handler) handleGetWeeklyAssembly(w http.ResponseWriter, r *http.Request) {
	actor, ok := users.CurrentUser(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth_required", "Authentication required")
		return
	}
	circleID, ok := pathInt64(w, r, "circleID")
	if !ok {
		return
	}

	assembly, err := h.service.GetWeeklyAssembly(r.Context(), actor, circleID)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotCircleMember):
			writeError(w, http.StatusForbidden, "forbidden", "You are not a member of this circle")
		default:
			writeError(w, http.StatusInternalServerError, "internal_error", "Could not load weekly assembly")
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"circle":       assembly.Circle,
		"season":       assembly.Season,
		"current_week": assembly.CurrentWeek,
		"standings":    assembly.Standings,
		"events":       assembly.Events,
	})
}

func (h *Handler) handleInviteToCircle(w http.ResponseWriter, r *http.Request) {
	actor, ok := users.CurrentUser(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth_required", "Authentication required")
		return
	}
	circleID, ok := pathInt64(w, r, "id")
	if !ok {
		return
	}

	var input InviteToCircleInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON")
		return
	}

	inv, err := h.service.InviteToCircle(r.Context(), actor, circleID, input)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidCircleInput):
			writeError(w, http.StatusBadRequest, "invalid_input", err.Error())
		case errors.Is(err, ErrNotCircleMember):
			writeError(w, http.StatusForbidden, "forbidden", "You are not a member of this circle")
		case errors.Is(err, ErrCircleNotFound):
			writeError(w, http.StatusNotFound, "circle_not_found", "Circle not found")
		default:
			writeError(w, http.StatusInternalServerError, "internal_error", "Could not send invitation")
		}
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"invitation": inv})
}

func (h *Handler) handleListMyInvitations(w http.ResponseWriter, r *http.Request) {
	actor, ok := users.CurrentUser(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth_required", "Authentication required")
		return
	}

	items, err := h.service.ListMyInvitations(r.Context(), actor)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Could not load invitations")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"invitations": items})
}

func (h *Handler) handleAcceptCircleInvitation(w http.ResponseWriter, r *http.Request) {
	actor, ok := users.CurrentUser(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth_required", "Authentication required")
		return
	}
	invitationID, ok := pathInt64(w, r, "id")
	if !ok {
		return
	}

	if err := h.service.AcceptCircleInvitation(r.Context(), actor, invitationID); err != nil {
		switch {
		case errors.Is(err, ErrCircleInvitationNotFound):
			writeError(w, http.StatusNotFound, "invitation_not_found", "Invitation not found")
		case errors.Is(err, ErrCircleInvitationAlreadyDone):
			writeError(w, http.StatusConflict, "invitation_already_done", "Invitation already accepted or declined")
		default:
			writeError(w, http.StatusInternalServerError, "internal_error", "Could not accept invitation")
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"accepted": true})
}

func (h *Handler) handleDeclineCircleInvitation(w http.ResponseWriter, r *http.Request) {
	actor, ok := users.CurrentUser(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth_required", "Authentication required")
		return
	}
	invitationID, ok := pathInt64(w, r, "id")
	if !ok {
		return
	}

	if err := h.service.DeclineCircleInvitation(r.Context(), actor, invitationID); err != nil {
		switch {
		case errors.Is(err, ErrCircleInvitationNotFound):
			writeError(w, http.StatusNotFound, "invitation_not_found", "Invitation not found")
		default:
			writeError(w, http.StatusInternalServerError, "internal_error", "Could not decline invitation")
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"declined": true})
}

func pathInt64(w http.ResponseWriter, r *http.Request, key string) (int64, bool) {
	raw := chi.URLParam(r, key)
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_id", "Path parameter must be a positive integer")
		return 0, false
	}
	return value, true
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func (h *Handler) handleEndSeason(w http.ResponseWriter, r *http.Request) {
	actor, ok := users.CurrentUser(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth_required", "Authentication required")
		return
	}

	circleID, ok := pathInt64(w, r, "id")
	if !ok {
		return
	}
	seasonID, ok := pathInt64(w, r, "sid")
	if !ok {
		return
	}

	var input SeasonEndInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON")
		return
	}

	if input.Action != SeasonEndActionExtend && input.Action != SeasonEndActionStartNew {
		writeError(w, http.StatusBadRequest, "invalid_action", "Action must be 'extend' or 'start_new'")
		return
	}

	result, err := h.service.EndSeason(r.Context(), actor, circleID, seasonID, input)
	if err != nil {
		switch {
		case errors.Is(err, ErrSeasonNotFound):
			writeError(w, http.StatusNotFound, "season_not_found", "Season not found")
		case errors.Is(err, ErrSeasonNotEnded):
			writeError(w, http.StatusConflict, "season_not_ended", "Season has not ended yet")
		case errors.Is(err, ErrSeasonAlreadyDone):
			writeError(w, http.StatusConflict, "season_already_done", "Season is already completed")
		case errors.Is(err, ErrNotCircleMember):
			writeError(w, http.StatusForbidden, "forbidden", "You are not a member of this circle")
		default:
			writeError(w, http.StatusInternalServerError, "internal_error", "Could not end season")
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"result": result})
}

func writeError(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}
