package personalization

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// CircuitBreakerStore persists per-feature breaker state.
type CircuitBreakerStore interface {
	GetState(ctx context.Context, feature Feature) (CircuitState, *time.Time, error)
	SetState(ctx context.Context, feature Feature, state CircuitState, until *time.Time, reason string) error
}

// PostgresCircuitBreakerStore is the production implementation.
type PostgresCircuitBreakerStore struct {
	pool *pgxpool.Pool
}

func NewPostgresCircuitBreakerStore(pool *pgxpool.Pool) *PostgresCircuitBreakerStore {
	return &PostgresCircuitBreakerStore{pool: pool}
}

func (s *PostgresCircuitBreakerStore) GetState(ctx context.Context, feature Feature) (CircuitState, *time.Time, error) {
	const sql = `SELECT state, opened_until FROM ai_circuit_breakers WHERE feature = $1`
	var state string
	var until *time.Time
	err := s.pool.QueryRow(ctx, sql, string(feature)).Scan(&state, &until)
	if err != nil {
		// No row means closed.
		return CircuitClosed, nil, nil
	}
	return CircuitState(state), until, nil
}

func (s *PostgresCircuitBreakerStore) SetState(ctx context.Context, feature Feature, state CircuitState, until *time.Time, reason string) error {
	const sql = `
		INSERT INTO ai_circuit_breakers (feature, state, opened_until, reason, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (feature)
		DO UPDATE SET state = EXCLUDED.state,
					  opened_until = EXCLUDED.opened_until,
					  reason = EXCLUDED.reason,
					  updated_at = EXCLUDED.updated_at
	`
	_, err := s.pool.Exec(ctx, sql, string(feature), string(state), until, reason, time.Now())
	return err
}

// CheckCircuit returns nil if the feature may proceed, otherwise ErrCircuitOpen.
func CheckCircuit(ctx context.Context, store CircuitBreakerStore, feature Feature, now time.Time) error {
	state, until, err := store.GetState(ctx, feature)
	if err != nil {
		return fmt.Errorf("circuit breaker get: %w", err)
	}
	if state == CircuitClosed {
		return nil
	}
	if until != nil && now.After(*until) {
		_ = store.SetState(ctx, feature, CircuitClosed, nil, "auto-recovered")
		return nil
	}
	return ErrCircuitOpen
}
