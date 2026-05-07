package dailylog

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sidnevart/proof-forge/backend/internal/personalization"
	"github.com/sidnevart/proof-forge/backend/internal/telegram"
	"github.com/sidnevart/proof-forge/backend/internal/telegram/bot"
)

// PromptWorker sends evening daily-log prompts to active memberships.
type PromptWorker struct {
	pool    *pgxpool.Pool
	sender  bot.Sender
	tgRepo  *telegram.Repository
	persSvc *personalization.Service
	log     *slog.Logger
}

// NewPromptWorker constructs the prompt worker.
func NewPromptWorker(pool *pgxpool.Pool, sender bot.Sender, tgRepo *telegram.Repository, log *slog.Logger) *PromptWorker {
	return &PromptWorker{pool: pool, sender: sender, tgRepo: tgRepo, log: log}
}

// WithPersonalization injects the personalization service for AI-generated pings.
func (w *PromptWorker) WithPersonalization(svc *personalization.Service) {
	w.persSvc = svc
}

// Run starts the prompt loop. It checks every 5 minutes.
func (w *PromptWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.tick(ctx); err != nil {
				w.log.Warn("prompt worker tick failed", "err", err)
			}
		}
	}
}

func (w *PromptWorker) tick(ctx context.Context) error {
	now := time.Now().UTC()

	// Select memberships where local hour is 19 and minute <= 5.
	const sql = `
		SELECT m.id, m.team_id, m.user_id, m.timezone, t.name, t.ai_mode, m.ai_consent, u.display_name
		FROM team_memberships m
		JOIN teams t ON t.id = m.team_id
		JOIN users u ON u.id = m.user_id
		WHERE m.status = 'active'
		  AND EXISTS (
			  SELECT 1 FROM telegram_links tl
			  WHERE tl.user_id = m.user_id AND tl.status = 'active'
		  )
	`
	rows, err := w.pool.Query(ctx, sql)
	if err != nil {
		return fmt.Errorf("query prompt candidates: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var memID, teamID, userID int64
		var tz, teamName, aiMode, alias string
		var consent bool
		if err := rows.Scan(&memID, &teamID, &userID, &tz, &teamName, &aiMode, &consent, &alias); err != nil {
			w.log.Warn("scan prompt candidate", "err", err)
			continue
		}

		loc, err := time.LoadLocation(tz)
		if err != nil {
			continue
		}
		local := now.In(loc)
		if local.Hour() != 19 || local.Minute() > 5 {
			continue
		}

		logDate := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)

		// Check if entry already exists.
		var exists bool
		_ = w.pool.QueryRow(ctx,
			`SELECT true FROM daily_log_entries WHERE user_id = $1 AND team_id = $2 AND log_date = $3`,
			userID, teamID, logDate,
		).Scan(&exists)
		if exists {
			continue
		}

		link, err := w.tgRepo.GetTelegramLinkByUserID(ctx, userID)
		if err != nil {
			continue
		}

		var streak int
		var weekLogCount int
		var goalTitles []string
		if w.persSvc != nil {
			streak, weekLogCount = w.getStreakAndWeekLogs(ctx, userID, teamID)
			goalTitles = w.getActiveGoalTitles(ctx, teamID)
		}

		w.sendPrompt(ctx, link.TelegramChatID, teamName, aiMode, consent, alias, streak, weekLogCount, goalTitles)
	}
	return rows.Err()
}

func (w *PromptWorker) getStreakAndWeekLogs(ctx context.Context, userID, teamID int64) (streak int, weekLogCount int) {
	const streakSQL = `SELECT COALESCE(current_streak, 0) FROM user_streak WHERE user_id = $1 AND team_id = $2`
	_ = w.pool.QueryRow(ctx, streakSQL, userID, teamID).Scan(&streak)
	weekAgo := time.Now().UTC().AddDate(0, 0, -7)
	const countSQL = `
		SELECT COUNT(*) FROM daily_log_entries
		WHERE user_id = $1 AND team_id = $2 AND log_date >= $3 AND status = 'logged'
	`
	_ = w.pool.QueryRow(ctx, countSQL, userID, teamID, weekAgo).Scan(&weekLogCount)
	return
}

func (w *PromptWorker) getActiveGoalTitles(ctx context.Context, teamID int64) []string {
	const sql = `SELECT title FROM goals WHERE team_id = $1 AND status = 'active'`
	rows, err := w.pool.Query(ctx, sql, teamID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var titles []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err == nil {
			titles = append(titles, t)
		}
	}
	return titles
}

func (w *PromptWorker) sendPrompt(ctx context.Context, chatID int64, teamName, aiMode string, consent bool, alias string, streak, weekLogCount int, goalTitles []string) {
	text := fmt.Sprintf("Что нового узнал сегодня?\n\nОдна строка достаточно. Если есть ссылка или скрин, прикрепи.\n\nКоманда: %s", teamName)

	if w.persSvc != nil {
		mode := w.resolveMode(aiMode, consent)
		if mode != personalization.ModeOff {
			system := `Ты дружелюбный AI, который помогает людям рефлексировать о ежедневном обучении.
Напиши один вопрос на русском языке, который побуждает пользователя поделиться, что он узнал сегодня.
8-25 слов. Должен заканчиваться вопросительным знаком.
Не используй похвалу/стыд/контроль.
Не используй: молодец, отлично, плохо, ты должен.`
			var goalsPart string
			if len(goalTitles) > 0 {
				goalsPart = "Цели: " + strings.Join(goalTitles, ", ") + "."
			}
			user := fmt.Sprintf("Пользователь: %s. Стрик: %d дней. Записей за неделю: %d. %s День недели: %s.",
				alias, streak, weekLogCount, goalsPart, time.Now().Weekday().String())
			res, _, err := w.persSvc.Prompt(ctx, personalization.FeatureEveningPing, mode, 2*time.Second, system, user, 64)
			if err == nil && res.Text != "" {
				if err := personalization.ValidateEveningPing(res.Text); err == nil {
					text = res.Text + fmt.Sprintf("\n\nКоманда: %s", teamName)
					if res.Provider == personalization.ProviderKimi {
						text += "\n✨ AI"
					}
				}
			}
		}
	}

	_ = w.sender.SendMessage(ctx, chatID, text)
}

func (w *PromptWorker) resolveMode(aiMode string, consent bool) personalization.Mode {
	switch aiMode {
	case "off":
		return personalization.ModeOff
	case "metadata-only":
		return personalization.ModeMetadataOnly
	case "full":
		if consent {
			return personalization.ModeFull
		}
		return personalization.ModeMetadataOnly
	}
	return personalization.ModeOff
}

// RolloverWorker marks missed days for memberships whose local date rolled past midnight.
type RolloverWorker struct {
	pool *pgxpool.Pool
	log  *slog.Logger
}

// NewRolloverWorker constructs the rollover worker.
func NewRolloverWorker(pool *pgxpool.Pool, log *slog.Logger) *RolloverWorker {
	return &RolloverWorker{pool: pool, log: log}
}

// Run starts the rollover loop. It checks every 10 minutes.
func (w *RolloverWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.tick(ctx); err != nil {
				w.log.Warn("rollover worker tick failed", "err", err)
			}
		}
	}
}

func (w *RolloverWorker) tick(ctx context.Context) error {
	service := NewService(NewPostgresRepository(w.pool))

	const sql = `
		SELECT team_id, user_id, timezone
		FROM team_memberships
		WHERE status = 'active'
	`
	rows, err := w.pool.Query(ctx, sql)
	if err != nil {
		return fmt.Errorf("query rollover candidates: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var teamID, userID int64
		var tz string
		if err := rows.Scan(&teamID, &userID, &tz); err != nil {
			continue
		}
		mem := MembershipInfo{Timezone: tz, Status: "active"}
		if err := service.DayRollover(ctx, mem, userID, teamID); err != nil {
			w.log.Warn("rollover failed", "user_id", userID, "team_id", teamID, "err", err)
		}
	}
	return rows.Err()
}
