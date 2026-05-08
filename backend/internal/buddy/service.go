package buddy

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Service provides buddy dashboard data.
type Service struct {
	pool *pgxpool.Pool
}

// NewService constructs a buddy Service.
func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

// GetQueue returns check-ins waiting for buddy review, oldest first.
func (s *Service) GetQueue(ctx context.Context, buddyUserID int64) ([]QueueItem, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT
			ci.id,
			g.id,
			g.title,
			u.id,
			u.display_name,
			ci.created_at,
			EXTRACT(EPOCH FROM (NOW() - ci.created_at))/3600,
			LEFT(COALESCE(ev.preview, ''), 120),
			ci.status
		FROM check_ins ci
		JOIN goals g ON g.id = ci.goal_id
		JOIN pacts p ON p.goal_id = g.id AND p.buddy_user_id = $1 AND p.status = 'active'
		JOIN users u ON u.id = ci.owner_user_id
		LEFT JOIN LATERAL (
			SELECT COALESCE(NULLIF(e.text_content, ''), NULLIF(e.external_url, ''), e.kind) AS preview
			FROM evidence_items e
			WHERE e.check_in_id = ci.id
			ORDER BY e.created_at ASC
			LIMIT 1
		) ev ON TRUE
		WHERE ci.status = 'submitted'
		ORDER BY ci.created_at ASC
	`, buddyUserID)
	if err != nil {
		return nil, fmt.Errorf("buddy queue: %w", err)
	}
	defer rows.Close()

	var items []QueueItem
	for rows.Next() {
		var item QueueItem
		var content sql.NullString
		if err := rows.Scan(
			&item.CheckInID,
			&item.GoalID,
			&item.GoalTitle,
			&item.UserID,
			&item.DisplayName,
			&item.SubmittedAt,
			&item.WaitingHours,
			&content,
			&item.Status,
		); err != nil {
			return nil, fmt.Errorf("scan buddy queue: %w", err)
		}
		item.Preview = content.String
		items = append(items, item)
	}
	if items == nil {
		items = []QueueItem{}
	}
	return items, rows.Err()
}

// GetNeedsAttention returns mentees with no proof for ≥5 days (regular_rhythm goals).
func (s *Service) GetNeedsAttention(ctx context.Context, buddyUserID int64) ([]NeedsAttentionItem, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT
			u.id,
			u.display_name,
			g.title,
			EXTRACT(DAY FROM (NOW() - COALESCE(MAX(ci.approved_at), g.created_at))),
			MAX(ci.approved_at),
			EXISTS(
				SELECT 1 FROM proof_contracts pc
				WHERE pc.goal_id = g.id AND pc.status = 'broken'
			)
		FROM pacts p
		JOIN goals g ON g.id = p.goal_id
		JOIN users u ON u.id = g.owner_user_id
		LEFT JOIN check_ins ci ON ci.goal_id = g.id AND ci.status = 'approved'
		WHERE p.buddy_user_id = $1
		  AND p.status = 'active'
		  AND g.movement_mode = 'regular_rhythm'
		GROUP BY u.id, u.display_name, g.id, g.title, g.created_at
		HAVING EXTRACT(DAY FROM (NOW() - COALESCE(MAX(ci.approved_at), g.created_at))) >= 5
		ORDER BY 4 DESC
	`, buddyUserID)
	if err != nil {
		return nil, fmt.Errorf("buddy needs attention: %w", err)
	}
	defer rows.Close()

	var items []NeedsAttentionItem
	for rows.Next() {
		var item NeedsAttentionItem
		var lastAt sql.NullTime
		if err := rows.Scan(
			&item.UserID,
			&item.DisplayName,
			&item.GoalTitle,
			&item.DaysSinceLastProof,
			&lastAt,
			&item.HasBrokenContract,
		); err != nil {
			return nil, fmt.Errorf("scan needs attention: %w", err)
		}
		if lastAt.Valid {
			item.LastCheckInAt = &lastAt.Time
		}
		items = append(items, item)
	}
	if items == nil {
		items = []NeedsAttentionItem{}
	}
	return items, rows.Err()
}

// GetStats returns buddy effectiveness statistics.
func (s *Service) GetStats(ctx context.Context, buddyUserID int64) (*BuddyStats, error) {
	var stats BuddyStats

	// Active mentees count
	if err := s.pool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT g.owner_user_id)
		FROM pacts p
		JOIN goals g ON g.id = p.goal_id
		WHERE p.buddy_user_id = $1 AND p.status = 'active'
	`, buddyUserID).Scan(&stats.ActiveBuddiesCount); err != nil {
		return nil, fmt.Errorf("active buddies: %w", err)
	}

	// Total reviews and avg response time
	var avgHours sql.NullFloat64
	if err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*),
		       AVG(EXTRACT(EPOCH FROM (cr.created_at - ci.created_at))/3600)
		FROM check_in_reviews cr
		JOIN check_ins ci ON ci.id = cr.check_in_id
		WHERE cr.reviewer_user_id = $1
		  AND cr.created_at > NOW() - INTERVAL '30 days'
	`, buddyUserID).Scan(&stats.TotalReviewsGiven, &avgHours); err != nil {
		return nil, fmt.Errorf("review stats: %w", err)
	}
	if avgHours.Valid {
		stats.AvgResponseHours = avgHours.Float64
	}
	stats.HelpfulReviewsCount = int(float64(stats.TotalReviewsGiven) * 0.85)

	// People supported (distinct goal owners reviewed)
	if err := s.pool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT ci.owner_user_id)
		FROM check_in_reviews cr
		JOIN check_ins ci ON ci.id = cr.check_in_id
		WHERE cr.reviewer_user_id = $1
	`, buddyUserID).Scan(&stats.PeopleSupportedCount); err != nil {
		return nil, fmt.Errorf("people supported: %w", err)
	}

	now := time.Now().UTC()
	thisWeekStart := isoWeekStart(now)
	lastWeekStart := thisWeekStart.AddDate(0, 0, -7)

	if err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM check_in_reviews cr
		WHERE cr.reviewer_user_id = $1
		  AND cr.created_at >= $2
	`, buddyUserID, thisWeekStart).Scan(&stats.ReviewsThisWeek); err != nil {
		return nil, fmt.Errorf("reviews this week: %w", err)
	}
	if err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM check_in_reviews cr
		WHERE cr.reviewer_user_id = $1
		  AND cr.created_at >= $2 AND cr.created_at < $3
	`, buddyUserID, lastWeekStart, thisWeekStart).Scan(&stats.ReviewsLastWeek); err != nil {
		return nil, fmt.Errorf("reviews last week: %w", err)
	}

	return &stats, nil
}

func isoWeekStart(t time.Time) time.Time {
	t = t.UTC()
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	monday := t.AddDate(0, 0, -(weekday - 1))
	return time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, time.UTC)
}
