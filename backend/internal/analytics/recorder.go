package analytics

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Recorder writes analytics events.
type Recorder interface {
	Record(ctx context.Context, name EventName, source Source, userID, teamID, goalID, proofID *int64, props map[string]any) error
}

// PostgresRecorder is the production implementation.
type PostgresRecorder struct {
	pool *pgxpool.Pool
}

func NewPostgresRecorder(pool *pgxpool.Pool) *PostgresRecorder {
	return &PostgresRecorder{pool: pool}
}

// Record inserts a single analytics event.
func (r *PostgresRecorder) Record(ctx context.Context, name EventName, source Source, userID, teamID, goalID, proofID *int64, props map[string]any) error {
	if err := ValidateProperties(props); err != nil {
		return err
	}
	const sql = `
		INSERT INTO analytics_events (ts, event_name, user_id, team_id, goal_id, proof_id, source, properties)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.pool.Exec(ctx, sql, time.Now(), string(name), userID, teamID, goalID, proofID, string(source), props)
	if err != nil {
		return fmt.Errorf("analytics record: %w", err)
	}
	return nil
}

// NoopRecorder discards all events silently.
type NoopRecorder struct{}

func (NoopRecorder) Record(context.Context, EventName, Source, *int64, *int64, *int64, *int64, map[string]any) error {
	return nil
}
