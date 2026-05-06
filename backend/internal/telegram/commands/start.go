package commands

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/sidnevart/proof-forge/backend/internal/telegram"
	"github.com/sidnevart/proof-forge/backend/internal/telegram/bot"
)

// StartHandler handles the /start command including the link flow.
type StartHandler struct {
	repo   *telegram.Repository
	sender bot.Sender
	log    *slog.Logger
}

func NewStartHandler(repo *telegram.Repository, sender bot.Sender, log *slog.Logger) *StartHandler {
	return &StartHandler{repo: repo, sender: sender, log: log}
}

// Handle dispatches /start or /start link_<token>.
func (h *StartHandler) Handle(ctx context.Context, msg *bot.Message) {
	text := strings.TrimSpace(msg.Text)
	chatID := msg.Chat.ID

	if msg.From == nil {
		return
	}

	const prefix = "/start link_"
	if strings.HasPrefix(text, prefix) {
		token := strings.TrimPrefix(text, prefix)
		h.handleLink(ctx, chatID, msg.From, token)
		return
	}

	if err := h.sender.SendMessage(ctx, chatID,
		"👋 <b>PROOFFORGE BOT</b>\n\nОтправь /help чтобы узнать что умею."); err != nil {
		h.log.Warn("start: send greeting", "err", err)
	}
}

func (h *StartHandler) handleLink(ctx context.Context, chatID int64, from *bot.User, token string) {
	pt, err := h.repo.RedeemPendingToken(ctx, token)
	if err != nil {
		if err == telegram.ErrTokenNotFound {
			_ = h.sender.SendMessage(ctx, chatID, "❌ <b>Ссылка устарела или недействительна.</b> Получи новую в настройках.")
			return
		}
		h.log.Error("start link: redeem token", "err", err)
		_ = h.sender.SendMessage(ctx, chatID, "Произошла ошибка. Попробуй ещё раз.")
		return
	}

	username := ""
	if from != nil {
		username = from.Username
	}

	if _, err := h.repo.CreateTelegramLink(ctx, pt.UserID, chatID, username); err != nil {
		h.log.Error("start link: create link", "err", err)
		_ = h.sender.SendMessage(ctx, chatID, "Не удалось привязать аккаунт. Попробуй ещё раз.")
		return
	}

	_ = h.sender.SendMessage(ctx, chatID, "✅ <b>ПРИВЯЗАН. БУДЕМ ДЕРЖАТЬ В ФОРМЕ.</b>")
}

// GenerateToken creates a pending link token for a user (called from web API).
// The token is a 32-char cryptographically random hex string — not predictable from userID/time.
func GenerateToken(ctx context.Context, repo *telegram.Repository, userID int64) (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	token := hex.EncodeToString(buf)
	expires := time.Now().Add(10 * time.Minute)
	if err := repo.CreatePendingToken(ctx, token, userID, expires); err != nil {
		return "", fmt.Errorf("store token: %w", err)
	}
	return token, nil
}
