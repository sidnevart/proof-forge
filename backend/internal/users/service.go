package users

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

type TokenGenerator func() (string, error)
type Clock func() time.Time

type Service struct {
	users         UserRepository
	sessions      SessionRepository
	refresh       RefreshTokenRepository
	sessionTTL    time.Duration
	refreshTTL    time.Duration
	tokenGenerate TokenGenerator
	clock         Clock
}

// NewService wires the users service. `refresh` may be nil — in that case the
// service runs in legacy single-token mode (Login/Register only issue an
// access cookie) which is what existing tests exercise. Production code paths
// always pass a real RefreshTokenRepository.
func NewService(usersRepo UserRepository, sessionsRepo SessionRepository, sessionTTL time.Duration, opts ...ServiceOption) *Service {
	s := &Service{
		users:         usersRepo,
		sessions:      sessionsRepo,
		sessionTTL:    sessionTTL,
		refreshTTL:    30 * 24 * time.Hour,
		tokenGenerate: randomToken,
		clock:         time.Now,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(s)
		}
	}
	return s
}

// ServiceOption tweaks the service after construction. Used to inject the
// refresh-token repo and override the refresh TTL without growing
// NewService's positional argument list.
type ServiceOption func(*Service)

// WithRefreshRepo enables the refresh-token rotation flow. Must be set in
// production wiring — without it Login/Register will leave RefreshTokenRaw
// empty in their result and the handler skips emitting the refresh cookie.
func WithRefreshRepo(repo RefreshTokenRepository) ServiceOption {
	return func(s *Service) {
		if repo != nil {
			s.refresh = repo
		}
	}
}

// WithRefreshTTL pins the refresh-token lifetime. Defaults to 30 days.
func WithRefreshTTL(d time.Duration) ServiceOption {
	return func(s *Service) {
		if d > 0 {
			s.refreshTTL = d
		}
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

	return s.issueTokens(ctx, user)
}

func (s *Service) Login(ctx context.Context, input LoginInput) (RegistrationResult, error) {
	email := strings.ToLower(strings.TrimSpace(input.Email))
	if email == "" || !strings.Contains(email, "@") {
		return RegistrationResult{}, errors.Join(ErrInvalidInput, errors.New("valid email is required"))
	}

	user, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return RegistrationResult{}, ErrNotFound
		}
		return RegistrationResult{}, fmt.Errorf("find user by email: %w", err)
	}

	return s.issueTokens(ctx, user)
}

// issueTokens is the common tail of Login and Register. It mints an access
// session row and (if the refresh repo is wired) a refresh-token row, and
// returns the raw values ready for the handler to set as cookies.
func (s *Service) issueTokens(ctx context.Context, user User) (RegistrationResult, error) {
	rawAccess, err := s.tokenGenerate()
	if err != nil {
		return RegistrationResult{}, fmt.Errorf("generate session token: %w", err)
	}

	now := s.clock().UTC()
	accessExpires := now.Add(s.sessionTTL)
	if err := s.sessions.CreateSession(ctx, Session{
		UserID:    user.ID,
		TokenHash: hashToken(rawAccess),
		ExpiresAt: accessExpires,
		CreatedAt: now,
	}); err != nil {
		return RegistrationResult{}, fmt.Errorf("%w: %v", ErrSessionFailed, err)
	}

	result := RegistrationResult{
		User:         user,
		SessionToken: rawAccess,
		ExpiresAt:    accessExpires,
	}

	// Refresh repo is optional — if it's not wired (legacy tests, telegram-only
	// flows) we just return the access cookie. The handler then skips Set-Cookie
	// for the refresh cookie.
	if s.refresh != nil {
		rawRefresh, err := s.tokenGenerate()
		if err != nil {
			return RegistrationResult{}, fmt.Errorf("generate refresh token: %w", err)
		}
		refreshExpires := now.Add(s.refreshTTL)
		if _, err := s.refresh.CreateRefreshToken(ctx, RefreshToken{
			UserID:    user.ID,
			TokenHash: hashToken(rawRefresh),
			ExpiresAt: refreshExpires,
		}); err != nil {
			return RegistrationResult{}, fmt.Errorf("%w: %v", ErrSessionFailed, err)
		}
		result.RefreshTokenRaw = rawRefresh
		result.RefreshExpiresAt = refreshExpires
	}

	return result, nil
}

// Refresh validates the supplied refresh token, rotates it (the old row is
// marked revoked, a fresh row is created with parent_id pointing at it), and
// issues a new access session. If the supplied token was already revoked the
// service treats it as a stolen-cookie signal and revokes the entire chain.
func (s *Service) Refresh(ctx context.Context, rawRefresh string) (RefreshResult, error) {
	if rawRefresh == "" || s.refresh == nil {
		return RefreshResult{}, ErrRefreshTokenInvalid
	}

	hash := hashToken(rawRefresh)
	stored, err := s.refresh.FindByTokenHash(ctx, hash)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return RefreshResult{}, ErrRefreshTokenInvalid
		}
		return RefreshResult{}, fmt.Errorf("find refresh token: %w", err)
	}

	now := s.clock().UTC()

	// Reuse detection: a row that was already revoked but is being presented
	// again means somebody is replaying an old cookie. Revoke everything in
	// the chain so neither attacker nor victim can keep using it.
	if stored.RevokedAt != nil {
		if err := s.refresh.RevokeChain(ctx, stored.ID); err != nil {
			return RefreshResult{}, fmt.Errorf("revoke chain on reuse: %w", err)
		}
		return RefreshResult{}, ErrRefreshTokenReused
	}

	if !stored.ExpiresAt.After(now) {
		return RefreshResult{}, ErrRefreshTokenInvalid
	}

	user, err := s.users.FindByID(ctx, stored.UserID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return RefreshResult{}, ErrRefreshTokenInvalid
		}
		return RefreshResult{}, fmt.Errorf("find user for refresh: %w", err)
	}

	// Mint the new pair before revoking the old one — if creation fails we
	// haven't burned the user's only valid token yet.
	rawAccess, err := s.tokenGenerate()
	if err != nil {
		return RefreshResult{}, fmt.Errorf("generate access token: %w", err)
	}
	accessExpires := now.Add(s.sessionTTL)
	if err := s.sessions.CreateSession(ctx, Session{
		UserID:    user.ID,
		TokenHash: hashToken(rawAccess),
		ExpiresAt: accessExpires,
		CreatedAt: now,
	}); err != nil {
		return RefreshResult{}, fmt.Errorf("%w: %v", ErrSessionFailed, err)
	}

	rawRefreshNew, err := s.tokenGenerate()
	if err != nil {
		return RefreshResult{}, fmt.Errorf("generate refresh token: %w", err)
	}
	refreshExpires := now.Add(s.refreshTTL)
	parent := stored.ID
	if _, err := s.refresh.CreateRefreshToken(ctx, RefreshToken{
		UserID:    user.ID,
		TokenHash: hashToken(rawRefreshNew),
		ParentID:  &parent,
		ExpiresAt: refreshExpires,
	}); err != nil {
		return RefreshResult{}, fmt.Errorf("%w: %v", ErrSessionFailed, err)
	}

	// Now that the new pair is in place, revoke the old refresh row so a replay
	// of the cookie we just rotated triggers reuse detection.
	if err := s.refresh.Revoke(ctx, stored.ID); err != nil {
		return RefreshResult{}, fmt.Errorf("revoke rotated refresh: %w", err)
	}

	return RefreshResult{
		User:             user,
		SessionToken:     rawAccess,
		ExpiresAt:        accessExpires,
		RefreshTokenRaw:  rawRefreshNew,
		RefreshExpiresAt: refreshExpires,
	}, nil
}

// Logout removes the access-cookie row and revokes the supplied refresh token
// so neither cookie can be replayed for its remaining lifetime. Both
// arguments are optional — if either is empty we just skip that part. Errors
// are logged-and-swallowed at the caller; a logout should never fail in a way
// the user has to react to.
func (s *Service) Logout(ctx context.Context, rawAccess, rawRefresh string) error {
	if rawAccess != "" {
		if err := s.sessions.DeleteSessionByTokenHash(ctx, hashToken(rawAccess)); err != nil {
			return fmt.Errorf("delete access session on logout: %w", err)
		}
	}
	if rawRefresh != "" && s.refresh != nil {
		stored, err := s.refresh.FindByTokenHash(ctx, hashToken(rawRefresh))
		if err == nil && stored.RevokedAt == nil {
			if err := s.refresh.Revoke(ctx, stored.ID); err != nil {
				return fmt.Errorf("revoke refresh on logout: %w", err)
			}
		}
	}
	return nil
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
