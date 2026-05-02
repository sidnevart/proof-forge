package goals

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/sidnevart/proof-forge/backend/internal/platform/email"
	"github.com/sidnevart/proof-forge/backend/internal/users"
)

type TokenGenerator func() (string, error)
type Clock func() time.Time

type Service struct {
	repo          Repository
	emailSender   email.Sender
	webOrigin     string
	log           *slog.Logger
	inviteTTL     time.Duration
	tokenGenerate TokenGenerator
	clock         Clock
}

func NewService(repo Repository, emailSender email.Sender, webOrigin string, log *slog.Logger, inviteTTL time.Duration) *Service {
	return &Service{
		repo:          repo,
		emailSender:   emailSender,
		webOrigin:     webOrigin,
		log:           log,
		inviteTTL:     inviteTTL,
		tokenGenerate: randomInviteToken,
		clock:         time.Now,
	}
}

func (s *Service) CreateGoal(ctx context.Context, owner users.User, input CreateInput) (GoalView, error) {
	if err := input.Validate(owner); err != nil {
		return GoalView{}, err
	}

	input = input.Normalize()
	rawToken, err := s.tokenGenerate()
	if err != nil {
		return GoalView{}, fmt.Errorf("generate invite token: %w", err)
	}

	goal, err := s.repo.CreateGoalWithInvite(ctx, CreateGoalParams{
		OwnerID:         owner.ID,
		OwnerEmail:      owner.Email,
		Title:           input.Title,
		Description:     input.Description,
		BuddyName:       input.BuddyName,
		BuddyEmail:      input.BuddyEmail,
		GoalStatus:      GoalStatusPendingBuddyAcceptance,
		PactStatus:      PactStatusInvited,
		InviteStatus:    InviteStatusPending,
		ProgressHealth:  ProgressHealthUnknown,
		InviteTokenHash: hashInviteToken(rawToken),
		InviteExpiresAt: s.clock().UTC().Add(s.inviteTTL),
	})
	if err != nil {
		return GoalView{}, fmt.Errorf("create goal with invite: %w", err)
	}

	goal.Invite.AcceptanceToken = rawToken

	ownerName := owner.DisplayName
	if ownerName == "" {
		ownerName = owner.Email
	}
	if err := s.emailSender.SendBuddyInvite(ctx, email.BuddyInviteParams{
		To:        input.BuddyEmail,
		OwnerName: ownerName,
		GoalTitle: input.Title,
		InviteURL: s.webOrigin + "/invites/" + rawToken,
	}); err != nil && s.log != nil {
		s.log.Warn("send buddy invite email", "to", input.BuddyEmail, "err", err)
	}

	return goal, nil
}

func (s *Service) Dashboard(ctx context.Context, actor users.User) (Dashboard, error) {
	goalViews, err := s.repo.ListGoalsForUser(ctx, actor.ID)
	if err != nil {
		return Dashboard{}, fmt.Errorf("list goals for user: %w", err)
	}

	// Summary counts only goals where the user is the owner — keeps the
	// "your goals" mental model. Buddy goals are still in the Goals slice
	// for the UI to render in a separate section.
	summary := DashboardSummary{}
	for _, item := range goalViews {
		if item.Role != RoleOwner {
			continue
		}
		summary.TotalGoals++
		switch item.Goal.Status {
		case GoalStatusPendingBuddyAcceptance:
			summary.PendingBuddyAcceptance++
		case GoalStatusActive:
			summary.ActiveGoals++
		}
	}

	return Dashboard{
		Summary: summary,
		Goals:   goalViews,
	}, nil
}

func (s *Service) GetGoal(ctx context.Context, actor users.User, goalID int64) (GoalView, error) {
	view, err := s.repo.FindGoalForUser(ctx, goalID, actor.ID)
	if err != nil {
		return GoalView{}, err
	}
	return view, nil
}

// GetInvitePreview looks up an invite by its raw token and returns the preview
// record. No auth is required — the token is the credential.
func (s *Service) GetInvitePreview(ctx context.Context, rawToken string) (InviteRecord, error) {
	record, err := s.repo.FindInviteByToken(ctx, hashInviteToken(rawToken))
	if err != nil {
		return InviteRecord{}, err
	}
	return record, nil
}

// AcceptInvite validates all domain invariants and, when they pass, atomically
// transitions invite → accepted, pact → active, goal → active.
func (s *Service) AcceptInvite(ctx context.Context, actor users.User, rawToken string) error {
	record, err := s.repo.FindInviteByToken(ctx, hashInviteToken(rawToken))
	if err != nil {
		return err
	}

	if record.InviteStatus == InviteStatusAccepted {
		return ErrInviteAlreadyAccepted
	}
	if record.InviteStatus != InviteStatusPending {
		return ErrInviteAlreadyAccepted
	}
	if s.clock().After(record.ExpiresAt) {
		return ErrInviteExpired
	}
	if !strings.EqualFold(actor.Email, record.InviteeEmail) {
		return ErrUnauthorizedAcceptance
	}

	return s.repo.AcceptInvite(ctx, AcceptInviteParams{
		InviteID:   record.InviteID,
		PactID:     record.PactID,
		GoalID:     record.GoalID,
		AcceptedAt: s.clock().UTC(),
	})
}

func hashInviteToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func randomInviteToken() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
