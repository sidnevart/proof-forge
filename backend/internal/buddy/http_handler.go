package buddy

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/sidnevart/proof-forge/backend/internal/users"
)

// Handler exposes buddy dashboard endpoints.
type Handler struct {
	service *Service
}

// NewHandler constructs a buddy Handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes installs buddy endpoints on the given router.
// The router must already be wrapped with the auth middleware.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/buddy/queue", h.handleQueue)
	r.Get("/buddy/needs-attention", h.handleNeedsAttention)
	r.Get("/buddy/stats", h.handleStats)
}

func (h *Handler) handleQueue(w http.ResponseWriter, r *http.Request) {
	actor, ok := users.CurrentUser(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	items, err := h.service.GetQueue(r.Context(), actor.ID)
	if err != nil {
		slog.Error("could not load buddy queue", "user_id", actor.ID, "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": items,
		"meta": map[string]int{"total": len(items)},
	})
}

func (h *Handler) handleNeedsAttention(w http.ResponseWriter, r *http.Request) {
	actor, ok := users.CurrentUser(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	items, err := h.service.GetNeedsAttention(r.Context(), actor.ID)
	if err != nil {
		slog.Error("could not load buddy needs attention", "user_id", actor.ID, "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"data": items})
}

func (h *Handler) handleStats(w http.ResponseWriter, r *http.Request) {
	actor, ok := users.CurrentUser(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	stats, err := h.service.GetStats(r.Context(), actor.ID)
	if err != nil {
		slog.Error("could not load buddy stats", "user_id", actor.ID, "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"data": stats})
}
