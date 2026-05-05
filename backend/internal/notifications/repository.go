package notifications

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository handles notification logging, domain events, and TG link lookups.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// Emit writes a domain event to the domain_events table.
// Implements checkins.DomainEventEmitter.
func (r *PostgresRepository) Emit(ctx context.Context, kind string, payload map[string]any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("notifications: marshal payload: %w", err)
	}
	_, err = r.pool.Exec(ctx,
		`INSERT INTO domain_events (kind, payload) VALUES ($1, $2)`,
		kind, data,
	)
	return err
}

// LogNotification inserts a notification log entry, ignoring duplicates (dedup_key).
func (r *PostgresRepository) LogNotification(ctx context.Context, userID int64, kind, dedupKey string) error {
	payload, _ := json.Marshal(map[string]string{"kind": kind})
	_, err := r.pool.Exec(ctx,
		`INSERT INTO notifications_log (user_id, kind, dedup_key, payload)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (dedup_key) DO NOTHING`,
		userID, kind, dedupKey, payload,
	)
	return err
}

// CountTodayNotifications returns the count of nudge notifications sent today for a user.
// quiet entries are not counted.
func (r *PostgresRepository) CountTodayNotifications(ctx context.Context, userID int64) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM notifications_log
		 WHERE user_id = $1
		   AND kind NOT IN ('quiet', 'digest')
		   AND sent_at >= date_trunc('day', now() AT TIME ZONE 'UTC')`,
		userID,
	).Scan(&count)
	return count, err
}

// IsQuietToday returns true if the user set quiet mode today.
func (r *PostgresRepository) IsQuietToday(ctx context.Context, userID int64) (bool, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM notifications_log
		 WHERE user_id = $1
		   AND kind = 'quiet'
		   AND sent_at >= date_trunc('day', now() AT TIME ZONE 'UTC')`,
		userID,
	).Scan(&count)
	return count > 0, err
}

// SetQuietToday logs a quiet marker for today.
func (r *PostgresRepository) SetQuietToday(ctx context.Context, userID int64) error {
	return r.LogNotification(ctx, userID, KindQuiet,
		fmt.Sprintf("quiet_%d_%s", userID, time.Now().UTC().Format("2006-01-02")))
}

// GetUnprocessedEvents returns domain events not yet processed, up to limit.
func (r *PostgresRepository) GetUnprocessedEvents(ctx context.Context, limit int) ([]DomainEvent, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, kind, payload, created_at
		 FROM domain_events
		 WHERE processed_at IS NULL
		 ORDER BY created_at
		 LIMIT $1`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []DomainEvent
	for rows.Next() {
		var e DomainEvent
		if err := rows.Scan(&e.ID, &e.Kind, &e.Payload, &e.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// MarkEventProcessed stamps processed_at on a domain event.
func (r *PostgresRepository) MarkEventProcessed(ctx context.Context, eventID int64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE domain_events SET processed_at = now() WHERE id = $1`,
		eventID,
	)
	return err
}

// GetGoalBuddyUserID returns the buddy_user_id for the active pact on a goal, or 0 if not found.
func (r *PostgresRepository) GetGoalBuddyUserID(ctx context.Context, goalID int64) (int64, error) {
	var buddyUserID int64
	err := r.pool.QueryRow(ctx,
		`SELECT p.buddy_user_id FROM pacts p WHERE p.goal_id = $1 AND p.status = 'active' LIMIT 1`,
		goalID,
	).Scan(&buddyUserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}
	return buddyUserID, nil
}

// GetOwnerDisplayName returns the display_name (or email fallback) for a user.
func (r *PostgresRepository) GetOwnerDisplayName(ctx context.Context, userID int64) (string, error) {
	var displayName, email string
	err := r.pool.QueryRow(ctx,
		`SELECT display_name, email FROM users WHERE id = $1`,
		userID,
	).Scan(&displayName, &email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	if displayName != "" {
		return displayName, nil
	}
	return email, nil
}

// GetTelegramChatID returns the telegram_chat_id for a user, or 0 if not linked.
func (r *PostgresRepository) GetTelegramChatID(ctx context.Context, userID int64) (int64, error) {
	var chatID int64
	err := r.pool.QueryRow(ctx,
		`SELECT telegram_chat_id FROM telegram_links WHERE user_id = $1 AND status = 'active'`,
		userID,
	).Scan(&chatID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}
	return chatID, nil
}

// DigestTargetRow is a user in a circle that needs a digest sent.
type DigestTargetRow struct {
	UserID      int64
	DisplayName string
	ChatID      int64
	CircleID    int64
	CircleName  string
	DayOfSeason int
	TotalDays   int
	Timezone    string
}

// GetDigestTargets returns users whose circle digest time has arrived locally.
func (r *PostgresRepository) GetDigestTargets(ctx context.Context) ([]DigestTargetRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			u.id,
			u.display_name,
			tl.telegram_chat_id,
			c.id AS circle_id,
			c.name AS circle_name,
			GREATEST(1, (EXTRACT(EPOCH FROM (now() AT TIME ZONE c.daily_window_tz - cs.starts_at AT TIME ZONE c.daily_window_tz)) / 86400)::int + 1) AS day_of_season,
			28 AS total_days,
			c.daily_window_tz
		FROM circles c
		JOIN circle_seasons cs ON cs.circle_id = c.id AND cs.status = 'active'
		JOIN circle_members cm ON cm.circle_id = c.id AND cm.status = 'active'
		JOIN users u ON u.id = cm.user_id
		JOIN telegram_links tl ON tl.user_id = u.id AND tl.status = 'active'
		WHERE
			EXTRACT(HOUR FROM now() AT TIME ZONE c.daily_window_tz) = 8
			AND EXTRACT(MINUTE FROM now() AT TIME ZONE c.daily_window_tz) BETWEEN 30 AND 34
			AND NOT EXISTS (
				SELECT 1 FROM notifications_log nl
				WHERE nl.user_id = u.id
				  AND nl.kind = 'digest'
				  AND nl.dedup_key = 'digest_' || c.id || '_' || (now() AT TIME ZONE c.daily_window_tz)::date
			)
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var targets []DigestTargetRow
	for rows.Next() {
		var t DigestTargetRow
		if err := rows.Scan(&t.UserID, &t.DisplayName, &t.ChatID, &t.CircleID, &t.CircleName,
			&t.DayOfSeason, &t.TotalDays, &t.Timezone); err != nil {
			return nil, err
		}
		targets = append(targets, t)
	}
	return targets, rows.Err()
}

// DigestMembersRow is standings data for the digest message.
type DigestMembersRow struct {
	UserID      int64
	DisplayName string
	Streak      int
	HasCheckin  bool
	Rank        int
}

// GetCircleDigestMembers returns member standings for a given circle today.
func (r *PostgresRepository) GetCircleDigestMembers(ctx context.Context, circleID int64) ([]DigestMembersRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			u.id,
			u.display_name,
			COALESCE(g.current_streak, 0) AS streak,
			EXISTS (
				SELECT 1 FROM check_ins ci
				JOIN goals gg ON gg.id = ci.goal_id
				WHERE gg.owner_user_id = u.id
				  AND gg.circle_id = $1
				  AND ci.status = 'approved'
				  AND ci.approved_at >= date_trunc('day', now())
			) AS has_checkin,
			ROW_NUMBER() OVER (ORDER BY COALESCE(g.current_streak,0) DESC, u.display_name) AS rank
		FROM circle_members cm
		JOIN users u ON u.id = cm.user_id
		LEFT JOIN goals g ON g.owner_user_id = u.id AND g.circle_id = $1 AND g.status = 'active'
		WHERE cm.circle_id = $1 AND cm.status = 'active'
		ORDER BY rank
	`, circleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []DigestMembersRow
	for rows.Next() {
		var m DigestMembersRow
		if err := rows.Scan(&m.UserID, &m.DisplayName, &m.Streak, &m.HasCheckin, &m.Rank); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, rows.Err()
}
