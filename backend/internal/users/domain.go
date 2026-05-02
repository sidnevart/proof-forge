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

type RegisterInput struct {
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
}

type LoginInput struct {
	Email string `json:"email"`
}

type RegistrationResult struct {
	User         User
	SessionToken string
	ExpiresAt    time.Time
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
