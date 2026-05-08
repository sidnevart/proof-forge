package leaderboards

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var errSelfLike = errors.New("cannot like own proof")

// BestProofsBoard ranks proof artifacts (not people) in a circle.
type BestProofsBoard struct {
	BoardType string           `json:"board_type"`
	Period    string           `json:"period"`
	Entries   []BestProofEntry `json:"entries"`
}

type BestProofEntry struct {
	Rank          int         `json:"rank"`
	CheckInID     int64       `json:"check_in_id"`
	GoalTitle     string      `json:"goal_title"`
	ProofPreview  string      `json:"proof_preview"`
	ProofURL      *string     `json:"proof_url"`
	SubmittedAt   time.Time   `json:"submitted_at"`
	Author        ProofAuthor `json:"author"`
	LikesCount    int         `json:"likes_count"`
	BuddyApproved bool        `json:"buddy_approved"`
	ProofType     string      `json:"proof_type"`
}

type ProofAuthor struct {
	UserID      int64   `json:"user_id"`
	DisplayName string  `json:"display_name"`
	AvatarURL   *string `json:"avatar_url"`
}

// LikeResult is the response for toggling a like.
type LikeResult struct {
	CheckInID  int64 `json:"check_in_id"`
	LikesCount int   `json:"likes_count"`
	LikedByMe  bool  `json:"liked_by_me"`
}

func getBestProofsBoard(ctx context.Context, pool *pgxpool.Pool, circleID, actorID int64, limit int, period string) (*BestProofsBoard, error) {
	board := &BestProofsBoard{
		BoardType: "best_proofs",
		Period:    period,
	}

	var since time.Time
	switch period {
	case "this_week":
		since = isoWeekStart(time.Now())
	case "last_4_weeks":
		since = time.Now().AddDate(0, 0, -28)
	default:
		since = isoWeekStart(time.Now())
	}

	rows, err := pool.Query(ctx, `
		SELECT
			ci.id,
			g.title,
			LEFT(COALESCE(ev.preview, ''), 200),
			ci.submitted_at,
			u.id,
			u.display_name,
			u.avatar_url,
			COUNT(DISTINCT pl.id) AS likes_count,
			(ci.approved_at IS NOT NULL) AS buddy_approved,
			COALESCE(ev.proof_type, 'text') AS proof_type,
			ev.proof_url
		FROM check_ins ci
		JOIN goals g ON g.id = ci.goal_id
		JOIN circle_memberships cm ON cm.user_id = g.owner_user_id AND cm.circle_id = $1 AND cm.status = 'active'
		JOIN users u ON u.id = g.owner_user_id
		LEFT JOIN proof_likes pl ON pl.check_in_id = ci.id
		LEFT JOIN LATERAL (
			SELECT
				COALESCE(NULLIF(ei.text_content, ''), NULLIF(ei.external_url, ''), ei.kind) AS preview,
				ei.kind AS proof_type,
				ei.external_url AS proof_url
			FROM evidence_items ei
			WHERE ei.check_in_id = ci.id
			ORDER BY ei.id
			LIMIT 1
		) ev ON TRUE
		WHERE ci.status = 'approved'
		  AND ci.submitted_at >= $2
		GROUP BY ci.id, g.title, ev.preview, ci.submitted_at, u.id, u.display_name, u.avatar_url, ci.approved_at, ev.proof_type, ev.proof_url
		ORDER BY buddy_approved DESC, likes_count DESC, ci.submitted_at DESC
		LIMIT $3
	`, circleID, since, limit)
	if err != nil {
		return nil, fmt.Errorf("best proofs board: %w", err)
	}
	defer rows.Close()

	rank := 0
	for rows.Next() {
		rank++
		var e BestProofEntry
		var content sql.NullString
		var avatarURL sql.NullString
		var proofURL sql.NullString
		if err := rows.Scan(
			&e.CheckInID,
			&e.GoalTitle,
			&content,
			&e.SubmittedAt,
			&e.Author.UserID,
			&e.Author.DisplayName,
			&avatarURL,
			&e.LikesCount,
			&e.BuddyApproved,
			&e.ProofType,
			&proofURL,
		); err != nil {
			return nil, err
		}
		e.Rank = rank
		e.ProofPreview = content.String
		if avatarURL.Valid {
			e.Author.AvatarURL = &avatarURL.String
		}
		if proofURL.Valid {
			e.ProofURL = &proofURL.String
		}
		board.Entries = append(board.Entries, e)
	}
	if board.Entries == nil {
		board.Entries = []BestProofEntry{}
	}
	return board, rows.Err()
}

func toggleLike(ctx context.Context, pool *pgxpool.Pool, checkInID, userID int64) (*LikeResult, error) {
	// Toggle: insert if not exists, delete if exists.
	var liked bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM proof_likes WHERE check_in_id = $1 AND user_id = $2)
	`, checkInID, userID).Scan(&liked)
	if err != nil {
		return nil, fmt.Errorf("toggle like: check: %w", err)
	}

	if liked {
		if _, err := pool.Exec(ctx, `
			DELETE FROM proof_likes WHERE check_in_id = $1 AND user_id = $2
		`, checkInID, userID); err != nil {
			return nil, fmt.Errorf("toggle like: delete: %w", err)
		}
		liked = false
	} else {
		// Prevent self-liking — check if the check-in belongs to the same user.
		var ownerID int64
		if err := pool.QueryRow(ctx, `SELECT owner_user_id FROM check_ins WHERE id = $1`, checkInID).Scan(&ownerID); err != nil {
			return nil, fmt.Errorf("toggle like: owner: %w", err)
		}
		if ownerID == userID {
			return nil, errSelfLike
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO proof_likes(check_in_id, user_id) VALUES($1, $2) ON CONFLICT DO NOTHING
		`, checkInID, userID); err != nil {
			return nil, fmt.Errorf("toggle like: insert: %w", err)
		}
		liked = true
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM proof_likes WHERE check_in_id = $1`, checkInID).Scan(&count); err != nil {
		return nil, fmt.Errorf("toggle like: count: %w", err)
	}

	return &LikeResult{CheckInID: checkInID, LikesCount: count, LikedByMe: liked}, nil
}
