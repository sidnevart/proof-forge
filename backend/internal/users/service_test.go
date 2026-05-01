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
}

func (s sessionRepoStub) CreateSession(ctx context.Context, session Session) error {
	return s.createSession(ctx, session)
}

func (s sessionRepoStub) FindUserBySessionTokenHash(ctx context.Context, token string) (User, error) {
	return s.findUser(ctx, token)
}

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
