package companion

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"log/slog"

	"github.com/sidnevart/proof-forge/backend/internal/users"
)

// Handler exposes companion endpoints.
type Handler struct {
	service *Service
	pool    *pgxpool.Pool
	log     *slog.Logger
}

// NewHandler constructs a Handler.
func NewHandler(service *Service, pool *pgxpool.Pool, log *slog.Logger) *Handler {
	return &Handler{service: service, pool: pool, log: log}
}

// RegisterRoutes installs companion endpoints.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/me/ai/notifications", h.handleListNotifications)
	r.Post("/me/ai/notifications/{id}/dismiss", h.handleDismissNotification)
	r.Get("/me/ai/drafts", h.handleListProofDrafts)
	r.Post("/me/ai/drafts/{id}/accept", h.handleAcceptProofDraft)
	r.Post("/me/ai/drafts/{id}/reject", h.handleRejectProofDraft)
	r.Get("/teams/{teamID}/ai/health", h.handleTeamHealth)
}

// --- DTOs ---

type notificationDTO struct {
	ID        string   `json:"id"`
	Feature   string   `json:"feature"`
	Title     string   `json:"title"`
	Body      string   `json:"body"`
	Actions   []NotificationAction `json:"actions"`
	CreatedAt string   `json:"created_at"`
}

type proofDraftDTO struct {
	ID         string  `json:"id"`
	TeamID     int64   `json:"team_id"`
	GoalID     *int64  `json:"goal_id,omitempty"`
	NoteIDs    []int64 `json:"note_ids"`
	Rationale  string  `json:"rationale"`
	Confidence string  `json:"confidence"`
	CreatedAt  string  `json:"created_at"`
}

type teamHealthDTO struct {
	TeamHealthScore int      `json:"team_health_score"`
	FairPlayStatus  string   `json:"fair_play_status"`
	PendingApprovals int     `json:"pending_approvals"`
	Alerts          []alertDTO `json:"alerts"`
}

type alertDTO struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// --- Handlers ---

func (h *Handler) handleListNotifications(w http.ResponseWriter, r *http.Request) {
	actor, ok := users.CurrentUser(r.Context())
	if !ok {
		writeErrJSON(w, http.StatusUnauthorized, "auth_required", "Authentication required")
		return
	}

	notes, err := h.service.GetActiveNotifications(r.Context(), actor.ID)
	if err != nil {
		h.log.Error("list notifications", "err", err)
		writeErrJSON(w, http.StatusInternalServerError, "db_error", "Could not fetch notifications")
		return
	}

	dtos := make([]notificationDTO, 0, len(notes))
	for _, n := range notes {
		dtos = append(dtos, notificationDTO{
			ID:        n.ID,
			Feature:   string(n.Feature),
			Title:     n.Title,
			Body:      n.Body,
			Actions:   n.Actions,
			CreatedAt: n.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"notifications": dtos})
}

func (h *Handler) handleDismissNotification(w http.ResponseWriter, r *http.Request) {
	actor, ok := users.CurrentUser(r.Context())
	if !ok {
		writeErrJSON(w, http.StatusUnauthorized, "auth_required", "Authentication required")
		return
	}

	id := chi.URLParam(r, "id")
	if err := h.service.DismissNotification(r.Context(), actor.ID, id); err != nil {
		h.log.Error("dismiss notification", "err", err)
		writeErrJSON(w, http.StatusInternalServerError, "db_error", "Could not dismiss notification")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) handleListProofDrafts(w http.ResponseWriter, r *http.Request) {
	actor, ok := users.CurrentUser(r.Context())
	if !ok {
		writeErrJSON(w, http.StatusUnauthorized, "auth_required", "Authentication required")
		return
	}

	drafts, err := h.service.GetProofDrafts(r.Context(), actor.ID)
	if err != nil {
		h.log.Error("list proof drafts", "err", err)
		writeErrJSON(w, http.StatusInternalServerError, "db_error", "Could not fetch drafts")
		return
	}

	dtos := make([]proofDraftDTO, 0, len(drafts))
	for _, d := range drafts {
		dtos = append(dtos, proofDraftDTO{
			ID:         d.ID,
			TeamID:     d.TeamID,
			GoalID:     d.GoalID,
			NoteIDs:    d.NoteIDs,
			Rationale:  d.Rationale,
			Confidence: d.Confidence,
			CreatedAt:  d.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"drafts": dtos})
}

func (h *Handler) handleAcceptProofDraft(w http.ResponseWriter, r *http.Request) {
	actor, ok := users.CurrentUser(r.Context())
	if !ok {
		writeErrJSON(w, http.StatusUnauthorized, "auth_required", "Authentication required")
		return
	}

	id := chi.URLParam(r, "id")
	draft, err := h.service.AcceptProofDraft(r.Context(), actor.ID, id)
	if err != nil {
		h.log.Error("accept proof draft", "err", err)
		writeErrJSON(w, http.StatusNotFound, "draft_not_found", "Draft not found or already consumed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"draft": proofDraftDTO{
			ID:         draft.ID,
			TeamID:     draft.TeamID,
			GoalID:     draft.GoalID,
			NoteIDs:    draft.NoteIDs,
			Rationale:  draft.Rationale,
			Confidence: draft.Confidence,
			CreatedAt:  draft.CreatedAt.Format("2006-01-02T15:04:05Z"),
		},
	})
}

func (h *Handler) handleRejectProofDraft(w http.ResponseWriter, r *http.Request) {
	actor, ok := users.CurrentUser(r.Context())
	if !ok {
		writeErrJSON(w, http.StatusUnauthorized, "auth_required", "Authentication required")
		return
	}

	id := chi.URLParam(r, "id")
	if err := h.service.RejectProofDraft(r.Context(), actor.ID, id); err != nil {
		h.log.Error("reject proof draft", "err", err)
		writeErrJSON(w, http.StatusNotFound, "draft_not_found", "Draft not found or already consumed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) handleTeamHealth(w http.ResponseWriter, r *http.Request) {
	actor, ok := users.CurrentUser(r.Context())
	if !ok {
		writeErrJSON(w, http.StatusUnauthorized, "auth_required", "Authentication required")
		return
	}

	teamIDStr := chi.URLParam(r, "teamID")
	teamID, err := strconv.ParseInt(teamIDStr, 10, 64)
	if err != nil {
		writeErrJSON(w, http.StatusBadRequest, "invalid_team_id", "Invalid team ID")
		return
	}

	// TODO: verify actor is lead/trusted for this team.
	_ = actor
	_ = teamID

	// Placeholder: will be implemented when team health metrics are available.
	writeJSON(w, http.StatusOK, teamHealthDTO{
		TeamHealthScore:  78,
		FairPlayStatus:   "green",
		PendingApprovals: 0,
		Alerts:           []alertDTO{},
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeErrJSON(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": code, "message": message})
}
