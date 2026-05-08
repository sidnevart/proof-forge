package companion

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/sidnevart/proof-forge/backend/internal/telegram/bot"
)

// SurfaceRouter delivers companion outputs to the right channels.
type SurfaceRouter struct {
	bot      bot.Sender
	email    EmailSender
	webSock  WebSocketPusher
	log      *slog.Logger
}

// EmailSender is the minimal interface for sending emails.
type EmailSender interface {
	Send(ctx context.Context, to string, subject string, body string) error
}

// WebSocketPusher sends real-time updates to connected clients.
type WebSocketPusher interface {
	Push(ctx context.Context, userID int64, payload any) error
}

// NewSurfaceRouter builds a router.
func NewSurfaceRouter(bot bot.Sender, email EmailSender, ws WebSocketPusher, log *slog.Logger) *SurfaceRouter {
	return &SurfaceRouter{bot: bot, email: email, webSock: ws, log: log}
}

// Deliver sends a payload to the specified surfaces.
func (r *SurfaceRouter) Deliver(ctx context.Context, userID int64, telegramChatID *int64, email string, surfaces []Surface, payload *DeliveryPayload) error {
	var lastErr error
	for _, s := range surfaces {
		switch s {
		case SurfaceTelegram:
			if telegramChatID != nil && r.bot != nil {
				if err := r.bot.SendMessage(ctx, *telegramChatID, payload.Text); err != nil {
					r.log.Warn("companion telegram delivery failed", "user_id", userID, "err", err)
					lastErr = err
				}
			}
		case SurfaceInApp:
			// In-app delivery is handled by SaveInAppNotification + frontend polling.
			// Nothing to do here.
		case SurfaceEmail:
			if email != "" && r.email != nil {
				if err := r.email.Send(ctx, email, payload.Text, payload.Text); err != nil {
					r.log.Warn("companion email delivery failed", "user_id", userID, "err", err)
					lastErr = err
				}
			}
		case SurfaceWebSocket:
			if r.webSock != nil {
				if err := r.webSock.Push(ctx, userID, payload); err != nil {
					r.log.Warn("companion websocket delivery failed", "user_id", userID, "err", err)
					lastErr = err
				}
			}
		default:
			r.log.Warn("unknown companion surface", "surface", s)
		}
	}
	if lastErr != nil {
		return fmt.Errorf("companion deliver: some surfaces failed: %w", lastErr)
	}
	return nil
}
