package users

import "context"

type UserRepository interface {
	FindByEmail(context.Context, string) (User, error)
	FindByID(context.Context, int64) (User, error)
	Create(context.Context, RegisterInput) (User, error)
	SetPlatformAdmin(ctx context.Context, userID int64, isAdmin bool) error
}

type SessionRepository interface {
	CreateSession(context.Context, Session) error
	FindUserBySessionTokenHash(context.Context, string) (User, error)
	// DeleteSessionByTokenHash removes an access session row. Called on
	// logout so a leaked access cookie can't be replayed for its remaining
	// lifetime.
	DeleteSessionByTokenHash(context.Context, string) error
}

// RefreshTokenRepository persists and retrieves refresh tokens. The service
// uses these primitives to implement rotation and reuse detection: every
// refresh call atomically marks the old row revoked and creates a new row
// with parent_id pointing at it.
//
// All revoke operations use Postgres NOW() internally so we don't need to
// trust the API server's wall clock to match the database's.
type RefreshTokenRepository interface {
	// CreateRefreshToken writes a new refresh token row and returns the
	// assigned id. Named explicitly (not `Create`) because the user repo on
	// the same struct already owns Create(...) for users.
	CreateRefreshToken(ctx context.Context, token RefreshToken) (int64, error)
	// FindByTokenHash looks up a refresh token without filtering on status —
	// the service inspects revoked_at and expires_at itself so it can
	// distinguish «expired», «unknown», and «reused after rotation».
	FindByTokenHash(ctx context.Context, tokenHash string) (RefreshToken, error)
	// Revoke marks a single row revoked at NOW(). Idempotent.
	Revoke(ctx context.Context, id int64) error
	// RevokeChain revokes every row reachable by walking parent_id from the
	// given id (both ancestors and descendants), as well as the row itself.
	// Used when reuse is detected: we assume the entire chain is compromised.
	RevokeChain(ctx context.Context, id int64) error
}
