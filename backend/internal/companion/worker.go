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

	// 3. Lead weekly brief: Mon 09:00 local time.
	if err := w.fireLeadBriefs(ctx, now); err != nil {
		w.log.Error("lead brief tick", "err", err)
	}

	// 4. Streak reminder: 4h before midnight if streak at risk.
	if err := w.fireStreakReminders(ctx, now); err != nil {
		w.log.Error("streak reminder tick", "err", err)
	}

	// 5. Proof draft: when user has ≥3 unconsumed daily log entries.
	if err := w.fireProofDrafts(ctx, now); err != nil {
		w.log.Error("proof draft tick", "err", err)
	}

	// 6. Buddy stalled: when proof is submitted but buddy hasn't responded in 72h.
	if err := w.fireBuddyStalled(ctx, now); err != nil {
		w.log.Error("buddy stalled tick", "err", err)
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

// fireLeadBriefs sends weekly briefs to team leads where it's Mon 09:00.
func (w *Worker) fireLeadBriefs(ctx context.Context, now time.Time) error {
	// Find team leads with Telegram linked.
	rows, err := w.pool.Query(ctx, `
		SELECT u.id, t.id, t.name, COALESCE(u.timezone, 'UTC'), tl.telegram_chat_id
		FROM users u
		JOIN team_memberships tm ON tm.user_id = u.id
		JOIN teams t ON t.id = tm.team_id
		JOIN telegram_links tl ON tl.user_id = u.id
		WHERE tm.role IN ('lead', 'trusted')
		  AND tm.status = 'active'
		  AND tl.status = 'active'
	`)
	if err != nil {
		return fmt.Errorf("lead brief: query leads: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var leaderID, teamID int64
		var teamName, tz string
		var chatID int64
		if err := rows.Scan(&leaderID, &teamID, &teamName, &tz, &chatID); err != nil {
			continue
		}

		loc, err := time.LoadLocation(tz)
		if err != nil {
			loc = time.UTC
		}
		local := now.In(loc)
		if local.Weekday() != time.Monday {
			continue
		}
		if local.Hour() != 9 || local.Minute() > 4 {
			continue
		}

		triggerID := BuildTriggerID(FeatureLeadWeeklyBrief, leaderID, local.Format("2006-W02"))
		shouldFire, err := w.service.ShouldFire(ctx, leaderID, FeatureLeadWeeklyBrief, triggerID)
		if err != nil {
			w.log.Warn("lead brief: should fire check", "leader", leaderID, "err", err)
			continue
		}
		if !shouldFire {
			continue
		}

		if err := w.sendLeadBrief(ctx, leaderID, teamID, chatID); err != nil {
			w.log.Warn("lead brief: send", "leader", leaderID, "err", err)
		}
	}
	return nil
}

// sendLeadBrief generates and delivers a lead weekly brief.
func (w *Worker) sendLeadBrief(ctx context.Context, leaderID, teamID, chatID int64) error {
	now := w.clock()
	from := now.AddDate(0, 0, -7)
	triggerID := BuildTriggerID(FeatureLeadWeeklyBrief, leaderID, now.Format("2006-W02"))

	lc, err := w.assembler.GetLeadBriefContext(ctx, leaderID, teamID, from, now)
	if err != nil {
		return fmt.Errorf("lead brief: assemble context: %w", err)
	}

	userPrompt := BuildLeadBriefPrompt(lc)

	res, _, err := w.service.Run(ctx, FeatureLeadWeeklyBrief, personalization.ModeMetadataOnly, 5*time.Second,
		func(ctx context.Context) (personalization.PromptResult, error) {
			llm := w.service.LLM()
			if llm == nil {
				return personalization.PromptResult{}, fmt.Errorf("llm not configured")
			}
			return llm.Prompt(ctx, LeadBriefSystemPrompt, userPrompt, 500)
		})
	if err != nil {
		w.log.Warn("lead brief: llm failed, using template", "leader", leaderID, "err", err)
		res.Text = LeadBriefTemplate(lc)
		res.Provider = personalization.ProviderTemplate
	}

	if err := ValidateLeadBrief(res.Text); err != nil {
		w.log.Warn("lead brief: validation failed, using template", "leader", leaderID, "err", err)
		res.Text = LeadBriefTemplate(lc)
		res.Provider = personalization.ProviderTemplate
	}

	// Deliver via Telegram.
	if w.bot != nil {
		if err := w.bot.SendMessage(ctx, chatID, res.Text); err != nil {
			return fmt.Errorf("lead brief: telegram send: %w", err)
		}
	}

	if err := w.service.RecordFired(ctx, leaderID, FeatureLeadWeeklyBrief, triggerID); err != nil {
		w.log.Warn("lead brief: record fired", "leader", leaderID, "err", err)
	}

	w.log.Info("lead brief sent", "leader", leaderID, "team", teamID, "provider", res.Provider)
	return nil
}

// fireStreakReminders sends reminders to users whose streak is at risk.
func (w *Worker) fireStreakReminders(ctx context.Context, now time.Time) error {
	// Find users with active streak who haven't submitted a proof today
	// and have 4 hours or less until midnight.
	rows, err := w.pool.Query(ctx, `
		SELECT u.id, COALESCE(u.timezone, 'UTC'), tl.telegram_chat_id
		FROM users u
		JOIN goals g ON g.owner_user_id = u.id
		  AND g.status = 'active'
		  AND g.archived_at IS NULL
		  AND g.current_streak_count > 0
		JOIN telegram_links tl ON tl.user_id = u.id
		WHERE tl.status = 'active'
		  AND NOT EXISTS (
		      SELECT 1 FROM check_ins ci
		      WHERE ci.owner_user_id = u.id
		        AND ci.status = 'approved'
		        AND ci.approved_at >= CURRENT_DATE
		  )
		GROUP BY u.id, u.timezone, tl.telegram_chat_id
	`)
	if err != nil {
		return fmt.Errorf("streak reminder: query users: %w", err)
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
		hoursLeft := 24 - local.Hour()
		// Only fire between 20:00 and 23:59 (4h before midnight).
		if local.Hour() < 20 {
			continue
		}

		triggerID := BuildTriggerID(FeatureStreakReminder, userID, local.Format("2006-01-02"))
		shouldFire, err := w.service.ShouldFire(ctx, userID, FeatureStreakReminder, triggerID)
		if err != nil {
			w.log.Warn("streak reminder: should fire check", "user", userID, "err", err)
			continue
		}
		if !shouldFire {
			continue
		}

		if err := w.sendStreakReminder(ctx, userID, chatID, hoursLeft); err != nil {
			w.log.Warn("streak reminder: send", "user", userID, "err", err)
		}
	}
	return nil
}

// sendStreakReminder delivers a streak risk reminder.
// Streak reminder uses template fallback only — no LLM needed.
func (w *Worker) sendStreakReminder(ctx context.Context, userID int64, chatID int64, hoursLeft int) error {
	triggerID := BuildTriggerID(FeatureStreakReminder, userID, w.clock().Format("2006-01-02"))

	sc, err := w.assembler.GetStreakContext(ctx, userID)
	if err != nil {
		return fmt.Errorf("streak reminder: assemble context: %w", err)
	}
	sc.HoursLeft = hoursLeft

	text := StreakReminderTemplate(sc)

	// Deliver via Telegram.
	if w.bot != nil {
		if err := w.bot.SendMessage(ctx, chatID, text); err != nil {
			return fmt.Errorf("streak reminder: telegram send: %w", err)
		}
	}

	// Also save in-app notification.
	_ = w.service.SaveInAppNotification(ctx, userID, FeatureStreakReminder,
		"Серия под угрозой", text, []NotificationAction{
			{Label: "Сделать пруф", Action: "open_dashboard", URL: "/dashboard"},
		})

	if err := w.service.RecordFired(ctx, userID, FeatureStreakReminder, triggerID); err != nil {
		w.log.Warn("streak reminder: record fired", "user", userID, "err", err)
	}

	w.log.Info("streak reminder sent", "user", userID, "streak", sc.CurrentStreak)
	return nil
}

// fireProofDrafts checks for users with ≥3 unconsumed daily log entries and creates draft suggestions.
func (w *Worker) fireProofDrafts(ctx context.Context, now time.Time) error {
	// Run every hour on the hour to avoid repeated polling.
	if now.Minute() > 4 {
		return nil
	}

	// Find users with active goals, teams, and ≥3 unconsumed daily log entries.
	rows, err := w.pool.Query(ctx, `
		SELECT u.id
		FROM users u
		WHERE EXISTS (
			SELECT 1 FROM goals g
			WHERE g.owner_user_id = u.id
			  AND g.status = 'active'
			  AND g.archived_at IS NULL
		)
		  AND EXISTS (
			SELECT 1 FROM team_memberships tm
			JOIN teams t ON t.id = tm.team_id
			WHERE tm.user_id = u.id AND tm.status = 'active'
		)
		  AND (
			SELECT COUNT(*) FROM daily_log_entries dle
			WHERE dle.user_id = u.id
			  AND dle.consumed_in_check_in_id IS NULL
			  AND dle.status = 'logged'
			  AND dle.log_date >= CURRENT_DATE - INTERVAL '7 days'
		) >= 3
		LIMIT 50
	`)
	if err != nil {
		return fmt.Errorf("proof draft: query users: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var userID int64
		if err := rows.Scan(&userID); err != nil {
			continue
		}

		triggerID := BuildTriggerID(FeatureProofDraft, userID, now.Format("2006-01-02"))
		shouldFire, err := w.service.ShouldFire(ctx, userID, FeatureProofDraft, triggerID)
		if err != nil {
			w.log.Warn("proof draft: should fire check", "user", userID, "err", err)
			continue
		}
		if !shouldFire {
			continue
		}

		if err := w.sendProofDraft(ctx, userID); err != nil {
			w.log.Warn("proof draft: send", "user", userID, "err", err)
		}
	}
	return nil
}

// sendProofDraft generates and delivers a proof draft suggestion.
func (w *Worker) sendProofDraft(ctx context.Context, userID int64) error {
	triggerID := BuildTriggerID(FeatureProofDraft, userID, w.clock().Format("2006-01-02"))

	pdc, err := w.assembler.GetProofDraftContext(ctx, userID)
	if err != nil {
		return fmt.Errorf("proof draft: assemble context: %w", err)
	}

	userPrompt := BuildProofDraftPrompt(pdc)

	res, _, err := w.service.Run(ctx, FeatureProofDraft, personalization.ModeMetadataOnly, 5*time.Second,
		func(ctx context.Context) (personalization.PromptResult, error) {
			llm := w.service.LLM()
			if llm == nil {
				return personalization.PromptResult{}, fmt.Errorf("llm not configured")
			}
			return llm.Prompt(ctx, ProofDraftSystemPrompt, userPrompt, 300)
		})
	if err != nil {
		w.log.Warn("proof draft: llm failed, using template", "user", userID, "err", err)
		res.Text = ProofDraftTemplate(pdc)
		res.Provider = personalization.ProviderTemplate
	}

	if err := ValidateProofDraft(res.Text); err != nil {
		w.log.Warn("proof draft: validation failed, using template", "user", userID, "err", err)
		res.Text = ProofDraftTemplate(pdc)
		res.Provider = personalization.ProviderTemplate
	}

	// Collect note IDs for the draft.
	var noteIDs []int64
	for _, n := range pdc.Notes {
		noteIDs = append(noteIDs, n.NoteID)
	}

	// Save proof draft for accept/reject flow.
	draft := &ProofDraft{
		ID:         fmt.Sprintf("draft-%d-%d", userID, w.clock().Unix()),
		UserID:     userID,
		TeamID:     pdc.TeamID,
		GoalID:     &pdc.GoalID,
		NoteIDs:    noteIDs,
		Rationale:  res.Text,
		Confidence: "medium",
		CreatedAt:  w.clock(),
	}
	if err := w.service.SaveProofDraft(ctx, draft); err != nil {
		w.log.Warn("proof draft: save draft", "user", userID, "err", err)
	}

	// Save as in-app notification for user to accept/reject.
	_ = w.service.SaveInAppNotification(ctx, userID, FeatureProofDraft,
		"Черновик пруфа", res.Text, []NotificationAction{
			{Label: "Использовать", Action: "accept_draft", URL: "/dashboard"},
			{Label: "Отменить", Action: "dismiss_draft", URL: "/dashboard"},
		})

	if err := w.service.RecordFired(ctx, userID, FeatureProofDraft, triggerID); err != nil {
		w.log.Warn("proof draft: record fired", "user", userID, "err", err)
	}

	w.log.Info("proof draft sent", "user", userID, "provider", res.Provider)
	return nil
}

// fireBuddyStalled finds proofs stuck in "submitted" state for >72h and alerts the buddy.
func (w *Worker) fireBuddyStalled(ctx context.Context, now time.Time) error {
	// Run once per hour.
	if now.Minute() > 4 {
		return nil
	}

	stalledList, err := w.assembler.GetBuddyStalledContext(ctx, 72)
	if err != nil {
		return fmt.Errorf("buddy stalled: assemble context: %w", err)
	}

	for _, bsc := range stalledList {
		triggerID := BuildTriggerID(FeatureBuddyStalled, bsc.BuddyID, fmt.Sprintf("%d-%s", bsc.ProofID, now.Format("2006-01-02")))
		shouldFire, err := w.service.ShouldFire(ctx, bsc.BuddyID, FeatureBuddyStalled, triggerID)
		if err != nil {
			w.log.Warn("buddy stalled: should fire check", "buddy", bsc.BuddyID, "err", err)
			continue
		}
		if !shouldFire {
			continue
		}

		if err := w.sendBuddyStalled(ctx, bsc); err != nil {
			w.log.Warn("buddy stalled: send", "buddy", bsc.BuddyID, "err", err)
		}
	}
	return nil
}

// sendBuddyStalled generates and delivers a buddy stalled alert.
func (w *Worker) sendBuddyStalled(ctx context.Context, bsc *BuddyStalledContext) error {
	triggerID := BuildTriggerID(FeatureBuddyStalled, bsc.BuddyID, fmt.Sprintf("%d-%s", bsc.ProofID, w.clock().Format("2006-01-02")))

	userPrompt := BuildBuddyStalledPrompt(bsc)

	res, _, err := w.service.Run(ctx, FeatureBuddyStalled, personalization.ModeMetadataOnly, 3*time.Second,
		func(ctx context.Context) (personalization.PromptResult, error) {
			llm := w.service.LLM()
			if llm == nil {
				return personalization.PromptResult{}, fmt.Errorf("llm not configured")
			}
			return llm.Prompt(ctx, BuddyStalledSystemPrompt, userPrompt, 200)
		})
	if err != nil {
		w.log.Warn("buddy stalled: llm failed, using template", "buddy", bsc.BuddyID, "err", err)
		res.Text = BuddyStalledTemplate(bsc)
		res.Provider = personalization.ProviderTemplate
	}

	if err := ValidateBuddyStalled(res.Text); err != nil {
		w.log.Warn("buddy stalled: validation failed, using template", "buddy", bsc.BuddyID, "err", err)
		res.Text = BuddyStalledTemplate(bsc)
		res.Provider = personalization.ProviderTemplate
	}

	// Try to deliver via Telegram to buddy.
	var chatID int64
	_ = w.pool.QueryRow(ctx,
		`SELECT telegram_chat_id FROM telegram_links WHERE user_id = $1 AND status = 'active'`, bsc.BuddyID,
	).Scan(&chatID)

	if w.bot != nil && chatID != 0 {
		if err := w.bot.SendMessage(ctx, chatID, res.Text); err != nil {
			w.log.Warn("buddy stalled: telegram send", "buddy", bsc.BuddyID, "err", err)
		}
	}

	// Also save in-app notification for buddy.
	_ = w.service.SaveInAppNotification(ctx, bsc.BuddyID, FeatureBuddyStalled,
		"Пруф ждёт фидбека", res.Text, []NotificationAction{
			{Label: "Посмотреть", Action: "open_dashboard", URL: "/dashboard"},
		})

	if err := w.service.RecordFired(ctx, bsc.BuddyID, FeatureBuddyStalled, triggerID); err != nil {
		w.log.Warn("buddy stalled: record fired", "buddy", bsc.BuddyID, "err", err)
	}

	w.log.Info("buddy stalled sent", "buddy", bsc.BuddyID, "proof", bsc.ProofID)
	return nil
}
