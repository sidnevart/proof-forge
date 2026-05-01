package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"github.com/sidnevart/proof-forge/backend/internal/platform/config"
)

func New(cfg config.LogConfig) *slog.Logger {
	level := parseLevel(cfg.Level)

	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	switch strings.ToLower(cfg.Format) {
	case "text":
		handler = slog.NewTextHandler(os.Stdout, opts)
	default:
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}

func WithComponent(log *slog.Logger, component string) *slog.Logger {
	return log.With("component", component)
}

func WithRequestID(_ context.Context, log *slog.Logger, requestID string) *slog.Logger {
	if requestID == "" {
		return log
	}
	return log.With("request_id", requestID)
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
