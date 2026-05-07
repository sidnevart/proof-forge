package dailylog

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository is the production storage for daily-log and streak.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository constructs the production repo.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// UpsertEntry inserts or updates a daily-log row, returning the final row.
func (r *PostgresRepository) UpsertEntry(ctx context.Context, e *Entry) (*Entry, error) {
	const sql = `
		INSERT INTO daily_log_entries
			(user_id, team_id, log_date, status, text_content, has_artifact,
			 external_url, storage_key, mime_type, file_size_bytes, submitted_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (user_id, team_id, log_date)
		DO UPDATE SET
			status = EXCLUDED.status,
			text_content = EXCLUDED.text_content,
			has_artifact = EXCLUDED.has_artifact,
			external_url = EXCLUDED.external_url,
			storage_key = EXCLUDED.storage_key,
			mime_type = EXCLUDED.mime_type,
			file_size_bytes = EXCLUDED.file_size_bytes,
			overwritten_at = NOW()
		RETURNING id, user_id, team_id, log_date, status, text_content, has_artifact,
			  external_url, storage_key, mime_type, file_size_bytes,
			  submitted_at, overwritten_at, consumed_in_check_in_id, consumed_at
	`
	row := r.pool.QueryRow(ctx, sql,
		e.UserID, e.TeamID, e.LogDate, string(e.Status), e.TextContent, e.HasArtifact,
		e.ExternalURL, e.StorageKey, e.MimeType, e.FileSizeBytes, e.SubmittedAt,
	)
	return scanEntry(row.Scan)
}

// GetEntry fetches a single entry by composite key.
func (r *PostgresRepository) GetEntry(ctx context.Context, userID, teamID int64, date time.Time) (*Entry, error) {
	const sql = `
		SELECT id, user_id, team_id, log_date, status, text_content, has_artifact,
			   external_url, storage_key, mime_type, file_size_bytes,
			   submitted_at, overwritten_at, consumed_in_check_in_id, consumed_at
		FROM daily_log_entries
		WHERE user_id = $1 AND team_id = $2 AND log_date = $3
	`
	row := r.pool.QueryRow(ctx, sql, userID, teamID, date)
	return scanEntry(row.Scan)
}

// ListEntries returns entries for a user/team in a date range.
func (r *PostgresRepository) ListEntries(ctx context.Context, filter ListFilter) ([]Entry, error) {
	const sql = `
		SELECT id, user_id, team_id, log_date, status, text_content, has_artifact,
			   external_url, storage_key, mime_type, file_size_bytes,
			   submitted_at, overwritten_at, consumed_in_check_in_id, consumed_at
		FROM daily_log_entries
		WHERE user_id = $1 AND team_id = $2 AND log_date BETWEEN $3 AND $4
		ORDER BY log_date DESC
	`
	rows, err := r.pool.Query(ctx, sql, filter.UserID, filter.TeamID, filter.From, filter.To)
	if err != nil {
		return nil, fmt.Errorf("list entries: %w", err)
	}
	defer rows.Close()

	out := make([]Entry, 0)
	for rows.Next() {
		e, err := scanEntry(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, *e)
	}
	return out, rows.Err()
}

// GetStreak returns the streak row or a zero-value streak if absent.
func (r *PostgresRepository) GetStreak(ctx context.Context, userID, teamID int64) (*Streak, error) {
	const sql = `
		SELECT user_id, team_id, current_streak, best_streak, last_active_date,
			   freezes_used_this_month, updated_at
		FROM user_streak
		WHERE user_id = $1 AND team_id = $2
	`
	row := r.pool.QueryRow(ctx, sql, userID, teamID)
	var s Streak
	var last *time.Time
	err := row.Scan(
		&s.UserID, &s.TeamID, &s.Current, &s.Best, &last,
		&s.FreezesUsedThisMonth, &s.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return &Streak{UserID: userID, TeamID: teamID}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get streak: %w", err)
	}
	s.LastActiveDate = last
	return &s, nil
}

// UpdateStreak upserts the streak row atomically.
func (r *PostgresRepository) UpdateStreak(ctx context.Context, s *Streak) error {
	const sql = `
		INSERT INTO user_streak
			(user_id, team_id, current_streak, best_streak, last_active_date,
			 freezes_used_this_month, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id, team_id)
		DO UPDATE SET
			current_streak = EXCLUDED.current_streak,
			best_streak = EXCLUDED.best_streak,
			last_active_date = EXCLUDED.last_active_date,
			freezes_used_this_month = EXCLUDED.freezes_used_this_month,
			updated_at = EXCLUDED.updated_at
	`
	_, err := r.pool.Exec(ctx, sql,
		s.UserID, s.TeamID, s.Current, s.Best, s.LastActiveDate,
		s.FreezesUsedThisMonth, s.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update streak: %w", err)
	}
	return nil
}

// GetMembership reads the active membership row for authorization pre-flight.
func (r *PostgresRepository) GetMembership(ctx context.Context, teamID, userID int64) (MembershipInfo, error) {
	const sql = `
		SELECT role, status, timezone, ai_consent
		FROM team_memberships
		WHERE team_id = $1 AND user_id = $2 AND status = 'active'
	`
	var m MembershipInfo
	err := r.pool.QueryRow(ctx, sql, teamID, userID).Scan(
		&m.Role, &m.Status, &m.Timezone, &m.AIConsent,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return MembershipInfo{}, ErrNotMember
	}
	if err != nil {
		return MembershipInfo{}, fmt.Errorf("get membership: %w", err)
	}
	return m, nil
}

// UpsertEntryAndStreak runs both upserts inside a single transaction.
func (r *PostgresRepository) UpsertEntryAndStreak(ctx context.Context, e *Entry, s *Streak) (*Entry, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin upsert tx: %w", err)
	}
	defer tx.Rollback(ctx)

	const entrySQL = `
		INSERT INTO daily_log_entries
			(user_id, team_id, log_date, status, text_content, has_artifact,
			 external_url, storage_key, mime_type, file_size_bytes, submitted_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (user_id, team_id, log_date)
		DO UPDATE SET
			status = EXCLUDED.status,
			text_content = EXCLUDED.text_content,
			has_artifact = EXCLUDED.has_artifact,
			external_url = EXCLUDED.external_url,
			storage_key = EXCLUDED.storage_key,
			mime_type = EXCLUDED.mime_type,
			file_size_bytes = EXCLUDED.file_size_bytes,
			overwritten_at = NOW()
		RETURNING id, user_id, team_id, log_date, status, text_content, has_artifact,
			  external_url, storage_key, mime_type, file_size_bytes,
			  submitted_at, overwritten_at, consumed_in_check_in_id, consumed_at
	`
	row := tx.QueryRow(ctx, entrySQL,
		e.UserID, e.TeamID, e.LogDate, string(e.Status), e.TextContent, e.HasArtifact,
		e.ExternalURL, e.StorageKey, e.MimeType, e.FileSizeBytes, e.SubmittedAt,
	)
	entry, err := scanEntry(row.Scan)
	if err != nil {
		return nil, fmt.Errorf("upsert entry tx: %w", err)
	}

	const streakSQL = `
		INSERT INTO user_streak
			(user_id, team_id, current_streak, best_streak, last_active_date,
			 freezes_used_this_month, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id, team_id)
		DO UPDATE SET
			current_streak = EXCLUDED.current_streak,
			best_streak = EXCLUDED.best_streak,
			last_active_date = EXCLUDED.last_active_date,
			freezes_used_this_month = EXCLUDED.freezes_used_this_month,
			updated_at = EXCLUDED.updated_at
	`
	_, err = tx.Exec(ctx, streakSQL,
		s.UserID, s.TeamID, s.Current, s.Best, s.LastActiveDate,
		s.FreezesUsedThisMonth, s.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("update streak tx: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit upsert tx: %w", err)
	}
	return entry, nil
}

func scanEntry(scan func(...any) error) (*Entry, error) {
	var e Entry
	var ext, stor, mime *string
	var fsize *int64
	var over *time.Time
	var consumedID *int64
	var consumedAt *time.Time
	err := scan(
		&e.ID, &e.UserID, &e.TeamID, &e.LogDate, &e.Status, &e.TextContent, &e.HasArtifact,
		&ext, &stor, &mime, &fsize,
		&e.SubmittedAt, &over, &consumedID, &consumedAt,
	)
	if err != nil {
		return nil, err
	}
	e.ExternalURL = ext
	e.StorageKey = stor
	e.MimeType = mime
	e.FileSizeBytes = fsize
	e.OverwrittenAt = over
	e.ConsumedInCheckInID = consumedID
	e.ConsumedAt = consumedAt
	return &e, nil
}
