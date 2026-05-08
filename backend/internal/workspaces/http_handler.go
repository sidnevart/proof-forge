package workspaces

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/sidnevart/proof-forge/backend/internal/users"
)

// Handler exposes the REST surface for workspace management.
type Handler struct {
	service *Service
}

// NewHandler constructs a Handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes installs workspace endpoints on the given router.
// The router must already be wrapped with the auth middleware.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/workspaces", h.handleCreate)
	r.Get("/workspaces", h.handleList)
	r.Get("/workspaces/{workspaceID}", h.handleGet)
	r.Get("/workspaces/slug/{slug}/available", h.handleSlugCheck)
}

// ── DTOs ──────────────────────────────────────────────────────────────

type createInput struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
	Type string `json:"type"`
}

type workspaceDTO struct {
	ID          int64      `json:"id"`
	OwnerUserID int64      `json:"owner_user_id"`
	Name        string     `json:"name"`
	Slug        string     `json:"slug"`
	Type        string     `json:"type"`
	IsActive    bool       `json:"is_active"`
	FrozenAt    *time.Time `json:"frozen_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

type slugAvailableDTO struct {
	Available bool `json:"available"`
}

func toDTO(w *Workspace) workspaceDTO {
	return workspaceDTO{
		ID:          w.ID,
		OwnerUserID: w.OwnerUserID,
		Name:        w.Name,
		Slug:        w.Slug,
		Type:        string(w.Type),
		IsActive:    w.IsActive,
		FrozenAt:    w.FrozenAt,
		CreatedAt:   w.CreatedAt,
	}
}

// ── Handlers ──────────────────────────────────────────────────────────

func (h *Handler) handleCreate(w http.ResponseWriter, r *http.Request) {
	actor, ok := users.CurrentUser(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var in createInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	workspace, err := h.service.Create(r.Context(), CreateInput{
		OwnerUserID: actor.ID,
		Name:        in.Name,
		Slug:        in.Slug,
		Type:        in.Type,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrSlugTaken):
			http.Error(w, "slug already taken", http.StatusConflict)
		case errors.Is(err, ErrInvalidWorkspace):
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(toDTO(workspace))
}

func (h *Handler) handleList(w http.ResponseWriter, r *http.Request) {
	actor, ok := users.CurrentUser(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	workspaces, err := h.service.ListByOwner(r.Context(), actor.ID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	dtos := make([]workspaceDTO, len(workspaces))
	for i, ws := range workspaces {
		dtos[i] = toDTO(ws)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(dtos)
}

func (h *Handler) handleGet(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "workspaceID"), 10, 64)
	if err != nil {
		http.Error(w, "invalid workspace id", http.StatusBadRequest)
		return
	}

	workspace, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrWorkspaceNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(toDTO(workspace))
}

func (h *Handler) handleSlugCheck(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	available, err := h.service.CheckSlugAvailable(r.Context(), slug)
	if err != nil {
		if errors.Is(err, ErrInvalidWorkspace) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(slugAvailableDTO{Available: available})
}
