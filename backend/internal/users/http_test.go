package users

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

func TestRegisterRouteSetsCookie(t *testing.T) {
	service := NewService(
		userRepoStub{
			findByEmail: func(_ context.Context, _ string) (User, error) { return User{}, ErrNotFound },
			create: func(_ context.Context, input RegisterInput) (User, error) {
				return User{ID: 1, Email: input.Email, DisplayName: input.DisplayName}, nil
			},
		},
		sessionRepoStub{
			createSession: func(_ context.Context, _ Session) error { return nil },
			findUser:      func(_ context.Context, _ string) (User, error) { return User{}, ErrNotFound },
		},
		24*time.Hour,
	)
	service.tokenGenerate = func() (string, error) { return "cookie-token", nil }
	service.clock = func() time.Time { return time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC) }

	handler := NewHandler(nil, service, "pf_session", false)
	router := chi.NewRouter()
	handler.RegisterPublicRoutes(router)

	body, _ := json.Marshal(RegisterInput{
		Email:       "user@example.com",
		DisplayName: "User",
	})

	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rec.Code)
	}

	found := false
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == "pf_session" && cookie.Value == "cookie-token" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected session cookie to be set")
	}
}
