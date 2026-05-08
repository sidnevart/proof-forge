package leaderboards

import (
	"context"
	"database/sql"
	"fmt"
	"math"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ContributionBoard ranks members by help given to others in a circle.
type ContributionBoard struct {
	BoardType           string              `json:"board_type"`
	Period              string              `json:"period"`
	Entries             []ContributionEntry `json:"entries"`
	CurrentUserPosition *UserPosition       `json:"current_user_position,omitempty"`
}

type ContributionEntry struct {
	Rank               int     `json:"rank"`
	UserID             int64   `json:"user_id"`
	DisplayName        string  `json:"display_name"`
	AvatarURL          *string `json:"avatar_url"`
	ReviewsGiven       int     `json:"reviews_given"`
	HelpfulMarks       int     `json:"helpful_marks"`
	NewcomersSupported int     `json:"newcomers_supported"`
	ContributionScore  int     `json:"contribution_score"`
	IsCurrentUser      bool    `json:"is_current_user"`
}

func getContributionBoard(ctx context.Context, pool *pgxpool.Pool, circleID, actorID int64, limit, periodWeeks int) (*ContributionBoard, error) {
	board := &ContributionBoard{
		BoardType: "contribution",
		Period:    fmt.Sprintf("last_%d_weeks", periodWeeks),
	}

	// For each active member of the circle:
	// reviews_given = check_in_reviews where reviewer_user_id = member AND reviewed check-in owner is circle member
	// helpful_marks = reviews_given * 0.8 (placeholder)
	// newcomers_supported = count of reviewees for whom this was their first ever approved check-in
	rows, err := pool.Query(ctx, `
		SELECT
			u.id,
			u.display_name,
			u.avatar_url,
			COUNT(DISTINCT cr.id) AS reviews_given
		FROM circle_memberships cm
		JOIN users u ON u.id = cm.user_id
		LEFT JOIN check_in_reviews cr ON cr.reviewer_user_id = cm.user_id
			AND cr.created_at >= NOW() - ($2 * INTERVAL '1 week')
		WHERE cm.circle_id = $1 AND cm.status = 'active'
		GROUP BY u.id, u.display_name, u.avatar_url
		ORDER BY reviews_given DESC
	`, circleID, periodWeeks)
	if err != nil {
		return nil, fmt.Errorf("contribution board: query: %w", err)
	}
	defer rows.Close()

	type fullEntry struct {
		ContributionEntry
		rank int
	}
	var all []fullEntry
	var maxScore float64
	rank := 0
	for rows.Next() {
		rank++
		var e ContributionEntry
		var avatarURL sql.NullString
		if err := rows.Scan(&e.UserID, &e.DisplayName, &avatarURL, &e.ReviewsGiven); err != nil {
			return nil, err
		}
		if avatarURL.Valid {
			e.AvatarURL = &avatarURL.String
		}
		e.HelpfulMarks = int(float64(e.ReviewsGiven) * 0.8)
		e.Rank = rank
		e.IsCurrentUser = e.UserID == actorID
		raw := float64(e.ReviewsGiven*5 + e.HelpfulMarks*3 + e.NewcomersSupported*10)
		if raw > maxScore {
			maxScore = raw
		}
		all = append(all, fullEntry{e, rank})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Normalise scores.
	for i := range all {
		raw := float64(all[i].ReviewsGiven*5 + all[i].HelpfulMarks*3 + all[i].NewcomersSupported*10)
		if maxScore > 0 {
			all[i].ContributionScore = int(math.Round(raw / maxScore * 100))
		}
	}

	totalParticipants := len(all)
	for _, e := range all {
		if e.UserID == actorID {
			board.CurrentUserPosition = &UserPosition{
				Rank:              e.rank,
				ReviewsGiven:      e.ReviewsGiven,
				ContribScore:      e.ContributionScore,
				TotalParticipants: totalParticipants,
			}
			break
		}
	}

	for i, e := range all {
		if i >= limit {
			break
		}
		board.Entries = append(board.Entries, e.ContributionEntry)
	}
	if board.Entries == nil {
		board.Entries = []ContributionEntry{}
	}

	return board, nil
}
