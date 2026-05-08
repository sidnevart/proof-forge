package stats

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Service computes personal metrics and the NowCard.
type Service struct {
	db *db
}

// NewService constructs a stats Service.
func NewService(pool *pgxpool.Pool) *Service {
	return &Service{db: &db{pool: pool}}
}

// GetStats returns all personal metrics for userID.
func (s *Service) GetStats(ctx context.Context, userID int64) (*UserStats, error) {
	now := time.Now().UTC()
	thisWeekStart := isoWeekStart(now)
	lastWeekStart := thisWeekStart.AddDate(0, 0, -7)

	streak, err := s.db.proofStreakWeeks(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("streak: %w", err)
	}

	thisWeek, err := s.db.proofsInWeek(ctx, userID, thisWeekStart)
	if err != nil {
		return nil, fmt.Errorf("this week: %w", err)
	}

	lastWeek, err := s.db.proofsInWeek(ctx, userID, lastWeekStart)
	if err != nil {
		return nil, fmt.Errorf("last week: %w", err)
	}

	activeWeeks, err := s.db.activeWeeksInLast12(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("active weeks: %w", err)
	}

	record, err := s.db.personalRecordWeek(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("personal record: %w", err)
	}

	seasonPct, err := s.db.seasonCompletionPct(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("season pct: %w", err)
	}

	nextAt, err := s.db.nextProofExpectedAt(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("next proof: %w", err)
	}

	activeGoals, err := s.db.activeGoalsCount(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("active goals: %w", err)
	}

	pendingContracts, err := s.db.pendingContractsCount(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("pending contracts: %w", err)
	}

	diff := thisWeek - lastWeek
	trend := weekTrend(thisWeek, lastWeek)

	return &UserStats{
		ProofStreak:           streak,
		ProofsThisWeek:        thisWeek,
		ProofsLastWeek:        lastWeek,
		WeekVsLastWeek:        diff,
		WeekTrend:             trend,
		ActiveWeeks:           activeWeeks,
		PersonalRecordWeek:    record,
		SeasonCompletionPct:   seasonPct,
		NextProofExpectedAt:   nextAt,
		ActiveGoalsCount:      activeGoals,
		PendingContractsCount: pendingContracts,
	}, nil
}

// GetNowCard returns the single highest-priority action for userID.
func (s *Service) GetNowCard(ctx context.Context, userID int64) (*NowCard, error) {
	// 1. Buddy waiting > 24h
	if pr, err := s.db.pendingBuddyReview(ctx, userID, 24*time.Hour); err != nil {
		return nil, err
	} else if pr != nil {
		return &NowCard{
			Type:     "buddy_waiting",
			Title:    "Бадди ждёт твоего ответа",
			Subtitle: fmt.Sprintf("%s оставил комментарий %s", pr.BuddyName, humanizeAge(pr.WaitingSince)),
			Urgency:  NowUrgencyWarn,
			Action: &NowAction{
				Label: "Ответить бадди",
				URL:   fmt.Sprintf("/goals/%d/checkins/%d", pr.GoalID, pr.CheckInID),
			},
		}, nil
	}

	// 2. Broken (overdue) contract
	if bc, err := s.db.brokenContract(ctx, userID); err != nil {
		return nil, err
	} else if bc != nil {
		return &NowCard{
			Type:     "contract_broken",
			Title:    "Пруф просрочен",
			Subtitle: fmt.Sprintf("По цели «%s» — %s", bc.GoalTitle, bc.WhatToProve),
			Urgency:  NowUrgencyDanger,
			Action: &NowAction{
				Label: "Сдать пруф или обновить контракт",
				URL:   fmt.Sprintf("/goals/%d", bc.GoalID),
			},
		}, nil
	}

	now := time.Now().UTC()
	todayEnd := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, time.UTC)
	tomorrowEnd := todayEnd.AddDate(0, 0, 1)

	// 3. Due today
	if dc, err := s.db.contractDueWithin(ctx, userID, now, todayEnd); err != nil {
		return nil, err
	} else if dc != nil {
		return &NowCard{
			Type:     "due_today",
			Title:    "Дедлайн сегодня",
			Subtitle: fmt.Sprintf("«%s» — %s", dc.GoalTitle, dc.WhatToProve),
			Urgency:  NowUrgencyFire,
			Action: &NowAction{
				Label: "Сдать пруф",
				URL:   fmt.Sprintf("/goals/%d/check-in", dc.GoalID),
			},
		}, nil
	}

	// 4. Due tomorrow
	if dc, err := s.db.contractDueWithin(ctx, userID, todayEnd, tomorrowEnd); err != nil {
		return nil, err
	} else if dc != nil {
		return &NowCard{
			Type:     "due_tomorrow",
			Title:    "Дедлайн завтра",
			Subtitle: fmt.Sprintf("«%s» — %s", dc.GoalTitle, dc.WhatToProve),
			Urgency:  NowUrgencyFire,
			Action: &NowAction{
				Label: "Начать сейчас",
				URL:   fmt.Sprintf("/goals/%d/check-in", dc.GoalID),
			},
		}, nil
	}

	// 5. No proof for 5+ days (regular rhythm)
	if ig, err := s.db.inactiveRegularGoal(ctx, userID, 5); err != nil {
		return nil, err
	} else if ig != nil {
		return &NowCard{
			Type:     "long_pause",
			Title:    fmt.Sprintf("Ты не сдавал пруф %d дней", ig.DaysSinceLastProof),
			Subtitle: fmt.Sprintf("По цели «%s»", ig.GoalTitle),
			Urgency:  NowUrgencyWarn,
			Action: &NowAction{
				Label: "Сдать пруф или сказать где застрял",
				URL:   fmt.Sprintf("/goals/%d/check-in", ig.GoalID),
			},
		}, nil
	}

	// 6. No proof for 3+ days
	if ig, err := s.db.inactiveRegularGoal(ctx, userID, 3); err != nil {
		return nil, err
	} else if ig != nil {
		return &NowCard{
			Type:     "small_pause",
			Title:    "Небольшой перерыв",
			Subtitle: fmt.Sprintf("Готов к следующему шагу по «%s»?", ig.GoalTitle),
			Urgency:  NowUrgencyNeutral,
			Action: &NowAction{
				Label: "Продолжить",
				URL:   fmt.Sprintf("/goals/%d/check-in", ig.GoalID),
			},
		}, nil
	}

	// 7. No active goals
	if hasGoals, err := s.db.hasActiveGoals(ctx, userID); err != nil {
		return nil, err
	} else if !hasGoals {
		return &NowCard{
			Type:     "no_goals",
			Title:    "Начни движение",
			Subtitle: "Создай первую цель или подключись к инициативе",
			Urgency:  NowUrgencyNeutral,
			Action: &NowAction{
				Label: "Создать цель",
				URL:   "/goals/new",
			},
		}, nil
	}

	// 8. All good
	nextAt, _ := s.db.nextProofExpectedAt(ctx, userID)
	return &NowCard{
		Type:     "on_track",
		Title:    "Всё идёт по плану",
		Subtitle: formatNextProof(nextAt),
		Urgency:  NowUrgencyNeutral,
		Action:   nil,
	}, nil
}

// ── helpers ───────────────────────────────────────────────────────────

func weekTrend(thisWeek, lastWeek int) WeekTrend {
	if lastWeek == 0 && thisWeek == 0 {
		return WeekTrendFirstWeek
	}
	switch {
	case thisWeek > lastWeek:
		return WeekTrendBetter
	case thisWeek < lastWeek:
		return WeekTrendWorse
	default:
		return WeekTrendSame
	}
}

func humanizeAge(t time.Time) string {
	d := time.Since(t)
	hours := int(d.Hours())
	switch {
	case hours < 2:
		return "час назад"
	case hours < 24:
		return fmt.Sprintf("%d часов назад", hours)
	default:
		return "вчера"
	}
}

func formatNextProof(t *time.Time) string {
	if t == nil {
		return "Нет запланированных пруфов"
	}
	return fmt.Sprintf("Следующий пруф ожидается %d %s", t.Day(), russianMonth(t.Month()))
}

func russianMonth(m time.Month) string {
	months := []string{"января", "февраля", "марта", "апреля", "мая", "июня",
		"июля", "августа", "сентября", "октября", "ноября", "декабря"}
	if int(m) < 1 || int(m) > 12 {
		return ""
	}
	return months[int(m)-1]
}
