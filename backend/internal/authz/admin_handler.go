package authz

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AdminHandler exposes platform-admin management endpoints.
type AdminHandler struct {
	pool *pgxpool.Pool
}

// NewAdminHandler constructs an AdminHandler.
func NewAdminHandler(pool *pgxpool.Pool) *AdminHandler {
	return &AdminHandler{pool: pool}
}

// RegisterAdminRoutes installs admin endpoints.
// The router MUST already be wrapped with auth + RequirePlatformAdmin middleware.
func (h *AdminHandler) RegisterAdminRoutes(r chi.Router) {
	r.Post("/users/{userID}/make-admin", h.handleMakeAdmin)
	r.Delete("/users/{userID}/make-admin", h.handleRevokeAdmin)
}

func (h *AdminHandler) handleMakeAdmin(w http.ResponseWriter, r *http.Request) {
	h.setAdmin(w, r, true)
}

func (h *AdminHandler) handleRevokeAdmin(w http.ResponseWriter, r *http.Request) {
	h.setAdmin(w, r, false)
}

func (h *AdminHandler) setAdmin(w http.ResponseWriter, r *http.Request, isAdmin bool) {
	userID, err := strconv.ParseInt(chi.URLParam(r, "userID"), 10, 64)
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	if _, err := h.pool.Exec(r.Context(),
		`UPDATE users SET is_platform_admin = $2, updated_at = NOW() WHERE id = $1`,
		userID, isAdmin,
	); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"user_id":          userID,
		"is_platform_admin": isAdmin,
	})
}
