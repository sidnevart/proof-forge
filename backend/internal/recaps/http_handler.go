package recaps

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
	r.Get("/goals/{goalID}/recaps", h.handleList)
	r.Get("/recaps/{recapID}", h.handleGet)
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

	recaps, err := h.service.ListForGoal(r.Context(), actor, goalID)
	if err != nil {
		h.writeServiceError(w, r, "list recaps", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"recaps": recaps})
}

func (h *Handler) handleGet(w http.ResponseWriter, r *http.Request) {
	actor, ok := currentUser(w, r)
	if !ok {
		return
	}
	recapID, ok := pathInt64(w, r, "recapID")
	if !ok {
		return
	}

	recap, err := h.service.GetRecap(r.Context(), actor, recapID)
	if err != nil {
		h.writeServiceError(w, r, "get recap", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"recap": recap})
}

func (h *Handler) writeServiceError(w http.ResponseWriter, r *http.Request, op string, err error) {
	switch {
	case errors.Is(err, ErrRecapNotFound):
		writeError(w, http.StatusNotFound, "not_found", "Recap not found")
	case errors.Is(err, ErrNotAuthorized):
		writeError(w, http.StatusForbidden, "forbidden", "Not authorized to access this recap")
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
