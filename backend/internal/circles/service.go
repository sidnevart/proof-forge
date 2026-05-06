package circles

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/sidnevart/proof-forge/backend/internal/users"
)

type CodeGenerator func() (string, error)

type Service struct {
	repo         Repository
	clock        func() time.Time
	codeGenerate CodeGenerator
}

func NewService(repo Repository) *Service {
	return &Service{
		repo:         repo,
		clock:        time.Now,
		codeGenerate: randomInviteCode,
	}
}

func (s *Service) CreateCircle(ctx context.Context, actor users.User, input CreateInput) (Detail, error) {
	if err := input.Validate(); err != nil {
		return Detail{}, err
	}
	input = input.Normalize()

	code, err := s.codeGenerate()
	if err != nil {
		return Detail{}, fmt.Errorf("generate invite code: %w", err)
	}

	startsAt := s.clock().UTC()
	endsAt := startsAt.Add(SeasonLengthDays * 24 * time.Hour)

	detail, err := s.repo.CreateCircle(ctx, CreateCircleParams{
		OwnerUserID: actor.ID,
		Name:        input.Name,
		InviteCode:  code,
		MemberLimit: DefaultMemberLimit,
		StartsAt:    startsAt,
		EndsAt:      endsAt,
	})
	if err != nil {
		return Detail{}, fmt.Errorf("create circle: %w", err)
	}
	return detail, nil
}

func (s *Service) ListCircles(ctx context.Context, actor users.User) ([]Detail, error) {
	items, err := s.repo.ListCirclesForUser(ctx, actor.ID)
	if err != nil {
		return nil, fmt.Errorf("list circles: %w", err)
	}
	return items, nil
}

func (s *Service) GetCircle(ctx context.Context, actor users.User, circleID int64) (Detail, error) {
	// Best-effort lazy completion of any overdue season before fetching circle detail.
	s.LazyCheckSeason(ctx, circleID)

	item, err := s.repo.GetCircleForUser(ctx, circleID, actor.ID)
	if err != nil {
		return Detail{}, err
	}
	return item, nil
}

func (s *Service) JoinCircle(ctx context.Context, actor users.User, input JoinInput) (Detail, error) {
	if err := input.Validate(); err != nil {
		return Detail{}, err
	}
	item, err := s.repo.JoinCircle(ctx, JoinCircleParams{
		UserID:     actor.ID,
		InviteCode: input.InviteCode,
		JoinedAt:   s.clock().UTC(),
	})
	if err != nil {
		return Detail{}, err
	}
	return item, nil
}

func (s *Service) GetStandings(ctx context.Context, actor users.User, circleID int64) ([]StandingEntry, error) {
	detail, err := s.repo.GetCircleForUser(ctx, circleID, actor.ID)
	if err != nil {
		return nil, err
	}

	currentWeek, currentStart, currentEnd, prevStart, prevEnd := seasonWindows(detail.ActiveSeason, s.clock().UTC())
	_ = currentWeek
	snapshots, err := s.repo.ListStandingSnapshots(
		ctx,
		circleID,
		detail.ActiveSeason.StartsAt,
		detail.ActiveSeason.EndsAt,
		currentStart,
		currentEnd,
		prevStart,
		prevEnd,
	)
	if err != nil {
		return nil, fmt.Errorf("load standing snapshots: %w", err)
	}
	return buildStandings(snapshots, currentWeek), nil
}

func (s *Service) GetWeeklyAssembly(ctx context.Context, actor users.User, circleID int64) (WeeklyAssembly, error) {
	detail, err := s.repo.GetCircleForUser(ctx, circleID, actor.ID)
	if err != nil {
		return WeeklyAssembly{}, err
	}

	currentWeek, currentStart, currentEnd, prevStart, prevEnd := seasonWindows(detail.ActiveSeason, s.clock().UTC())
	snapshots, err := s.repo.ListStandingSnapshots(
		ctx,
		circleID,
		detail.ActiveSeason.StartsAt,
		detail.ActiveSeason.EndsAt,
		currentStart,
		currentEnd,
		prevStart,
		prevEnd,
	)
	if err != nil {
		return WeeklyAssembly{}, fmt.Errorf("load standing snapshots: %w", err)
	}

	standings := buildStandings(snapshots, currentWeek)
	return WeeklyAssembly{
		Circle:      detail.Circle,
		Season:      detail.ActiveSeason,
		CurrentWeek: currentWeek,
		Standings:   standings,
		Events:      buildEvents(standings, s.clock().UTC()),
	}, nil
}

func buildStandings(snapshots []StandingSnapshot, currentWeek int) []StandingEntry {
	entries := make([]StandingEntry, 0, len(snapshots))
	for _, snapshot := range snapshots {
		effectiveMissedWeeks := 0
		if snapshot.GoalsCount > 0 {
			effectiveMissedWeeks = currentWeek - snapshot.ApprovedWeeks
			if effectiveMissedWeeks < 0 {
				effectiveMissedWeeks = 0
			}
		}

		snapshot.MissedWeeks = effectiveMissedWeeks
		status := deriveWeeklyStatus(snapshot)
		score := snapshot.ApprovedWeeks*10 + snapshot.CurrentStreak*3 - effectiveMissedWeeks*2
		switch status {
		case WeeklyStatusComeback:
			score += 5
		case WeeklyStatusApproved:
			score += 3
		case WeeklyStatusDropped:
			score -= 4
		case WeeklyStatusAtRisk:
			score -= 2
		}

		entries = append(entries, StandingEntry{
			UserID:              snapshot.UserID,
			UserEmail:           snapshot.UserEmail,
			DisplayName:         snapshot.DisplayName,
			WeeklyStatus:        status,
			ApprovedWeeks:       snapshot.ApprovedWeeks,
			MissedWeeks:         effectiveMissedWeeks,
			CurrentStreak:       snapshot.CurrentStreak,
			Score:               score,
			GoalsCount:          snapshot.GoalsCount,
			HasActivityThisWeek: snapshot.HasApprovedThisWeek || snapshot.HasSubmittedThisWeek,
		})
	}

	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Score != entries[j].Score {
			return entries[i].Score > entries[j].Score
		}
		if entries[i].ApprovedWeeks != entries[j].ApprovedWeeks {
			return entries[i].ApprovedWeeks > entries[j].ApprovedWeeks
		}
		if entries[i].CurrentStreak != entries[j].CurrentStreak {
			return entries[i].CurrentStreak > entries[j].CurrentStreak
		}
		return entries[i].UserEmail < entries[j].UserEmail
	})
	for i := range entries {
		entries[i].Rank = i + 1
	}
	return entries
}

func deriveWeeklyStatus(snapshot StandingSnapshot) WeeklyStatus {
	if snapshot.GoalsCount == 0 {
		return WeeklyStatusAtRisk
	}
	if snapshot.HasApprovedThisWeek && !snapshot.HasApprovedPrevWeek && snapshot.MissedWeeks > 0 {
		return WeeklyStatusComeback
	}
	if snapshot.HasApprovedThisWeek {
		return WeeklyStatusApproved
	}
	if snapshot.HasSubmittedThisWeek {
		return WeeklyStatusWaitingReview
	}
	if snapshot.MissedWeeks >= 2 {
		return WeeklyStatusDropped
	}
	return WeeklyStatusAtRisk
}

func buildEvents(standings []StandingEntry, now time.Time) []Event {
	events := make([]Event, 0, len(standings))
	for _, item := range standings {
		switch item.WeeklyStatus {
		case WeeklyStatusApproved:
			events = append(events, Event{
				Kind:       EventKindApproved,
				UserID:     item.UserID,
				UserEmail:  item.UserEmail,
				Message:    item.DisplayName + " подтвердил шаг недели.",
				OccurredAt: now,
			})
		case WeeklyStatusComeback:
			events = append(events, Event{
				Kind:       EventKindComeback,
				UserID:     item.UserID,
				UserEmail:  item.UserEmail,
				Message:    item.DisplayName + " вернулся в ритм.",
				OccurredAt: now,
			})
		case WeeklyStatusDropped:
			events = append(events, Event{
				Kind:       EventKindDropped,
				UserID:     item.UserID,
				UserEmail:  item.UserEmail,
				Message:    item.DisplayName + " выпал из недельного ритма.",
				OccurredAt: now,
			})
		case WeeklyStatusAtRisk:
			events = append(events, Event{
				Kind:       EventKindRisk,
				UserID:     item.UserID,
				UserEmail:  item.UserEmail,
				Message:    item.DisplayName + " рискует сорвать неделю.",
				OccurredAt: now,
			})
		}
	}
	return events
}

func seasonWindows(season Season, now time.Time) (int, time.Time, time.Time, time.Time, time.Time) {
	if now.Before(season.StartsAt) {
		now = season.StartsAt
	}
	if now.After(season.EndsAt) {
		now = season.EndsAt.Add(-time.Second)
	}

	weekIndex := int(now.Sub(season.StartsAt)/(7*24*time.Hour)) + 1
	if weekIndex < 1 {
		weekIndex = 1
	}
	if weekIndex > 4 {
		weekIndex = 4
	}

	currentStart := season.StartsAt.Add(time.Duration(weekIndex-1) * 7 * 24 * time.Hour)
	currentEnd := currentStart.Add(7 * 24 * time.Hour)

	prevStart := currentStart.Add(-7 * 24 * time.Hour)
	prevEnd := currentStart
	if weekIndex == 1 {
		prevStart = season.StartsAt
		prevEnd = season.StartsAt
	}

	return weekIndex, currentStart, currentEnd, prevStart, prevEnd
}

func (s *Service) InviteToCircle(ctx context.Context, actor users.User, circleID int64, input InviteToCircleInput) (CircleInvitation, error) {
	targetEmail := strings.TrimSpace(input.TargetEmail)
	if !strings.Contains(targetEmail, "@") {
		return CircleInvitation{}, errors.Join(ErrInvalidCircleInput, errors.New("target_email must be a valid email address"))
	}
	if strings.EqualFold(targetEmail, actor.Email) {
		return CircleInvitation{}, errors.Join(ErrInvalidCircleInput, errors.New("cannot invite yourself"))
	}

	// verify actor is a circle member
	if _, err := s.repo.GetCircleForUser(ctx, circleID, actor.ID); err != nil {
		return CircleInvitation{}, err
	}

	inv, err := s.repo.InviteToCircle(ctx, InviteToCircleParams{
		CircleID:      circleID,
		InviterUserID: actor.ID,
		TargetEmail:   targetEmail,
		Message:       strings.TrimSpace(input.Message),
	})
	if err != nil {
		return CircleInvitation{}, fmt.Errorf("invite to circle: %w", err)
	}
	return inv, nil
}

func (s *Service) ListMyInvitations(ctx context.Context, actor users.User) ([]CircleInvitation, error) {
	items, err := s.repo.ListInvitationsForUser(ctx, actor.Email)
	if err != nil {
		return nil, fmt.Errorf("list my invitations: %w", err)
	}
	return items, nil
}

func (s *Service) AcceptCircleInvitation(ctx context.Context, actor users.User, invitationID int64) error {
	if err := s.repo.AcceptCircleInvitation(ctx, invitationID, actor.ID); err != nil {
		return err
	}
	return nil
}

func (s *Service) DeclineCircleInvitation(ctx context.Context, actor users.User, invitationID int64) error {
	if err := s.repo.DeclineCircleInvitation(ctx, invitationID, actor.ID); err != nil {
		return err
	}
	return nil
}

// EndSeason processes an end-of-season action (extend or start_new) for a circle season.
// The actor must be a member of the circle, the season must have ended, and must not be
// already completed.
func (s *Service) EndSeason(ctx context.Context, actor users.User, circleID, seasonID int64, input SeasonEndInput) (SeasonEndResult, error) {
	// 1. Verify actor is a member of the circle.
	if _, err := s.repo.GetCircleForUser(ctx, circleID, actor.ID); err != nil {
		return SeasonEndResult{}, err
	}

	// 2. Fetch the season.
	season, err := s.repo.GetSeason(ctx, circleID, seasonID)
	if err != nil {
		return SeasonEndResult{}, err
	}

	// 3. Already completed?
	if season.Status == SeasonStatusCompleted {
		return SeasonEndResult{}, ErrSeasonAlreadyDone
	}

	// 4. Has the season ended yet?
	if season.EndsAt.After(s.clock().UTC()) {
		return SeasonEndResult{}, ErrSeasonNotEnded
	}

	// 5. Mark season completed with the chosen action.
	if err := s.repo.MarkSeasonCompleted(ctx, seasonID, input.Action); err != nil {
		return SeasonEndResult{}, fmt.Errorf("mark season completed: %w", err)
	}

	// 6. Perform action-specific work.
	switch input.Action {
	case SeasonEndActionExtend:
		now := s.clock().UTC()
		newSeason, err := s.repo.CreateExtendedSeason(ctx, circleID, now, now.Add(SeasonLengthDays*24*time.Hour))
		if err != nil {
			return SeasonEndResult{}, fmt.Errorf("create extended season: %w", err)
		}
		return SeasonEndResult{Action: SeasonEndActionExtend, NewSeasonID: &newSeason.ID}, nil

	case SeasonEndActionStartNew:
		return SeasonEndResult{Action: SeasonEndActionStartNew, RedirectToGoals: true}, nil

	default:
		return SeasonEndResult{}, fmt.Errorf("unknown season end action: %s", input.Action)
	}
}

// LazyCheckSeason silently marks any overdue active season for a circle as completed.
// Errors are logged but not surfaced — this is a best-effort background hygiene call.
func (s *Service) LazyCheckSeason(ctx context.Context, circleID int64) {
	_ = s.repo.LazyCompleteExpiredSeason(ctx, circleID)
}

// IsCircleMemberForGoal returns true when userID is an active member of the
// circle that owns the goal identified by goalID. Used by the checkins service
// to authorise circle-wide proof reviews (Round A pivot).
func (s *Service) IsCircleMemberForGoal(ctx context.Context, userID, goalID int64) (bool, error) {
	return s.repo.IsActiveMemberOfGoalCircle(ctx, userID, goalID)
}

func randomInviteCode() (string, error) {
	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
