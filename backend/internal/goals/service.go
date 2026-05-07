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
	"unicode/utf8"

	"github.com/sidnevart/proof-forge/backend/internal/ai"
	"github.com/sidnevart/proof-forge/backend/internal/circles"
	"github.com/sidnevart/proof-forge/backend/internal/platform/email"
	"github.com/sidnevart/proof-forge/backend/internal/users"
)

type TokenGenerator func() (string, error)
type Clock func() time.Time
type ServiceOption func(*Service)

// DomainEventEmitter writes domain events for downstream consumers (nudge engine, etc).
type DomainEventEmitter interface {
	Emit(ctx context.Context, kind string, payload map[string]any) error
}

// noopGoalsEmitter satisfies DomainEventEmitter without doing anything.
type noopGoalsEmitter struct{}

func (noopGoalsEmitter) Emit(_ context.Context, _ string, _ map[string]any) error { return nil }

// CircleLister is the dashboard's read-only window into the circles domain.
// We accept any implementation (real circles.Service, fake in tests) to keep
// goals decoupled from circles wiring.
type CircleLister interface {
	ListCircles(ctx context.Context, actor users.User) ([]circles.Detail, error)
}

const (
	goalRefineRateLimit       = 5
	goalRefineRateWindow      = 24 * time.Hour
	goalRefineCacheTTL        = 30 * 24 * time.Hour
	goalRefineMaxProofExample = 3
)

type Service struct {
	repo               Repository
	emailSender        email.Sender
	webOrigin          string
	log                *slog.Logger
	inviteTTL          time.Duration
	refineProvider     ai.RefineProvider
	tokenGenerate      TokenGenerator
	circleCodeGenerate TokenGenerator
	clock              Clock
	emitter            DomainEventEmitter
	circleLister       CircleLister
}

func NewService(repo Repository, emailSender email.Sender, webOrigin string, log *slog.Logger, inviteTTL time.Duration, opts ...ServiceOption) *Service {
	service := &Service{
		repo:               repo,
		emailSender:        emailSender,
		webOrigin:          webOrigin,
		log:                log,
		inviteTTL:          inviteTTL,
		refineProvider:     ai.NewFakeGoalRefineProvider(),
		tokenGenerate:      randomInviteToken,
		circleCodeGenerate: randomCircleInviteCode,
		clock:              time.Now,
		emitter:            noopGoalsEmitter{},
	}
	for _, opt := range opts {
		if opt != nil {
			opt(service)
		}
	}
	return service
}

func WithRefineProvider(provider ai.RefineProvider) ServiceOption {
	return func(s *Service) {
		if provider != nil {
			s.refineProvider = provider
		}
	}
}

func WithNotifEmitter(emitter DomainEventEmitter) ServiceOption {
	return func(s *Service) {
		if emitter != nil {
			s.emitter = emitter
		}
	}
}

// WithCircleLister wires the circles read-side into the goals service so the
// dashboard can return a CircleSummary alongside goals. If unset, Dashboard
// returns an empty circles slice.
func WithCircleLister(lister CircleLister) ServiceOption {
	return func(s *Service) {
		if lister != nil {
			s.circleLister = lister
		}
	}
}

func (s *Service) CreateGoal(ctx context.Context, owner users.User, input CreateInput) (GoalView, error) {
	if err := input.Validate(owner); err != nil {
		return GoalView{}, err
	}

	input = input.Normalize()

	params := CreateGoalParams{
		OwnerID:         owner.ID,
		OwnerEmail:      owner.Email,
		Title:           input.Title,
		Description:     input.Description,
		BuddyName:       input.BuddyName,
		BuddyEmail:      input.BuddyEmail,
		ProofExamples:   input.ProofExamples,
		Category:        input.Category,
		GoalStatus:      GoalStatusPendingBuddyAcceptance,
		PactStatus:      PactStatusInvited,
		InviteStatus:    InviteStatusPending,
		ProgressHealth:  ProgressHealthUnknown,
		InviteExpiresAt: s.clock().UTC().Add(s.inviteTTL),
	}

	if input.CircleID > 0 {
		// Existing circle path: validate owner+buddy membership and ensure
		// the owner does not already have an active goal in this circle
		// («1 круг = 1 цель на сезон» invariant).
		ownerInCircle, err := s.repo.IsCircleMember(ctx, input.CircleID, owner.ID)
		if err != nil {
			return GoalView{}, fmt.Errorf("check owner circle membership: %w", err)
		}
		if !ownerInCircle {
			return GoalView{}, ErrOwnerNotInCircle
		}

		buddyInCircle, err := s.repo.IsCircleMemberByEmail(ctx, input.CircleID, input.BuddyEmail)
		if err != nil {
			return GoalView{}, fmt.Errorf("check buddy circle membership: %w", err)
		}
		if !buddyInCircle {
			return GoalView{}, ErrBuddyNotInCircle
		}

		hasActive, err := s.repo.HasActiveGoalInCircle(ctx, input.CircleID, owner.ID)
		if err != nil {
			return GoalView{}, fmt.Errorf("check active goal in circle: %w", err)
		}
		if hasActive {
			return GoalView{}, ErrActiveGoalAlreadyExists
		}

		circleID := input.CircleID
		params.CircleID = &circleID
	} else {
		// Auto-create path: a fresh circle is born together with the goal.
		// «Объяви цель → круг рождается автоматически». A 7-day season starts now.
		code, err := s.circleCodeGenerate()
		if err != nil {
			return GoalView{}, fmt.Errorf("generate circle invite code: %w", err)
		}
		now := s.clock().UTC()
		params.AutoCircle = &AutoCircleParams{
			Name:        deriveCircleName(input.Title),
			InviteCode:  code,
			MemberLimit: circles.DefaultMemberLimit,
			StartsAt:    now,
			EndsAt:      now.Add(circles.SeasonLengthDays * 24 * time.Hour),
		}
	}

	rawToken, err := s.tokenGenerate()
	if err != nil {
		return GoalView{}, fmt.Errorf("generate invite token: %w", err)
	}
	params.InviteTokenHash = hashInviteToken(rawToken)

	goal, err := s.repo.CreateGoalWithInvite(ctx, params)
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

func (s *Service) RefineGoal(ctx context.Context, actor users.User, input RefineInput) (GoalRefineResponse, error) {
	if err := input.Validate(); err != nil {
		return GoalRefineResponse{}, err
	}

	input = input.Normalize()
	draftHash := hashGoalRefineDraft(input.DraftText)
	now := s.clock().UTC()
	since := now.Add(-goalRefineRateWindow)

	requestCount, err := s.repo.CountGoalRefineRequestsSince(ctx, actor.ID, since)
	if err != nil {
		return GoalRefineResponse{}, fmt.Errorf("count refine requests: %w", err)
	}
	if requestCount >= goalRefineRateLimit {
		return GoalRefineResponse{}, ErrGoalRefineRateLimited
	}

	if err := s.repo.InsertGoalRefineRequest(ctx, GoalRefineRequestLogParams{
		UserID:    actor.ID,
		DraftHash: draftHash,
	}); err != nil {
		return GoalRefineResponse{}, fmt.Errorf("insert refine request log: %w", err)
	}

	cached, found, err := s.repo.FindGoalRefineCache(ctx, draftHash, now.Add(-goalRefineCacheTTL))
	if err != nil {
		return GoalRefineResponse{}, fmt.Errorf("find refine cache: %w", err)
	}
	if found {
		return cached, nil
	}

	providerResult, err := s.refineProvider.RefineGoal(ctx, input.DraftText)
	if err != nil {
		return GoalRefineResponse{}, fmt.Errorf("refine goal with provider: %w", err)
	}

	response, err := normalizeGoalRefineResponse(providerResult)
	if err != nil {
		return GoalRefineResponse{}, fmt.Errorf("normalize refine response: %w", err)
	}

	if err := s.repo.SaveGoalRefineCache(ctx, draftHash, response); err != nil {
		return GoalRefineResponse{}, fmt.Errorf("save refine cache: %w", err)
	}

	return response, nil
}

// deriveCircleName turns the goal title into a short circle name. We cap at
// 60 runes so the name fits inside circle headers without wrapping.
func deriveCircleName(title string) string {
	trimmed := strings.TrimSpace(title)
	if utf8.RuneCountInString(trimmed) <= 60 {
		return trimmed
	}
	runes := []rune(trimmed)
	return string(runes[:60])
}

func randomCircleInviteCode() (string, error) {
	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func (s *Service) Dashboard(ctx context.Context, owner users.User) (Dashboard, error) {
	// ListGoalsForUser returns goals where the caller is the owner OR the
	// invited buddy, so a user who accepted an invite sees that goal too.
	goalViews, err := s.repo.ListGoalsForUser(ctx, owner.ID)
	if err != nil {
		return Dashboard{}, fmt.Errorf("list goals for user: %w", err)
	}

	summary := DashboardSummary{
		TotalGoals: len(goalViews),
	}
	for _, item := range goalViews {
		switch item.Goal.Status {
		case GoalStatusPendingBuddyAcceptance:
			summary.PendingBuddyAcceptance++
		case GoalStatusActive:
			summary.ActiveGoals++
		}
	}

	circleSummaries := make([]CircleSummary, 0)
	if s.circleLister != nil {
		details, err := s.circleLister.ListCircles(ctx, owner)
		if err != nil {
			// Don't fail the whole dashboard just because circles can't be listed.
			// The empty state will fall back to the generic "НОВЫЙ КРУГ" eyebrow.
			if s.log != nil {
				s.log.Warn("dashboard: list circles", "err", err)
			}
		} else {
			for _, d := range details {
				circleSummaries = append(circleSummaries, CircleSummary{
					ID:          d.Circle.ID,
					Name:        d.Circle.Name,
					MemberCount: len(d.Members),
				})
			}
		}
	}

	return Dashboard{
		Summary: summary,
		Goals:   goalViews,
		Circles: circleSummaries,
	}, nil
}

// GetGoal returns a single goal view scoped to the actor. The check «is the
// actor allowed to see this goal?» reuses ListGoalsForUser semantics (owner
// OR buddy), so a buddy who accepted an invite can read the goal page just
// like the owner. Anything else returns ErrGoalNotFound — we don't leak the
// existence of goals the user has no relationship to.
func (s *Service) GetGoal(ctx context.Context, actor users.User, goalID int64) (GoalView, error) {
	views, err := s.repo.ListGoalsForUser(ctx, actor.ID)
	if err != nil {
		return GoalView{}, fmt.Errorf("list goals for user: %w", err)
	}
	for _, v := range views {
		if v.Goal.ID == goalID {
			return v, nil
		}
	}
	return GoalView{}, ErrGoalNotFound
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

	if err := s.repo.AcceptInvite(ctx, AcceptInviteParams{
		InviteID:   record.InviteID,
		PactID:     record.PactID,
		GoalID:     record.GoalID,
		AcceptedAt: s.clock().UTC(),
	}); err != nil {
		return err
	}

	// Notify goal owner: buddy accepted the invite.
	buddyName := actor.DisplayName
	if buddyName == "" {
		buddyName = actor.Email
	}
	if err := s.emailSender.SendBuddyAccepted(ctx, email.BuddyAcceptedParams{
		To:           record.OwnerEmail,
		BuddyName:    buddyName,
		GoalTitle:    record.GoalTitle,
		DashboardURL: s.webOrigin + "/dashboard",
	}); err != nil && s.log != nil {
		s.log.Warn("send buddy accepted email", "to", record.OwnerEmail, "err", err)
	}

	_ = s.emitter.Emit(ctx, "invite.accepted", map[string]any{
		"goal_id":       record.GoalID,
		"owner_user_id": record.InviterID,
		"buddy_name":    buddyName,
	})

	return nil
}

func hashInviteToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func hashGoalRefineDraft(draftText string) string {
	normalized := strings.ToLower(strings.TrimSpace(draftText))
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}

func normalizeGoalRefineResponse(result ai.GoalRefineResult) (GoalRefineResponse, error) {
	response := GoalRefineResponse{
		Category: strings.TrimSpace(result.Category),
		Variants: make([]GoalRefineVariant, 0, len(result.Variants)),
	}
	if response.Category == "" || len([]rune(response.Category)) > 32 {
		return GoalRefineResponse{}, fmt.Errorf("%w: provider category is invalid", ErrInvalidGoalRefineInput)
	}
	if len(result.Variants) != 3 {
		return GoalRefineResponse{}, fmt.Errorf("%w: provider must return exactly 3 variants", ErrInvalidGoalRefineInput)
	}

	for _, variant := range result.Variants {
		normalized := GoalRefineVariant{
			Title:         strings.TrimSpace(variant.Title),
			Smart:         strings.TrimSpace(variant.Smart),
			ProofExamples: make([]string, 0, len(variant.ProofExamples)),
		}
		if normalized.Title == "" || normalized.Smart == "" {
			return GoalRefineResponse{}, fmt.Errorf("%w: provider returned empty variant fields", ErrInvalidGoalRefineInput)
		}
		if len(variant.ProofExamples) != goalRefineMaxProofExample {
			return GoalRefineResponse{}, fmt.Errorf("%w: provider must return exactly %d proof examples", ErrInvalidGoalRefineInput, goalRefineMaxProofExample)
		}

		for _, example := range variant.ProofExamples {
			example = strings.TrimSpace(example)
			if example == "" {
				return GoalRefineResponse{}, fmt.Errorf("%w: provider returned empty proof example", ErrInvalidGoalRefineInput)
			}
			normalized.ProofExamples = append(normalized.ProofExamples, example)
		}

		response.Variants = append(response.Variants, normalized)
	}

	return response, nil
}

func randomInviteToken() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
