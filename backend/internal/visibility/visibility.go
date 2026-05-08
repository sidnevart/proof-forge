// Package visibility implements role-based data filtering.
// Every handler that returns user data should resolve the observer role
// and apply the appropriate filter before writing the response.
package visibility

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sidnevart/proof-forge/backend/internal/authz"
	"github.com/sidnevart/proof-forge/backend/internal/goals"
)

// ObserverRole describes the relationship of the viewer to the subject's data.
type ObserverRole string

const (
	ObserverRoleSelf            ObserverRole = "self"
	ObserverRoleBuddy           ObserverRole = "buddy"
	ObserverRoleCircleMember    ObserverRole = "circle_member"
	ObserverRoleTeamspaceLead   ObserverRole = "teamspace_lead"
	ObserverRoleCommunityLeader ObserverRole = "community_leader"
	ObserverRoleWorkspaceOwner  ObserverRole = "workspace_owner"
	ObserverRolePlatformAdmin   ObserverRole = "platform_admin"
	ObserverRoleStranger        ObserverRole = "stranger"
)

// Service resolves visibility rules between an observer and a subject.
type Service struct {
	pool *pgxpool.Pool
	az   *authz.Authorizer
}

// New constructs a visibility Service.
func New(pool *pgxpool.Pool, az *authz.Authorizer) *Service {
	return &Service{pool: pool, az: az}
}

// ResolveRole determines the highest-privilege role the observer has relative
// to the subject. Roles are checked in priority order.
func (s *Service) ResolveRole(ctx context.Context, observerID, subjectID int64) (ObserverRole, error) {
	if observerID == subjectID {
		return ObserverRoleSelf, nil
	}

	if isAdmin, err := s.az.IsPlatformAdmin(ctx, observerID); err != nil {
		return ObserverRoleStranger, fmt.Errorf("resolve role: %w", err)
	} else if isAdmin {
		return ObserverRolePlatformAdmin, nil
	}

	if isBuddy, err := s.hasActivePact(ctx, observerID, subjectID); err != nil {
		return ObserverRoleStranger, fmt.Errorf("resolve role: %w", err)
	} else if isBuddy {
		return ObserverRoleBuddy, nil
	}

	if sameCircle, err := s.shareCircle(ctx, observerID, subjectID); err != nil {
		return ObserverRoleStranger, fmt.Errorf("resolve role: %w", err)
	} else if sameCircle {
		return ObserverRoleCircleMember, nil
	}

	if isLead, err := s.isTeamspaceLeadOver(ctx, observerID, subjectID); err != nil {
		return ObserverRoleStranger, fmt.Errorf("resolve role: %w", err)
	} else if isLead {
		return ObserverRoleTeamspaceLead, nil
	}

	if isLeader, err := s.isCommunityLeaderOver(ctx, observerID, subjectID); err != nil {
		return ObserverRoleStranger, fmt.Errorf("resolve role: %w", err)
	} else if isLeader {
		return ObserverRoleCommunityLeader, nil
	}

	return ObserverRoleStranger, nil
}

// ── DB helpers ────────────────────────────────────────────────────────

func (s *Service) hasActivePact(ctx context.Context, observerID, subjectID int64) (bool, error) {
	var ok bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM pacts p
			JOIN goals g ON g.id = p.goal_id
			WHERE p.status = 'active'
			  AND ((g.owner_user_id = $1 AND g.buddy_user_id = $2)
			    OR (g.owner_user_id = $2 AND g.buddy_user_id = $1))
		)`, observerID, subjectID).Scan(&ok)
	return ok, err
}

func (s *Service) shareCircle(ctx context.Context, observerID, subjectID int64) (bool, error) {
	var ok bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM circle_memberships a
			JOIN circle_memberships b ON a.circle_id = b.circle_id
			WHERE a.user_id = $1 AND b.user_id = $2
			  AND a.status = 'active' AND b.status = 'active'
		)`, observerID, subjectID).Scan(&ok)
	return ok, err
}

// isTeamspaceLeadOver checks if observer is a lead in any team where subject is a member.
func (s *Service) isTeamspaceLeadOver(ctx context.Context, observerID, subjectID int64) (bool, error) {
	var ok bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM team_memberships lead_m
			JOIN team_memberships sub_m ON sub_m.team_id = lead_m.team_id
			WHERE lead_m.user_id = $1 AND lead_m.role = 'lead'
			  AND sub_m.user_id = $2
		)`, observerID, subjectID).Scan(&ok)
	return ok, err
}

// isCommunityLeaderOver checks if observer is a community_leader in any community where subject is a member.
func (s *Service) isCommunityLeaderOver(ctx context.Context, observerID, subjectID int64) (bool, error) {
	var ok bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM community_memberships lm
			JOIN community_memberships sm ON sm.community_space_id = lm.community_space_id
			WHERE lm.user_id = $1 AND lm.role = 'community_leader' AND lm.status = 'active'
			  AND sm.user_id = $2 AND sm.status = 'active'
		)`, observerID, subjectID).Scan(&ok)
	return ok, err
}

// ── Filtered views ────────────────────────────────────────────────────

// GoalView is the role-filtered projection of a goal.
type GoalView struct {
	ID           int64   `json:"id"`
	Title        string  `json:"title"`
	MovementMode string  `json:"movement_mode"`
	Status       string  `json:"status"`
	Description  *string `json:"description,omitempty"`
}

// FilterGoal returns a role-appropriate view of a goal.
// Description is only visible to the owner and their buddy.
func FilterGoal(g goals.Goal, role ObserverRole) GoalView {
	view := GoalView{
		ID:           g.ID,
		Title:        g.Title,
		MovementMode: string(g.MovementMode),
		Status:       string(g.Status),
	}
	switch role {
	case ObserverRoleSelf, ObserverRoleBuddy, ObserverRolePlatformAdmin:
		view.Description = &g.Description
	}
	return view
}

// StreakView expresses streak visibility rules.
type StreakView struct {
	// Exact value (nil = hidden).
	Count *int `json:"streak_count,omitempty"`
	// Aggregated label ("active" / "inactive") when exact value is hidden.
	Label *string `json:"streak_label,omitempty"`
}

// FilterStreak returns streak data appropriate for the observer role.
func FilterStreak(streakCount int, role ObserverRole) StreakView {
	switch role {
	case ObserverRoleSelf, ObserverRoleBuddy, ObserverRolePlatformAdmin:
		return StreakView{Count: &streakCount}
	case ObserverRoleCircleMember, ObserverRoleTeamspaceLead,
		ObserverRoleCommunityLeader, ObserverRoleWorkspaceOwner:
		label := "active"
		if streakCount == 0 {
			label = "inactive"
		}
		return StreakView{Label: &label}
	default:
		return StreakView{}
	}
}
