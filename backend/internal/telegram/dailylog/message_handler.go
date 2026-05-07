package dailylog

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/sidnevart/proof-forge/backend/internal/dailylog"
	"github.com/sidnevart/proof-forge/backend/internal/telegram"
	"github.com/sidnevart/proof-forge/backend/internal/telegram/bot"
)

// MessageHandler processes free-text Telegram replies as daily-log submissions.
type MessageHandler struct {
	dlSvc  *dailylog.Service
	tgRepo *telegram.Repository
	sender bot.Sender
	log    *slog.Logger
}

// NewMessageHandler builds a MessageHandler.
func NewMessageHandler(dlSvc *dailylog.Service, tgRepo *telegram.Repository, sender bot.Sender, log *slog.Logger) *MessageHandler {
	return &MessageHandler{dlSvc: dlSvc, tgRepo: tgRepo, sender: sender, log: log}
}

// Handle treats any non-command text as a daily-log entry for the user's primary team.
func (h *MessageHandler) Handle(ctx context.Context, msg *bot.Message) {
	chatID := msg.Chat.ID
	text := strings.TrimSpace(msg.Text)
	if text == "" {
		return
	}

	link, err := h.tgRepo.GetTelegramLinkByChatID(ctx, chatID)
	if err != nil {
		_ = h.sender.SendMessage(ctx, chatID, "Аккаунт не привязан. Получи ссылку в настройках ProofForge.")
		return
	}

	// For MVP: use the first active team membership.
	// In production this should track pending prompt state per-membership.
	teamID, err := h.tgRepo.GetFirstActiveTeamMembership(ctx, link.UserID)
	if err != nil {
		h.log.Warn("dailylog message: no active team", "user_id", link.UserID, "err", err)
		_ = h.sender.SendMessage(ctx, chatID, "Ты не состоишь ни в одной команде. Сначала создай или присоединись к команде.")
		return
	}

	_, su, err := h.dlSvc.SubmitLog(ctx, dailylog.SubmitInput{
		UserID:       link.UserID,
		TeamID:       teamID,
		LogDate:      time.Now(),
		TextContent:  text,
		ClientSource: "telegram_bot",
	})
	if err != nil {
		switch {
		case errors.Is(err, dailylog.ErrTextTooShort):
			_ = h.sender.SendMessage(ctx, chatID, "Слишком коротко. Минимум 10 символов.")
		default:
			h.log.Warn("dailylog message: submit failed", "err", err)
			_ = h.sender.SendMessage(ctx, chatID, "Не удалось записать. Попробуй позже.")
		}
		return
	}

	reply := fmt.Sprintf("Записано.\n\nСтрик: %d дней.", su.NewCurrent)
	if su.IsNewRecord {
		reply += " Новый рекорд."
	}
	_ = h.sender.SendMessage(ctx, chatID, reply)
}
