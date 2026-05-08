package leaderboards

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// GrowthBoard ranks members by improvement vs the previous week.
// Only positive growth_delta entries appear.
type GrowthBoard struct {
	BoardType           string        `json:"board_type"`
	Entries             []GrowthEntry `json:"entries"`
	CurrentUserPosition *UserPosition `json:"current_user_position,omitempty"`
}

type GrowthEntry struct {
	Rank           int     `json:"rank"`
	UserID         int64   `json:"user_id"`
	DisplayName    string  `json:"display_name"`
	AvatarURL      *string `json:"avatar_url"`
	ProofsThisWeek int     `json:"proofs_this_week"`
	ProofsLastWeek int     `json:"proofs_last_week"`
	GrowthDelta    int     `json:"growth_delta"`
	GrowthLabel    string  `json:"growth_label"`
	IsCurrentUser  bool    `json:"is_current_user"`
}

func getGrowthBoard(ctx context.Context, pool *pgxpool.Pool, circleID, actorID int64, limit int) (*GrowthBoard, error) {
	board := &GrowthBoard{BoardType: "growth"}

	rows, err := pool.Query(ctx, `
		WITH this_week AS (
			SELECT ci.owner_user_id, COUNT(*) AS cnt
			FROM check_ins ci
			WHERE ci.approved_at >= DATE_TRUNC('week', NOW())
			  AND ci.status = 'approved'
			GROUP BY ci.owner_user_id
		),
		last_week AS (
			SELECT ci.owner_user_id, COUNT(*) AS cnt
			FROM check_ins ci
			WHERE ci.approved_at >= DATE_TRUNC('week', NOW()) - INTERVAL '1 week'
			  AND ci.approved_at < DATE_TRUNC('week', NOW())
			  AND ci.status = 'approved'
			GROUP BY ci.owner_user_id
		)
		SELECT
			u.id,
			u.display_name,
			u.avatar_url,
			COALESCE(tw.cnt, 0) AS proofs_this_week,
			COALESCE(lw.cnt, 0) AS proofs_last_week,
			COALESCE(tw.cnt, 0) - COALESCE(lw.cnt, 0) AS growth_delta
		FROM circle_memberships cm
		JOIN users u ON u.id = cm.user_id
		LEFT JOIN this_week tw ON tw.owner_user_id = cm.user_id
		LEFT JOIN last_week lw ON lw.owner_user_id = cm.user_id
		WHERE cm.circle_id = $1 AND cm.status = 'active'
		  AND COALESCE(tw.cnt, 0) > COALESCE(lw.cnt, 0)
		ORDER BY growth_delta DESC, proofs_this_week DESC
	`, circleID)
	if err != nil {
		return nil, fmt.Errorf("growth board: query: %w", err)
	}
	defer rows.Close()

	type fullEntry struct {
		GrowthEntry
	}
	var all []fullEntry
	for rows.Next() {
		var e GrowthEntry
		var avatarURL sql.NullString
		if err := rows.Scan(
			&e.UserID, &e.DisplayName, &avatarURL,
			&e.ProofsThisWeek, &e.ProofsLastWeek, &e.GrowthDelta,
		); err != nil {
			return nil, err
		}
		if avatarURL.Valid {
			e.AvatarURL = &avatarURL.String
		}
		e.GrowthLabel = growthLabel(e.GrowthDelta, e.ProofsLastWeek)
		e.IsCurrentUser = e.UserID == actorID
		all = append(all, fullEntry{e})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	totalParticipants := len(all)
	for rank, e := range all {
		if e.UserID == actorID {
			board.CurrentUserPosition = &UserPosition{
				Rank:              rank + 1,
				GrowthDelta:       e.GrowthDelta,
				TotalParticipants: totalParticipants,
			}
			break
		}
	}

	for i, e := range all {
		if i >= limit {
			break
		}
		e.Rank = i + 1
		board.Entries = append(board.Entries, e.GrowthEntry)
	}
	if board.Entries == nil {
		board.Entries = []GrowthEntry{}
	}

	return board, nil
}

func growthLabel(delta, lastWeek int) string {
	if lastWeek == 0 && delta > 0 {
		return fmt.Sprintf("С нуля до %d пруфов", delta)
	}
	if delta >= lastWeek && lastWeek > 0 {
		return "Удвоил активность"
	}
	return fmt.Sprintf("+%d к прошлой неделе", delta)
}
