package goals

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/sidnevart/proof-forge/backend/internal/users"
)

func newTestRouter(service *Service) *chi.Mux {
	handler := NewHandler(nil, service)
	router := chi.NewRouter()
	handler.RegisterPublicRoutes(router)
	handler.RegisterRoutes(router)
	return router
}

func TestCreateGoalRoute(t *testing.T) {
	stub := newTestStub()
	stub.createGoal = func(_ context.Context, params CreateGoalParams) (GoalView, error) {
		// Pretend the repository auto-created a circle with id=42 for this goal.
		return GoalView{
			Goal:   Goal{ID: 9, CircleID: 42, Title: params.Title, Status: params.GoalStatus},
			Buddy:  Buddy{ID: 3, Email: params.BuddyEmail, DisplayName: params.BuddyName},
			Pact:   Pact{ID: 11, Status: params.PactStatus},
			Invite: Invite{ID: 13, Status: params.InviteStatus, ExpiresAt: params.InviteExpiresAt},
		}, nil
	}

	service := NewService(stub, noopEmailSender{}, "", nil, 7*24*time.Hour)
	service.tokenGenerate = func() (string, error) { return "invite-token", nil }
	service.circleCodeGenerate = func() (string, error) { return "auto-code", nil }

	router := newTestRouter(service)

	body, _ := json.Marshal(CreateInput{
		Title:      "Ship MVP",
		BuddyName:  "Peer",
		BuddyEmail: "peer@example.com",
	})
	req := httptest.NewRequest(http.MethodPost, "/goals", bytes.NewReader(body))
	req = req.WithContext(users.WithAuthenticatedUser(req.Context(), users.User{
		ID:    1,
		Email: "owner@example.com",
	}))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify the response includes circle_id so the frontend can route the user
	// into their freshly-created circle without an extra round trip.
	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	goalPayload, ok := resp["goal"].(map[string]any)
	if !ok {
		t.Fatalf("missing goal payload, got %#v", resp)
	}
	innerGoal, ok := goalPayload["goal"].(map[string]any)
	if !ok {
		t.Fatalf("missing inner goal payload, got %#v", goalPayload)
	}
	circleIDValue, ok := innerGoal["circle_id"].(float64)
	if !ok {
		t.Fatalf("expected goal.circle_id to be a number, got %#v", innerGoal["circle_id"])
	}
	if circleIDValue != 42 {
		t.Fatalf("expected goal.circle_id=42, got %v", circleIDValue)
	}
}

func TestCreateGoalRouteConflictWhenActiveGoalExists(t *testing.T) {
	stub := newTestStub()
	stub.isCircleMember = func(context.Context, int64, int64) (bool, error) { return true, nil }
	stub.isCircleMemberByEmail = func(context.Context, int64, string) (bool, error) { return true, nil }
	stub.hasActiveGoalInCircle = func(context.Context, int64, int64) (bool, error) { return true, nil }

	service := NewService(stub, noopEmailSender{}, "", nil, 7*24*time.Hour)
	router := newTestRouter(service)

	body, _ := json.Marshal(CreateInput{
		Title:      "Вторая цель",
		BuddyName:  "Peer",
		BuddyEmail: "peer@example.com",
		CircleID:   1,
	})
	req := httptest.NewRequest(http.MethodPost, "/goals", bytes.NewReader(body))
	req = req.WithContext(users.WithAuthenticatedUser(req.Context(), users.User{
		ID:    1,
		Email: "owner@example.com",
	}))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	errPayload, _ := resp["error"].(map[string]any)
	if errPayload["code"] != "active_goal_already_exists" {
		t.Fatalf("expected error code active_goal_already_exists, got %v", errPayload["code"])
	}
}

func TestRefineGoalRouteInvalidInputReturns400(t *testing.T) {
	service := NewService(newTestStub(), noopEmailSender{}, "", nil, 7*24*time.Hour)
	router := newTestRouter(service)

	body, _ := json.Marshal(map[string]any{
		"draft_text": "  ",
	})
	req := httptest.NewRequest(http.MethodPost, "/goals/refine", bytes.NewReader(body))
	req = req.WithContext(users.WithAuthenticatedUser(req.Context(), users.User{
		ID:    1,
		Email: "owner@example.com",
	}))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	errPayload, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error payload, got %#v", resp)
	}
	if errPayload["code"] != "invalid_input" {
		t.Fatalf("expected invalid_input code, got %v", errPayload["code"])
	}
}

func TestGetInviteRouteReturnsPreview(t *testing.T) {
	stub := newTestStub()
	stub.findInvite = func(_ context.Context, _ string) (InviteRecord, error) {
		return InviteRecord{
			InviteID:     10,
			InviteStatus: InviteStatusPending,
			InviteeEmail: "buddy@example.com",
			GoalTitle:    "Ship MVP",
			OwnerName:    "Alice",
			ExpiresAt:    time.Now().Add(7 * 24 * time.Hour),
		}, nil
	}

	service := NewService(stub, noopEmailSender{}, "", nil, 7*24*time.Hour)
	router := newTestRouter(service)

	req := httptest.NewRequest(http.MethodGet, "/invites/validtoken", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	invite, ok := resp["invite"].(map[string]any)
	if !ok {
		t.Fatalf("missing invite key in response")
	}
	if invite["goal_title"] != "Ship MVP" {
		t.Fatalf("expected goal_title 'Ship MVP', got %v", invite["goal_title"])
	}
}

func TestGetInviteRouteNotFoundReturns404(t *testing.T) {
	service := NewService(newTestStub(), noopEmailSender{}, "", nil, 7*24*time.Hour)
	router := newTestRouter(service)

	req := httptest.NewRequest(http.MethodGet, "/invites/badtoken", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rec.Code)
	}
}

func TestAcceptInviteRouteHappyPath(t *testing.T) {
	stub := newTestStub()
	stub.findInvite = func(_ context.Context, _ string) (InviteRecord, error) {
		return InviteRecord{
			InviteID:     10,
			PactID:       20,
			GoalID:       30,
			InviteStatus: InviteStatusPending,
			InviteeEmail: "buddy@example.com",
			ExpiresAt:    time.Now().Add(7 * 24 * time.Hour),
		}, nil
	}

	service := NewService(stub, noopEmailSender{}, "", nil, 7*24*time.Hour)
	router := newTestRouter(service)

	req := httptest.NewRequest(http.MethodPost, "/invites/validtoken/accept", nil)
	req = req.WithContext(users.WithAuthenticatedUser(req.Context(), users.User{
		ID:    2,
		Email: "buddy@example.com",
	}))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAcceptInviteRouteUnauthenticatedReturns401(t *testing.T) {
	service := NewService(newTestStub(), noopEmailSender{}, "", nil, 7*24*time.Hour)
	router := newTestRouter(service)

	req := httptest.NewRequest(http.MethodPost, "/invites/sometoken/accept", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
}

func TestAcceptInviteRouteWrongEmailReturns403(t *testing.T) {
	stub := newTestStub()
	stub.findInvite = func(_ context.Context, _ string) (InviteRecord, error) {
		return InviteRecord{
			InviteID:     10,
			InviteStatus: InviteStatusPending,
			InviteeEmail: "buddy@example.com",
			ExpiresAt:    time.Now().Add(7 * 24 * time.Hour),
		}, nil
	}

	service := NewService(stub, noopEmailSender{}, "", nil, 7*24*time.Hour)
	router := newTestRouter(service)

	req := httptest.NewRequest(http.MethodPost, "/invites/validtoken/accept", nil)
	req = req.WithContext(users.WithAuthenticatedUser(req.Context(), users.User{
		ID:    99,
		Email: "impostor@example.com",
	}))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAcceptInviteRouteExpiredReturns410(t *testing.T) {
	stub := newTestStub()
	stub.findInvite = func(_ context.Context, _ string) (InviteRecord, error) {
		return InviteRecord{
			InviteID:     10,
			InviteStatus: InviteStatusPending,
			InviteeEmail: "buddy@example.com",
			ExpiresAt:    time.Now().Add(-1 * time.Hour),
		}, nil
	}

	service := NewService(stub, noopEmailSender{}, "", nil, 7*24*time.Hour)
	router := newTestRouter(service)

	req := httptest.NewRequest(http.MethodPost, "/invites/expiredtoken/accept", nil)
	req = req.WithContext(users.WithAuthenticatedUser(req.Context(), users.User{
		ID:    2,
		Email: "buddy@example.com",
	}))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusGone {
		t.Fatalf("expected status 410, got %d: %s", rec.Code, rec.Body.String())
	}
}
