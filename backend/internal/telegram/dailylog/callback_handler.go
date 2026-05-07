package dailylog

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/sidnevart/proof-forge/backend/internal/dailylog"
	"github.com/sidnevart/proof-forge/backend/internal/telegram"
	"github.com/sidnevart/proof-forge/backend/internal/telegram/bot"
)

// CallbackHandler processes Telegram inline callbacks for daily-log actions.
type CallbackHandler struct {
	dlSvc  *dailylog.Service
	tgRepo *telegram.Repository
	sender bot.Sender
	log    *slog.Logger
}

// NewCallbackHandler builds a CallbackHandler.
func NewCallbackHandler(dlSvc *dailylog.Service, tgRepo *telegram.Repository, sender bot.Sender, log *slog.Logger) *CallbackHandler {
	return &CallbackHandler{dlSvc: dlSvc, tgRepo: tgRepo, sender: sender, log: log}
}

// Handle dispatches callback_data prefixed with "log:".
func (h *CallbackHandler) Handle(ctx context.Context, cq *bot.CallbackQuery) {
	if err := h.sender.AnswerCallbackQuery(ctx, cq.ID, ""); err != nil {
		h.log.Warn("dailylog callback: answer", "err", err)
	}

	chatID := cq.From.ID
	link, err := h.tgRepo.GetTelegramLinkByChatID(ctx, chatID)
	if err != nil {
		h.log.Warn("dailylog callback: get link", "err", err)
		return
	}

	action, teamID, date, ok := parseCallbackData(cq.Data)
	if !ok {
		h.log.Warn("dailylog callback: invalid data", "data", cq.Data)
		return
	}

	switch action {
	case "skip":
		h.handleSkip(ctx, cq, link.UserID, teamID, date)
	case "freeze":
		h.handleFreeze(ctx, cq, link.UserID, teamID, date)
	default:
		h.log.Warn("dailylog callback: unknown action", "action", action)
	}
}

func (h *CallbackHandler) handleSkip(ctx context.Context, cq *bot.CallbackQuery, userID, teamID int64, date time.Time) {
	entry := &dailylog.Entry{
		UserID:      userID,
		TeamID:      teamID,
		LogDate:     date,
		Status:      dailylog.StatusSkipped,
		TextContent: "",
		SubmittedAt: time.Now(),
	}
	streak, err := h.dlSvc.GetMyStreak(ctx, userID, teamID)
	if err != nil {
		h.editOrSend(ctx, cq, cq.From.ID, "❌ Не удалось пропустить.")
		return
	}
	su := dailylog.ComputeStreakUpdate(*streak, dailylog.StatusSkipped, date)
	streak.Current = su.NewCurrent
	streak.Best = su.NewBest
	streak.UpdatedAt = time.Now()

	repo := h.dlSvc.Repo()
	_, err = repo.UpsertEntryAndStreak(ctx, entry, streak)
	if err != nil {
		h.editOrSend(ctx, cq, cq.From.ID, "❌ Не удалось пропустить.")
		return
	}
	h.editOrSend(ctx, cq, cq.From.ID, "Пропущено. Стрик сброшен.")
}

func (h *CallbackHandler) handleFreeze(ctx context.Context, cq *bot.CallbackQuery, userID, teamID int64, date time.Time) {
	_, err := h.dlSvc.FreezeDay(ctx, dailylog.FreezeInput{
		UserID:  userID,
		TeamID:  teamID,
		LogDate: date,
		Reason:  "telegram",
	})
	if err != nil {
		msg := "❌ Не удалось заморозить."
		if errors.Is(err, dailylog.ErrFreezeLimitReached) {
			msg = "❌ Лимит заморозок на месяц исчерпан."
		}
		h.editOrSend(ctx, cq, cq.From.ID, msg)
		return
	}
	h.editOrSend(ctx, cq, cq.From.ID, "❄ Заморожено. Стрик сохранен.")
}

func (h *CallbackHandler) editOrSend(ctx context.Context, cq *bot.CallbackQuery, chatID int64, text string) {
	if cq.Message != nil {
		if err := h.sender.EditMessageText(ctx, chatID, cq.Message.MessageID, text); err != nil {
			h.log.Warn("dailylog callback: edit message", "err", err)
			_ = h.sender.SendMessage(ctx, chatID, text)
		}
		return
	}
	_ = h.sender.SendMessage(ctx, chatID, text)
}

// parseCallbackData parses "log:skip:7:2026-05-07" into (action, teamID, date).
func parseCallbackData(data string) (string, int64, time.Time, bool) {
	parts := strings.SplitN(data, ":", 4)
	if len(parts) != 4 || parts[0] != "log" {
		return "", 0, time.Time{}, false
	}
	teamID, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return "", 0, time.Time{}, false
	}
	date, err := time.Parse("2006-01-02", parts[3])
	if err != nil {
		return "", 0, time.Time{}, false
	}
	return parts[1], teamID, date, true
}
