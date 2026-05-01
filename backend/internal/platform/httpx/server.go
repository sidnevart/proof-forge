package httpx

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	platformlogger "github.com/sidnevart/proof-forge/backend/internal/platform/logger"
	"github.com/sidnevart/proof-forge/backend/internal/platform/readiness"
)

func NewRouter(log *slog.Logger, checker readiness.Checker, webOrigin string) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.RealIP)
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(cors(webOrigin))
	r.Use(requestLogging(log))

	r.Get("/healthz", readiness.HealthHandler())
	r.Get("/readyz", readiness.ReadyHandler(checker, log))

	return r
}

func cors(webOrigin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if webOrigin != "" {
				w.Header().Set("Access-Control-Allow-Origin", webOrigin)
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
				w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PATCH,OPTIONS")
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func requestLogging(log *slog.Logger) func(http.Handler) http.Handler {
	base := log
	if base == nil {
		base = slog.Default()
	}
	base = platformlogger.WithComponent(base, "http")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			requestID := middleware.GetReqID(r.Context())
			reqLog := platformlogger.WithRequestID(r.Context(), base, requestID)

			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)

			reqLog.Info("request completed",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
				"bytes", ww.BytesWritten(),
				"duration_ms", time.Since(start).Milliseconds(),
			)
		})
	}
}
