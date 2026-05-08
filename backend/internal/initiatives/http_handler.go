package initiatives

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
	// Space-scoped
	r.Post("/spaces/{type}/{spaceID}/initiatives", h.handleCreate)
	r.Get("/spaces/{type}/{spaceID}/initiatives", h.handleList)

	// Initiative-scoped
	r.Get("/initiatives/{id}", h.handleGetDetail)
	r.Post("/initiatives/{id}/join", h.handleJoin)
	r.Get("/initiatives/{id}/pending-proofs", h.handlePendingProofs)
	r.Post("/initiatives/{id}/proofs/{checkinID}/approve", h.handleApprove)
}

func (h *Handler) handleCreate(w http.ResponseWriter, r *http.Request) {
	actor, ok := users.CurrentUser(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "auth_required", "Authentication required")
		return
	}

	spaceType := chi.URLParam(r, "type")
	spaceID, err := strconv.ParseInt(chi.URLParam(r, "spaceID"), 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_space_id", "Invalid space ID")
		return
	}

	var body struct {
		Title         string `json:"title"`
		Description   string `json:"description"`
		ProofCriteria string `json:"proof_criteria"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_body", "Invalid request body")
		return
	}

	initiative, err := h.service.Create(r.Context(), actor, CreateInput{
		SpaceType:     spaceType,
		SpaceID:       spaceID,
		Title:         body.Title,
		Description:   body.Description,
		ProofCriteria: body.ProofCriteria,
	})
	if err != nil {
		h.mapErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, initiative)
}

func (h *Handler) handleList(w http.ResponseWriter, r *http.Request) {
	spaceType := chi.URLParam(r, "type")
	spaceID, err := strconv.ParseInt(chi.URLParam(r, "spaceID"), 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_space_id", "Invalid space ID")
		return
	}

	list, err := h.service.List(r.Context(), spaceType, spaceID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal_error", "Could not list initiatives")
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *Handler) handleGetDetail(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_id", "Invalid initiative ID")
		return
	}
	detail, err := h.service.GetDetail(r.Context(), id)
	if err != nil {
		h.mapErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (h *Handler) handleJoin(w http.ResponseWriter, r *http.Request) {
	actor, ok := users.CurrentUser(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "auth_required", "Authentication required")
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_id", "Invalid initiative ID")
		return
	}
	result, err := h.service.Join(r.Context(), actor, id)
	if err != nil {
		h.mapErr(w, err)
		return
	}
	status := http.StatusOK
	if result.Created {
		status = http.StatusCreated
	}
	writeJSON(w, status, result)
}

func (h *Handler) handlePendingProofs(w http.ResponseWriter, r *http.Request) {
	actor, ok := users.CurrentUser(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "auth_required", "Authentication required")
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_id", "Invalid initiative ID")
		return
	}
	proofs, err := h.service.PendingProofs(r.Context(), actor, id)
	if err != nil {
		h.mapErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, proofs)
}

func (h *Handler) handleApprove(w http.ResponseWriter, r *http.Request) {
	actor, ok := users.CurrentUser(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "auth_required", "Authentication required")
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_id", "Invalid initiative ID")
		return
	}
	checkinID, err := strconv.ParseInt(chi.URLParam(r, "checkinID"), 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_checkin_id", "Invalid checkin ID")
		return
	}

	var body struct {
		Comment string `json:"comment"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	if err := h.service.Approve(r.Context(), actor, id, checkinID, body.Comment); err != nil {
		h.mapErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"approved": true})
}

func (h *Handler) mapErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInitiativeNotFound):
		writeErr(w, http.StatusNotFound, "initiative_not_found", "Initiative not found")
	case errors.Is(err, ErrNotSpaceMember):
		writeErr(w, http.StatusForbidden, "not_space_member", "You are not a member of this space")
	case errors.Is(err, ErrNotInitiativeMember):
		writeErr(w, http.StatusForbidden, "not_initiative_member", "You are not a participant of this initiative")
	case errors.Is(err, ErrCannotApproveSelf):
		writeErr(w, http.StatusForbidden, "cannot_approve_self", "You cannot approve your own proof")
	case errors.Is(err, ErrInitiativeArchived):
		writeErr(w, http.StatusGone, "initiative_archived", "This initiative is archived")
	case errors.Is(err, ErrCheckinNotFound):
		writeErr(w, http.StatusNotFound, "checkin_not_found", "Proof not found or already approved")
	case errors.Is(err, ErrInvalidInput):
		writeErr(w, http.StatusBadRequest, "invalid_input", err.Error())
	default:
		writeErr(w, http.StatusInternalServerError, "internal_error", "An error occurred")
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeErr(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": code, "message": message})
}
