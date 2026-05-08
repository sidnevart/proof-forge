package leaderboards

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NeedsHelpReport is the private report for leads/leaders about members needing support.
type NeedsHelpReport struct {
	Summary      NeedsHelpSummary `json:"summary"`
	Members      []NeedsHelpMember `json:"members"`
	AtRiskCircles []AtRiskCircle  `json:"at_risk_circles"`
}

type NeedsHelpSummary struct {
	MembersNeedingAttention int `json:"members_needing_attention"`
	PendingReviewsTotal     int `json:"pending_reviews_total"`
	BrokenContractsTotal    int `json:"broken_contracts_total"`
}

type NeedsHelpMember struct {
	UserID               int64   `json:"user_id"`
	DisplayName          string  `json:"display_name"`
	AvatarURL            *string `json:"avatar_url"`
	DaysSinceLastProof   int     `json:"days_since_last_proof"`
	HasBrokenContract    bool    `json:"has_broken_contract"`
	PendingReviewsCount  int     `json:"pending_reviews_count"`
	AttentionReason      string  `json:"attention_reason"`
}

type AtRiskCircle struct {
	CircleID      int64      `json:"circle_id"`
	CircleName    string     `json:"circle_name"`
	ActiveMembers int        `json:"active_members"`
	TotalMembers  int        `json:"total_members"`
	LastProofAt   *time.Time `json:"last_proof_at"`
}

// getTeamspaceNeedsHelp returns the needs-help report for a teamspace lead.
func getTeamspaceNeedsHelp(ctx context.Context, pool *pgxpool.Pool, teamID int64) (*NeedsHelpReport, error) {
	report := &NeedsHelpReport{}

	// Members with attention needed: no proof for 7+ days, or broken contract, or pending review.
	rows, err := pool.Query(ctx, `
		SELECT
			u.id,
			u.display_name,
			u.avatar_url,
			EXTRACT(DAY FROM NOW() - MAX(COALESCE(ci.approved_at, g.created_at)))::int AS days_since,
			EXISTS(
				SELECT 1 FROM proof_contracts pc
				JOIN goals g2 ON g2.id = pc.goal_id
				WHERE g2.owner_user_id = u.id AND pc.status = 'broken'
			) AS has_broken,
			COUNT(DISTINCT ci2.id) FILTER (WHERE ci2.status = 'submitted') AS pending_reviews
		FROM team_memberships tm
		JOIN users u ON u.id = tm.user_id
		LEFT JOIN goals g ON g.owner_user_id = u.id AND g.status = 'active'
		LEFT JOIN check_ins ci ON ci.goal_id = g.id AND ci.status = 'approved'
		LEFT JOIN check_ins ci2 ON ci2.goal_id = g.id AND ci2.status = 'submitted'
		WHERE tm.team_id = $1
		GROUP BY u.id, u.display_name, u.avatar_url
		HAVING
			EXTRACT(DAY FROM NOW() - MAX(COALESCE(ci.approved_at, g.created_at))) >= 7
			OR EXISTS(
				SELECT 1 FROM proof_contracts pc
				JOIN goals g2 ON g2.id = pc.goal_id
				WHERE g2.owner_user_id = u.id AND pc.status = 'broken'
			)
			OR COUNT(DISTINCT ci2.id) FILTER (WHERE ci2.status = 'submitted') > 0
		ORDER BY days_since DESC NULLS LAST
	`, teamID)
	if err != nil {
		return nil, fmt.Errorf("needs help: members: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var m NeedsHelpMember
		var avatarURL sql.NullString
		var daysSince sql.NullInt32
		if err := rows.Scan(
			&m.UserID, &m.DisplayName, &avatarURL,
			&daysSince, &m.HasBrokenContract, &m.PendingReviewsCount,
		); err != nil {
			return nil, err
		}
		if avatarURL.Valid {
			m.AvatarURL = &avatarURL.String
		}
		if daysSince.Valid {
			m.DaysSinceLastProof = int(daysSince.Int32)
		}
		m.AttentionReason = attentionReason(m.DaysSinceLastProof, m.HasBrokenContract, m.PendingReviewsCount)
		report.Members = append(report.Members, m)
	}
	if report.Members == nil {
		report.Members = []NeedsHelpMember{}
	}

	// Summary counts.
	report.Summary.MembersNeedingAttention = len(report.Members)
	for _, m := range report.Members {
		report.Summary.PendingReviewsTotal += m.PendingReviewsCount
		if m.HasBrokenContract {
			report.Summary.BrokenContractsTotal++
		}
	}

	// At-risk circles: circles where active_members / total_members < 0.5.
	// For teamspaces, circles are circles linked to teams via goals.
	circleRows, err := pool.Query(ctx, `
		SELECT
			c.id,
			c.name,
			COUNT(DISTINCT cm.user_id) AS total_members,
			COUNT(DISTINCT ci.owner_user_id) FILTER (WHERE ci.approved_at >= NOW() - INTERVAL '14 days') AS active_members,
			MAX(ci.approved_at) AS last_proof_at
		FROM circles c
		JOIN circle_memberships cm ON cm.circle_id = c.id AND cm.status = 'active'
		JOIN team_memberships tm ON tm.user_id = cm.user_id AND tm.team_id = $1
		LEFT JOIN check_ins ci ON ci.owner_user_id = cm.user_id AND ci.status = 'approved'
		GROUP BY c.id, c.name
		HAVING COUNT(DISTINCT ci.owner_user_id) FILTER (WHERE ci.approved_at >= NOW() - INTERVAL '14 days')::float /
		       NULLIF(COUNT(DISTINCT cm.user_id), 0) < 0.5
		ORDER BY active_members ASC
	`, teamID)
	if err != nil {
		return nil, fmt.Errorf("needs help: at-risk circles: %w", err)
	}
	defer circleRows.Close()

	for circleRows.Next() {
		var arc AtRiskCircle
		var lastProofAt sql.NullTime
		if err := circleRows.Scan(&arc.CircleID, &arc.CircleName,
			&arc.TotalMembers, &arc.ActiveMembers, &lastProofAt); err != nil {
			return nil, err
		}
		if lastProofAt.Valid {
			arc.LastProofAt = &lastProofAt.Time
		}
		report.AtRiskCircles = append(report.AtRiskCircles, arc)
	}
	if report.AtRiskCircles == nil {
		report.AtRiskCircles = []AtRiskCircle{}
	}

	return report, nil
}

// getCommunityNeedsHelp returns the needs-help report for a community leader.
func getCommunityNeedsHelp(ctx context.Context, pool *pgxpool.Pool, communityID int64) (*NeedsHelpReport, error) {
	report := &NeedsHelpReport{}

	rows, err := pool.Query(ctx, `
		SELECT
			u.id,
			u.display_name,
			u.avatar_url,
			EXTRACT(DAY FROM NOW() - MAX(COALESCE(ci.approved_at, g.created_at)))::int AS days_since,
			FALSE AS has_broken,
			COUNT(DISTINCT ci2.id) FILTER (WHERE ci2.status = 'submitted') AS pending_reviews
		FROM community_memberships cm2
		JOIN users u ON u.id = cm2.user_id
		LEFT JOIN goals g ON g.owner_user_id = u.id AND g.status = 'active'
		LEFT JOIN check_ins ci ON ci.goal_id = g.id AND ci.status = 'approved'
		LEFT JOIN check_ins ci2 ON ci2.goal_id = g.id AND ci2.status = 'submitted'
		WHERE cm2.community_space_id = $1 AND cm2.status = 'active'
		GROUP BY u.id, u.display_name, u.avatar_url
		HAVING
			EXTRACT(DAY FROM NOW() - MAX(COALESCE(ci.approved_at, g.created_at))) >= 7
			OR COUNT(DISTINCT ci2.id) FILTER (WHERE ci2.status = 'submitted') > 0
		ORDER BY days_since DESC NULLS LAST
	`, communityID)
	if err != nil {
		return nil, fmt.Errorf("community needs help: members: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var m NeedsHelpMember
		var avatarURL sql.NullString
		var daysSince sql.NullInt32
		if err := rows.Scan(
			&m.UserID, &m.DisplayName, &avatarURL,
			&daysSince, &m.HasBrokenContract, &m.PendingReviewsCount,
		); err != nil {
			return nil, err
		}
		if avatarURL.Valid {
			m.AvatarURL = &avatarURL.String
		}
		if daysSince.Valid {
			m.DaysSinceLastProof = int(daysSince.Int32)
		}
		m.AttentionReason = attentionReason(m.DaysSinceLastProof, false, m.PendingReviewsCount)
		report.Members = append(report.Members, m)
	}
	if report.Members == nil {
		report.Members = []NeedsHelpMember{}
	}

	report.Summary.MembersNeedingAttention = len(report.Members)
	for _, m := range report.Members {
		report.Summary.PendingReviewsTotal += m.PendingReviewsCount
	}

	// At-risk circles within the community.
	circleRows, err := pool.Query(ctx, `
		SELECT
			c.id,
			c.name,
			COUNT(DISTINCT cm.user_id) AS total_members,
			COUNT(DISTINCT ci.owner_user_id) FILTER (WHERE ci.approved_at >= NOW() - INTERVAL '14 days') AS active_members,
			MAX(ci.approved_at) AS last_proof_at
		FROM circles c
		JOIN circle_memberships cm ON cm.circle_id = c.id AND cm.status = 'active'
		LEFT JOIN check_ins ci ON ci.owner_user_id = cm.user_id AND ci.status = 'approved'
		WHERE c.community_space_id = $1
		GROUP BY c.id, c.name
		HAVING COUNT(DISTINCT ci.owner_user_id) FILTER (WHERE ci.approved_at >= NOW() - INTERVAL '14 days')::float /
		       NULLIF(COUNT(DISTINCT cm.user_id), 0) < 0.5
		ORDER BY active_members ASC
	`, communityID)
	if err != nil {
		return nil, fmt.Errorf("community needs help: circles: %w", err)
	}
	defer circleRows.Close()

	for circleRows.Next() {
		var arc AtRiskCircle
		var lastProofAt sql.NullTime
		if err := circleRows.Scan(&arc.CircleID, &arc.CircleName,
			&arc.TotalMembers, &arc.ActiveMembers, &lastProofAt); err != nil {
			return nil, err
		}
		if lastProofAt.Valid {
			arc.LastProofAt = &lastProofAt.Time
		}
		report.AtRiskCircles = append(report.AtRiskCircles, arc)
	}
	if report.AtRiskCircles == nil {
		report.AtRiskCircles = []AtRiskCircle{}
	}

	return report, nil
}

func attentionReason(daysSince int, hasBroken bool, pendingReviews int) string {
	if hasBroken {
		return "broken_contract"
	}
	if pendingReviews > 0 {
		return "waiting_for_review"
	}
	if daysSince >= 14 {
		return "no_proof_14_days"
	}
	return "no_proof_7_days"
}
