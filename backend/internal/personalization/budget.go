package personalization

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const DefaultDailyBudgetTokens = 500000

// BudgetStore tracks daily token consumption.
type BudgetStore interface {
	GetOrCreate(ctx context.Context, date time.Time) (*BudgetRow, error)
	Increment(ctx context.Context, date time.Time, tokensIn, tokensOut int64, fallback bool) error
}

// BudgetRow is a single day's budget aggregate.
type BudgetRow struct {
	BudgetDate      time.Time
	TokensUsed      int64
	RequestsCount   int
	FallbackCount   int
	UpdatedAt       time.Time
}

// PostgresBudgetStore is the production implementation.
type PostgresBudgetStore struct {
	pool *pgxpool.Pool
}

func NewPostgresBudgetStore(pool *pgxpool.Pool) *PostgresBudgetStore {
	return &PostgresBudgetStore{pool: pool}
}

func (s *PostgresBudgetStore) GetOrCreate(ctx context.Context, date time.Time) (*BudgetRow, error) {
	const sql = `
		INSERT INTO ai_daily_budget (budget_date, tokens_used, requests_count, fallback_count, updated_at)
		VALUES ($1, 0, 0, 0, $2)
		ON CONFLICT (budget_date)
		DO UPDATE SET updated_at = EXCLUDED.updated_at
		RETURNING budget_date, tokens_used, requests_count, fallback_count, updated_at
	`
	var row BudgetRow
	err := s.pool.QueryRow(ctx, sql, date, time.Now()).Scan(
		&row.BudgetDate, &row.TokensUsed, &row.RequestsCount, &row.FallbackCount, &row.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("budget get or create: %w", err)
	}
	return &row, nil
}

func (s *PostgresBudgetStore) Increment(ctx context.Context, date time.Time, tokensIn, tokensOut int64, fallback bool) error {
	const sql = `
		UPDATE ai_daily_budget
		   SET tokens_used = tokens_used + $2,
		       requests_count = requests_count + 1,
		       fallback_count = fallback_count + $3,
		       updated_at = $4
		 WHERE budget_date = $1
	`
	fb := 0
	if fallback {
		fb = 1
	}
	_, err := s.pool.Exec(ctx, sql, date, tokensIn+tokensOut, fb, time.Now())
	return err
}

// CheckBudget returns nil if the daily budget is not exceeded.
func CheckBudget(ctx context.Context, store BudgetStore, date time.Time, budgetLimit int64) error {
	row, err := store.GetOrCreate(ctx, date)
	if err != nil {
		return fmt.Errorf("budget check: %w", err)
	}
	if row.TokensUsed >= budgetLimit {
		return ErrBudgetExceeded
	}
	return nil
}
