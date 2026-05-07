// Package teams owns the corporate-mode "team" aggregate that lives parallel
// to circles/. A team has exactly one lead, optional trusted_approvers, and
// regular members. Team-bound goals/check-ins are XOR-isolated from circle
// surfaces by a CHECK constraint at the DB level (migration 00012).
//
// All authorization decisions are concentrated in authz.go as pure functions
// — never inline a role-check inside a handler or service method.
package teams

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

// Role enumerates the three membership roles inside a team.
//
// RoleLead is the team owner — exactly one per team is enforced via partial
// unique index in migration 00011. RoleTrustedApprover has approval rights
// delegated by the lead. RoleMember can submit proofs and comment but cannot
// approve.
type Role string

const (
	RoleLead            Role = "lead"
	RoleTrustedApprover Role = "trusted_approver"
	RoleMember          Role = "member"
)

// MembershipStatus mirrors the BD CHECK constraint on team_memberships.status.
type MembershipStatus string

const (
	MembershipStatusActive  MembershipStatus = "active"
	MembershipStatusLeft    MembershipStatus = "left"
	MembershipStatusRemoved MembershipStatus = "removed"
)

// AIMode defines per-team privacy posture for the personalization layer
// (added in phase 3). On phase 0 the field is stored but unused.
//
//	AIModeOff          — никаких AI-вызовов, только шаблоны.
//	AIModeMetadataOnly — в LLM идут алиасы и заголовки, но не содержимое (default).
//	AIModeFull         — содержимое заметок и proof'ов передаётся в LLM
//	                     (требует ai_consent от каждого user'а индивидуально).
type AIMode string

const (
	AIModeOff          AIMode = "off"
	AIModeMetadataOnly AIMode = "metadata-only"
	AIModeFull         AIMode = "full"
)

const (
	DefaultMemberLimit = 25
	MaxTeamNameLength  = 80
)

// Domain errors. Map to HTTP via http_handler.go.
var (
	ErrTeamNotFound                = errors.New("team not found")
	ErrTeamArchived                = errors.New("team archived")
	ErrTeamFull                    = errors.New("team full")
	ErrAlreadyMember               = errors.New("already a member")
	ErrNotMember                   = errors.New("not a member of this team")
	ErrNotLead                     = errors.New("only the lead can do this")
	ErrMustHaveLead                = errors.New("team must have a lead")
	ErrCannotDemoteSelf            = errors.New("lead cannot demote themselves")
	ErrCannotRemoveSelfAsLead      = errors.New("lead cannot remove themselves; transfer first")
	ErrCannotLeaveAsOnlyLead       = errors.New("lead cannot leave the only-lead team; transfer first")
	ErrCannotChangeOthersAIConsent = errors.New("only the user themselves can change ai_consent")
	ErrInvalidInviteCode           = errors.New("invalid invite code")
	ErrInvalidTeamName             = errors.New("invalid team name")
	ErrInvalidRole                 = errors.New("invalid role")
	ErrInvalidAIMode               = errors.New("invalid ai_mode")
	ErrInvalidTimezone             = errors.New("invalid timezone")
)

// Team is the persisted aggregate row of `teams`.
type Team struct {
	ID          int64
	LeadUserID  int64
	Name        string
	InviteCode  string
	MemberLimit int
	AIMode      AIMode
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ArchivedAt  *time.Time
}

// Membership is the persisted row of `team_memberships`.
type Membership struct {
	ID        int64
	TeamID    int64
	UserID    int64
	Role      Role
	Status    MembershipStatus
	AIConsent bool
	Timezone  string
	JoinedAt  time.Time
	LeftAt    *time.Time
}

// Detail aggregates a team with the caller's own membership and a member count.
// It is the canonical read-model shape returned from /v1/teams and /v1/teams/:id.
type Detail struct {
	Team         Team
	MyMembership Membership
	MemberCount  int
}

// ParseRole validates and converts an external string into a Role. Used at
// the handler boundary when decoding request bodies and in repo when scanning
// rows that may have arbitrary text (defence-in-depth against schema drift).
func ParseRole(s string) (Role, error) {
	switch s {
	case string(RoleLead), string(RoleTrustedApprover), string(RoleMember):
		return Role(s), nil
	default:
		return "", fmt.Errorf("%w: %q", ErrInvalidRole, s)
	}
}

// ParseAIMode validates and converts an external string into an AIMode.
func ParseAIMode(s string) (AIMode, error) {
	switch s {
	case string(AIModeOff), string(AIModeMetadataOnly), string(AIModeFull):
		return AIMode(s), nil
	default:
		return "", fmt.Errorf("%w: %q", ErrInvalidAIMode, s)
	}
}

// ValidateTeamName enforces the length CHECK from migration 00011.
// Trims whitespace before counting runes to reject all-space names.
func ValidateTeamName(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return fmt.Errorf("%w: empty", ErrInvalidTeamName)
	}
	if utf8.RuneCountInString(trimmed) > MaxTeamNameLength {
		return fmt.Errorf("%w: longer than %d chars", ErrInvalidTeamName, MaxTeamNameLength)
	}
	return nil
}
