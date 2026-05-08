// Package authz provides centralized authorization helpers for role checks.
// Space-level roles (teamspace, community) are always checked against the DB
// per request — they are not cached in sessions.
package authz

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrForbidden = errors.New("forbidden")

// Authorizer performs authorization checks against the database.
type Authorizer struct {
	pool *pgxpool.Pool
}

// New constructs an Authorizer backed by the given pool.
func New(pool *pgxpool.Pool) *Authorizer {
	return &Authorizer{pool: pool}
}

// IsPlatformAdmin reports whether the user has the global platform_admin flag.
func (a *Authorizer) IsPlatformAdmin(ctx context.Context, userID int64) (bool, error) {
	var ok bool
	err := a.pool.QueryRow(ctx,
		`SELECT is_platform_admin FROM users WHERE id = $1`,
		userID,
	).Scan(&ok)
	if err != nil {
		return false, fmt.Errorf("authz: check platform admin: %w", err)
	}
	return ok, nil
}

// WorkspaceOwnerRole returns the caller's role in a workspace.
// Returns "owner" if the caller is the workspace owner, "none" otherwise.
func (a *Authorizer) IsWorkspaceOwner(ctx context.Context, userID, workspaceID int64) (bool, error) {
	var ok bool
	err := a.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM workspaces WHERE id = $1 AND owner_user_id = $2)`,
		workspaceID, userID,
	).Scan(&ok)
	if err != nil {
		return false, fmt.Errorf("authz: check workspace owner: %w", err)
	}
	return ok, nil
}

// IsTeamspaceLead reports whether the user has the 'lead' role in the given team (teamspace).
func (a *Authorizer) IsTeamspaceLead(ctx context.Context, userID, teamspaceID int64) (bool, error) {
	var ok bool
	err := a.pool.QueryRow(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM team_memberships
			WHERE team_id = $1 AND user_id = $2 AND role = 'lead'
		)`,
		teamspaceID, userID,
	).Scan(&ok)
	if err != nil {
		return false, fmt.Errorf("authz: check teamspace lead: %w", err)
	}
	return ok, nil
}

// CanApproveProofs reports whether the user can approve proofs in the given teamspace
// (lead or trusted_approver role).
func (a *Authorizer) CanApproveProofs(ctx context.Context, userID, teamspaceID int64) (bool, error) {
	var ok bool
	err := a.pool.QueryRow(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM team_memberships
			WHERE team_id = $1 AND user_id = $2 AND role IN ('lead', 'trusted_approver')
		)`,
		teamspaceID, userID,
	).Scan(&ok)
	if err != nil {
		return false, fmt.Errorf("authz: check approve proofs: %w", err)
	}
	return ok, nil
}

// CanViewTeamAnalytics reports whether the user can view analytics for the given teamspace
// (lead or workspace owner or platform admin).
func (a *Authorizer) CanViewTeamAnalytics(ctx context.Context, userID, teamspaceID int64) (bool, error) {
	isLead, err := a.IsTeamspaceLead(ctx, userID, teamspaceID)
	if err != nil {
		return false, err
	}
	if isLead {
		return true, nil
	}
	isAdmin, err := a.IsPlatformAdmin(ctx, userID)
	return isAdmin, err
}

// IsCommunityLeader reports whether the user has the 'community_leader' role in the given community.
func (a *Authorizer) IsCommunityLeader(ctx context.Context, userID, communityID int64) (bool, error) {
	var ok bool
	err := a.pool.QueryRow(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM community_memberships
			WHERE community_space_id = $1 AND user_id = $2 AND role = 'community_leader' AND status = 'active'
		)`,
		communityID, userID,
	).Scan(&ok)
	if err != nil {
		return false, fmt.Errorf("authz: check community leader: %w", err)
	}
	return ok, nil
}
