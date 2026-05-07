package users

import (
	"context"
	"errors"
	"strings"
	"time"
)

const (
	minDisplayNameLength = 2
)

var (
	ErrNotFound      = errors.New("user not found")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrEmailTaken    = errors.New("email already taken")
	ErrInvalidInput  = errors.New("invalid input")
	ErrSessionFailed = errors.New("session creation failed")
	// ErrRefreshTokenInvalid covers both expired and unknown refresh tokens —
	// the API always reports them as 401 so an attacker can't distinguish.
	ErrRefreshTokenInvalid = errors.New("refresh token invalid")
	// ErrRefreshTokenReused fires when a token was already rotated. Triggers
	// the chain-revocation path (treat as compromised credentials).
	ErrRefreshTokenReused = errors.New("refresh token reused")
)

type User struct {
	ID          int64     `json:"id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Session struct {
	UserID    int64
	TokenHash string
	ExpiresAt time.Time
	CreatedAt time.Time
}

// RefreshToken is the rotation chain primitive. Each /v1/auth/refresh issues a
// new row with parent_id pointing at the row it replaces; the old row is
// marked revoked. Re-using a revoked-but-rotated row indicates the cookie was
// stolen — the service revokes the entire chain when it sees this.
type RefreshToken struct {
	ID        int64
	UserID    int64
	TokenHash string
	ParentID  *int64
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}

type RegisterInput struct {
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
}

type LoginInput struct {
	Email string `json:"email"`
}

// RegistrationResult is what Login/Register return: an access cookie value
// (alongside its expiry) plus the refresh cookie value (alongside its
// expiry). The handler turns these into two Set-Cookie headers.
type RegistrationResult struct {
	User             User
	SessionToken     string
	ExpiresAt        time.Time
	RefreshTokenRaw  string
	RefreshExpiresAt time.Time
}

// RefreshResult is what Service.Refresh returns: a fresh access cookie value
// + expiry, and the rotated refresh cookie value + expiry. The handler always
// re-issues both cookies so the client doesn't have to coordinate.
type RefreshResult struct {
	User             User
	SessionToken     string
	ExpiresAt        time.Time
	RefreshTokenRaw  string
	RefreshExpiresAt time.Time
}

func (in RegisterInput) Normalize() RegisterInput {
	return RegisterInput{
		Email:       strings.ToLower(strings.TrimSpace(in.Email)),
		DisplayName: strings.TrimSpace(in.DisplayName),
	}
}

func (in RegisterInput) Validate() error {
	normalized := in.Normalize()

	if normalized.Email == "" || !strings.Contains(normalized.Email, "@") {
		return errors.Join(ErrInvalidInput, errors.New("valid email is required"))
	}
	if len(normalized.DisplayName) < minDisplayNameLength {
		return errors.Join(ErrInvalidInput, errors.New("display_name must be at least 2 characters"))
	}

	return nil
}

type AuthenticatedUserKey struct{}

func WithAuthenticatedUser(ctx context.Context, user User) context.Context {
	return context.WithValue(ctx, AuthenticatedUserKey{}, user)
}

func CurrentUser(ctx context.Context) (User, bool) {
	user, ok := ctx.Value(AuthenticatedUserKey{}).(User)
	return user, ok
}
