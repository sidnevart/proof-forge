package checkins

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) CreateCheckIn(ctx context.Context, params CreateCheckInParams) (CheckIn, error) {
	// Verify the goal is active and the actor is the owner in the same statement.
	const query = `
		INSERT INTO check_ins (goal_id, owner_user_id, status)
		SELECT $1, $2, $3
		FROM goals
		WHERE id = $1
		  AND owner_user_id = $2
		  AND status = 'active'
		RETURNING id, goal_id, owner_user_id, status, submitted_at, approved_at, rejected_at, changes_requested_at, created_at, updated_at
	`

	var ci CheckIn
	var submittedAt, approvedAt, rejectedAt, changesAt sql.NullTime
	err := r.pool.QueryRow(ctx, query, params.GoalID, params.OwnerUserID, params.Status).Scan(
		&ci.ID, &ci.GoalID, &ci.OwnerUserID, &ci.Status,
		&submittedAt, &approvedAt, &rejectedAt, &changesAt,
		&ci.CreatedAt, &ci.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CheckIn{}, ErrGoalNotEligible
		}
		return CheckIn{}, fmt.Errorf("insert check-in: %w", err)
	}
	applyNullTimes(&ci, submittedAt, approvedAt, rejectedAt, changesAt)
	return ci, nil
}

func (r *PostgresRepository) GetCheckIn(ctx context.Context, checkInID int64) (CheckInView, error) {
	const query = `
		SELECT
			c.id, c.goal_id, c.owner_user_id, c.status,
			c.submitted_at, c.approved_at, c.rejected_at, c.changes_requested_at,
			c.created_at, c.updated_at,
			g.buddy_user_id,
			e.id, e.kind, e.text_content, e.external_url,
			e.storage_key, e.mime_type, e.file_size_bytes, e.created_at
		FROM check_ins c
		JOIN goals g ON g.id = c.goal_id
		LEFT JOIN evidence_items e ON e.check_in_id = c.id
		WHERE c.id = $1
		ORDER BY e.created_at ASC
	`

	rows, err := r.pool.Query(ctx, query, checkInID)
	if err != nil {
		return CheckInView{}, fmt.Errorf("query check-in: %w", err)
	}
	defer rows.Close()

	var view CheckInView
	found := false
	for rows.Next() {
		var (
			submittedAt, approvedAt, rejectedAt, changesAt sql.NullTime
			eID                                            sql.NullInt64
			eKind                                          sql.NullString
			eText, eURL, eKey, eMIME                       sql.NullString
			eSize                                          sql.NullInt64
			eCreatedAt                                     sql.NullTime
		)
		if err := rows.Scan(
			&view.CheckIn.ID, &view.CheckIn.GoalID, &view.CheckIn.OwnerUserID, &view.CheckIn.Status,
			&submittedAt, &approvedAt, &rejectedAt, &changesAt,
			&view.CheckIn.CreatedAt, &view.CheckIn.UpdatedAt,
			&view.BuddyUserID,
			&eID, &eKind, &eText, &eURL, &eKey, &eMIME, &eSize, &eCreatedAt,
		); err != nil {
			return CheckInView{}, fmt.Errorf("scan check-in row: %w", err)
		}
		if !found {
			applyNullTimes(&view.CheckIn, submittedAt, approvedAt, rejectedAt, changesAt)
			found = true
		}
		if eID.Valid {
			view.Evidence = append(view.Evidence, EvidenceItem{
				ID:            eID.Int64,
				CheckInID:     checkInID,
				Kind:          EvidenceKind(eKind.String),
				TextContent:   eText.String,
				ExternalURL:   eURL.String,
				StorageKey:    eKey.String,
				MIMEType:      eMIME.String,
				FileSizeBytes: eSize.Int64,
				CreatedAt:     eCreatedAt.Time,
			})
		}
	}
	if err := rows.Err(); err != nil {
		return CheckInView{}, fmt.Errorf("iterate check-in rows: %w", err)
	}
	if !found {
		return CheckInView{}, ErrCheckInNotFound
	}
	return view, nil
}

func (r *PostgresRepository) ListCheckInsByGoal(ctx context.Context, goalID int64, actorID int64) ([]CheckIn, error) {
	const query = `
		SELECT c.id, c.goal_id, c.owner_user_id, c.status,
		       c.submitted_at, c.approved_at, c.rejected_at, c.changes_requested_at,
		       c.created_at, c.updated_at
		FROM check_ins c
		JOIN goals g ON g.id = c.goal_id
		WHERE c.goal_id = $1
		  AND (c.owner_user_id = $2 OR g.buddy_user_id = $2)
		ORDER BY c.created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, goalID, actorID)
	if err != nil {
		return nil, fmt.Errorf("query check-ins for goal: %w", err)
	}
	defer rows.Close()

	var list []CheckIn
	for rows.Next() {
		var ci CheckIn
		var submittedAt, approvedAt, rejectedAt, changesAt sql.NullTime
		if err := rows.Scan(
			&ci.ID, &ci.GoalID, &ci.OwnerUserID, &ci.Status,
			&submittedAt, &approvedAt, &rejectedAt, &changesAt,
			&ci.CreatedAt, &ci.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan check-in list row: %w", err)
		}
		applyNullTimes(&ci, submittedAt, approvedAt, rejectedAt, changesAt)
		list = append(list, ci)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate check-in list: %w", err)
	}
	return list, nil
}

func (r *PostgresRepository) UpdateCheckInStatus(ctx context.Context, params UpdateStatusParams) error {
	const query = `
		UPDATE check_ins
		SET status = $2,
		    submitted_at         = COALESCE($3, submitted_at),
		    approved_at          = COALESCE($4, approved_at),
		    rejected_at          = COALESCE($5, rejected_at),
		    changes_requested_at = COALESCE($6, changes_requested_at),
		    updated_at           = $7
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, query,
		params.ID, params.Status,
		params.SubmittedAt, params.ApprovedAt,
		params.RejectedAt, params.ChangesRequestedAt,
		params.Now,
	)
	if err != nil {
		return fmt.Errorf("update check-in status: %w", err)
	}
	return nil
}

func (r *PostgresRepository) InsertEvidence(ctx context.Context, params InsertEvidenceParams) (EvidenceItem, error) {
	const query = `
		INSERT INTO evidence_items
			(check_in_id, kind, text_content, external_url, storage_key, mime_type, file_size_bytes)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, check_in_id, kind, text_content, external_url, storage_key, mime_type, file_size_bytes, created_at
	`

	var (
		item                                    EvidenceItem
		textContent, externalURL, storageKey sql.NullString
		mimeType                                sql.NullString
		fileSize                                sql.NullInt64
	)
	err := r.pool.QueryRow(ctx, query,
		params.CheckInID, params.Kind,
		emptyToNull(params.TextContent),
		emptyToNull(params.ExternalURL),
		emptyToNull(params.StorageKey),
		emptyToNull(params.MIMEType),
		nullableInt64(params.FileSizeBytes),
	).Scan(
		&item.ID, &item.CheckInID, &item.Kind,
		&textContent, &externalURL, &storageKey, &mimeType, &fileSize,
		&item.CreatedAt,
	)
	if err != nil {
		return EvidenceItem{}, fmt.Errorf("insert evidence: %w", err)
	}
	item.TextContent = textContent.String
	item.ExternalURL = externalURL.String
	item.StorageKey = storageKey.String
	item.MIMEType = mimeType.String
	item.FileSizeBytes = fileSize.Int64
	return item, nil
}

func (r *PostgresRepository) CountEvidence(ctx context.Context, checkInID int64) (int, error) {
	var count int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM evidence_items WHERE check_in_id = $1`, checkInID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count evidence: %w", err)
	}
	return count, nil
}

func (r *PostgresRepository) RecordReview(ctx context.Context, params RecordReviewParams) (ReviewRecord, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return ReviewRecord{}, fmt.Errorf("begin review tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var rec ReviewRecord
	err = tx.QueryRow(ctx, `
		INSERT INTO check_in_reviews (check_in_id, reviewer_user_id, decision, comment)
		VALUES ($1, $2, $3, $4)
		RETURNING id, check_in_id, reviewer_user_id, decision, comment, created_at
	`, params.CheckInID, params.ReviewerUserID, params.Decision, params.Comment).Scan(
		&rec.ID, &rec.CheckInID, &rec.ReviewerUserID, &rec.Decision, &rec.Comment, &rec.CreatedAt,
	)
	if err != nil {
		return ReviewRecord{}, fmt.Errorf("insert review record: %w", err)
	}

	var approvedAt, rejectedAt, changesAt *time.Time
	switch params.Decision {
	case DecisionApprove:
		approvedAt = &params.Now
	case DecisionReject:
		rejectedAt = &params.Now
	case DecisionRequestChanges:
		changesAt = &params.Now
	}

	if _, err = tx.Exec(ctx, `
		UPDATE check_ins
		SET status                = $2,
		    approved_at           = COALESCE($3, approved_at),
		    rejected_at           = COALESCE($4, rejected_at),
		    changes_requested_at  = COALESCE($5, changes_requested_at),
		    updated_at            = $6
		WHERE id = $1
	`, params.CheckInID, params.Decision, approvedAt, rejectedAt, changesAt, params.Now); err != nil {
		return ReviewRecord{}, fmt.Errorf("update check-in status: %w", err)
	}

	switch params.Decision {
	case DecisionApprove:
		if _, err = tx.Exec(ctx, `
			UPDATE goals
			SET current_streak_count    = current_streak_count + 1,
			    current_progress_health = 'stable',
			    updated_at              = $2
			WHERE id = $1
		`, params.GoalID, params.Now); err != nil {
			return ReviewRecord{}, fmt.Errorf("update goal progress on approve: %w", err)
		}
	case DecisionReject:
		if _, err = tx.Exec(ctx, `
			UPDATE goals
			SET current_streak_count    = 0,
			    current_progress_health = 'at_risk',
			    updated_at              = $2
			WHERE id = $1
		`, params.GoalID, params.Now); err != nil {
			return ReviewRecord{}, fmt.Errorf("update goal progress on reject: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return ReviewRecord{}, fmt.Errorf("commit review tx: %w", err)
	}
	return rec, nil
}

func applyNullTimes(ci *CheckIn, submitted, approved, rejected, changes sql.NullTime) {
	if submitted.Valid {
		t := submitted.Time
		ci.SubmittedAt = &t
	}
	if approved.Valid {
		t := approved.Time
		ci.ApprovedAt = &t
	}
	if rejected.Valid {
		t := rejected.Time
		ci.RejectedAt = &t
	}
	if changes.Valid {
		t := changes.Time
		ci.ChangesRequestedAt = &t
	}
}

func emptyToNull(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func nullableInt64(n int64) any {
	if n == 0 {
		return nil
	}
	return n
}

