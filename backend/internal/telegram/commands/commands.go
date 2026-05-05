package commands

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/sidnevart/proof-forge/backend/internal/notifications"
	"github.com/sidnevart/proof-forge/backend/internal/telegram"
	"github.com/sidnevart/proof-forge/backend/internal/telegram/bot"
)

// CommandHandler handles /quiet, /today, /help commands.
type CommandHandler struct {
	tgRepo    *telegram.Repository
	notifRepo *notifications.PostgresRepository
	sender    bot.Sender
	webOrigin string
	log       *slog.Logger
}

func NewCommandHandler(
	tgRepo *telegram.Repository,
	notifRepo *notifications.PostgresRepository,
	sender bot.Sender,
	webOrigin string,
	log *slog.Logger,
) *CommandHandler {
	return &CommandHandler{
		tgRepo:    tgRepo,
		notifRepo: notifRepo,
		sender:    sender,
		webOrigin: webOrigin,
		log:       log,
	}
}

// Handle dispatches text commands.
func (h *CommandHandler) Handle(ctx context.Context, msg *bot.Message) {
	chatID := msg.Chat.ID
	text := strings.TrimSpace(msg.Text)

	link, err := h.tgRepo.GetTelegramLinkByChatID(ctx, chatID)
	if err != nil {
		_ = h.sender.SendMessage(ctx, chatID, "Аккаунт не привязан. Получи ссылку в настройках ProofForge.")
		return
	}

	switch {
	case text == "/quiet":
		h.handleQuiet(ctx, chatID, link.UserID)
	case text == "/today":
		h.handleToday(ctx, chatID)
	case text == "/help":
		h.handleHelp(ctx, chatID)
	}
}

func (h *CommandHandler) handleQuiet(ctx context.Context, chatID int64, userID int64) {
	if err := h.notifRepo.SetQuietToday(ctx, userID); err != nil {
		h.log.Warn("quiet: set quiet", "err", err)
	}
	_ = h.sender.SendMessage(ctx, chatID, "🔕 <b>ТИХО НА СЕГОДНЯ.</b> Нуджи выключены до полуночи.")
}

func (h *CommandHandler) handleToday(ctx context.Context, chatID int64) {
	url := fmt.Sprintf("%s/dashboard", h.webOrigin)
	_ = h.sender.SendMessage(ctx, chatID,
		fmt.Sprintf("📊 <b>ОТКРОЙ ДАШБОРД</b>\n<a href=\"%s\">%s</a>", url, url))
}

func (h *CommandHandler) handleHelp(ctx context.Context, chatID int64) {
	_ = h.sender.SendMessage(ctx, chatID,
		"<b>КОМАНДЫ:</b>\n"+
			"/today — открыть дашборд\n"+
			"/quiet — не беспокоить сегодня\n"+
			"/help — эта справка")
}
