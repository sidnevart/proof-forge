package stats

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/sidnevart/proof-forge/backend/internal/users"
)

// Handler exposes personal stats and the NowCard endpoint.
type Handler struct {
	service *Service
}

// NewHandler constructs a stats Handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes installs the stats endpoints on the given router.
// The router must already be wrapped with the auth middleware.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/me/stats", h.handleStats)
	r.Get("/me/now", h.handleNow)
}

func (h *Handler) handleStats(w http.ResponseWriter, r *http.Request) {
	actor, ok := users.CurrentUser(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	stats, err := h.service.GetStats(r.Context(), actor.ID)
	if err != nil {
		slog.Error("could not load user stats", "user_id", actor.ID, "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"data": stats})
}

func (h *Handler) handleNow(w http.ResponseWriter, r *http.Request) {
	actor, ok := users.CurrentUser(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	card, err := h.service.GetNowCard(r.Context(), actor.ID)
	if err != nil {
		slog.Error("could not load now card", "user_id", actor.ID, "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"data": card})
}
