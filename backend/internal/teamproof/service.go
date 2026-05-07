package teamproof

import (
	"context"
	"fmt"
	"time"

	"github.com/sidnevart/proof-forge/backend/internal/analytics"
)

// Service is the use-case layer for team proofs.
type Service struct {
	repo     Repository
	clock    func() time.Time
	recorder analytics.Recorder
}

// ServiceOption customises a Service.
type ServiceOption func(*Service)

// WithClock injects a deterministic clock for tests.
func WithClock(f func() time.Time) ServiceOption {
	return func(s *Service) { s.clock = f }
}

// WithRecorder injects an analytics recorder.
func WithRecorder(r analytics.Recorder) ServiceOption {
	return func(s *Service) { s.recorder = r }
}

// NewService builds a Service.
func NewService(repo Repository, opts ...ServiceOption) *Service {
	s := &Service{repo: repo, clock: time.Now}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// SubmitProof creates a team proof from daily log entries and evidence.
func (s *Service) SubmitProof(ctx context.Context, in SubmitInput) (*TeamProof, error) {
	// Verify active membership.
	mem, err := s.repo.GetMembership(ctx, in.TeamID, in.OwnerUserID)
	if err != nil {
		return nil, fmt.Errorf("membership: %w", err)
	}
	if mem.Status != "active" {
		return nil, fmt.Errorf("not an active member")
	}

	// Verify goal belongs to team and caller is owner.
	goalTeamID, goalOwnerID, err := s.repo.GetGoalTeamOwner(ctx, in.GoalID)
	if err != nil {
		return nil, err
	}
	if goalTeamID != in.TeamID {
		return nil, ErrInvalidGoal
	}
	if goalOwnerID != in.OwnerUserID {
		return nil, ErrNotAuthor
	}

	// Verify entries ownership and consumption.
	hasConsumedEntries := false
	if len(in.DailyLogEntryIDs) > 0 {
		entries, err := s.repo.GetDailyLogEntries(ctx, in.DailyLogEntryIDs, in.OwnerUserID, in.TeamID)
		if err != nil {
			return nil, err
		}
		if len(entries) != len(in.DailyLogEntryIDs) {
			return nil, ErrEntriesNotOwned
		}
		for _, e := range entries {
			if e.ConsumedInCheckInID != nil {
				return nil, ErrEntryAlreadyUsed
			}
			if e.Status != "logged" {
				return nil, ErrEntriesNotOwned
			}
		}
		hasConsumedEntries = true
	}

	// Require at least one artifact.
	if err := ArtifactRequired(in.Evidence, hasConsumedEntries); err != nil {
		return nil, err
	}

	now := s.clock()
	proof := &TeamProof{
		TeamID:      in.TeamID,
		GoalID:      in.GoalID,
		OwnerUserID: in.OwnerUserID,
		Status:      "submitted",
		SubmittedAt: &now,
	}
	p, err := s.repo.CreateProof(ctx, proof, in.Evidence, in.DailyLogEntryIDs)
	if err != nil {
		return nil, err
	}

	if s.recorder != nil {
		_ = s.recorder.Record(ctx, analytics.EventProofSubmitted, analytics.SourceWeb, &in.OwnerUserID, &in.TeamID, &in.GoalID, &p.ID, map[string]any{
			"evidence_count": len(in.Evidence),
			"consumed_entries": len(in.DailyLogEntryIDs),
		})
	}

	return p, nil
}

// ApproveProof allows lead or trusted approver to approve a submitted proof.
func (s *Service) ApproveProof(ctx context.Context, approverID int64, proofID int64) error {
	proof, err := s.repo.GetProof(ctx, proofID)
	if err != nil {
		return err
	}
	if proof.Status != "submitted" {
		return ErrAlreadyReviewed
	}
	if proof.OwnerUserID == approverID {
		return ErrCannotReviewSelf
	}

	mem, err := s.repo.GetMembership(ctx, proof.TeamID, approverID)
	if err != nil {
		return err
	}
	if mem.Role != "lead" && mem.Role != "trusted_approver" {
		return ErrNotApprover
	}

	if err := s.repo.UpdateProofStatus(ctx, proofID, "approved", mem.Role, s.clock()); err != nil {
		return err
	}

	if s.recorder != nil {
		_ = s.recorder.Record(ctx, analytics.EventProofApproved, analytics.SourceWeb, &approverID, &proof.TeamID, &proof.GoalID, &proofID, map[string]any{
			"approver_role": mem.Role,
		})
	}

	return nil
}

// RejectProof allows lead or trusted approver to reject with a required comment.
func (s *Service) RejectProof(ctx context.Context, approverID int64, proofID int64, comment string) error {
	if len(comment) == 0 {
		return ErrRejectCommentRequired
	}
	proof, err := s.repo.GetProof(ctx, proofID)
	if err != nil {
		return err
	}
	if proof.Status != "submitted" {
		return ErrAlreadyReviewed
	}
	if proof.OwnerUserID == approverID {
		return ErrCannotReviewSelf
	}

	mem, err := s.repo.GetMembership(ctx, proof.TeamID, approverID)
	if err != nil {
		return err
	}
	if mem.Role != "lead" && mem.Role != "trusted_approver" {
		return ErrNotApprover
	}

	// Insert review record via check_in_reviews (reused from circles logic).
	// Phase 2 stores rejection comment in check_in_reviews.
	const insertReview = `
		INSERT INTO check_in_reviews (check_in_id, reviewer_user_id, decision, comment, created_at)
		VALUES ($1, $2, 'rejected', $3, $4)
	`
	if _, err := s.repo.(*PostgresRepository).pool.Exec(ctx, insertReview, proofID, approverID, comment, s.clock()); err != nil {
		return fmt.Errorf("record reject review: %w", err)
	}

	if err := s.repo.UpdateProofStatus(ctx, proofID, "rejected", mem.Role, s.clock()); err != nil {
		return err
	}

	if s.recorder != nil {
		_ = s.recorder.Record(ctx, analytics.EventProofRejected, analytics.SourceWeb, &approverID, &proof.TeamID, &proof.GoalID, &proofID, map[string]any{
			"approver_role": mem.Role,
		})
	}

	return nil
}

// ListFeed returns the team feed scoped by viewer role.
func (s *Service) ListFeed(ctx context.Context, teamID int64, viewerUserID int64, cursor int64, limit int) ([]FeedItem, error) {
	mem, err := s.repo.GetMembership(ctx, teamID, viewerUserID)
	if err != nil {
		return nil, err
	}
	if mem.Status != "active" {
		return nil, fmt.Errorf("not an active member")
	}
	return s.repo.ListFeed(ctx, teamID, viewerUserID, mem.Role, cursor, limit)
}

// ListComments returns comments for a proof if viewer is a team member.
func (s *Service) ListComments(ctx context.Context, proofID int64, viewerUserID int64) ([]Comment, error) {
	proof, err := s.repo.GetProof(ctx, proofID)
	if err != nil {
		return nil, err
	}
	mem, err := s.repo.GetMembership(ctx, proof.TeamID, viewerUserID)
	if err != nil {
		return nil, err
	}
	if mem.Status != "active" {
		return nil, fmt.Errorf("not an active member")
	}
	return s.repo.ListComments(ctx, proofID)
}

// AddComment adds a comment if the viewer is permitted.
func (s *Service) AddComment(ctx context.Context, proofID int64, userID int64, text string) (*Comment, error) {
	proof, err := s.repo.GetProof(ctx, proofID)
	if err != nil {
		return nil, err
	}

	mem, err := s.repo.GetMembership(ctx, proof.TeamID, userID)
	if err != nil {
		return nil, err
	}
	if mem.Status != "active" {
		return nil, fmt.Errorf("not an active member")
	}

	// Permission rules:
	// - Anyone can comment on approved proofs.
	// - Author, lead and trusted can comment on pending/rejected.
	canComment := false
	switch proof.Status {
	case "approved":
		canComment = true
	case "submitted", "rejected":
		if proof.OwnerUserID == userID || mem.Role == "lead" || mem.Role == "trusted_approver" {
			canComment = true
		}
	}
	if !canComment {
		return nil, ErrCannotComment
	}

	c, err := s.repo.AddComment(ctx, proofID, userID, text, s.clock())
	if err != nil {
		return nil, err
	}

	if s.recorder != nil {
		_ = s.recorder.Record(ctx, analytics.EventProofCommented, analytics.SourceWeb, &userID, &proof.TeamID, &proof.GoalID, &proofID, map[string]any{
			"comment_id": c.ID,
		})
	}

	return c, nil
}
