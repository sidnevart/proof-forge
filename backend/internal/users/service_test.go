package users

import (
	"context"
	"errors"
	"testing"
	"time"
)

type userRepoStub struct {
	findByEmail func(context.Context, string) (User, error)
	create      func(context.Context, RegisterInput) (User, error)
}

func (s userRepoStub) FindByEmail(ctx context.Context, email string) (User, error) {
	return s.findByEmail(ctx, email)
}

func (s userRepoStub) FindByID(context.Context, int64) (User, error) {
	return User{}, ErrNotFound
}

func (s userRepoStub) Create(ctx context.Context, input RegisterInput) (User, error) {
	return s.create(ctx, input)
}

type sessionRepoStub struct {
	createSession func(context.Context, Session) error
	findUser      func(context.Context, string) (User, error)
	deleteSession func(context.Context, string) error
}

func (s sessionRepoStub) CreateSession(ctx context.Context, session Session) error {
	return s.createSession(ctx, session)
}

func (s sessionRepoStub) FindUserBySessionTokenHash(ctx context.Context, token string) (User, error) {
	return s.findUser(ctx, token)
}

func (s sessionRepoStub) DeleteSessionByTokenHash(ctx context.Context, token string) error {
	if s.deleteSession == nil {
		return nil
	}
	return s.deleteSession(ctx, token)
}

// refreshRepoStub is an in-memory RefreshTokenRepository for the rotation
// tests. It uses an int counter to mint ids, mirroring the production schema
// (BIGSERIAL) closely enough that parent_id chains work.
type refreshRepoStub struct {
	rows   map[int64]*RefreshToken
	byHash map[string]int64
	nextID int64
	now    func() time.Time
}

func newRefreshRepoStub(now func() time.Time) *refreshRepoStub {
	return &refreshRepoStub{
		rows:   make(map[int64]*RefreshToken),
		byHash: make(map[string]int64),
		nextID: 0,
		now:    now,
	}
}

func (r *refreshRepoStub) CreateRefreshToken(_ context.Context, t RefreshToken) (int64, error) {
	r.nextID++
	id := r.nextID
	t.ID = id
	t.CreatedAt = r.now()
	stored := t
	r.rows[id] = &stored
	r.byHash[t.TokenHash] = id
	return id, nil
}

func (r *refreshRepoStub) FindByTokenHash(_ context.Context, hash string) (RefreshToken, error) {
	id, ok := r.byHash[hash]
	if !ok {
		return RefreshToken{}, ErrNotFound
	}
	return *r.rows[id], nil
}

func (r *refreshRepoStub) Revoke(_ context.Context, id int64) error {
	row, ok := r.rows[id]
	if !ok || row.RevokedAt != nil {
		return nil
	}
	now := r.now()
	row.RevokedAt = &now
	return nil
}

func (r *refreshRepoStub) RevokeChain(_ context.Context, id int64) error {
	visited := make(map[int64]bool)
	queue := []int64{id}
	for len(queue) > 0 {
		head := queue[0]
		queue = queue[1:]
		if visited[head] {
			continue
		}
		visited[head] = true
		row, ok := r.rows[head]
		if !ok {
			continue
		}
		if row.RevokedAt == nil {
			now := r.now()
			row.RevokedAt = &now
		}
		// Walk ancestors.
		if row.ParentID != nil {
			queue = append(queue, *row.ParentID)
		}
		// Walk descendants.
		for _, candidate := range r.rows {
			if candidate.ParentID != nil && *candidate.ParentID == head {
				queue = append(queue, candidate.ID)
			}
		}
	}
	return nil
}

func TestServiceRefreshRotatesAndRevokes(t *testing.T) {
	now := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }
	tokens := []string{"access-1", "refresh-1", "access-2", "refresh-2"}
	idx := 0

	refreshRepo := newRefreshRepoStub(clock)
	service := NewService(
		userRepoStub{
			findByEmail: func(context.Context, string) (User, error) { return User{}, ErrNotFound },
			create: func(_ context.Context, in RegisterInput) (User, error) {
				return User{ID: 11, Email: in.Email, DisplayName: in.DisplayName}, nil
			},
		},
		sessionRepoStub{
			createSession: func(context.Context, Session) error { return nil },
			findUser:      func(context.Context, string) (User, error) { return User{}, ErrNotFound },
		},
		15*time.Minute,
		WithRefreshRepo(refreshRepo),
		WithRefreshTTL(30*24*time.Hour),
	)
	service.tokenGenerate = func() (string, error) {
		v := tokens[idx]
		idx++
		return v, nil
	}
	service.clock = clock
	// Override the user repo's FindByID since the stub above returns NotFound.
	service.users = userRepoStub{
		findByEmail: func(context.Context, string) (User, error) { return User{}, ErrNotFound },
		create:      func(_ context.Context, in RegisterInput) (User, error) { return User{ID: 11}, nil },
	}

	reg, err := service.Register(context.Background(), RegisterInput{Email: "a@b.com", DisplayName: "Tester"})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if reg.RefreshTokenRaw != "refresh-1" {
		t.Fatalf("expected refresh-1, got %q", reg.RefreshTokenRaw)
	}

	// We need FindByID to succeed during refresh — patch the userRepo.
	service.users = userByIDStub{user: User{ID: 11, Email: "a@b.com"}}

	res, err := service.Refresh(context.Background(), "refresh-1")
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if res.SessionToken != "access-2" || res.RefreshTokenRaw != "refresh-2" {
		t.Fatalf("expected access-2/refresh-2, got %q/%q", res.SessionToken, res.RefreshTokenRaw)
	}

	// Old refresh row should now be revoked.
	old, err := refreshRepo.FindByTokenHash(context.Background(), hashToken("refresh-1"))
	if err != nil {
		t.Fatalf("FindByTokenHash: %v", err)
	}
	if old.RevokedAt == nil {
		t.Fatalf("expected refresh-1 revoked after rotation")
	}

	// Replaying refresh-1 must trip reuse detection and revoke the chain.
	if _, err := service.Refresh(context.Background(), "refresh-1"); !errors.Is(err, ErrRefreshTokenReused) {
		t.Fatalf("expected ErrRefreshTokenReused on replay, got %v", err)
	}
	newRow, err := refreshRepo.FindByTokenHash(context.Background(), hashToken("refresh-2"))
	if err != nil {
		t.Fatalf("FindByTokenHash refresh-2: %v", err)
	}
	if newRow.RevokedAt == nil {
		t.Fatalf("expected refresh-2 revoked after reuse detected on its parent")
	}
}

func TestServiceRefreshRejectsExpired(t *testing.T) {
	now := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }
	refreshRepo := newRefreshRepoStub(clock)

	// Pre-seed an expired refresh row directly.
	expired := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC) // a month before "now"
	if _, err := refreshRepo.CreateRefreshToken(context.Background(), RefreshToken{
		UserID:    11,
		TokenHash: hashToken("old"),
		ExpiresAt: expired,
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	service := NewService(
		userByIDStub{user: User{ID: 11}},
		sessionRepoStub{
			createSession: func(context.Context, Session) error { return nil },
			findUser:      func(context.Context, string) (User, error) { return User{}, ErrNotFound },
		},
		15*time.Minute,
		WithRefreshRepo(refreshRepo),
	)
	service.clock = clock

	if _, err := service.Refresh(context.Background(), "old"); !errors.Is(err, ErrRefreshTokenInvalid) {
		t.Fatalf("expected ErrRefreshTokenInvalid for expired refresh, got %v", err)
	}
}

// userByIDStub satisfies UserRepository for tests that need FindByID to
// succeed without exercising FindByEmail / Create.
type userByIDStub struct{ user User }

func (s userByIDStub) FindByEmail(context.Context, string) (User, error) { return User{}, ErrNotFound }
func (s userByIDStub) FindByID(_ context.Context, id int64) (User, error) {
	if id == s.user.ID {
		return s.user, nil
	}
	return User{}, ErrNotFound
}
func (s userByIDStub) Create(context.Context, RegisterInput) (User, error) { return User{}, nil }

func TestServiceRegister(t *testing.T) {
	service := NewService(
		userRepoStub{
			findByEmail: func(context.Context, string) (User, error) { return User{}, ErrNotFound },
			create: func(_ context.Context, input RegisterInput) (User, error) {
				return User{ID: 7, Email: input.Email, DisplayName: input.DisplayName}, nil
			},
		},
		sessionRepoStub{
			createSession: func(context.Context, Session) error { return nil },
			findUser:      func(context.Context, string) (User, error) { return User{}, ErrNotFound },
		},
		24*time.Hour,
	)
	service.tokenGenerate = func() (string, error) { return "session-token", nil }
	service.clock = func() time.Time { return time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC) }

	result, err := service.Register(context.Background(), RegisterInput{
		Email:       " User@example.com ",
		DisplayName: "  Artem ",
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if result.User.Email != "user@example.com" {
		t.Fatalf("expected normalized email, got %q", result.User.Email)
	}
	if result.User.DisplayName != "Artem" {
		t.Fatalf("expected trimmed display name, got %q", result.User.DisplayName)
	}
	if result.SessionToken != "session-token" {
		t.Fatalf("expected session token to be returned")
	}
}

func TestServiceRegisterRejectsDuplicateEmail(t *testing.T) {
	service := NewService(
		userRepoStub{
			findByEmail: func(context.Context, string) (User, error) { return User{ID: 1}, nil },
			create:      func(context.Context, RegisterInput) (User, error) { return User{}, nil },
		},
		sessionRepoStub{
			createSession: func(context.Context, Session) error { return nil },
			findUser:      func(context.Context, string) (User, error) { return User{}, ErrNotFound },
		},
		24*time.Hour,
	)

	_, err := service.Register(context.Background(), RegisterInput{
		Email:       "user@example.com",
		DisplayName: "Artem",
	})
	if !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("expected ErrEmailTaken, got %v", err)
	}
}

func TestServiceAuthenticateRejectsUnknownToken(t *testing.T) {
	service := NewService(
		userRepoStub{
			findByEmail: func(context.Context, string) (User, error) { return User{}, ErrNotFound },
			create:      func(context.Context, RegisterInput) (User, error) { return User{}, nil },
		},
		sessionRepoStub{
			createSession: func(context.Context, Session) error { return nil },
			findUser:      func(context.Context, string) (User, error) { return User{}, ErrNotFound },
		},
		24*time.Hour,
	)

	_, err := service.Authenticate(context.Background(), "missing")
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}
