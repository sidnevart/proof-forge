// Package admin provides platform admin endpoints.
package admin

import (
	"database/sql"
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Handler exposes platform admin management endpoints.
type Handler struct {
	pool *pgxpool.Pool
}

// NewHandler constructs an admin Handler.
func NewHandler(pool *pgxpool.Pool) *Handler {
	return &Handler{pool: pool}
}

// RegisterRoutes installs admin endpoints.
// The router MUST already be wrapped with RequirePlatformAdmin middleware.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/stats", h.handleStats)
	r.Get("/workspaces", h.handleListWorkspaces)
	r.Post("/workspaces/{workspaceID}/freeze", h.handleFreezeWorkspace)
	r.Post("/workspaces/{workspaceID}/unfreeze", h.handleUnfreezeWorkspace)
	r.Get("/users", h.handleListUsers)
	r.Get("/pilot-report", h.handlePilotReport)
}

// ── Stats ─────────────────────────────────────────────────────────────

func (h *Handler) handleStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var totalUsers, activeUsers30d, totalWorkspaces, activeWorkspaces,
		totalTeams, totalCommunities, totalProofs30d int
	_ = h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&totalUsers)
	_ = h.pool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT owner_user_id) FROM goals
		WHERE updated_at > NOW() - INTERVAL '30 days'`).Scan(&activeUsers30d)
	_ = h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM workspaces`).Scan(&totalWorkspaces)
	_ = h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM workspaces WHERE is_active = TRUE`).Scan(&activeWorkspaces)
	_ = h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM teams`).Scan(&totalTeams)
	_ = h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM community_spaces`).Scan(&totalCommunities)
	_ = h.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM check_ins
		WHERE status = 'approved' AND approved_at > NOW() - INTERVAL '30 days'`).Scan(&totalProofs30d)

	avgProofs := 0.0
	if activeUsers30d > 0 {
		avgProofs = float64(totalProofs30d) / float64(activeUsers30d)
	}

	// Health per workspace
	wsRows, err := h.pool.Query(ctx, `
		SELECT
			w.id, w.name,
			COUNT(DISTINCT tm.user_id) as total_members,
			COUNT(DISTINCT CASE WHEN ci.approved_at > NOW() - INTERVAL '30 days' THEN g.owner_user_id END) as active_members,
			COUNT(DISTINCT ci.id) FILTER (WHERE ci.approved_at > NOW() - INTERVAL '30 days') as proofs_30d
		FROM workspaces w
		LEFT JOIN teams t ON t.workspace_id = w.id
		LEFT JOIN team_memberships tm ON tm.team_id = t.id
		LEFT JOIN goals g ON g.owner_user_id = tm.user_id
		LEFT JOIN check_ins ci ON ci.goal_id = g.id AND ci.status = 'approved'
		GROUP BY w.id, w.name
		ORDER BY w.created_at DESC
	`)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer wsRows.Close()

	type wsHealth struct {
		WorkspaceID   int64  `json:"workspace_id"`
		WorkspaceName string `json:"workspace_name"`
		HealthScore   int    `json:"health_score"`
		ActiveMembers int    `json:"active_members"`
		TotalMembers  int    `json:"total_members"`
		Proofs30d     int    `json:"proofs_last_30d"`
		ChurnRisk     string `json:"churn_risk"`
	}
	var healthList []wsHealth
	for wsRows.Next() {
		var ws wsHealth
		if err := wsRows.Scan(&ws.WorkspaceID, &ws.WorkspaceName, &ws.TotalMembers, &ws.ActiveMembers, &ws.Proofs30d); err != nil {
			continue
		}
		ws.HealthScore = computeHealthScore(ws.ActiveMembers, ws.TotalMembers, ws.Proofs30d)
		ws.ChurnRisk = churnRisk(ws.HealthScore)
		healthList = append(healthList, ws)
	}
	if healthList == nil {
		healthList = []wsHealth{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": map[string]any{
			"platform": map[string]any{
				"total_users":                totalUsers,
				"active_users_last_30d":      activeUsers30d,
				"total_workspaces":           totalWorkspaces,
				"active_workspaces":          activeWorkspaces,
				"total_teamspaces":           totalTeams,
				"total_community_spaces":     totalCommunities,
				"total_proofs_last_30d":      totalProofs30d,
				"avg_proofs_per_active_user": math.Round(avgProofs*10) / 10,
			},
			"health": healthList,
			"revenue": map[string]int{
				"paid_workspaces_count":  0,
				"trial_workspaces_count": 0,
				"free_workspaces_count":  totalWorkspaces,
			},
		},
	})
}

// ── Workspaces ────────────────────────────────────────────────────────

func (h *Handler) handleListWorkspaces(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	limit, offset := parsePagination(r)

	statusFilter := r.URL.Query().Get("status")
	query := `
		SELECT w.id, w.name, w.slug, w.type, w.is_active, w.created_at,
		       u.id, u.display_name, u.email,
		       COUNT(DISTINCT t.id), COUNT(DISTINCT tm.user_id)
		FROM workspaces w
		JOIN users u ON u.id = w.owner_user_id
		LEFT JOIN teams t ON t.workspace_id = w.id
		LEFT JOIN team_memberships tm ON tm.team_id = t.id
	`
	var args []any
	if statusFilter == "active" {
		query += ` WHERE w.is_active = TRUE`
	} else if statusFilter == "frozen" {
		query += ` WHERE w.is_active = FALSE`
	}
	query += ` GROUP BY w.id, w.name, w.slug, w.type, w.is_active, w.created_at, u.id, u.display_name, u.email`
	query += ` ORDER BY w.created_at DESC LIMIT $1 OFFSET $2`
	args = append(args, limit, offset)

	rows, err := h.pool.Query(ctx, query, args...)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type wsItem struct {
		ID              int64     `json:"id"`
		Name            string    `json:"name"`
		Slug            string    `json:"slug"`
		Type            string    `json:"type"`
		IsActive        bool      `json:"is_active"`
		CreatedAt       time.Time `json:"created_at"`
		Owner           any       `json:"owner"`
		TeamspacesCount int       `json:"teamspaces_count"`
		MembersCount    int       `json:"members_count"`
	}

	var items []wsItem
	for rows.Next() {
		var item wsItem
		var ownerID int64
		var ownerName, ownerEmail string
		if err := rows.Scan(
			&item.ID, &item.Name, &item.Slug, &item.Type, &item.IsActive, &item.CreatedAt,
			&ownerID, &ownerName, &ownerEmail,
			&item.TeamspacesCount, &item.MembersCount,
		); err != nil {
			continue
		}
		item.Owner = map[string]any{"user_id": ownerID, "display_name": ownerName, "email": ownerEmail}
		items = append(items, item)
	}
	if items == nil {
		items = []wsItem{}
	}

	var total int
	_ = h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM workspaces`).Scan(&total)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": items,
		"meta": map[string]int{"total": total},
	})
}

func (h *Handler) handleFreezeWorkspace(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	wsID, err := strconv.ParseInt(chi.URLParam(r, "workspaceID"), 10, 64)
	if err != nil {
		http.Error(w, "invalid workspace id", http.StatusBadRequest)
		return
	}

	var body struct {
		Reason string `json:"reason"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	if _, err := h.pool.Exec(ctx, `
		UPDATE workspaces
		SET is_active = FALSE, frozen_at = NOW(), frozen_reason = $2, updated_at = NOW()
		WHERE id = $1
	`, wsID, body.Reason); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": map[string]any{"id": wsID, "is_active": false, "frozen_reason": body.Reason},
	})
}

func (h *Handler) handleUnfreezeWorkspace(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	wsID, err := strconv.ParseInt(chi.URLParam(r, "workspaceID"), 10, 64)
	if err != nil {
		http.Error(w, "invalid workspace id", http.StatusBadRequest)
		return
	}

	if _, err := h.pool.Exec(ctx, `
		UPDATE workspaces
		SET is_active = TRUE, frozen_at = NULL, frozen_reason = NULL, updated_at = NOW()
		WHERE id = $1
	`, wsID); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": map[string]any{"id": wsID, "is_active": true},
	})
}

// ── Users ─────────────────────────────────────────────────────────────

func (h *Handler) handleListUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	limit, offset := parsePagination(r)
	q := strings.TrimSpace(r.URL.Query().Get("q"))

	var args []any
	query := `
		SELECT u.id, u.display_name, u.email, u.is_platform_admin, u.created_at,
		       COUNT(DISTINCT g.id) as goals_count,
		       COUNT(DISTINCT ci.id) as proofs_count
		FROM users u
		LEFT JOIN goals g ON g.owner_user_id = u.id
		LEFT JOIN check_ins ci ON ci.goal_id = g.id AND ci.status = 'approved'
	`
	if q != "" {
		query += ` WHERE u.display_name ILIKE $1 OR u.email ILIKE $1`
		args = append(args, "%"+q+"%")
	}
	query += ` GROUP BY u.id, u.display_name, u.email, u.is_platform_admin, u.created_at`
	query += ` ORDER BY u.created_at DESC`

	limitArg := len(args) + 1
	offsetArg := len(args) + 2
	query += ` LIMIT $` + strconv.Itoa(limitArg) + ` OFFSET $` + strconv.Itoa(offsetArg)
	args = append(args, limit, offset)

	rows, err := h.pool.Query(ctx, query, args...)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type userItem struct {
		ID              int64     `json:"id"`
		DisplayName     string    `json:"display_name"`
		Email           string    `json:"email"`
		IsPlatformAdmin bool      `json:"is_platform_admin"`
		CreatedAt       time.Time `json:"created_at"`
		GoalsCount      int       `json:"goals_count"`
		ProofsCount     int       `json:"proofs_count"`
	}

	var items []userItem
	for rows.Next() {
		var item userItem
		var nullCreatedAt sql.NullTime
		if err := rows.Scan(
			&item.ID, &item.DisplayName, &item.Email, &item.IsPlatformAdmin,
			&nullCreatedAt, &item.GoalsCount, &item.ProofsCount,
		); err != nil {
			continue
		}
		if nullCreatedAt.Valid {
			item.CreatedAt = nullCreatedAt.Time
		}
		items = append(items, item)
	}
	if items == nil {
		items = []userItem{}
	}

	var total int
	_ = h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&total)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": items,
		"meta": map[string]int{"total": total, "limit": limit, "offset": offset},
	})
}

// ── helpers ───────────────────────────────────────────────────────────

func computeHealthScore(activeMembers, totalMembers, proofs30d int) int {
	if totalMembers == 0 {
		return 0
	}
	actRatio := float64(activeMembers) / float64(totalMembers)
	proofRatio := math.Min(float64(proofs30d)/float64(totalMembers), 10) / 10
	score := actRatio*50 + proofRatio*50
	return int(math.Round(score))
}

func churnRisk(score int) string {
	switch {
	case score >= 70:
		return "low"
	case score >= 40:
		return "medium"
	default:
		return "high"
	}
}

func parsePagination(r *http.Request) (limit, offset int) {
	limit = 20
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 && l <= 100 {
		limit = l
	}
	if o, err := strconv.Atoi(r.URL.Query().Get("offset")); err == nil && o >= 0 {
		offset = o
	}
	return
}
