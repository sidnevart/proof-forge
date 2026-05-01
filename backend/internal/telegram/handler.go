package telegram

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Handler receives Telegram webhook updates. It validates the secret token
// header and logs the raw update. Bot-logic dispatch is a follow-up slice.
type Handler struct {
	secret string
	log    *slog.Logger
}

func NewHandler(secret string, log *slog.Logger) *Handler {
	return &Handler{secret: secret, log: log}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/telegram/webhook", h.handleUpdate)
}

func (h *Handler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	if h.secret != "" && r.Header.Get("X-Telegram-Bot-Api-Secret-Token") != h.secret {
		h.log.Warn("telegram webhook: invalid secret token", "remote", r.RemoteAddr)
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1 MB cap
	if err != nil {
		h.log.Error("telegram webhook: read body", "err", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	var raw json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		h.log.Error("telegram webhook: invalid json", "err", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	h.log.Info("telegram webhook: update received", "bytes", len(body))
	w.WriteHeader(http.StatusOK)
}
