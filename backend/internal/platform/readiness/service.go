package readiness

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"
)

type Checker interface {
	Ready(context.Context) error
}

type Pinger interface {
	Ping(context.Context) error
}

type Service struct {
	db      Pinger
	timeout time.Duration
}

func NewService(db Pinger, timeout time.Duration) *Service {
	return &Service{db: db, timeout: timeout}
}

func (s *Service) Ready(ctx context.Context) error {
	if s == nil || s.db == nil {
		return errors.New("readiness checker is not configured")
	}

	checkCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	return s.db.Ping(checkCtx)
}

func HealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

func ReadyHandler(checker Checker, log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := checker.Ready(r.Context()); err != nil {
			if log != nil {
				log.Warn("readiness check failed", "error", err.Error())
			}
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{
				"status": "not_ready",
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
