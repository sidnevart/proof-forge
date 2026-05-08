package companion

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository implements Repository using pgx.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgresRepository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// --- ai_companion_fired ---

func (r *PostgresRepository) HasFired(ctx context.Context, userID int64, triggerID string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM ai_companion_fired WHERE user_id = $1 AND trigger_id = $2)`,
		userID, triggerID,
	).Scan(&exists)
	return exists, err
}

func (r *PostgresRepository) RecordFired(ctx context.Context, userID int64, feature Feature, triggerID string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO ai_companion_fired (user_id, feature, trigger_id) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`,
		userID, string(feature), triggerID,
	)
	return err
}

func (r *PostgresRepository) CountFiredToday(ctx context.Context, userID int64, feature Feature) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM ai_companion_fired WHERE user_id = $1 AND feature = $2 AND fired_at >= CURRENT_DATE`,
		userID, string(feature),
	).Scan(&count)
	return count, err
}

func (r *PostgresRepository) CountFiredThisWeek(ctx context.Context, userID int64, feature Feature) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM ai_companion_fired WHERE user_id = $1 AND feature = $2 AND fired_at >= DATE_TRUNC('week', CURRENT_DATE)`,
		userID, string(feature),
	).Scan(&count)
	return count, err
}

// --- ai_proof_drafts ---

func (r *PostgresRepository) SaveProofDraft(ctx context.Context, draft *ProofDraft) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO ai_proof_drafts (id, user_id, team_id, goal_id, note_ids, rationale, confidence)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (id) DO UPDATE SET
		   goal_id = EXCLUDED.goal_id,
		   note_ids = EXCLUDED.note_ids,
		   rationale = EXCLUDED.rationale,
		   confidence = EXCLUDED.confidence`,
		draft.ID, draft.UserID, draft.TeamID, draft.GoalID, draft.NoteIDs, draft.Rationale, draft.Confidence,
	)
	return err
}

func (r *PostgresRepository) GetProofDrafts(ctx context.Context, userID int64, activeOnly bool) ([]*ProofDraft, error) {
	query := `SELECT id, user_id, team_id, goal_id, note_ids, rationale, confidence, created_at FROM ai_proof_drafts WHERE user_id = $1`
	if activeOnly {
		query += ` AND consumed_at IS NULL`
	}
	query += ` ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var drafts []*ProofDraft
	for rows.Next() {
		var d ProofDraft
		var goalID sql.NullInt64
		err := rows.Scan(&d.ID, &d.UserID, &d.TeamID, &goalID, &d.NoteIDs, &d.Rationale, &d.Confidence, &d.CreatedAt)
		if err != nil {
			continue
		}
		if goalID.Valid {
			v := goalID.Int64
			d.GoalID = &v
		}
		drafts = append(drafts, &d)
	}
	return drafts, nil
}

func (r *PostgresRepository) GetProofDraftByID(ctx context.Context, userID int64, id string) (*ProofDraft, error) {
	var d ProofDraft
	var goalID sql.NullInt64
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, team_id, goal_id, note_ids, rationale, confidence, created_at FROM ai_proof_drafts WHERE user_id = $1 AND id = $2 AND consumed_at IS NULL`,
		userID, id,
	).Scan(&d.ID, &d.UserID, &d.TeamID, &goalID, &d.NoteIDs, &d.Rationale, &d.Confidence, &d.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("draft not found: %w", err)
	}
	if goalID.Valid {
		v := goalID.Int64
		d.GoalID = &v
	}
	return &d, nil
}

func (r *PostgresRepository) ConsumeProofDraft(ctx context.Context, userID int64, id string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE ai_proof_drafts SET consumed_at = NOW() WHERE user_id = $1 AND id = $2`,
		userID, id,
	)
	return err
}

// --- ai_companion_cache ---

func (r *PostgresRepository) GetCache(ctx context.Context, cacheKey string) (*CacheEntry, error) {
	var e CacheEntry
	err := r.pool.QueryRow(ctx,
		`SELECT cache_key, feature, payload, expires_at, created_at FROM ai_companion_cache WHERE cache_key = $1 AND expires_at > NOW()`,
		cacheKey,
	).Scan(&e.CacheKey, &e.Feature, &e.Payload, &e.ExpiresAt, &e.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *PostgresRepository) SetCache(ctx context.Context, entry *CacheEntry) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO ai_companion_cache (cache_key, feature, payload, expires_at)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (cache_key) DO UPDATE SET
		   feature = EXCLUDED.feature,
		   payload = EXCLUDED.payload,
		   expires_at = EXCLUDED.expires_at,
		   created_at = NOW()`,
		entry.CacheKey, string(entry.Feature), entry.Payload, entry.ExpiresAt,
	)
	return err
}

func (r *PostgresRepository) DeleteCache(ctx context.Context, cacheKey string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM ai_companion_cache WHERE cache_key = $1`, cacheKey)
	return err
}

// --- ai_notifications ---

func (r *PostgresRepository) SaveNotification(ctx context.Context, n *Notification) error {
	actionsJSON, err := json.Marshal(n.Actions)
	if err != nil {
		return fmt.Errorf("marshal actions: %w", err)
	}
	_, err = r.pool.Exec(ctx,
		`INSERT INTO ai_notifications (id, user_id, feature, title, body, actions)
		 VALUES (gen_random_uuid(), $1, $2, $3, $4, $5)`,
		n.UserID, string(n.Feature), n.Title, n.Body, actionsJSON,
	)
	return err
}

func (r *PostgresRepository) GetNotifications(ctx context.Context, userID int64, activeOnly bool) ([]*Notification, error) {
	query := `SELECT id, user_id, feature, title, body, actions, dismissed_at, created_at FROM ai_notifications WHERE user_id = $1`
	if activeOnly {
		query += ` AND dismissed_at IS NULL`
	}
	query += ` ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []*Notification
	for rows.Next() {
		var n Notification
		var actionsRaw []byte
		var dismissedAt sql.NullTime
		err := rows.Scan(&n.ID, &n.UserID, &n.Feature, &n.Title, &n.Body, &actionsRaw, &dismissedAt, &n.CreatedAt)
		if err != nil {
			continue
		}
		if dismissedAt.Valid {
			n.DismissedAt = &dismissedAt.Time
		}
		_ = json.Unmarshal(actionsRaw, &n.Actions)
		notes = append(notes, &n)
	}
	return notes, nil
}

func (r *PostgresRepository) DismissNotification(ctx context.Context, userID int64, id string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE ai_notifications SET dismissed_at = NOW() WHERE user_id = $1 AND id = $2`,
		userID, id,
	)
	return err
}
