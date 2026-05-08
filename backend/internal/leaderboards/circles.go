package leaderboards

import (
	"context"
	"fmt"
	"math"

	"github.com/jackc/pgx/v5/pgxpool"
)

// CirclesBoard compares circles within a community space.
// Shows all circles — teams compete, individuals are not ranked.
type CirclesBoard struct {
	BoardType        string          `json:"board_type"`
	Period           string          `json:"period"`
	Entries          []CircleEntry   `json:"entries"`
	MyCirclePosition *CirclePosition `json:"my_circle_position,omitempty"`
}

type CircleEntry struct {
	Rank            int     `json:"rank"`
	CircleID        int64   `json:"circle_id"`
	CircleName      string  `json:"circle_name"`
	MembersCount    int     `json:"members_count"`
	ProofsSubmitted int     `json:"proofs_submitted"`
	ProofsApproved  int     `json:"proofs_approved"`
	CompletionPct   float64 `json:"completion_pct"`
	ActiveMembers   int     `json:"active_members"`
	CircleScore     int     `json:"circle_score"`
	IsMyCircle      bool    `json:"is_my_circle"`
}

type CirclePosition struct {
	Rank         int `json:"rank"`
	CircleScore  int `json:"circle_score"`
	TotalCircles int `json:"total_circles"`
}

func getCirclesBoard(ctx context.Context, pool *pgxpool.Pool, communityID, actorID int64, periodWeeks int) (*CirclesBoard, error) {
	board := &CirclesBoard{
		BoardType: "circles",
		Period:    fmt.Sprintf("last_%d_weeks", periodWeeks),
	}

	// Find which circle(s) the actor belongs to within this community.
	var actorCircleID int64
	_ = pool.QueryRow(ctx, `
		SELECT cm.circle_id FROM circle_memberships cm
		JOIN circles c ON c.id = cm.circle_id
		WHERE c.community_space_id = $1 AND cm.user_id = $2 AND cm.status = 'active'
		LIMIT 1
	`, communityID, actorID).Scan(&actorCircleID)

	rows, err := pool.Query(ctx, `
		SELECT
			c.id,
			c.name,
			COUNT(DISTINCT cm.user_id) AS members_count,
			COUNT(DISTINCT ci.id) FILTER (WHERE ci.status IN ('submitted','approved')) AS proofs_submitted,
			COUNT(DISTINCT ci.id) FILTER (WHERE ci.status = 'approved') AS proofs_approved,
			COUNT(DISTINCT ci.owner_user_id) FILTER (WHERE ci.status = 'approved' AND ci.approved_at >= NOW() - ($2 * INTERVAL '1 week')) AS active_members
		FROM circles c
		JOIN circle_memberships cm ON cm.circle_id = c.id AND cm.status = 'active'
		LEFT JOIN check_ins ci ON ci.owner_user_id = cm.user_id
			AND ci.created_at >= NOW() - ($2 * INTERVAL '1 week')
		WHERE c.community_space_id = $1
		GROUP BY c.id, c.name
		ORDER BY c.id
	`, communityID, periodWeeks)
	if err != nil {
		return nil, fmt.Errorf("circles board: query: %w", err)
	}
	defer rows.Close()

	type rawEntry struct {
		circleID        int64
		circleName      string
		membersCount    int
		proofsSubmitted int
		proofsApproved  int
		activeMembers   int
	}
	var raws []rawEntry
	for rows.Next() {
		var re rawEntry
		if err := rows.Scan(&re.circleID, &re.circleName, &re.membersCount,
			&re.proofsSubmitted, &re.proofsApproved, &re.activeMembers); err != nil {
			return nil, err
		}
		raws = append(raws, re)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Compute scores.
	type scored struct {
		rawEntry
		score float64
	}
	var scoreds []scored
	for _, re := range raws {
		completionPct := 0.0
		if re.proofsSubmitted > 0 {
			completionPct = float64(re.proofsApproved) / float64(re.proofsSubmitted) * 100
		}
		activePct := 0.0
		if re.membersCount > 0 {
			activePct = float64(re.activeMembers) / float64(re.membersCount) * 100
		}
		approvalPct := 0.0
		if re.proofsSubmitted > 0 {
			approvalPct = float64(re.proofsApproved) / float64(re.proofsSubmitted) * 100
		}
		score := completionPct*0.5 + activePct*0.3 + approvalPct*0.2
		scoreds = append(scoreds, scored{re, score})
	}

	// Sort by score descending.
	for i := 0; i < len(scoreds); i++ {
		for j := i + 1; j < len(scoreds); j++ {
			if scoreds[j].score > scoreds[i].score {
				scoreds[i], scoreds[j] = scoreds[j], scoreds[i]
			}
		}
	}

	totalCircles := len(scoreds)
	for rank, s := range scoreds {
		completionPct := 0.0
		if s.proofsSubmitted > 0 {
			completionPct = float64(s.proofsApproved) / float64(s.proofsSubmitted) * 100
		}
		e := CircleEntry{
			Rank:            rank + 1,
			CircleID:        s.circleID,
			CircleName:      s.circleName,
			MembersCount:    s.membersCount,
			ProofsSubmitted: s.proofsSubmitted,
			ProofsApproved:  s.proofsApproved,
			CompletionPct:   math.Round(completionPct*10) / 10,
			ActiveMembers:   s.activeMembers,
			CircleScore:     int(math.Round(s.score)),
			IsMyCircle:      s.circleID == actorCircleID,
		}
		board.Entries = append(board.Entries, e)

		if e.IsMyCircle {
			board.MyCirclePosition = &CirclePosition{
				Rank:         rank + 1,
				CircleScore:  e.CircleScore,
				TotalCircles: totalCircles,
			}
		}
	}
	if board.Entries == nil {
		board.Entries = []CircleEntry{}
	}

	return board, nil
}
