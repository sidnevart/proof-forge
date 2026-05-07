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

func (r *PostgresRepository) DeleteSessionByTokenHash(ctx context.Context, tokenHash string) error {
	if _, err := r.pool.Exec(ctx, `DELETE FROM user_sessions WHERE token_hash = $1`, tokenHash); err != nil {
		return fmt.Errorf("delete user session: %w", err)
	}
	return nil
}

// CreateRefreshToken inserts a new row in refresh_tokens. Returns the new id
// so the caller can chain rotations.
func (r *PostgresRepository) CreateRefreshToken(ctx context.Context, token RefreshToken) (int64, error) {
	const query = `
		INSERT INTO refresh_tokens (user_id, token_hash, parent_id, expires_at, created_at)
		VALUES ($1, $2, $3, $4, NOW())
		RETURNING id
	`
	var id int64
	if err := r.pool.QueryRow(ctx, query, token.UserID, token.TokenHash, token.ParentID, token.ExpiresAt).Scan(&id); err != nil {
		return 0, fmt.Errorf("insert refresh token: %w", err)
	}
	return id, nil
}

// FindByTokenHash returns the row exactly as stored — the service layer is
// responsible for inspecting expires_at / revoked_at to decide validity.
func (r *PostgresRepository) FindByTokenHash(ctx context.Context, tokenHash string) (RefreshToken, error) {
	const query = `
		SELECT id, user_id, token_hash, parent_id, expires_at, revoked_at, created_at
		FROM refresh_tokens
		WHERE token_hash = $1
	`
	var t RefreshToken
	err := r.pool.QueryRow(ctx, query, tokenHash).Scan(
		&t.ID,
		&t.UserID,
		&t.TokenHash,
		&t.ParentID,
		&t.ExpiresAt,
		&t.RevokedAt,
		&t.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return RefreshToken{}, ErrNotFound
		}
		return RefreshToken{}, fmt.Errorf("query refresh token: %w", err)
	}
	return t, nil
}

// Revoke marks one refresh-token row as revoked at NOW(). Idempotent: a
// re-revocation just updates revoked_at to the latest timestamp without
// failing.
func (r *PostgresRepository) Revoke(ctx context.Context, id int64) error {
	const query = `UPDATE refresh_tokens SET revoked_at = NOW() WHERE id = $1 AND revoked_at IS NULL`
	if _, err := r.pool.Exec(ctx, query, id); err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	return nil
}

// RevokeChain walks the parent_id linked list in BOTH directions starting from
// `id` and revokes every visited row. We do this in one recursive CTE to avoid
// round-trips: when a reuse is detected we want every co-conspirator token
// revoked atomically.
func (r *PostgresRepository) RevokeChain(ctx context.Context, id int64) error {
	const query = `
		WITH RECURSIVE
		ancestors(id) AS (
			SELECT id FROM refresh_tokens WHERE id = $1
			UNION ALL
			SELECT t.parent_id FROM refresh_tokens t
			JOIN ancestors a ON t.id = a.id
			WHERE t.parent_id IS NOT NULL
		),
		descendants(id) AS (
			SELECT id FROM refresh_tokens WHERE id = $1
			UNION ALL
			SELECT t.id FROM refresh_tokens t
			JOIN descendants d ON t.parent_id = d.id
		)
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE revoked_at IS NULL
		  AND id IN (SELECT id FROM ancestors UNION SELECT id FROM descendants)
	`
	if _, err := r.pool.Exec(ctx, query, id); err != nil {
		return fmt.Errorf("revoke refresh token chain: %w", err)
	}
	return nil
}
