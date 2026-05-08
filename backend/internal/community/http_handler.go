package community

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/sidnevart/proof-forge/backend/internal/users"
)

// Handler exposes the REST surface for community space management.
type Handler struct {
	service *Service
}

// NewHandler constructs a Handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes installs community space endpoints.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/community-spaces", h.handleCreate)
	r.Get("/community-spaces/{id}", h.handleGet)
	r.Post("/community-spaces/join", h.handleJoin)
}

// ── DTOs ──────────────────────────────────────────────────────────────

type createInput struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	IsPublic    bool   `json:"is_public"`
	WorkspaceID *int64 `json:"workspace_id,omitempty"`
}

type joinInput struct {
	InviteCode string `json:"invite_code"`
}

type communityDTO struct {
	ID          int64      `json:"id"`
	WorkspaceID *int64     `json:"workspace_id,omitempty"`
	OwnerUserID int64      `json:"owner_user_id"`
	Name        string     `json:"name"`
	Slug        string     `json:"slug"`
	Description string     `json:"description"`
	InviteCode  string     `json:"invite_code"`
	IsPublic    bool       `json:"is_public"`
	CreatedAt   time.Time  `json:"created_at"`
}

type membershipDTO struct {
	CommunitySpaceID int64     `json:"community_space_id"`
	UserID           int64     `json:"user_id"`
	Role             string    `json:"role"`
	Status           string    `json:"status"`
	JoinedAt         time.Time `json:"joined_at"`
}

func toDTO(cs *CommunitySpace) communityDTO {
	return communityDTO{
		ID:          cs.ID,
		WorkspaceID: cs.WorkspaceID,
		OwnerUserID: cs.OwnerUserID,
		Name:        cs.Name,
		Slug:        cs.Slug,
		Description: cs.Description,
		InviteCode:  cs.InviteCode,
		IsPublic:    cs.IsPublic,
		CreatedAt:   cs.CreatedAt,
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

	cs, err := h.service.Create(r.Context(), CreateInput{
		OwnerUserID: actor.ID,
		WorkspaceID: in.WorkspaceID,
		Name:        in.Name,
		Slug:        in.Slug,
		Description: in.Description,
		IsPublic:    in.IsPublic,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrSlugTaken):
			http.Error(w, "slug already taken", http.StatusConflict)
		case errors.Is(err, ErrInvalidCommunity):
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(toDTO(cs))
}

func (h *Handler) handleGet(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	cs, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrCommunityNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(toDTO(cs))
}

func (h *Handler) handleJoin(w http.ResponseWriter, r *http.Request) {
	actor, ok := users.CurrentUser(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var in joinInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	m, err := h.service.Join(r.Context(), actor.ID, in.InviteCode)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInviteCode):
			http.Error(w, "invalid invite code", http.StatusBadRequest)
		case errors.Is(err, ErrAlreadyMember):
			http.Error(w, "already a member", http.StatusConflict)
		default:
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(membershipDTO{
		CommunitySpaceID: m.CommunitySpaceID,
		UserID:           m.UserID,
		Role:             string(m.Role),
		Status:           string(m.Status),
		JoinedAt:         m.JoinedAt,
	})
}
