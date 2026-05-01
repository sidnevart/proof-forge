package users

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

type TokenGenerator func() (string, error)
type Clock func() time.Time

type Service struct {
	users         UserRepository
	sessions      SessionRepository
	sessionTTL    time.Duration
	tokenGenerate TokenGenerator
	clock         Clock
}

func NewService(usersRepo UserRepository, sessionsRepo SessionRepository, sessionTTL time.Duration) *Service {
	return &Service{
		users:         usersRepo,
		sessions:      sessionsRepo,
		sessionTTL:    sessionTTL,
		tokenGenerate: randomToken,
		clock:         time.Now,
	}
}

func (s *Service) Register(ctx context.Context, input RegisterInput) (RegistrationResult, error) {
	if err := input.Validate(); err != nil {
		return RegistrationResult{}, err
	}

	input = input.Normalize()
	existing, err := s.users.FindByEmail(ctx, input.Email)
	switch {
	case err == nil && existing.ID > 0:
		return RegistrationResult{}, ErrEmailTaken
	case err != nil && !errors.Is(err, ErrNotFound):
		return RegistrationResult{}, fmt.Errorf("find user by email: %w", err)
	}

	user, err := s.users.Create(ctx, input)
	if err != nil {
		return RegistrationResult{}, fmt.Errorf("create user: %w", err)
	}

	rawToken, err := s.tokenGenerate()
	if err != nil {
		return RegistrationResult{}, fmt.Errorf("generate session token: %w", err)
	}

	now := s.clock().UTC()
	expiresAt := now.Add(s.sessionTTL)
	session := Session{
		UserID:    user.ID,
		TokenHash: hashToken(rawToken),
		ExpiresAt: expiresAt,
		CreatedAt: now,
	}
	if err := s.sessions.CreateSession(ctx, session); err != nil {
		return RegistrationResult{}, fmt.Errorf("%w: %v", ErrSessionFailed, err)
	}

	return RegistrationResult{
		User:         user,
		SessionToken: rawToken,
		ExpiresAt:    expiresAt,
	}, nil
}

func (s *Service) Authenticate(ctx context.Context, sessionToken string) (User, error) {
	if sessionToken == "" {
		return User{}, ErrUnauthorized
	}

	user, err := s.sessions.FindUserBySessionTokenHash(ctx, hashToken(sessionToken))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return User{}, ErrUnauthorized
		}
		return User{}, fmt.Errorf("find user by session token: %w", err)
	}

	return user, nil
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func randomToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
