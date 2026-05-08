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

// PilotEvent is used for pilot-instrumentation tracking with workspace context.
type PilotEvent struct {
	UserID      int64
	WorkspaceID *int64
	TeamID      *int64
	GoalID      *int64
	Name        EventName
	Props       map[string]any
}

// Tracker is a fire-and-forget analytics emitter safe to call from handlers.
// Errors are silently swallowed so analytics never blocks business logic.
type Tracker interface {
	TrackAsync(ev PilotEvent)
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
	const q = `
		INSERT INTO analytics_events (ts, event_name, user_id, team_id, goal_id, proof_id, source, properties)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.pool.Exec(ctx, q, time.Now(), string(name), userID, teamID, goalID, proofID, string(source), props)
	if err != nil {
		return fmt.Errorf("analytics record: %w", err)
	}
	return nil
}

// TrackAsync writes a pilot event in a background goroutine.
// Never returns an error — analytics failures must not block callers.
func (r *PostgresRecorder) TrackAsync(ev PilotEvent) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = r.pool.Exec(ctx, `
			INSERT INTO analytics_events (ts, event_name, user_id, team_id, goal_id, workspace_id, source, properties)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`,
			time.Now(), string(ev.Name), ev.UserID, ev.TeamID, ev.GoalID, ev.WorkspaceID,
			string(SourceSystem), ev.Props,
		)
	}()
}

// NoopRecorder discards all events silently.
type NoopRecorder struct{}

func (NoopRecorder) Record(context.Context, EventName, Source, *int64, *int64, *int64, *int64, map[string]any) error {
	return nil
}

func (NoopRecorder) TrackAsync(PilotEvent) {}
