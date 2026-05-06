package callbacks

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/sidnevart/proof-forge/backend/internal/checkins"
	"github.com/sidnevart/proof-forge/backend/internal/telegram"
	"github.com/sidnevart/proof-forge/backend/internal/telegram/bot"
	"github.com/sidnevart/proof-forge/backend/internal/users"
)

// ReviewService is the subset of checkins.Service needed for inline approval.
type ReviewService interface {
	Review(ctx context.Context, actor users.User, checkInID int64, input checkins.ReviewInput) (checkins.ReviewRecord, error)
}

// ReviewHandler handles inline approval callbacks from the Telegram bot.
type ReviewHandler struct {
	svc    ReviewService
	tgRepo *telegram.Repository
	sender bot.Sender
	log    *slog.Logger
}

func NewReviewHandler(svc ReviewService, tgRepo *telegram.Repository, sender bot.Sender, log *slog.Logger) *ReviewHandler {
	return &ReviewHandler{svc: svc, tgRepo: tgRepo, sender: sender, log: log}
}

// Handle processes a callback_data string of the form "review_<checkinID>_<action>".
func (h *ReviewHandler) Handle(ctx context.Context, cq *bot.CallbackQuery) {
	if err := h.sender.AnswerCallbackQuery(ctx, cq.ID, ""); err != nil {
		h.log.Warn("review callback: answer", "err", err)
	}

	chatID := cq.From.ID

	link, err := h.tgRepo.GetTelegramLinkByChatID(ctx, chatID)
	if err != nil {
		h.log.Warn("review callback: get link", "err", err)
		return
	}

	checkInID, action, ok := parseCallbackData(cq.Data)
	if !ok {
		h.log.Warn("review callback: invalid data", "data", cq.Data)
		return
	}

	actor := users.User{ID: link.UserID}
	input := checkins.ReviewInput{}

	switch action {
	case "approve":
		input.Decision = checkins.DecisionApprove
	case "reject":
		input.Decision = checkins.DecisionReject
		input.Comment = "Отклонено через Telegram."
	default:
		h.log.Warn("review callback: unknown action", "action", action)
		return
	}

	if _, err := h.svc.Review(ctx, actor, checkInID, input); err != nil {
		h.editOrSend(ctx, cq, chatID, fmt.Sprintf("❌ Ошибка: %s", err.Error()))
		if errors.Is(err, checkins.ErrNotBuddy) || errors.Is(err, checkins.ErrCannotReview) {
			return
		}
		return
	}

	var resultText string
	if action == "approve" {
		resultText = "✅ <b>ОДОБРЕНО</b>"
	} else {
		resultText = "❌ <b>ОТКЛОНЕНО</b>"
	}

	h.editOrSend(ctx, cq, chatID, resultText)
}

func (h *ReviewHandler) editOrSend(ctx context.Context, cq *bot.CallbackQuery, chatID int64, text string) {
	if cq.Message != nil {
		if err := h.sender.EditMessageText(ctx, chatID, cq.Message.MessageID, text); err != nil {
			h.log.Warn("review callback: edit message", "err", err)
			_ = h.sender.SendMessage(ctx, chatID, text)
		}
		return
	}
	_ = h.sender.SendMessage(ctx, chatID, text)
}

func parseCallbackData(data string) (int64, string, bool) {
	parts := strings.SplitN(data, "_", 3)
	if len(parts) != 3 || parts[0] != "review" {
		return 0, "", false
	}
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return 0, "", false
	}
	return id, parts[2], true
}

// BuildApprovalKeyboard returns the inline keyboard for a check-in approval request.
func BuildApprovalKeyboard(checkInID int64) bot.InlineKeyboardMarkup {
	return bot.InlineKeyboardMarkup{
		InlineKeyboard: [][]bot.InlineKeyboardButton{
			{
				{Text: "✅ ОДОБРИТЬ", CallbackData: fmt.Sprintf("review_%d_approve", checkInID)},
				{Text: "❌ ОТКЛОНИТЬ", CallbackData: fmt.Sprintf("review_%d_reject", checkInID)},
			},
		},
	}
}
