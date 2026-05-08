// Package leaderboards provides endpoints for all leaderboard and board types.
package leaderboards

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sidnevart/proof-forge/backend/internal/authz"
	"github.com/sidnevart/proof-forge/backend/internal/users"
)

// Handler provides all leaderboard and needs-help endpoints.
type Handler struct {
	pool *pgxpool.Pool
	az   *authz.Authorizer
}

// NewHandler constructs a Handler.
func NewHandler(pool *pgxpool.Pool, az *authz.Authorizer) *Handler {
	return &Handler{pool: pool, az: az}
}

// RegisterRoutes installs all leaderboard routes on an auth-wrapped router.
func (h *Handler) RegisterRoutes(r chi.Router) {
	// Personal boards
	r.Get("/me/leaderboard", h.handlePersonalLeaderboard)
	r.Get("/me/achievements", h.handleAchievements)

	// Circle boards
	r.Get("/circles/{circleID}/stability-board", h.handleStabilityBoard)
	r.Get("/circles/{circleID}/contribution-board", h.handleContributionBoard)
	r.Get("/circles/{circleID}/best-proofs-board", h.handleBestProofsBoard)
	r.Get("/circles/{circleID}/growth-board", h.handleGrowthBoard)

	// Community boards
	r.Get("/community-spaces/{communityID}/circles-board", h.handleCirclesBoard)
	r.Get("/community-spaces/{communityID}/needs-help", h.handleCommunityNeedsHelp)

	// Teamspace boards
	r.Get("/teamspaces/{teamID}/needs-help", h.handleTeamspaceNeedsHelp)

	// Like toggle (proof artifacts)
	r.Post("/check-ins/{checkInID}/like", h.handleLike)
}

// ── Personal ─────────────────────────────────────────────────────────────────

func (h *Handler) handlePersonalLeaderboard(w http.ResponseWriter, r *http.Request) {
	actor, ok := currentUser(w, r)
	if !ok {
		return
	}
	lb, err := getPersonalLeaderboard(r.Context(), h.pool, actor.ID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"data": lb})
}

func (h *Handler) handleAchievements(w http.ResponseWriter, r *http.Request) {
	actor, ok := currentUser(w, r)
	if !ok {
		return
	}
	resp, err := getAchievements(r.Context(), h.pool, actor.ID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"data": resp})
}

// ── Circle boards ────────────────────────────────────────────────────────────

func (h *Handler) handleStabilityBoard(w http.ResponseWriter, r *http.Request) {
	actor, ok := currentUser(w, r)
	if !ok {
		return
	}
	circleID, ok := pathInt64(w, r, "circleID")
	if !ok {
		return
	}
	if !h.requireCircleMember(w, r, circleID, actor.ID) {
		return
	}
	limit := queryIntDefault(r, "limit", 5, 1, 10)
	board, err := getStabilityBoard(r.Context(), h.pool, circleID, actor.ID, limit, 4)
	if err != nil {
		slog.Error("failed to load stability board", "circle_id", circleID, "user_id", actor.ID, "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"data": board})
}

func (h *Handler) handleContributionBoard(w http.ResponseWriter, r *http.Request) {
	actor, ok := currentUser(w, r)
	if !ok {
		return
	}
	circleID, ok := pathInt64(w, r, "circleID")
	if !ok {
		return
	}
	if !h.requireCircleMember(w, r, circleID, actor.ID) {
		return
	}
	limit := queryIntDefault(r, "limit", 5, 1, 10)
	periodWeeks := periodToWeeks(r.URL.Query().Get("period"))
	board, err := getContributionBoard(r.Context(), h.pool, circleID, actor.ID, limit, periodWeeks)
	if err != nil {
		slog.Error("failed to load contribution board", "circle_id", circleID, "user_id", actor.ID, "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"data": board})
}

func (h *Handler) handleBestProofsBoard(w http.ResponseWriter, r *http.Request) {
	actor, ok := currentUser(w, r)
	if !ok {
		return
	}
	circleID, ok := pathInt64(w, r, "circleID")
	if !ok {
		return
	}
	if !h.requireCircleMember(w, r, circleID, actor.ID) {
		return
	}
	limit := queryIntDefault(r, "limit", 5, 1, 20)
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "this_week"
	}
	board, err := getBestProofsBoard(r.Context(), h.pool, circleID, actor.ID, limit, period)
	if err != nil {
		slog.Error("failed to load best proofs board", "circle_id", circleID, "user_id", actor.ID, "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"data": board})
}

func (h *Handler) handleGrowthBoard(w http.ResponseWriter, r *http.Request) {
	actor, ok := currentUser(w, r)
	if !ok {
		return
	}
	circleID, ok := pathInt64(w, r, "circleID")
	if !ok {
		return
	}
	if !h.requireCircleMember(w, r, circleID, actor.ID) {
		return
	}
	limit := queryIntDefault(r, "limit", 5, 1, 10)
	board, err := getGrowthBoard(r.Context(), h.pool, circleID, actor.ID, limit)
	if err != nil {
		slog.Error("failed to load growth board", "circle_id", circleID, "user_id", actor.ID, "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"data": board})
}

// ── Community boards ─────────────────────────────────────────────────────────

func (h *Handler) handleCirclesBoard(w http.ResponseWriter, r *http.Request) {
	actor, ok := currentUser(w, r)
	if !ok {
		return
	}
	communityID, ok := pathInt64(w, r, "communityID")
	if !ok {
		return
	}
	if !h.requireCommunityMember(w, r, communityID, actor.ID) {
		return
	}
	periodWeeks := periodToWeeks(r.URL.Query().Get("period"))
	board, err := getCirclesBoard(r.Context(), h.pool, communityID, actor.ID, periodWeeks)
	if err != nil {
		slog.Error("failed to load circles board", "community_id", communityID, "user_id", actor.ID, "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"data": board})
}

func (h *Handler) handleCommunityNeedsHelp(w http.ResponseWriter, r *http.Request) {
	actor, ok := currentUser(w, r)
	if !ok {
		return
	}
	communityID, ok := pathInt64(w, r, "communityID")
	if !ok {
		return
	}
	isLeader, err := h.az.IsCommunityLeader(r.Context(), actor.ID, communityID)
	if err != nil || !isLeader {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	report, err := getCommunityNeedsHelp(r.Context(), h.pool, communityID)
	if err != nil {
		slog.Error("failed to load community needs help", "community_id", communityID, "user_id", actor.ID, "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"data": report})
}

// ── Teamspace boards ─────────────────────────────────────────────────────────

func (h *Handler) handleTeamspaceNeedsHelp(w http.ResponseWriter, r *http.Request) {
	actor, ok := currentUser(w, r)
	if !ok {
		return
	}
	teamID, ok := pathInt64(w, r, "teamID")
	if !ok {
		return
	}
	isLead, err := h.az.IsTeamspaceLead(r.Context(), actor.ID, teamID)
	if err != nil || !isLead {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	report, err := getTeamspaceNeedsHelp(r.Context(), h.pool, teamID)
	if err != nil {
		slog.Error("failed to load teamspace needs help", "team_id", teamID, "user_id", actor.ID, "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"data": report})
}

// ── Like toggle ──────────────────────────────────────────────────────────────

func (h *Handler) handleLike(w http.ResponseWriter, r *http.Request) {
	actor, ok := currentUser(w, r)
	if !ok {
		return
	}
	checkInID, ok := pathInt64(w, r, "checkInID")
	if !ok {
		return
	}
	result, err := toggleLike(r.Context(), h.pool, checkInID, actor.ID)
	if err != nil {
		if errors.Is(err, errSelfLike) {
			http.Error(w, "cannot like own proof", http.StatusUnprocessableEntity)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"data": result})
}

// ── helpers ──────────────────────────────────────────────────────────────────

func (h *Handler) requireCircleMember(w http.ResponseWriter, r *http.Request, circleID, userID int64) bool {
	var ok bool
	err := h.pool.QueryRow(r.Context(), `
		SELECT EXISTS(SELECT 1 FROM circle_memberships WHERE circle_id = $1 AND user_id = $2 AND status = 'active')
	`, circleID, userID).Scan(&ok)
	if err != nil || !ok {
		http.Error(w, "forbidden", http.StatusForbidden)
		return false
	}
	return true
}

func (h *Handler) requireCommunityMember(w http.ResponseWriter, r *http.Request, communityID, userID int64) bool {
	var ok bool
	err := h.pool.QueryRow(r.Context(), `
		SELECT EXISTS(SELECT 1 FROM community_memberships WHERE community_space_id = $1 AND user_id = $2 AND status = 'active')
	`, communityID, userID).Scan(&ok)
	if err != nil || !ok {
		http.Error(w, "forbidden", http.StatusForbidden)
		return false
	}
	return true
}

func currentUser(w http.ResponseWriter, r *http.Request) (users.User, bool) {
	actor, ok := users.CurrentUser(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}
	return actor, ok
}

func pathInt64(w http.ResponseWriter, r *http.Request, key string) (int64, bool) {
	v, err := strconv.ParseInt(chi.URLParam(r, key), 10, 64)
	if err != nil {
		http.Error(w, "invalid "+key, http.StatusBadRequest)
		return 0, false
	}
	return v, true
}

func queryIntDefault(r *http.Request, key string, def, min, max int) int {
	v, err := strconv.Atoi(r.URL.Query().Get(key))
	if err != nil || v < min || v > max {
		return def
	}
	return v
}

func periodToWeeks(period string) int {
	switch period {
	case "last_week":
		return 1
	case "last_12_weeks":
		return 12
	default:
		return 4
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
