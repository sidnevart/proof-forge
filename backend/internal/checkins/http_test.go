package checkins

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/sidnevart/proof-forge/backend/internal/users"
)

func withActor(r *http.Request, u users.User) *http.Request {
	return r.WithContext(users.WithAuthenticatedUser(r.Context(), u))
}

func newRouter(svc *Service) *chi.Mux {
	h := NewHandler(nil, svc)
	r := chi.NewRouter()
	h.RegisterRoutes(r)
	return r
}

func TestCreateCheckInRoute201(t *testing.T) {
	svc := NewService(repoStub{}, storageStub{})
	router := newRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/goals/10/check-ins", nil)
	req = withActor(req, owner())
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateCheckInRouteUnauthenticated401(t *testing.T) {
	svc := NewService(repoStub{}, storageStub{})
	router := newRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/goals/10/check-ins", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestCreateCheckInRouteGoalNotEligible403(t *testing.T) {
	repo := repoStub{
		createCheckIn: func(_ context.Context, _ CreateCheckInParams) (CheckIn, error) {
			return CheckIn{}, ErrGoalNotEligible
		},
	}
	svc := NewService(repo, storageStub{})
	router := newRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/goals/10/check-ins", nil)
	req = withActor(req, owner())
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetCheckInRoute200(t *testing.T) {
	repo := repoStub{
		getCheckIn: func(_ context.Context, _ int64) (CheckInView, error) {
			return pendingView(1, 2), nil
		},
	}
	svc := NewService(repo, storageStub{})
	router := newRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/check-ins/1", nil)
	req = withActor(req, owner())
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetCheckInRouteUnauthorized403(t *testing.T) {
	repo := repoStub{
		getCheckIn: func(_ context.Context, _ int64) (CheckInView, error) {
			return pendingView(1, 99), nil // buddy is 99, not 2
		},
	}
	svc := NewService(repo, storageStub{})
	router := newRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/check-ins/1", nil)
	req = withActor(req, buddy()) // buddy.ID=2
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestSubmitRoute200(t *testing.T) {
	repo := repoStub{
		getCheckIn: func(_ context.Context, _ int64) (CheckInView, error) {
			return pendingView(1, 2), nil
		},
	}
	svc := NewService(repo, storageStub{})
	router := newRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/check-ins/1/submit", nil)
	req = withActor(req, owner())
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAddTextEvidenceRoute201(t *testing.T) {
	repo := repoStub{
		getCheckIn: func(_ context.Context, _ int64) (CheckInView, error) {
			return pendingView(1, 2), nil
		},
	}
	svc := NewService(repo, storageStub{})
	router := newRouter(svc)

	body, _ := json.Marshal(AddTextInput{Content: "Shipped the feature"})
	req := httptest.NewRequest(http.MethodPost, "/check-ins/1/evidence/text", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withActor(req, owner())
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAddLinkEvidenceRoute201(t *testing.T) {
	repo := repoStub{
		getCheckIn: func(_ context.Context, _ int64) (CheckInView, error) {
			return pendingView(1, 2), nil
		},
	}
	svc := NewService(repo, storageStub{})
	router := newRouter(svc)

	body, _ := json.Marshal(AddLinkInput{URL: "https://github.com/proof"})
	req := httptest.NewRequest(http.MethodPost, "/check-ins/1/evidence/link", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withActor(req, owner())
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAddFileEvidenceRouteMultipart201(t *testing.T) {
	repo := repoStub{
		getCheckIn: func(_ context.Context, _ int64) (CheckInView, error) {
			return pendingView(1, 2), nil
		},
	}
	svc := NewService(repo, storageStub{})
	router := newRouter(svc)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", "screenshot.png")
	fw.Write(bytes.Repeat([]byte("x"), 512))
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/check-ins/1/evidence/file", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	// set Content-Type of the file part via the form
	req = withActor(req, owner())

	// The multipart part Content-Type is set via CreateFormFile which uses
	// application/octet-stream. Simulate image/png via the handler's
	// header-based MIME read path by using a custom request.
	req2 := buildMultipartRequest("/check-ins/1/evidence/file", "image/png", pngData())
	req2 = withActor(req2, owner())

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req2)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAddFileEvidenceUnsupportedMIME415(t *testing.T) {
	repo := repoStub{
		getCheckIn: func(_ context.Context, _ int64) (CheckInView, error) {
			return pendingView(1, 2), nil
		},
	}
	svc := NewService(repo, storageStub{})
	router := newRouter(svc)

	req := buildMultipartRequest("/check-ins/1/evidence/file", "video/mp4", []byte("data"))
	req = withActor(req, owner())
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("expected 415, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestListCheckInsRoute200(t *testing.T) {
	repo := repoStub{
		listCheckIns: func(_ context.Context, _ int64, _ int64) ([]CheckIn, error) {
			return []CheckIn{{ID: 1, Status: StatusDraft, CreatedAt: time.Now()}}, nil
		},
	}
	svc := NewService(repo, storageStub{})
	router := newRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/goals/10/check-ins", nil)
	req = withActor(req, owner())
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestApproveRoute200(t *testing.T) {
	repo := repoStub{
		getCheckIn: func(_ context.Context, _ int64) (CheckInView, error) {
			v := pendingView(1, 2) // owner=1, buddy=2
			v.CheckIn.Status = StatusSubmitted
			return v, nil
		},
	}
	svc := NewService(repo, storageStub{})
	router := newRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/check-ins/1/approve", nil)
	req = withActor(req, buddy()) // buddy.ID=2
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestApproveRouteNotBuddy403(t *testing.T) {
	repo := repoStub{
		getCheckIn: func(_ context.Context, _ int64) (CheckInView, error) {
			v := pendingView(1, 2)
			v.CheckIn.Status = StatusSubmitted
			return v, nil
		},
	}
	svc := NewService(repo, storageStub{})
	router := newRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/check-ins/1/approve", nil)
	req = withActor(req, owner()) // owner.ID=1, not the buddy
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestApproveRouteCannotReview409(t *testing.T) {
	repo := repoStub{
		getCheckIn: func(_ context.Context, _ int64) (CheckInView, error) {
			return pendingView(1, 2), nil // status is draft, not submitted
		},
	}
	svc := NewService(repo, storageStub{})
	router := newRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/check-ins/1/approve", nil)
	req = withActor(req, buddy())
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRejectRoute200(t *testing.T) {
	repo := repoStub{
		getCheckIn: func(_ context.Context, _ int64) (CheckInView, error) {
			v := pendingView(1, 2)
			v.CheckIn.Status = StatusSubmitted
			return v, nil
		},
	}
	svc := NewService(repo, storageStub{})
	router := newRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/check-ins/1/reject", nil)
	req = withActor(req, buddy())
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRequestChangesRoute200(t *testing.T) {
	repo := repoStub{
		getCheckIn: func(_ context.Context, _ int64) (CheckInView, error) {
			v := pendingView(1, 2)
			v.CheckIn.Status = StatusSubmitted
			return v, nil
		},
	}
	svc := NewService(repo, storageStub{})
	router := newRouter(svc)

	body, _ := json.Marshal(ReviewInput{Comment: "needs more detail"})
	req := httptest.NewRequest(http.MethodPost, "/check-ins/1/request-changes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withActor(req, buddy())
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

// pngData returns minimal PNG magic bytes followed by padding.
func pngData() []byte {
	return append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 8)...)
}

func buildMultipartRequest(path string, mimeType string, data []byte) *http.Request {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	h := make(map[string][]string)
	h["Content-Disposition"] = []string{`form-data; name="file"; filename="upload"`}
	h["Content-Type"] = []string{mimeType}
	part, _ := mw.CreatePart(h)
	part.Write(data)
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, path, &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req
}
