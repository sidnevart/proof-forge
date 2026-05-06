package inspiration

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

// RegisterPublicRoutes mounts read-only feed routes — no auth required.
func (h *Handler) RegisterPublicRoutes(r chi.Router) {
	r.Get("/inspiration/proofs", h.handleListProofs)
	r.Get("/library/templates", h.handleListTemplates)
}

// RegisterRoutes mounts authenticated mutation routes.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/feed/circles", h.handleCirclesFeed)
	r.Post("/goals/{id}/visibility", h.handleGoalVisibility)
	r.Post("/checkins/{id}/visibility", h.handleCheckInVisibility)
	r.Post("/me/sharing", h.handleSharingPrefs)
	r.Post("/inspiration/report", h.handleReport)
}

func (h *Handler) handleCirclesFeed(w http.ResponseWriter, r *http.Request) {
	actor, ok := users.CurrentUser(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth_required", "Требуется авторизация")
		return
	}
	q := r.URL.Query()
	cursor, _ := strconv.ParseInt(q.Get("cursor"), 10, 64)
	limit, _ := strconv.Atoi(q.Get("limit"))

	items, err := h.service.ListCirclesFeed(r.Context(), actor.ID, CircleFeedParams{
		Cursor: cursor,
		Limit:  limit,
	})
	if err != nil {
		h.log.Error("list circles feed", "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "Не удалось загрузить ленту")
		return
	}
	if items == nil {
		items = []CircleFeedItem{}
	}

	var nextCursor *int64
	if len(items) > 0 {
		last := items[len(items)-1].ID
		nextCursor = &last
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items":       items,
		"next_cursor": nextCursor,
	})
}

func (h *Handler) handleListProofs(w http.ResponseWriter, r *http.Request) {
	p := parseFeedParams(r)
	proofs, err := h.service.ListProofs(r.Context(), p)
	if err != nil {
		h.log.Error("list proofs", "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "Не удалось загрузить ленту")
		return
	}
	if proofs == nil {
		proofs = []PublicProof{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"proofs": proofs})
}

func (h *Handler) handleListTemplates(w http.ResponseWriter, r *http.Request) {
	p := parseFeedParams(r)
	goals, err := h.service.ListTemplates(r.Context(), p)
	if err != nil {
		h.log.Error("list templates", "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "Не удалось загрузить библиотеку")
		return
	}
	if goals == nil {
		goals = []PublicGoal{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"templates": goals})
}

func (h *Handler) handleGoalVisibility(w http.ResponseWriter, r *http.Request) {
	actor, ok := users.CurrentUser(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth_required", "Требуется авторизация")
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "Неверный ID цели")
		return
	}
	var in VisibilityInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Неверный формат запроса")
		return
	}
	if err := h.service.SetGoalVisibility(r.Context(), id, actor.ID, in); err != nil {
		if errors.Is(err, ErrNotAuthorized) {
			writeError(w, http.StatusForbidden, "forbidden", "Нет прав")
			return
		}
		h.log.Error("set goal visibility", "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "Ошибка")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) handleCheckInVisibility(w http.ResponseWriter, r *http.Request) {
	actor, ok := users.CurrentUser(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth_required", "Требуется авторизация")
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "Неверный ID чек-ина")
		return
	}
	var in VisibilityInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Неверный формат запроса")
		return
	}
	if err := h.service.SetCheckInVisibility(r.Context(), id, actor.ID, in); err != nil {
		if errors.Is(err, ErrNotAuthorized) {
			writeError(w, http.StatusForbidden, "forbidden", "Нет прав")
			return
		}
		h.log.Error("set checkin visibility", "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "Ошибка")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) handleSharingPrefs(w http.ResponseWriter, r *http.Request) {
	actor, ok := users.CurrentUser(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth_required", "Требуется авторизация")
		return
	}
	var in SharingPrefsInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Неверный формат запроса")
		return
	}
	if err := h.service.UpdateSharingPrefs(r.Context(), actor.ID, in); err != nil {
		h.log.Error("update sharing prefs", "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "Ошибка")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) handleReport(w http.ResponseWriter, r *http.Request) {
	actor, ok := users.CurrentUser(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "auth_required", "Требуется авторизация")
		return
	}
	var in ReportInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Неверный формат запроса")
		return
	}
	if err := h.service.CreateReport(r.Context(), actor.ID, in); err != nil {
		if errors.Is(err, ErrInvalidReportInput) {
			writeError(w, http.StatusBadRequest, "invalid_report", "Неверный формат жалобы")
			return
		}
		h.log.Error("create report", "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "Ошибка")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func parseFeedParams(r *http.Request) FeedParams {
	q := r.URL.Query()
	cursor, _ := strconv.ParseInt(q.Get("cursor"), 10, 64)
	limit, _ := strconv.Atoi(q.Get("limit"))
	similar, _ := strconv.ParseInt(q.Get("similar_to_goal_id"), 10, 64)
	return FeedParams{
		Query:           q.Get("q"),
		Category:        q.Get("category"),
		SimilarToGoalID: similar,
		Cursor:          cursor,
		Limit:           limit,
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, map[string]string{"error": code, "message": msg})
}
