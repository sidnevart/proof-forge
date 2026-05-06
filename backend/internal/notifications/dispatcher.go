package notifications

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/sidnevart/proof-forge/backend/internal/telegram/bot"
)

// Dispatcher sends Telegram notifications and records them in notifications_log.
type Dispatcher struct {
	sender bot.Sender
	repo   *PostgresRepository
	policy *Policy
	log    *slog.Logger
}

func NewDispatcher(sender bot.Sender, repo *PostgresRepository, log *slog.Logger) *Dispatcher {
	return &Dispatcher{
		sender: sender,
		repo:   repo,
		policy: NewPolicy(repo),
		log:    log,
	}
}

// SendNudge sends a nudge message to the user if allowed by policy.
// kind is used for throttle accounting; dedupKey prevents duplicate sends.
func (d *Dispatcher) SendNudge(ctx context.Context, userID int64, kind, dedupKey, text string) error {
	ok, err := d.policy.CanSendNudge(ctx, userID)
	if err != nil {
		return fmt.Errorf("dispatcher: policy check: %w", err)
	}
	if !ok {
		return nil
	}

	chatID, err := d.repo.GetTelegramChatID(ctx, userID)
	if err != nil {
		return fmt.Errorf("dispatcher: get chat id: %w", err)
	}
	if chatID == 0 {
		return nil
	}

	if err := d.sendWithRetry(ctx, chatID, text); err != nil {
		return err
	}

	if logErr := d.repo.LogNotification(ctx, userID, kind, dedupKey); logErr != nil {
		d.log.Warn("dispatcher: log notification", "err", logErr)
	}
	return nil
}

// SendDigest sends a digest message to a specific chat without throttling.
func (d *Dispatcher) SendDigest(ctx context.Context, userID int64, chatID int64, dedupKey, text string) error {
	if err := d.sendWithRetry(ctx, chatID, text); err != nil {
		return err
	}
	if logErr := d.repo.LogNotification(ctx, userID, KindDigest, dedupKey); logErr != nil {
		d.log.Warn("dispatcher: log digest", "err", logErr)
	}
	return nil
}

// SendApprovalRequest sends an inline keyboard message to a buddy for review.
func (d *Dispatcher) SendApprovalRequest(ctx context.Context, chatID int64, text string, kb bot.InlineKeyboardMarkup) (int, error) {
	var msgID int
	var err error
	for attempt := 1; attempt <= 3; attempt++ {
		msgID, err = d.sender.SendMessageWithKeyboard(ctx, chatID, text, kb)
		if err == nil {
			return msgID, nil
		}
		d.log.Warn("dispatcher: send approval request", "attempt", attempt, "err", err)
		if attempt < 3 {
			time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
		}
	}
	return 0, fmt.Errorf("dispatcher: send approval request: %w", err)
}

func (d *Dispatcher) sendWithRetry(ctx context.Context, chatID int64, text string) error {
	var err error
	for attempt := 1; attempt <= 3; attempt++ {
		err = d.sender.SendMessage(ctx, chatID, text)
		if err == nil {
			return nil
		}
		d.log.Warn("dispatcher: send message", "attempt", attempt, "err", err)
		if attempt < 3 {
			time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
		}
	}
	return fmt.Errorf("dispatcher: send message: %w", err)
}
