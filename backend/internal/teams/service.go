package teams

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"strings"
	"time"
)

// Service is the use-case layer for teams. It orchestrates Repository
// operations and enforces invariants that span multiple steps:
//
//   - input validation (team name, role, ai_mode);
//   - "lead cannot demote/remove/leave self while the only lead";
//   - "only the user themselves can flip their ai_consent";
//   - invite-code generation.
//
// Authorization decisions (who can do what) are NOT made here directly —
// they are read off authz.go pure functions, so the same policy applies on
// every surface (HTTP, telegram bot, future channels).
type Service struct {
	repo  Repository
	clock func() time.Time
}

// ServiceOption customises a Service at construction time.
type ServiceOption func(*Service)

// WithClock injects a deterministic clock for tests. The default is time.Now.
func WithClock(f func() time.Time) ServiceOption {
	return func(s *Service) { s.clock = f }
}

// NewService builds a Service over the given Repository.
func NewService(repo Repository, opts ...ServiceOption) *Service {
	s := &Service{repo: repo, clock: time.Now}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// CreateTeam — every authenticated user can create their own team.
// Returns Detail with my_role=lead, member_count=1.
func (s *Service) CreateTeam(ctx context.Context, leadUserID int64, name string, mode AIMode) (Detail, error) {
	if err := ValidateTeamName(name); err != nil {
		return Detail{}, err
	}
	if mode == "" {
		mode = AIModeMetadataOnly
	}
	if _, err := ParseAIMode(string(mode)); err != nil {
		return Detail{}, err
	}
	code, err := generateInviteCode()
	if err != nil {
		return Detail{}, err
	}
	return s.repo.CreateTeam(ctx, CreateTeamParams{
		LeadUserID:  leadUserID,
		Name:        strings.TrimSpace(name),
		InviteCode:  code,
		MemberLimit: DefaultMemberLimit,
		AIMode:      mode,
		CreatedAt:   s.clock(),
	})
}

// ListMyTeams returns all teams the user is currently active in.
func (s *Service) ListMyTeams(ctx context.Context, userID int64) ([]Detail, error) {
	return s.repo.ListMyTeams(ctx, userID)
}

// GetTeamForUser is the canonical /teams/:id read.
func (s *Service) GetTeamForUser(ctx context.Context, teamID, userID int64) (Detail, error) {
	return s.repo.GetTeamForUser(ctx, teamID, userID)
}

// JoinTeam — user joins by invite_code. Idempotent for existing members.
func (s *Service) JoinTeam(ctx context.Context, userID int64, inviteCode string) (Detail, error) {
	code := strings.TrimSpace(inviteCode)
	if code == "" {
		return Detail{}, ErrInvalidInviteCode
	}
	return s.repo.JoinTeam(ctx, JoinTeamParams{
		UserID:     userID,
		InviteCode: code,
		JoinedAt:   s.clock(),
	})
}

// ChangeMemberRole — only lead. Cannot demote self.
func (s *Service) ChangeMemberRole(ctx context.Context, callerID, teamID, targetUserID int64, newRole Role) error {
	if _, err := ParseRole(string(newRole)); err != nil {
		return err
	}
	mem, err := s.repo.GetMembership(ctx, teamID, callerID)
	if err != nil {
		return err
	}
	if !CanManageTeam(mem) {
		return ErrNotLead
	}
	if callerID == targetUserID && newRole != RoleLead {
		return ErrCannotDemoteSelf
	}
	return s.repo.ChangeMemberRole(ctx, ChangeMemberRoleParams{
		TeamID:  teamID,
		UserID:  targetUserID,
		NewRole: newRole,
	})
}

// RemoveMember — lead only. Cannot remove self (must transfer or archive).
func (s *Service) RemoveMember(ctx context.Context, callerID, teamID, targetUserID int64) error {
	mem, err := s.repo.GetMembership(ctx, teamID, callerID)
	if err != nil {
		return err
	}
	if !CanManageTeam(mem) {
		return ErrNotLead
	}
	if callerID == targetUserID {
		return ErrCannotRemoveSelfAsLead
	}
	return s.repo.RemoveMember(ctx, RemoveMemberParams{
		TeamID: teamID,
		UserID: targetUserID,
		At:     s.clock(),
	})
}

// LeaveTeam — anyone can leave, except the only lead.
//
// The "only lead" check is enforced by the repository: it counts other
// active leads atomically and returns ErrCannotLeaveAsOnlyLead when there is
// no replacement. The service layer trusts that signal and propagates it.
func (s *Service) LeaveTeam(ctx context.Context, callerID, teamID int64) error {
	mem, err := s.repo.GetMembership(ctx, teamID, callerID)
	if err != nil {
		return err
	}
	_ = mem // GetMembership is a pre-flight; the actual leave-or-refuse is in repo.
	return s.repo.LeaveTeam(ctx, teamID, callerID, s.clock())
}

// SetMyAIConsent — caller must be the same user as the target.
func (s *Service) SetMyAIConsent(ctx context.Context, callerID, teamID, targetUserID int64, consent bool) error {
	if err := EnsureCanChangeAIConsent(callerID, targetUserID); err != nil {
		return err
	}
	// Also confirm caller has an active membership before flipping the flag.
	if _, err := s.repo.GetMembership(ctx, teamID, callerID); err != nil {
		return err
	}
	return s.repo.SetAIConsent(ctx, teamID, targetUserID, consent)
}

// RegenerateInvite — lead only. Returns the new code.
func (s *Service) RegenerateInvite(ctx context.Context, callerID, teamID int64) (string, error) {
	mem, err := s.repo.GetMembership(ctx, teamID, callerID)
	if err != nil {
		return "", err
	}
	if !CanManageTeam(mem) {
		return "", ErrNotLead
	}
	code, err := generateInviteCode()
	if err != nil {
		return "", err
	}
	if err := s.repo.RegenerateInviteCode(ctx, teamID, code); err != nil {
		return "", err
	}
	return code, nil
}

// ArchiveTeam — lead only.
func (s *Service) ArchiveTeam(ctx context.Context, callerID, teamID int64) error {
	mem, err := s.repo.GetMembership(ctx, teamID, callerID)
	if err != nil {
		return err
	}
	if !CanManageTeam(mem) {
		return ErrNotLead
	}
	return s.repo.ArchiveTeam(ctx, teamID, s.clock())
}

// generateInviteCode returns a 12-char base32 (no padding, uppercase) code.
// Source: 10 bytes of crypto/rand → 16 base32 chars → first 12.
// Collision space is ~32^12 ≈ 1.15e18 — negligible at MVP scale.
func generateInviteCode() (string, error) {
	var b [10]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	enc := base32.StdEncoding.WithPadding(base32.NoPadding)
	out := strings.ToUpper(enc.EncodeToString(b[:]))
	return out[:12], nil
}
