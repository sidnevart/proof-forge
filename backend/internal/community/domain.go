// Package community owns the CommunitySpace aggregate — open/semi-open spaces
// for Telegram clubs and external communities running proof-based seasons.
package community

import (
	"crypto/rand"
	"encoding/base32"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

// MemberRole enumerates community-space roles.
type MemberRole string

const (
	RoleCommunityLeader MemberRole = "community_leader"
	RoleMember          MemberRole = "member"
)

// MemberStatus mirrors the DB CHECK constraint.
type MemberStatus string

const (
	MemberStatusActive  MemberStatus = "active"
	MemberStatusLeft    MemberStatus = "left"
	MemberStatusRemoved MemberStatus = "removed"
)

var (
	ErrCommunityNotFound = errors.New("community space not found")
	ErrSlugTaken         = errors.New("slug already taken")
	ErrAlreadyMember     = errors.New("already a member")
	ErrNotMember         = errors.New("not a member")
	ErrInvalidCommunity  = errors.New("invalid community space")
	ErrInvalidInviteCode = errors.New("invalid invite code")
)

// CommunitySpace is the persisted aggregate.
type CommunitySpace struct {
	ID          int64
	WorkspaceID *int64
	OwnerUserID int64
	Name        string
	Slug        string
	Description string
	InviteCode  string
	IsPublic    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Membership is the persisted row of community_memberships.
type Membership struct {
	ID                int64
	CommunitySpaceID  int64
	UserID            int64
	Role              MemberRole
	Status            MemberStatus
	JoinedAt          time.Time
	LeftAt            *time.Time
}

// ValidateName enforces length constraints.
func ValidateName(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return fmt.Errorf("%w: name is empty", ErrInvalidCommunity)
	}
	if utf8.RuneCountInString(trimmed) > 80 {
		return fmt.Errorf("%w: name too long", ErrInvalidCommunity)
	}
	return nil
}

// GenerateInviteCode produces a short, human-friendly code: 2-char prefix + 4 random chars.
// E.g. "ml-X7K2A".
func GenerateInviteCode(prefix string) (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("rand: %w", err)
	}
	// base32 without padding, upper case
	code := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b)[:6]
	return fmt.Sprintf("%s-%s", prefix[:2], code), nil
}
