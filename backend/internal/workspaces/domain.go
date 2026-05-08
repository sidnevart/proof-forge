// Package workspaces owns the Workspace aggregate — the top-level container
// that groups teams (teamspaces) under a single organisation or community.
package workspaces

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

// WorkspaceType distinguishes corporate organisations from open communities.
type WorkspaceType string

const (
	WorkspaceTypeOrganization WorkspaceType = "organization"
	WorkspaceTypeCommunity    WorkspaceType = "community"
)

var (
	ErrWorkspaceNotFound   = errors.New("workspace not found")
	ErrSlugTaken           = errors.New("slug already taken")
	ErrInvalidWorkspace    = errors.New("invalid workspace")
	ErrForbidden           = errors.New("forbidden")
	ErrWorkspaceFrozen     = errors.New("workspace is frozen")
)

var slugRe = regexp.MustCompile(`^[a-z0-9][a-z0-9\-]{1,48}[a-z0-9]$`)

// Workspace is the persisted aggregate row.
type Workspace struct {
	ID           int64
	OwnerUserID  int64
	Name         string
	Slug         string
	Type         WorkspaceType
	IsActive     bool
	FrozenAt     *time.Time
	FrozenReason *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// IsFrozen reports whether the workspace has been frozen by a platform admin.
func (w *Workspace) IsFrozen() bool { return w.FrozenAt != nil }

// ValidateName checks the name constraints from the migration.
func ValidateName(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return fmt.Errorf("%w: name is empty", ErrInvalidWorkspace)
	}
	if utf8.RuneCountInString(trimmed) > 80 {
		return fmt.Errorf("%w: name exceeds 80 chars", ErrInvalidWorkspace)
	}
	return nil
}

// ValidateSlug checks the slug format from the migration CHECK constraint.
func ValidateSlug(slug string) error {
	if !slugRe.MatchString(slug) {
		return fmt.Errorf("%w: slug must be 3-50 lowercase alphanumeric + hyphens, no leading/trailing hyphen", ErrInvalidWorkspace)
	}
	return nil
}

// ParseType converts an external string to WorkspaceType.
func ParseType(s string) (WorkspaceType, error) {
	switch s {
	case string(WorkspaceTypeOrganization), string(WorkspaceTypeCommunity):
		return WorkspaceType(s), nil
	default:
		return "", fmt.Errorf("%w: unknown type %q", ErrInvalidWorkspace, s)
	}
}
