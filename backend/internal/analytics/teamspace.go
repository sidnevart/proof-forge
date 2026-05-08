package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sidnevart/proof-forge/backend/internal/authz"
	"github.com/sidnevart/proof-forge/backend/internal/users"
)

// TeamspaceHandler serves teamspace-level analytics.
type TeamspaceHandler struct {
	pool *pgxpool.Pool
	az   *authz.Authorizer
}

// NewTeamspaceHandler constructs a TeamspaceHandler.
func NewTeamspaceHandler(pool *pgxpool.Pool, az *authz.Authorizer) *TeamspaceHandler {
	return &TeamspaceHandler{pool: pool, az: az}
}

// RegisterTeamspaceRoutes installs teamspace analytics endpoints.
func (h *TeamspaceHandler) RegisterTeamspaceRoutes(r chi.Router) {
	r.Get("/teamspaces/{teamID}/analytics", h.handleTeamspaceAnalytics)
	r.Get("/teamspaces/{teamID}/analytics/members", h.handleMemberActivity)
}

// ── domain types ──────────────────────────────────────────────────────

type WeekActivity struct {
	Week          string `json:"week"`
	ProofsCount   int    `json:"proofs_count"`
	ActiveMembers int    `json:"active_members"`
}

type PopularTopic struct {
	Tag         string `json:"tag"`
	ProofCount  int    `json:"proof_count"`
	MemberCount int    `json:"member_count"`
}

type BestProofPreview struct {
	CheckInID       int64     `json:"check_in_id"`
	GoalTitle       string    `json:"goal_title"`
	ProofPreview    string    `json:"proof_preview"`
	SubmittedAt     time.Time `json:"submitted_at"`
	UserDisplayName string    `json:"user_display_name"`
}

type TeamspaceSummary struct {
	ActiveMembers        int     `json:"active_members"`
	TotalMembers         int     `json:"total_members"`
	TotalProofsSubmitted int     `json:"total_proofs_submitted"`
	TotalProofsApproved  int     `json:"total_proofs_approved"`
	AvgApprovalTimeHours float64 `json:"avg_approval_time_hours"`
}

type AttentionNeeded struct {
	Count       int    `json:"count"`
	Description string `json:"description"`
}

type TeamspaceAnalytics struct {
	Period           string             `json:"period"`
	TeamspaceID      int64              `json:"teamspace_id"`
	TeamspaceName    string             `json:"teamspace_name"`
	Summary          TeamspaceSummary   `json:"summary"`
	ActivityTrend    []WeekActivity     `json:"activity_trend"`
	PopularTopics    []PopularTopic     `json:"popular_topics"`
	IPRProofsCount   int                `json:"ipr_eligible_proofs"`
	AttentionNeeded  AttentionNeeded    `json:"attention_needed"`
	BestProofPreview []BestProofPreview `json:"best_proofs_preview"`
}

type MemberActivity struct {
	UserID        int64  `json:"user_id"`
	DisplayName   string `json:"display_name"`
	ProofsCount   int    `json:"proofs_count,omitempty"`
	ActiveWeeks   int    `json:"active_weeks,omitempty"`
	GoalsCount    int    `json:"goals_count,omitempty"`
	HasIPRGoal    bool   `json:"has_ipr_goal,omitempty"`
	ConsentShared bool   `json:"consent_shared"`
}

// ── handlers ──────────────────────────────────────────────────────────

func (h *TeamspaceHandler) handleTeamspaceAnalytics(w http.ResponseWriter, r *http.Request) {
	actor, ok := users.CurrentUser(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	teamID, err := strconv.ParseInt(chi.URLParam(r, "teamID"), 10, 64)
	if err != nil {
		http.Error(w, "invalid team id", http.StatusBadRequest)
		return
	}

	canView, err := h.az.CanViewTeamAnalytics(r.Context(), actor.ID, teamID)
	if err != nil || !canView {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	period := r.URL.Query().Get("period")
	if period == "" {
		period = "last_4_weeks"
	}
	weeks := periodToWeeks(period)

	analytics, err := h.getTeamspaceAnalytics(r.Context(), teamID, period, weeks)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"data": analytics})
}

func (h *TeamspaceHandler) handleMemberActivity(w http.ResponseWriter, r *http.Request) {
	actor, ok := users.CurrentUser(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	teamID, err := strconv.ParseInt(chi.URLParam(r, "teamID"), 10, 64)
	if err != nil {
		http.Error(w, "invalid team id", http.StatusBadRequest)
		return
	}

	// Only teamspace_lead can see member activity.
	isLead, err := h.az.IsTeamspaceLead(r.Context(), actor.ID, teamID)
	if err != nil || !isLead {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	members, err := h.getMemberActivity(r.Context(), teamID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"data": members})
}

// ── queries ───────────────────────────────────────────────────────────

func (h *TeamspaceHandler) getTeamspaceAnalytics(ctx context.Context, teamID int64, period string, weeks int) (*TeamspaceAnalytics, error) {
	analytics := &TeamspaceAnalytics{
		Period:      period,
		TeamspaceID: teamID,
	}

	// Team name
	if err := h.pool.QueryRow(ctx, `SELECT name FROM teams WHERE id = $1`, teamID).
		Scan(&analytics.TeamspaceName); err != nil {
		return nil, fmt.Errorf("team name: %w", err)
	}

	// Summary
	if err := h.pool.QueryRow(ctx, `
		SELECT
			COUNT(DISTINCT CASE WHEN ci.approved_at > NOW() - ($1 * INTERVAL '1 day') THEN tm.user_id END),
			COUNT(DISTINCT tm.user_id),
			COUNT(DISTINCT ci.id) FILTER (WHERE ci.status IN ('submitted', 'approved')),
			COUNT(DISTINCT ci.id) FILTER (WHERE ci.status = 'approved'),
			COALESCE(AVG(EXTRACT(EPOCH FROM (cr.created_at - ci.created_at))/3600) FILTER (WHERE cr.id IS NOT NULL), 0)
		FROM team_memberships tm
		LEFT JOIN goals g ON g.owner_user_id = tm.user_id
		LEFT JOIN check_ins ci ON ci.goal_id = g.id AND ci.created_at > NOW() - ($1 * INTERVAL '1 day')
		LEFT JOIN check_in_reviews cr ON cr.check_in_id = ci.id
		WHERE tm.team_id = $2
	`, weeks*7, teamID).Scan(
		&analytics.Summary.ActiveMembers,
		&analytics.Summary.TotalMembers,
		&analytics.Summary.TotalProofsSubmitted,
		&analytics.Summary.TotalProofsApproved,
		&analytics.Summary.AvgApprovalTimeHours,
	); err != nil {
		return nil, fmt.Errorf("summary: %w", err)
	}

	// Activity trend (per week)
	rows, err := h.pool.Query(ctx, `
		SELECT
			TO_CHAR(date_trunc('week', ci.created_at), 'IYYY-"W"IW') as week,
			COUNT(DISTINCT ci.id),
			COUNT(DISTINCT ci.owner_user_id)
		FROM check_ins ci
		JOIN goals g ON g.id = ci.goal_id
		JOIN team_memberships tm ON tm.user_id = g.owner_user_id AND tm.team_id = $1
		WHERE ci.created_at > NOW() - ($2 * INTERVAL '1 day')
		GROUP BY 1
		ORDER BY 1
	`, teamID, weeks*7)
	if err != nil {
		return nil, fmt.Errorf("activity trend: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var wa WeekActivity
		if err := rows.Scan(&wa.Week, &wa.ProofsCount, &wa.ActiveMembers); err != nil {
			return nil, err
		}
		analytics.ActivityTrend = append(analytics.ActivityTrend, wa)
	}
	if analytics.ActivityTrend == nil {
		analytics.ActivityTrend = []WeekActivity{}
	}

	// Popular topics — using goal category as proxy (no skill_tags yet)
	topicRows, err := h.pool.Query(ctx, `
		SELECT
			COALESCE(NULLIF(g.category, ''), 'Другое') as tag,
			COUNT(DISTINCT ci.id) as proof_count,
			COUNT(DISTINCT g.owner_user_id) as member_count
		FROM check_ins ci
		JOIN goals g ON g.id = ci.goal_id
		JOIN team_memberships tm ON tm.user_id = g.owner_user_id AND tm.team_id = $1
		WHERE ci.created_at > NOW() - ($2 * INTERVAL '1 day')
		  AND ci.status = 'approved'
		GROUP BY 1
		ORDER BY proof_count DESC
		LIMIT 10
	`, teamID, weeks*7)
	if err != nil {
		return nil, fmt.Errorf("popular topics: %w", err)
	}
	defer topicRows.Close()
	for topicRows.Next() {
		var pt PopularTopic
		if err := topicRows.Scan(&pt.Tag, &pt.ProofCount, &pt.MemberCount); err != nil {
			return nil, err
		}
		analytics.PopularTopics = append(analytics.PopularTopics, pt)
	}
	if analytics.PopularTopics == nil {
		analytics.PopularTopics = []PopularTopic{}
	}

	// IPR-eligible proofs (approved, with evidence)
	if err := h.pool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT ci.id)
		FROM check_ins ci
		JOIN evidence_items ei ON ei.check_in_id = ci.id
		JOIN goals g ON g.id = ci.goal_id
		JOIN team_memberships tm ON tm.user_id = g.owner_user_id AND tm.team_id = $1
		WHERE ci.status = 'approved'
		  AND ci.created_at > NOW() - ($2 * INTERVAL '1 day')
	`, teamID, weeks*7).Scan(&analytics.IPRProofsCount); err != nil {
		return nil, fmt.Errorf("ipr proofs: %w", err)
	}

	// Attention needed (no proof for 14+ days)
	var attentionCount int
	if err := h.pool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT g.owner_user_id)
		FROM team_memberships tm
		JOIN goals g ON g.owner_user_id = tm.user_id AND g.status = 'active'
		LEFT JOIN check_ins ci ON ci.goal_id = g.id AND ci.status = 'approved'
		WHERE tm.team_id = $1
		GROUP BY g.owner_user_id
		HAVING COALESCE(MAX(ci.approved_at), g.created_at) < NOW() - INTERVAL '14 days'
	`, teamID).Scan(&attentionCount); err != nil && err.Error() != "no rows in result set" {
		attentionCount = 0
	}
	analytics.AttentionNeeded = AttentionNeeded{
		Count:       attentionCount,
		Description: fmt.Sprintf("%d участников не сдавали пруф более 2 недель", attentionCount),
	}

	// Best proofs preview (top 3 recent)
	bpRows, err := h.pool.Query(ctx, `
		SELECT ci.id, g.title, LEFT(COALESCE(ev.preview, ''), 120), ci.created_at, u.display_name
		FROM check_ins ci
		JOIN goals g ON g.id = ci.goal_id
		JOIN team_memberships tm ON tm.user_id = g.owner_user_id AND tm.team_id = $1
		JOIN users u ON u.id = g.owner_user_id
		LEFT JOIN LATERAL (
			SELECT COALESCE(NULLIF(ei.text_content, ''), NULLIF(ei.external_url, ''), ei.kind) AS preview
			FROM evidence_items ei
			WHERE ei.check_in_id = ci.id
			ORDER BY ei.id
			LIMIT 1
		) ev ON TRUE
		WHERE ci.status = 'approved'
		ORDER BY ci.created_at DESC
		LIMIT 3
	`, teamID)
	if err != nil {
		return nil, fmt.Errorf("best proofs: %w", err)
	}
	defer bpRows.Close()
	for bpRows.Next() {
		var bp BestProofPreview
		if err := bpRows.Scan(&bp.CheckInID, &bp.GoalTitle, &bp.ProofPreview, &bp.SubmittedAt, &bp.UserDisplayName); err != nil {
			return nil, err
		}
		analytics.BestProofPreview = append(analytics.BestProofPreview, bp)
	}
	if analytics.BestProofPreview == nil {
		analytics.BestProofPreview = []BestProofPreview{}
	}

	return analytics, nil
}

func (h *TeamspaceHandler) getMemberActivity(ctx context.Context, teamID int64) ([]MemberActivity, error) {
	rows, err := h.pool.Query(ctx, `
		SELECT
			u.id,
			u.display_name,
			COUNT(DISTINCT ci.id) FILTER (WHERE ci.status = 'approved'),
			COUNT(DISTINCT date_trunc('week', ci.approved_at)) FILTER (WHERE ci.status = 'approved' AND ci.approved_at > NOW() - INTERVAL '84 days'),
			COUNT(DISTINCT g.id) FILTER (WHERE g.status = 'active'),
			FALSE as has_ipr_goal,
			TRUE as consent_shared
		FROM team_memberships tm
		JOIN users u ON u.id = tm.user_id
		LEFT JOIN goals g ON g.owner_user_id = tm.user_id
		LEFT JOIN check_ins ci ON ci.goal_id = g.id
		WHERE tm.team_id = $1
		GROUP BY u.id, u.display_name
		ORDER BY u.display_name
	`, teamID)
	if err != nil {
		return nil, fmt.Errorf("member activity: %w", err)
	}
	defer rows.Close()

	var members []MemberActivity
	for rows.Next() {
		var m MemberActivity
		if err := rows.Scan(
			&m.UserID,
			&m.DisplayName,
			&m.ProofsCount,
			&m.ActiveWeeks,
			&m.GoalsCount,
			&m.HasIPRGoal,
			&m.ConsentShared,
		); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	if members == nil {
		members = []MemberActivity{}
	}
	return members, rows.Err()
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
