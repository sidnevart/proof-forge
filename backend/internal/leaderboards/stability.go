package leaderboards

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// StabilityBoard shows top consistent members of a circle.
type StabilityBoard struct {
	CircleID            int64            `json:"circle_id"`
	CircleName          string           `json:"circle_name"`
	BoardType           string           `json:"board_type"`
	Period              string           `json:"period"`
	Entries             []StabilityEntry `json:"entries"`
	CurrentUserPosition *UserPosition    `json:"current_user_position,omitempty"`
}

type StabilityEntry struct {
	Rank             int     `json:"rank"`
	UserID           int64   `json:"user_id"`
	DisplayName      string  `json:"display_name"`
	AvatarURL        *string `json:"avatar_url"`
	ProofStreakWeeks int     `json:"proof_streak_weeks"`
	ActiveWeeks      int     `json:"active_weeks"`
	ConsistencyScore int     `json:"consistency_score"`
	IsCurrentUser    bool    `json:"is_current_user"`
}

type UserPosition struct {
	Rank              int `json:"rank"`
	ConsistencyScore  int `json:"consistency_score,omitempty"`
	GrowthDelta       int `json:"growth_delta,omitempty"`
	ReviewsGiven      int `json:"reviews_given,omitempty"`
	ContribScore      int `json:"contribution_score,omitempty"`
	TotalParticipants int `json:"total_participants"`
}

func getStabilityBoard(ctx context.Context, pool *pgxpool.Pool, circleID, actorID int64, limit, periodWeeks int) (*StabilityBoard, error) {
	board := &StabilityBoard{
		CircleID:  circleID,
		BoardType: "stability",
		Period:    fmt.Sprintf("last_%d_weeks", periodWeeks),
	}

	if err := pool.QueryRow(ctx, `SELECT name FROM circles WHERE id = $1`, circleID).
		Scan(&board.CircleName); err != nil {
		return nil, fmt.Errorf("stability board: circle name: %w", err)
	}

	// All members with consistency_score, ordered desc.
	rows, err := pool.Query(ctx, `
		SELECT
			u.id,
			u.display_name,
			u.avatar_url,
			COUNT(DISTINCT DATE_TRUNC('week', ci.approved_at)) AS active_weeks,
				ROUND(
				COUNT(DISTINCT DATE_TRUNC('week', ci.approved_at))::numeric /
				NULLIF($2, 0) * 100
			) AS consistency_score
		FROM circle_memberships cm
		JOIN users u ON u.id = cm.user_id
		LEFT JOIN check_ins ci ON ci.owner_user_id = cm.user_id
			AND ci.approved_at >= NOW() - ($2 * INTERVAL '1 week')
			AND ci.status = 'approved'
		WHERE cm.circle_id = $1 AND cm.status = 'active'
		GROUP BY u.id, u.display_name, u.avatar_url
		ORDER BY consistency_score DESC, active_weeks DESC
	`, circleID, periodWeeks)
	if err != nil {
		return nil, fmt.Errorf("stability board: query: %w", err)
	}
	defer rows.Close()

	type fullEntry struct {
		StabilityEntry
		rank int
	}
	var all []fullEntry
	rank := 0
	for rows.Next() {
		rank++
		var e StabilityEntry
		var avatarURL sql.NullString
		var score sql.NullFloat64
		if err := rows.Scan(&e.UserID, &e.DisplayName, &avatarURL, &e.ActiveWeeks, &score); err != nil {
			return nil, err
		}
		if avatarURL.Valid {
			e.AvatarURL = &avatarURL.String
		}
		if score.Valid {
			e.ConsistencyScore = int(score.Float64)
		}
		e.Rank = rank
		e.IsCurrentUser = e.UserID == actorID
		all = append(all, fullEntry{e, rank})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	totalParticipants := len(all)
	var currentUserPos *UserPosition

	for _, e := range all {
		if e.UserID == actorID {
			currentUserPos = &UserPosition{
				Rank:              e.rank,
				ConsistencyScore:  e.ConsistencyScore,
				TotalParticipants: totalParticipants,
			}
			break
		}
	}

	// Only return top limit entries.
	for i, e := range all {
		if i >= limit {
			break
		}
		// Compute streak for top entries.
		streak, _, _ := computeStreakAndRecord(ctx, pool, e.UserID)
		e.ProofStreakWeeks = streak
		board.Entries = append(board.Entries, e.StabilityEntry)
	}
	if board.Entries == nil {
		board.Entries = []StabilityEntry{}
	}

	board.CurrentUserPosition = currentUserPos
	return board, nil
}
