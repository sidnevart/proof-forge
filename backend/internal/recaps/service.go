package recaps

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/sidnevart/proof-forge/backend/internal/users"
)

type Service struct {
	repo Repository
	ai   AIProvider
	log  *slog.Logger
}

func NewService(repo Repository, ai AIProvider, log *slog.Logger) *Service {
	return &Service{repo: repo, ai: ai, log: log}
}

func (s *Service) GetRecap(ctx context.Context, actor users.User, recapID int64) (WeeklyRecap, error) {
	recap, buddyID, err := s.repo.GetRecap(ctx, recapID)
	if err != nil {
		return WeeklyRecap{}, err
	}
	if actor.ID != recap.OwnerUserID && actor.ID != buddyID {
		return WeeklyRecap{}, ErrNotAuthorized
	}
	return recap, nil
}

func (s *Service) ListForGoal(ctx context.Context, actor users.User, goalID int64) ([]WeeklyRecap, error) {
	recaps, err := s.repo.ListRecapsByGoal(ctx, goalID, actor.ID)
	if err != nil {
		return nil, fmt.Errorf("list recaps: %w", err)
	}
	return recaps, nil
}

// SweepAndGenerate is called by the worker. It finds active goals with approved
// check-ins in the current ISO week that lack a recap, generates AI summaries, and
// persists them. One failed goal does not block the rest.
func (s *Service) SweepAndGenerate(ctx context.Context, now time.Time) error {
	start, end := isoWeek(now)

	goals, err := s.repo.FindGoalsNeedingRecap(ctx, start, end)
	if err != nil {
		return fmt.Errorf("find goals needing recap: %w", err)
	}

	for _, g := range goals {
		if err := s.generateOne(ctx, g, start, end); err != nil {
			s.log.Error("recap generation failed", "goal_id", g.ID, "err", err)
		}
	}
	return nil
}

func (s *Service) generateOne(ctx context.Context, goal GoalForRecap, start, end time.Time) error {
	recap, err := s.repo.InsertRecap(ctx, InsertRecapParams{
		GoalID:      goal.ID,
		OwnerUserID: goal.OwnerUserID,
		PeriodStart: start,
		PeriodEnd:   end,
		Status:      StatusGenerating,
	})
	if err != nil {
		return fmt.Errorf("insert recap: %w", err)
	}

	checkIns, err := s.repo.FindApprovedCheckIns(ctx, goal.ID, start, end)
	if err != nil {
		_ = s.repo.UpdateRecap(ctx, UpdateRecapParams{ID: recap.ID, Status: StatusFailed})
		return fmt.Errorf("find approved check-ins: %w", err)
	}

	prompt := BuildPrompt(goal, [2]time.Time{start, end}, checkIns)
	summary, modelName, err := s.ai.Summarize(ctx, prompt)
	if err != nil {
		_ = s.repo.UpdateRecap(ctx, UpdateRecapParams{ID: recap.ID, Status: StatusFailed})
		return fmt.Errorf("ai summarize: %w", err)
	}

	return s.repo.UpdateRecap(ctx, UpdateRecapParams{
		ID:          recap.ID,
		Status:      StatusDone,
		SummaryText: summary,
		ModelName:   modelName,
		GeneratedAt: time.Now().UTC(),
	})
}

// isoWeek returns the Monday 00:00 UTC start and the following Monday 00:00 UTC end
// for the ISO week that contains now.
func isoWeek(now time.Time) (start, end time.Time) {
	t := now.UTC().Truncate(24 * time.Hour)
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday → day 7 (belongs to previous ISO week)
	}
	start = t.AddDate(0, 0, -(weekday - 1))
	end = start.AddDate(0, 0, 7)
	return
}
