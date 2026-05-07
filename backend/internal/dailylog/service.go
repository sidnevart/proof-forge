package dailylog

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sidnevart/proof-forge/backend/internal/analytics"
)

// Service is the use-case layer for daily-log and streak.
type Service struct {
	repo     Repository
	clock    func() time.Time
	hc       HolidayChecker
	recorder analytics.Recorder
}

// ServiceOption customises a Service at construction time.
type ServiceOption func(*Service)

// WithClock injects a deterministic clock for tests.
func WithClock(f func() time.Time) ServiceOption {
	return func(s *Service) { s.clock = f }
}

// WithHolidayChecker swaps the default RussianHolidayChecker.
func WithHolidayChecker(hc HolidayChecker) ServiceOption {
	return func(s *Service) { s.hc = hc }
}

// WithRecorder injects an analytics recorder.
func WithRecorder(r analytics.Recorder) ServiceOption {
	return func(s *Service) { s.recorder = r }
}

// NewService builds a Service over the given Repository.
func NewService(repo Repository, opts ...ServiceOption) *Service {
	s := &Service{repo: repo, clock: time.Now, hc: RussianHolidayChecker{}}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Repo returns the underlying repository.
func (s *Service) Repo() Repository {
	return s.repo
}

// SubmitLog records a daily-log entry and updates the streak atomically.
func (s *Service) SubmitLog(ctx context.Context, in SubmitInput) (*Entry, StreakUpdate, error) {
	mem, err := s.repo.GetMembership(ctx, in.TeamID, in.UserID)
	if err != nil {
		return nil, StreakUpdate{}, err
	}

	loc, err := time.LoadLocation(mem.Timezone)
	if err != nil {
		return nil, StreakUpdate{}, fmt.Errorf("%w: %q", ErrInvalidTimezone, mem.Timezone)
	}

	logDate := NormalizeDate(in.LogDate.In(loc))
	if err := IsValidLogDate(logDate, loc, s.clock()); err != nil {
		return nil, StreakUpdate{}, err
	}

	if err := ValidateText(in.TextContent); err != nil {
		return nil, StreakUpdate{}, err
	}

	// Check existing entry to decide whether streak should increment.
	existing, _ := s.repo.GetEntry(ctx, in.UserID, in.TeamID, logDate)
	alreadyCounted := existing != nil && (existing.Status == StatusLogged || existing.Status == StatusFrozen)

	entry := &Entry{
		UserID:      in.UserID,
		TeamID:      in.TeamID,
		LogDate:     logDate,
		Status:      StatusLogged,
		TextContent: in.TextContent,
		SubmittedAt: s.clock(),
	}
	if in.ExternalURL != nil && *in.ExternalURL != "" {
		entry.ExternalURL = in.ExternalURL
		entry.HasArtifact = true
	}

	streak, err := s.repo.GetStreak(ctx, in.UserID, in.TeamID)
	if err != nil {
		return nil, StreakUpdate{}, err
	}

	// Reset freeze counter on month rollover.
	if MonthReset(streak.UpdatedAt, s.clock()) {
		streak.FreezesUsedThisMonth = 0
	}

	var su StreakUpdate
	if !alreadyCounted {
		su = ComputeStreakUpdate(*streak, StatusLogged, logDate)
		streak.Current = su.NewCurrent
		streak.Best = su.NewBest
		streak.LastActiveDate = &logDate
		streak.UpdatedAt = s.clock()
	} else {
		// Overwrite — streak already counted for this day.
		su = StreakUpdate{OldCurrent: streak.Current, NewCurrent: streak.Current,
			OldBest: streak.Best, NewBest: streak.Best, Reason: "overwrite"}
	}

	entry, err = s.repo.UpsertEntryAndStreak(ctx, entry, streak)
	if err != nil {
		return nil, StreakUpdate{}, err
	}

	if s.recorder != nil {
		_ = s.recorder.Record(ctx, analytics.EventDailyLogSubmitted, analytics.SourceWeb, &in.UserID, &in.TeamID, nil, nil, map[string]any{
			"client_source": in.ClientSource,
		})
	}

	return entry, su, nil
}

// FreezeDay freezes a working day, preserving the streak.
func (s *Service) FreezeDay(ctx context.Context, in FreezeInput) (*Entry, error) {
	mem, err := s.repo.GetMembership(ctx, in.TeamID, in.UserID)
	if err != nil {
		return nil, err
	}

	loc, err := time.LoadLocation(mem.Timezone)
	if err != nil {
		return nil, fmt.Errorf("%w: %q", ErrInvalidTimezone, mem.Timezone)
	}

	logDate := NormalizeDate(in.LogDate.In(loc))
	if err := IsValidLogDate(logDate, loc, s.clock()); err != nil {
		return nil, err
	}

	streak, err := s.repo.GetStreak(ctx, in.UserID, in.TeamID)
	if err != nil {
		return nil, err
	}

	if MonthReset(streak.UpdatedAt, s.clock()) {
		streak.FreezesUsedThisMonth = 0
	}

	if err := FreezeAllowed(streak.FreezesUsedThisMonth); err != nil {
		return nil, err
	}

	existing, _ := s.repo.GetEntry(ctx, in.UserID, in.TeamID, logDate)
	if existing != nil && existing.Status == StatusFrozen {
		return existing, nil // idempotent
	}

	entry := &Entry{
		UserID:      in.UserID,
		TeamID:      in.TeamID,
		LogDate:     logDate,
		Status:      StatusFrozen,
		TextContent: "",
		SubmittedAt: s.clock(),
	}

	// Frozen preserves streak but does not increase best.
	su := ComputeStreakUpdate(*streak, StatusFrozen, logDate)
	streak.Current = su.NewCurrent
	streak.Best = su.NewBest
	streak.LastActiveDate = &logDate
	streak.FreezesUsedThisMonth++
	streak.UpdatedAt = s.clock()

	entry, err = s.repo.UpsertEntryAndStreak(ctx, entry, streak)
	if err != nil {
		return nil, err
	}

	if s.recorder != nil {
		_ = s.recorder.Record(ctx, analytics.EventStreakFrozen, analytics.SourceWeb, &in.UserID, &in.TeamID, nil, nil, map[string]any{
			"freezes_used": streak.FreezesUsedThisMonth,
		})
	}

	return entry, nil
}

// ListMyEntries returns the caller's own entries with full content.
func (s *Service) ListMyEntries(ctx context.Context, userID, teamID int64, from, to time.Time) ([]Entry, error) {
	if _, err := s.repo.GetMembership(ctx, teamID, userID); err != nil {
		return nil, err
	}
	return s.repo.ListEntries(ctx, ListFilter{
		UserID:      userID,
		TeamID:      teamID,
		From:        NormalizeDate(from),
		To:          NormalizeDate(to),
		WithContent: true,
	})
}

// GetMyStreak returns the caller's streak for a team.
func (s *Service) GetMyStreak(ctx context.Context, userID, teamID int64) (*Streak, error) {
	if _, err := s.repo.GetMembership(ctx, teamID, userID); err != nil {
		return nil, err
	}
	return s.repo.GetStreak(ctx, userID, teamID)
}

// DayRollover finds active memberships whose local date rolled past midnight
// and inserts missed entries when the previous working day was not logged,
// frozen, or skipped.
func (s *Service) DayRollover(ctx context.Context, membership MembershipInfo, userID, teamID int64) error {
	loc, err := time.LoadLocation(membership.Timezone)
	if err != nil {
		return fmt.Errorf("%w: %q", ErrInvalidTimezone, membership.Timezone)
	}

	now := s.clock().In(loc)
	yesterday := NormalizeDate(now.AddDate(0, 0, -1))

	if !IsWorkingDay(yesterday, s.hc) {
		return nil // weekends/holidays never create missed
	}

	existing, err := s.repo.GetEntry(ctx, userID, teamID, yesterday)
	if err != nil && !errors.Is(err, ErrEntryNotFound) {
		return err
	}
	if existing != nil && (existing.Status == StatusLogged || existing.Status == StatusFrozen || existing.Status == StatusSkipped) {
		return nil
	}

	// Insert missed and break streak.
	entry := &Entry{
		UserID:      userID,
		TeamID:      teamID,
		LogDate:     yesterday,
		Status:      StatusMissed,
		TextContent: "",
		SubmittedAt: s.clock(),
	}

	streak, err := s.repo.GetStreak(ctx, userID, teamID)
	if err != nil {
		return err
	}

	if MonthReset(streak.UpdatedAt, s.clock()) {
		streak.FreezesUsedThisMonth = 0
	}

	su := ComputeStreakUpdate(*streak, StatusMissed, yesterday)
	streak.Current = su.NewCurrent
	streak.Best = su.NewBest
	streak.UpdatedAt = s.clock()

	_, err = s.repo.UpsertEntryAndStreak(ctx, entry, streak)
	if err != nil {
		return err
	}

	if s.recorder != nil {
		_ = s.recorder.Record(ctx, analytics.EventStreakBroken, analytics.SourceCron, &userID, &teamID, nil, nil, map[string]any{
			"log_date": yesterday.Format("2006-01-02"),
		})
	}

	return nil
}
