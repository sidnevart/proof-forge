package httpx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

type readyChecker struct {
	err error
}

func (r readyChecker) Ready(context.Context) error {
	return r.err
}

func TestRouterHealthAndReady(t *testing.T) {
	router := NewRouter(nil, readyChecker{}, "http://localhost:3000")

	for _, path := range []string{"/healthz", "/readyz"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 for %s, got %d", path, rec.Code)
		}
	}
}

func TestRouterCORSPreflight(t *testing.T) {
	router := NewRouter(nil, readyChecker{}, "http://localhost:3000")
	req := httptest.NewRequest(http.MethodOptions, "/readyz", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("expected allow origin header, got %q", got)
	}
}
