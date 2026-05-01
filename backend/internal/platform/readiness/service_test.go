package readiness

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type stubChecker struct {
	err error
}

func (s stubChecker) Ready(context.Context) error {
	return s.err
}

type stubPinger struct {
	err error
}

func (s stubPinger) Ping(context.Context) error {
	return s.err
}

func TestServiceReadySuccess(t *testing.T) {
	service := NewService(stubPinger{}, time.Second)
	if err := service.Ready(context.Background()); err != nil {
		t.Fatalf("Ready() error = %v", err)
	}
}

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	HealthHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestReadyHandlerUnavailable(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()

	ReadyHandler(stubChecker{err: errors.New("db down")}, nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}
