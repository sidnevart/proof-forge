package telegram

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/sidnevart/proof-forge/backend/internal/telegram/bot"
)

// MessageHandler processes a Telegram message.
type MessageHandler interface {
	Handle(ctx context.Context, msg *bot.Message)
}

// CallbackHandler processes a Telegram callback query.
type CallbackHandler interface {
	Handle(ctx context.Context, cq *bot.CallbackQuery)
}

// Handler receives Telegram webhook updates and dispatches to sub-handlers.
type Handler struct {
	secret         string
	startHandler   MessageHandler
	commandHandler MessageHandler
	reviewCallback CallbackHandler
	log            *slog.Logger
}

// HandlerConfig wires sub-handlers into the top-level webhook handler.
type HandlerConfig struct {
	Secret         string
	StartHandler   MessageHandler
	CommandHandler MessageHandler
	ReviewCallback CallbackHandler
	Log            *slog.Logger
}

func NewHandler(cfg HandlerConfig) *Handler {
	return &Handler{
		secret:         cfg.Secret,
		startHandler:   cfg.StartHandler,
		commandHandler: cfg.CommandHandler,
		reviewCallback: cfg.ReviewCallback,
		log:            cfg.Log,
	}
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

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		h.log.Error("telegram webhook: read body", "err", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	var update bot.Update
	if err := json.Unmarshal(body, &update); err != nil {
		h.log.Error("telegram webhook: unmarshal update", "err", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	h.dispatch(r.Context(), update)
}

func (h *Handler) dispatch(ctx context.Context, update bot.Update) {
	switch {
	case update.CallbackQuery != nil:
		cq := update.CallbackQuery
		if strings.HasPrefix(cq.Data, "review_") && h.reviewCallback != nil {
			h.reviewCallback.Handle(ctx, cq)
		}

	case update.Message != nil:
		msg := update.Message
		text := strings.TrimSpace(msg.Text)

		switch {
		case strings.HasPrefix(text, "/start"):
			if h.startHandler != nil {
				h.startHandler.Handle(ctx, msg)
			}
		case strings.HasPrefix(text, "/quiet") ||
			strings.HasPrefix(text, "/today") ||
			strings.HasPrefix(text, "/help"):
			if h.commandHandler != nil {
				h.commandHandler.Handle(ctx, msg)
			}
		default:
			h.log.Debug("telegram webhook: unhandled message", "text", text)
		}
	}
}
