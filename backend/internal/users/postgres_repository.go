package users

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) FindByEmail(ctx context.Context, email string) (User, error) {
	const query = `
		SELECT id, email, display_name, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	var user User
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.DisplayName,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, fmt.Errorf("query user by email: %w", err)
	}

	return user, nil
}

func (r *PostgresRepository) FindByID(ctx context.Context, id int64) (User, error) {
	const query = `
		SELECT id, email, display_name, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var user User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.DisplayName,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, fmt.Errorf("query user by id: %w", err)
	}

	return user, nil
}

func (r *PostgresRepository) Create(ctx context.Context, input RegisterInput) (User, error) {
	const query = `
		INSERT INTO users (email, display_name)
		VALUES ($1, $2)
		RETURNING id, email, display_name, created_at, updated_at
	`

	var user User
	err := r.pool.QueryRow(ctx, query, input.Email, input.DisplayName).Scan(
		&user.ID,
		&user.Email,
		&user.DisplayName,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return User{}, fmt.Errorf("insert user: %w", err)
	}

	return user, nil
}

func (r *PostgresRepository) CreateSession(ctx context.Context, session Session) error {
	const query = `
		INSERT INTO user_sessions (user_id, token_hash, expires_at, created_at, last_seen_at)
		VALUES ($1, $2, $3, $4, $4)
	`

	if _, err := r.pool.Exec(ctx, query, session.UserID, session.TokenHash, session.ExpiresAt, session.CreatedAt); err != nil {
		return fmt.Errorf("insert user session: %w", err)
	}

	return nil
}

func (r *PostgresRepository) FindUserBySessionTokenHash(ctx context.Context, tokenHash string) (User, error) {
	const query = `
		SELECT u.id, u.email, u.display_name, u.created_at, u.updated_at
		FROM user_sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = $1
		  AND s.expires_at > NOW()
	`

	var user User
	err := r.pool.QueryRow(ctx, query, tokenHash).Scan(
		&user.ID,
		&user.Email,
		&user.DisplayName,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, fmt.Errorf("query user by session token: %w", err)
	}

	return user, nil
}
