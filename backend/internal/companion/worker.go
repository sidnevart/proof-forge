package companion

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sidnevart/proof-forge/backend/internal/personalization"
	"github.com/sidnevart/proof-forge/backend/internal/telegram/bot"
)

// Worker runs scheduled companion triggers.
type Worker struct {
	service   *Service
	assembler *ContextAssembler
	bot       bot.Sender
	pool      *pgxpool.Pool
	log       *slog.Logger
	clock     func() time.Time
}

// NewWorker creates a companion worker.
func NewWorker(service *Service, pool *pgxpool.Pool, bot bot.Sender, log *slog.Logger) *Worker {
	return &Worker{
		service:   service,
		assembler: NewContextAssembler(pool),
		bot:       bot,
		pool:      pool,
		log:       log,
		clock:     time.Now,
	}
}

// Run starts the worker loop.
func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	w.log.Info("companion worker started")

	for {
		select {
		case <-ctx.Done():
			w.log.Info("companion worker shutting down")
			return
		case <-ticker.C:
			if err := w.tick(ctx); err != nil {
				w.log.Error("companion worker tick", "err", err)
			}
		}
	}
}

func (w *Worker) tick(ctx context.Context) error {
	now := w.clock()

	// 1. Evening ping: Mon-Fri 19:00 local time.
	if err := w.fireEveningPings(ctx, now); err != nil {
		w.log.Error("evening ping tick", "err", err)
	}

	// 2. Weekly recap: Fri 18:00 local time.
	if err := w.fireWeeklyRecaps(ctx, now); err != nil {
		w.log.Error("weekly recap tick", "err", err)
	}

	return nil
}

// fireEveningPings sends evening pings to users where it's 19:00 Mon-Fri.
func (w *Worker) fireEveningPings(ctx context.Context, now time.Time) error {
	// Find users for whom it's currently 19:00 on a weekday.
	// We check users who have telegram linked and active goals.
	rows, err := w.pool.Query(ctx, `
		SELECT u.id, COALESCE(u.timezone, 'UTC'), tl.telegram_chat_id
		FROM users u
		JOIN telegram_links tl ON tl.user_id = u.id
		WHERE tl.status = 'active'
		  AND EXISTS (
		      SELECT 1 FROM goals g
		      WHERE g.owner_user_id = u.id
		        AND g.status = 'active'
		        AND g.archived_at IS NULL
		  )
	`)
	if err != nil {
		return fmt.Errorf("evening ping: query users: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var userID int64
		var tz string
		var chatID int64
		if err := rows.Scan(&userID, &tz, &chatID); err != nil {
			continue
		}

		// Check if local time is ~19:00 on a weekday.
		loc, err := time.LoadLocation(tz)
		if err != nil {
			loc = time.UTC
		}
		local := now.In(loc)
		if local.Weekday() == time.Saturday || local.Weekday() == time.Sunday {
			continue
		}
		if local.Hour() != 19 || local.Minute() > 4 {
			continue // only fire in the 19:00-19:04 window
		}

		triggerID := BuildTriggerID(FeatureEveningPing, userID, local.Format("2006-01-02"))
		shouldFire, err := w.service.ShouldFire(ctx, userID, FeatureEveningPing, triggerID)
		if err != nil {
			w.log.Warn("evening ping: should fire check", "user", userID, "err", err)
			continue
		}
		if !shouldFire {
			continue
		}

		if err := w.sendEveningPing(ctx, userID, chatID); err != nil {
			w.log.Warn("evening ping: send", "user", userID, "err", err)
		}
	}
	return nil
}

// sendEveningPing generates and delivers an evening ping.
func (w *Worker) sendEveningPing(ctx context.Context, userID, chatID int64) error {
	triggerID := BuildTriggerID(FeatureEveningPing, userID, w.clock().Format("2006-01-02"))

	uc, err := w.assembler.GetUserContext(ctx, userID)
	if err != nil {
		return fmt.Errorf("evening ping: assemble context: %w", err)
	}

	// Build prompt and run through personalization service.
	userPrompt := BuildEveningPingPrompt(uc)

	res, _, err := w.service.Run(ctx, FeatureEveningPing, personalization.ModeMetadataOnly, 2*time.Second,
		func(ctx context.Context) (personalization.PromptResult, error) {
			llm := w.service.LLM()
			if llm == nil {
				return personalization.PromptResult{}, fmt.Errorf("llm not configured")
			}
			return llm.Prompt(ctx, EveningPingSystemPrompt, userPrompt, 200)
		})
	if err != nil {
		w.log.Warn("evening ping: llm failed, using template", "user", userID, "err", err)
		res.Text = SelectEveningPingTemplate(uc.CurrentStreak)
		res.Provider = personalization.ProviderTemplate
	}

	// Validate.
	if err := ValidateEveningPing(res.Text); err != nil {
		w.log.Warn("evening ping: validation failed, using template", "user", userID, "err", err)
		res.Text = SelectEveningPingTemplate(uc.CurrentStreak)
		res.Provider = personalization.ProviderTemplate
	}

	// Deliver via Telegram.
	if w.bot != nil {
		if err := w.bot.SendMessage(ctx, chatID, res.Text); err != nil {
			return fmt.Errorf("evening ping: telegram send: %w", err)
		}
	}

	// Record as fired.
	if err := w.service.RecordFired(ctx, userID, FeatureEveningPing, triggerID); err != nil {
		w.log.Warn("evening ping: record fired", "user", userID, "err", err)
	}

	w.log.Info("evening ping sent", "user", userID, "provider", res.Provider)
	return nil
}

// fireWeeklyRecaps sends weekly recaps to users where it's Fri 18:00.
func (w *Worker) fireWeeklyRecaps(ctx context.Context, now time.Time) error {
	rows, err := w.pool.Query(ctx, `
		SELECT u.id, COALESCE(u.timezone, 'UTC'), tl.telegram_chat_id
		FROM users u
		JOIN telegram_links tl ON tl.user_id = u.id
		WHERE tl.status = 'active'
		  AND EXISTS (
		      SELECT 1 FROM check_ins ci
		      WHERE ci.owner_user_id = u.id
		        AND ci.submitted_at >= NOW() - INTERVAL '7 days'
		  )
	`)
	if err != nil {
		return fmt.Errorf("weekly recap: query users: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var userID int64
		var tz string
		var chatID int64
		if err := rows.Scan(&userID, &tz, &chatID); err != nil {
			continue
		}

		loc, err := time.LoadLocation(tz)
		if err != nil {
			loc = time.UTC
		}
		local := now.In(loc)
		if local.Weekday() != time.Friday {
			continue
		}
		if local.Hour() != 18 || local.Minute() > 4 {
			continue
		}

		triggerID := BuildTriggerID(FeatureWeeklyRecap, userID, local.Format("2006-W02"))
		shouldFire, err := w.service.ShouldFire(ctx, userID, FeatureWeeklyRecap, triggerID)
		if err != nil {
			w.log.Warn("weekly recap: should fire check", "user", userID, "err", err)
			continue
		}
		if !shouldFire {
			continue
		}

		if err := w.sendWeeklyRecap(ctx, userID, chatID); err != nil {
			w.log.Warn("weekly recap: send", "user", userID, "err", err)
		}
	}
	return nil
}

// sendWeeklyRecap generates and delivers a weekly recap.
func (w *Worker) sendWeeklyRecap(ctx context.Context, userID, chatID int64) error {
	now := w.clock()
	from := now.AddDate(0, 0, -7)
	triggerID := BuildTriggerID(FeatureWeeklyRecap, userID, now.Format("2006-W02"))

	wc, err := w.assembler.GetWeeklyContext(ctx, userID, from, now)
	if err != nil {
		return fmt.Errorf("weekly recap: assemble context: %w", err)
	}

	userPrompt := BuildWeeklyRecapPrompt(wc)

	res, _, err := w.service.Run(ctx, FeatureWeeklyRecap, personalization.ModeMetadataOnly, 5*time.Second,
		func(ctx context.Context) (personalization.PromptResult, error) {
			llm := w.service.LLM()
			if llm == nil {
				return personalization.PromptResult{}, fmt.Errorf("llm not configured")
			}
			return llm.Prompt(ctx, WeeklyRecapSystemPrompt, userPrompt, 400)
		})
	if err != nil {
		w.log.Warn("weekly recap: llm failed, using template", "user", userID, "err", err)
		res.Text = WeeklyRecapTemplate(wc)
		res.Provider = personalization.ProviderTemplate
	}

	if err := ValidateWeeklyRecap(res.Text); err != nil {
		w.log.Warn("weekly recap: validation failed, using template", "user", userID, "err", err)
		res.Text = WeeklyRecapTemplate(wc)
		res.Provider = personalization.ProviderTemplate
	}

	// Deliver via Telegram.
	if w.bot != nil {
		if err := w.bot.SendMessage(ctx, chatID, res.Text); err != nil {
			return fmt.Errorf("weekly recap: telegram send: %w", err)
		}
	}

	// Also save in-app notification.
	_ = w.service.SaveInAppNotification(ctx, userID, FeatureWeeklyRecap,
		"Итог недели", res.Text, []NotificationAction{
			{Label: "Посмотреть", Action: "open_dashboard", URL: "/dashboard"},
		})

	if err := w.service.RecordFired(ctx, userID, FeatureWeeklyRecap, triggerID); err != nil {
		w.log.Warn("weekly recap: record fired", "user", userID, "err", err)
	}

	w.log.Info("weekly recap sent", "user", userID, "provider", res.Provider)
	return nil
}
