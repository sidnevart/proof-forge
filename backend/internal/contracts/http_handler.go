package contracts

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/sidnevart/proof-forge/backend/internal/analytics"
	"github.com/sidnevart/proof-forge/backend/internal/users"
)

// Handler exposes the REST surface for proof contract management.
type Handler struct {
	service *Service
	tracker analytics.Tracker
}

// NewHandler constructs a Handler.
func NewHandler(service *Service, opts ...func(*Handler)) *Handler {
	h := &Handler{service: service, tracker: analytics.NoopRecorder{}}
	for _, o := range opts {
		o(h)
	}
	return h
}

// WithTracker injects an analytics tracker into the contracts handler.
func WithTracker(t analytics.Tracker) func(*Handler) {
	return func(h *Handler) { h.tracker = t }
}

// RegisterRoutes installs contract endpoints on the given router.
// The router must already be wrapped with the auth middleware.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/goals/{goalID}/contracts", h.handleCreate)
	r.Get("/goals/{goalID}/contracts", h.handleListByGoal)
	r.Get("/contracts/{contractID}", h.handleGet)
	r.Get("/contracts", h.handleListActive)
	r.Post("/contracts/{contractID}/activate", h.handleActivate)
	r.Post("/contracts/{contractID}/fulfill", h.handleFulfill)
	r.Post("/contracts/{contractID}/cancel", h.handleCancel)
}

// ── DTOs ──────────────────────────────────────────────────────────────

type createContractInput struct {
	BuddyUserID *int64 `json:"buddy_user_id,omitempty"`
	WhatToProve string `json:"what_to_prove"`
	HowToProve  string `json:"how_to_prove"`
	DueAt       string `json:"due_at"` // RFC3339
}

type contractDTO struct {
	ID          int64      `json:"id"`
	GoalID      int64      `json:"goal_id"`
	UserID      int64      `json:"user_id"`
	BuddyUserID *int64     `json:"buddy_user_id,omitempty"`
	WhatToProve string     `json:"what_to_prove"`
	HowToProve  string     `json:"how_to_prove"`
	DueAt       time.Time  `json:"due_at"`
	Status      string     `json:"status"`
	Urgency     string     `json:"urgency"`
	FulfilledAt *time.Time `json:"fulfilled_at,omitempty"`
	BrokenAt    *time.Time `json:"broken_at,omitempty"`
	CancelledAt *time.Time `json:"cancelled_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func toContractDTO(c *ProofContract) contractDTO {
	return contractDTO{
		ID:          c.ID,
		GoalID:      c.GoalID,
		UserID:      c.UserID,
		BuddyUserID: c.BuddyUserID,
		WhatToProve: c.WhatToProve,
		HowToProve:  c.HowToProve,
		DueAt:       c.DueAt,
		Status:      string(c.Status),
		Urgency:     string(c.Urgency()),
		FulfilledAt: c.FulfilledAt,
		BrokenAt:    c.BrokenAt,
		CancelledAt: c.CancelledAt,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}
}

// ── Handlers ──────────────────────────────────────────────────────────

func (h *Handler) handleCreate(w http.ResponseWriter, r *http.Request) {
	actor, ok := users.CurrentUser(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	goalID, err := strconv.ParseInt(chi.URLParam(r, "goalID"), 10, 64)
	if err != nil {
		http.Error(w, "invalid goal id", http.StatusBadRequest)
		return
	}

	var in createContractInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	dueAt, err := time.Parse(time.RFC3339, in.DueAt)
	if err != nil {
		http.Error(w, "due_at must be RFC3339", http.StatusBadRequest)
		return
	}

	contract, err := h.service.Create(r.Context(), CreateInput{
		GoalID:      goalID,
		UserID:      actor.ID,
		BuddyUserID: in.BuddyUserID,
		WhatToProve: in.WhatToProve,
		HowToProve:  in.HowToProve,
		DueAt:       dueAt,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidContract):
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}

	dueInDays := int(time.Until(contract.DueAt).Hours() / 24)
	h.tracker.TrackAsync(analytics.PilotEvent{
		UserID: actor.ID,
		GoalID: &contract.GoalID,
		Name:   analytics.EventProofContractCreated,
		Props: map[string]any{
			"goal_id":     contract.GoalID,
			"due_in_days": dueInDays,
		},
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(toContractDTO(contract))
}

func (h *Handler) handleListByGoal(w http.ResponseWriter, r *http.Request) {
	goalID, err := strconv.ParseInt(chi.URLParam(r, "goalID"), 10, 64)
	if err != nil {
		http.Error(w, "invalid goal id", http.StatusBadRequest)
		return
	}

	list, err := h.service.ListByGoal(r.Context(), goalID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	dtos := make([]contractDTO, len(list))
	for i, c := range list {
		dtos[i] = toContractDTO(c)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(dtos)
}

func (h *Handler) handleGet(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "contractID"), 10, 64)
	if err != nil {
		http.Error(w, "invalid contract id", http.StatusBadRequest)
		return
	}

	contract, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrContractNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(toContractDTO(contract))
}

func (h *Handler) handleListActive(w http.ResponseWriter, r *http.Request) {
	actor, ok := users.CurrentUser(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	list, err := h.service.ListActive(r.Context(), actor.ID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	dtos := make([]contractDTO, len(list))
	for i, c := range list {
		dtos[i] = toContractDTO(c)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(dtos)
}

func (h *Handler) handleActivate(w http.ResponseWriter, r *http.Request) {
	actor, ok := users.CurrentUser(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "contractID"), 10, 64)
	if err != nil {
		http.Error(w, "invalid contract id", http.StatusBadRequest)
		return
	}

	if err := h.service.Activate(r.Context(), id, actor.ID); err != nil {
		switch {
		case errors.Is(err, ErrContractNotFound):
			http.Error(w, "not found", http.StatusNotFound)
		case errors.Is(err, ErrForbidden):
			http.Error(w, "forbidden", http.StatusForbidden)
		case errors.Is(err, ErrInvalidTransition):
			http.Error(w, err.Error(), http.StatusConflict)
		default:
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleFulfill(w http.ResponseWriter, r *http.Request) {
	actor, ok := users.CurrentUser(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "contractID"), 10, 64)
	if err != nil {
		http.Error(w, "invalid contract id", http.StatusBadRequest)
		return
	}

	if err := h.service.Fulfill(r.Context(), id, actor.ID); err != nil {
		switch {
		case errors.Is(err, ErrContractNotFound):
			http.Error(w, "not found", http.StatusNotFound)
		case errors.Is(err, ErrForbidden):
			http.Error(w, "forbidden", http.StatusForbidden)
		case errors.Is(err, ErrInvalidTransition):
			http.Error(w, err.Error(), http.StatusConflict)
		default:
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleCancel(w http.ResponseWriter, r *http.Request) {
	actor, ok := users.CurrentUser(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "contractID"), 10, 64)
	if err != nil {
		http.Error(w, "invalid contract id", http.StatusBadRequest)
		return
	}

	if err := h.service.Cancel(r.Context(), id, actor.ID); err != nil {
		switch {
		case errors.Is(err, ErrContractNotFound):
			http.Error(w, "not found", http.StatusNotFound)
		case errors.Is(err, ErrForbidden):
			http.Error(w, "forbidden", http.StatusForbidden)
		case errors.Is(err, ErrInvalidTransition):
			http.Error(w, err.Error(), http.StatusConflict)
		default:
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
