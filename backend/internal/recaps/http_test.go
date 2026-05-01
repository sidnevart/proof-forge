package recaps

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/sidnevart/proof-forge/backend/internal/users"
)

func newTestRouter(svc *Service) *chi.Mux {
	h := NewHandler(nil, svc)
	r := chi.NewRouter()
	h.RegisterRoutes(r)
	return r
}

func withActor(r *http.Request, u users.User) *http.Request {
	return r.WithContext(users.WithAuthenticatedUser(r.Context(), u))
}

func TestListRecaps_200(t *testing.T) {
	recap := WeeklyRecap{ID: 1, GoalID: 10, OwnerUserID: 1, Status: StatusDone, SummaryText: "Good week", CreatedAt: time.Now()}
	svc := newSvc(repoStub{
		listRecapsByGoal: func(_ context.Context, _, _ int64) ([]WeeklyRecap, error) {
			return []WeeklyRecap{recap}, nil
		},
	}, aiStub{})

	req := httptest.NewRequest(http.MethodGet, "/goals/10/recaps", nil)
	req = withActor(req, ownerUser())
	rec := httptest.NewRecorder()
	newTestRouter(svc).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	_ = json.NewDecoder(rec.Body).Decode(&body)
	if body["recaps"] == nil {
		t.Error("expected recaps key in response")
	}
}

func TestListRecaps_401Unauthenticated(t *testing.T) {
	svc := newSvc(repoStub{}, aiStub{})
	req := httptest.NewRequest(http.MethodGet, "/goals/10/recaps", nil)
	rec := httptest.NewRecorder()
	newTestRouter(svc).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestGetRecap_200(t *testing.T) {
	recap := WeeklyRecap{ID: 7, GoalID: 10, OwnerUserID: 1, Status: StatusDone, SummaryText: "Good week", CreatedAt: time.Now()}
	svc := newSvc(repoStub{
		getRecap: func(_ context.Context, _ int64) (WeeklyRecap, int64, error) {
			return recap, 2, nil
		},
	}, aiStub{})

	req := httptest.NewRequest(http.MethodGet, "/recaps/7", nil)
	req = withActor(req, ownerUser())
	rec := httptest.NewRecorder()
	newTestRouter(svc).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetRecap_404NotFound(t *testing.T) {
	svc := newSvc(repoStub{}, aiStub{})
	req := httptest.NewRequest(http.MethodGet, "/recaps/999", nil)
	req = withActor(req, ownerUser())
	rec := httptest.NewRecorder()
	newTestRouter(svc).ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestGetRecap_403Forbidden(t *testing.T) {
	recap := WeeklyRecap{ID: 7, GoalID: 10, OwnerUserID: 1, Status: StatusDone}
	svc := newSvc(repoStub{
		getRecap: func(_ context.Context, _ int64) (WeeklyRecap, int64, error) {
			return recap, 2, nil // owner=1, buddy=2
		},
	}, aiStub{})

	req := httptest.NewRequest(http.MethodGet, "/recaps/7", nil)
	req = withActor(req, otherUser()) // other=3
	rec := httptest.NewRecorder()
	newTestRouter(svc).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetRecap_401Unauthenticated(t *testing.T) {
	svc := newSvc(repoStub{}, aiStub{})
	req := httptest.NewRequest(http.MethodGet, "/recaps/7", nil)
	rec := httptest.NewRecorder()
	newTestRouter(svc).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}
