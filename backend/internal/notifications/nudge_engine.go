package notifications

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/sidnevart/proof-forge/backend/internal/telegram/callbacks"
)

// NudgeEngine polls domain_events and dispatches nudges for each unprocessed event.
type NudgeEngine struct {
	repo       *PostgresRepository
	dispatcher *Dispatcher
	log        *slog.Logger
}

func NewNudgeEngine(repo *PostgresRepository, dispatcher *Dispatcher, log *slog.Logger) *NudgeEngine {
	return &NudgeEngine{repo: repo, dispatcher: dispatcher, log: log}
}

// Run polls the domain_events table every 60 seconds.
func (e *NudgeEngine) Run(ctx context.Context) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := e.tick(ctx); err != nil {
				e.log.Error("nudge engine tick", "err", err)
			}
		}
	}
}

// Tick processes all unprocessed domain events once. Exported for use in tests.
func (e *NudgeEngine) Tick(ctx context.Context) error {
	return e.tick(ctx)
}

func (e *NudgeEngine) tick(ctx context.Context) error {
	events, err := e.repo.GetUnprocessedEvents(ctx, 100)
	if err != nil {
		return fmt.Errorf("nudge engine: get events: %w", err)
	}

	for _, ev := range events {
		if err := e.handle(ctx, ev); err != nil {
			e.log.Warn("nudge engine: handle event", "id", ev.ID, "kind", ev.Kind, "err", err)
		}
		if err := e.repo.MarkEventProcessed(ctx, ev.ID); err != nil {
			e.log.Warn("nudge engine: mark processed", "id", ev.ID, "err", err)
		}
	}
	return nil
}

func (e *NudgeEngine) handle(ctx context.Context, ev DomainEvent) error {
	var payload map[string]any
	if err := json.Unmarshal(ev.Payload, &payload); err != nil {
		return nil
	}
	getInt64 := func(key string) int64 {
		if v, ok := payload[key]; ok {
			switch x := v.(type) {
			case float64:
				return int64(x)
			case int64:
				return x
			}
		}
		return 0
	}
	getString := func(key string) string {
		if v, ok := payload[key]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
		return ""
	}

	switch ev.Kind {
	case "checkin.submitted":
		// Buddy needs to review — send approval keyboard to buddy via Telegram.
		checkInID := getInt64("check_in_id")
		goalID := getInt64("goal_id")
		ownerName := getString("owner_display_name")

		buddyUserID, err := e.repo.GetGoalBuddyUserID(ctx, goalID)
		if err != nil {
			e.log.Warn("nudge: get buddy user id", "goal", goalID, "err", err)
			break
		}
		if buddyUserID == 0 {
			break // no active pact yet
		}

		chatID, err := e.repo.GetTelegramChatID(ctx, buddyUserID)
		if err != nil {
			e.log.Warn("nudge: get buddy chat id", "user", buddyUserID, "err", err)
			break
		}
		if chatID == 0 {
			break // buddy has no Telegram linked
		}

		if ownerName == "" {
			ownerName, _ = e.repo.GetOwnerDisplayName(ctx, getInt64("owner_user_id"))
		}
		text := fmt.Sprintf("📋 <b>НОВЫЙ ПРУФ</b>\n%s ждёт твоей оценки.", ownerName)
		kb := callbacks.BuildApprovalKeyboard(checkInID)
		if _, err := e.dispatcher.SendApprovalRequest(ctx, chatID, text, kb); err != nil {
			e.log.Warn("nudge: send approval request", "checkin", checkInID, "err", err)
		}

	case "checkin.approved":
		ownerID := getInt64("owner_user_id")
		ownerName := getString("owner_display_name")

		// 1. Notify the owner: their check-in was approved.
		ownerText := "✅ <b>ОДОБРЕНО. Продолжай!</b>"
		ownerKey := NudgeDedupKey(ownerID, fmt.Sprintf("%s_%d", KindCheckinApproved, ev.ID))
		if err := e.dispatcher.SendNudge(ctx, ownerID, KindCheckinApproved, ownerKey, ownerText); err != nil {
			e.log.Warn("nudge: checkin approved owner", "user", ownerID, "err", err)
		}

		// 2. Notify other circle members: first check-in today.
		memberIDs := toInt64Slice(payload["all_member_ids"])
		if len(memberIDs) > 0 && ownerName != "" {
			for _, memberID := range memberIDs {
				if memberID == ownerID {
					continue
				}
				text := fmt.Sprintf("🟢 <b>%s ПЕРВЫЙ. ТЫ ЕЩЁ НЕТ.</b>", ownerName)
				key := NudgeDedupKey(memberID, fmt.Sprintf("%s_%d", KindNudgeFirst, ownerID))
				if err := e.dispatcher.SendNudge(ctx, memberID, KindNudgeFirst, key, text); err != nil {
					e.log.Warn("nudge: first checkin", "user", memberID, "err", err)
				}
			}
		}

	case "checkin.rejected":
		ownerID := getInt64("owner_user_id")
		ownerText := "❌ <b>ОТКЛОНЕНО. Пересмотри и пересдай.</b>"
		ownerKey := NudgeDedupKey(ownerID, fmt.Sprintf("%s_%d", KindCheckinRejected, ev.ID))
		if err := e.dispatcher.SendNudge(ctx, ownerID, KindCheckinRejected, ownerKey, ownerText); err != nil {
			e.log.Warn("nudge: checkin rejected owner", "user", ownerID, "err", err)
		}

	case "invite.accepted":
		ownerID := getInt64("owner_user_id")
		buddyName := getString("buddy_name")
		if buddyName == "" {
			buddyName = "Партнёр"
		}
		text := fmt.Sprintf("🤝 <b>%s ПРИНЯЛ ПРИГЛАШЕНИЕ. Цель активна.</b>", buddyName)
		key := NudgeDedupKey(ownerID, fmt.Sprintf("%s_%d", KindInviteAccepted, ev.ID))
		if err := e.dispatcher.SendNudge(ctx, ownerID, KindInviteAccepted, key, text); err != nil {
			e.log.Warn("nudge: invite accepted", "user", ownerID, "err", err)
		}

	case "member.frozen":
		frozenID := getInt64("user_id")
		frozenName := getString("display_name")
		memberIDs := toInt64Slice(payload["all_member_ids"])

		// Notify the frozen user.
		selfText := "❄️ <b>ТЫ ЗАМОРОЖЕН. ОДИН ПРУФ — СНИМЕТ.</b>"
		key := NudgeDedupKey(frozenID, KindNudgeSelf)
		if err := e.dispatcher.SendNudge(ctx, frozenID, KindNudgeSelf, key, selfText); err != nil {
			e.log.Warn("nudge: self frozen", "user", frozenID, "err", err)
		}

		// Notify others.
		for _, memberID := range memberIDs {
			if memberID == frozenID {
				continue
			}
			text := fmt.Sprintf("🥶 <b>%s ЗАМОРОЖЕН. НЕ ПРИСОЕДИНЯЙСЯ.</b>", frozenName)
			k := NudgeDedupKey(memberID, fmt.Sprintf("%s_%d", KindNudgeFrozen, frozenID))
			if err := e.dispatcher.SendNudge(ctx, memberID, KindNudgeFrozen, k, text); err != nil {
				e.log.Warn("nudge: member frozen", "user", memberID, "err", err)
			}
		}

	case "daily.deadline_2h":
		userID := getInt64("user_id")
		text := "⏱ <b>2 ЧАСА ДО ТИСКОВ. СДАЙ.</b>"
		key := NudgeDedupKey(userID, KindNudge2h)
		if err := e.dispatcher.SendNudge(ctx, userID, KindNudge2h, key, text); err != nil {
			e.log.Warn("nudge: 2h deadline", "user", userID, "err", err)
		}

	case "rank.changed":
		userID := getInt64("user_id")
		oldRank := getInt64("old_rank")
		newRank := getInt64("new_rank")
		beatenName := getString("beaten_display_name")
		text := fmt.Sprintf("⬆️ <b>ОБОГНАЛ %s. РАНГ %d → %d.</b>", beatenName, oldRank, newRank)
		key := NudgeDedupKey(userID, KindNudgeRank)
		if err := e.dispatcher.SendNudge(ctx, userID, KindNudgeRank, key, text); err != nil {
			e.log.Warn("nudge: rank up", "user", userID, "err", err)
		}
	}
	return nil
}

func toInt64Slice(v any) []int64 {
	if v == nil {
		return nil
	}
	raw, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]int64, 0, len(raw))
	for _, item := range raw {
		switch x := item.(type) {
		case float64:
			out = append(out, int64(x))
		case int64:
			out = append(out, x)
		}
	}
	return out
}
