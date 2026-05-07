package teams

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PgUniqueViolation is the PostgreSQL SQLSTATE for "unique_violation".
// We map specific constraints to domain errors below.
const pgUniqueViolation = "23505"

// PostgresRepository is the production Repository backed by pgxpool.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository constructs the production repo.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// CreateTeam atomically inserts the team row and the lead's membership.
// Returns Detail with my_role=lead, member_count=1.
func (r *PostgresRepository) CreateTeam(ctx context.Context, params CreateTeamParams) (Detail, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Detail{}, fmt.Errorf("begin team tx: %w", err)
	}
	defer tx.Rollback(ctx)

	const insertTeam = `
		INSERT INTO teams (lead_user_id, name, invite_code, member_limit, ai_mode, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $6)
		RETURNING id, lead_user_id, name, invite_code, member_limit, ai_mode, created_at, updated_at, archived_at
	`
	var team Team
	if err := tx.QueryRow(ctx, insertTeam,
		params.LeadUserID,
		params.Name,
		params.InviteCode,
		params.MemberLimit,
		string(params.AIMode),
		params.CreatedAt,
	).Scan(
		&team.ID,
		&team.LeadUserID,
		&team.Name,
		&team.InviteCode,
		&team.MemberLimit,
		&team.AIMode,
		&team.CreatedAt,
		&team.UpdatedAt,
		&team.ArchivedAt,
	); err != nil {
		return Detail{}, fmt.Errorf("insert team: %w", err)
	}

	const insertMembership = `
		INSERT INTO team_memberships
			(team_id, user_id, role, status, ai_consent, timezone, joined_at)
		VALUES ($1, $2, 'lead', 'active', FALSE, 'Europe/Moscow', $3)
		RETURNING id, team_id, user_id, role, status, ai_consent, timezone, joined_at, left_at
	`
	var mem Membership
	if err := tx.QueryRow(ctx, insertMembership, team.ID, params.LeadUserID, params.CreatedAt).Scan(
		&mem.ID,
		&mem.TeamID,
		&mem.UserID,
		&mem.Role,
		&mem.Status,
		&mem.AIConsent,
		&mem.Timezone,
		&mem.JoinedAt,
		&mem.LeftAt,
	); err != nil {
		return Detail{}, fmt.Errorf("insert lead membership: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return Detail{}, fmt.Errorf("commit team tx: %w", err)
	}

	return Detail{Team: team, MyMembership: mem, MemberCount: 1}, nil
}

// ListMyTeams returns all teams where the user has an ACTIVE membership.
// Member count is computed inline via correlated subquery — small N, no JOIN
// magic needed.
func (r *PostgresRepository) ListMyTeams(ctx context.Context, userID int64) ([]Detail, error) {
	const query = `
		SELECT
			t.id, t.lead_user_id, t.name, t.invite_code, t.member_limit, t.ai_mode,
			t.created_at, t.updated_at, t.archived_at,
			m.id, m.team_id, m.user_id, m.role, m.status, m.ai_consent,
			m.timezone, m.joined_at, m.left_at,
			(
				SELECT COUNT(*) FROM team_memberships mc
				WHERE mc.team_id = t.id AND mc.status = 'active'
			) AS member_count
		FROM teams t
		JOIN team_memberships m ON m.team_id = t.id AND m.user_id = $1 AND m.status = 'active'
		ORDER BY t.created_at DESC
	`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list my teams: %w", err)
	}
	defer rows.Close()

	out := []Detail{}
	for rows.Next() {
		d, err := scanDetail(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// GetTeamForUser is the canonical /teams/:id read.
func (r *PostgresRepository) GetTeamForUser(ctx context.Context, teamID, userID int64) (Detail, error) {
	const query = `
		SELECT
			t.id, t.lead_user_id, t.name, t.invite_code, t.member_limit, t.ai_mode,
			t.created_at, t.updated_at, t.archived_at,
			m.id, m.team_id, m.user_id, m.role, m.status, m.ai_consent,
			m.timezone, m.joined_at, m.left_at,
			(SELECT COUNT(*) FROM team_memberships mc WHERE mc.team_id = t.id AND mc.status = 'active')
		FROM teams t
		JOIN team_memberships m ON m.team_id = t.id AND m.user_id = $2 AND m.status = 'active'
		WHERE t.id = $1
	`
	row := r.pool.QueryRow(ctx, query, teamID, userID)
	d, err := scanDetail(row.Scan)
	if errors.Is(err, pgx.ErrNoRows) {
		// Either team doesn't exist or user is not an active member — both
		// surface as ErrNotMember from the user's perspective.
		return Detail{}, ErrNotMember
	}
	if err != nil {
		return Detail{}, fmt.Errorf("get team for user: %w", err)
	}
	return d, nil
}

// JoinTeam — resolve invite_code → team, lock the row, check capacity,
// insert (or re-activate) membership, all atomically. Idempotent for
// active members.
func (r *PostgresRepository) JoinTeam(ctx context.Context, params JoinTeamParams) (Detail, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Detail{}, fmt.Errorf("begin join tx: %w", err)
	}
	defer tx.Rollback(ctx)

	const lockTeam = `
		SELECT id, lead_user_id, name, invite_code, member_limit, ai_mode,
		       created_at, updated_at, archived_at
		FROM teams
		WHERE invite_code = $1
		FOR UPDATE
	`
	var team Team
	if err := tx.QueryRow(ctx, lockTeam, params.InviteCode).Scan(
		&team.ID,
		&team.LeadUserID,
		&team.Name,
		&team.InviteCode,
		&team.MemberLimit,
		&team.AIMode,
		&team.CreatedAt,
		&team.UpdatedAt,
		&team.ArchivedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Detail{}, ErrInvalidInviteCode
		}
		return Detail{}, fmt.Errorf("lock team: %w", err)
	}
	if team.ArchivedAt != nil {
		return Detail{}, ErrTeamArchived
	}

	// Check existing membership for idempotence / re-activation.
	const existingQuery = `
		SELECT id, team_id, user_id, role, status, ai_consent, timezone, joined_at, left_at
		FROM team_memberships
		WHERE team_id = $1 AND user_id = $2
	`
	var existing Membership
	hasExisting := true
	if err := tx.QueryRow(ctx, existingQuery, team.ID, params.UserID).Scan(
		&existing.ID,
		&existing.TeamID,
		&existing.UserID,
		&existing.Role,
		&existing.Status,
		&existing.AIConsent,
		&existing.Timezone,
		&existing.JoinedAt,
		&existing.LeftAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			hasExisting = false
		} else {
			return Detail{}, fmt.Errorf("check existing membership: %w", err)
		}
	}

	const countActive = `SELECT COUNT(*) FROM team_memberships WHERE team_id = $1 AND status = 'active'`
	var activeCount int
	if err := tx.QueryRow(ctx, countActive, team.ID).Scan(&activeCount); err != nil {
		return Detail{}, fmt.Errorf("count active: %w", err)
	}

	var mem Membership
	switch {
	case hasExisting && existing.Status == MembershipStatusActive:
		// Idempotent — return current state.
		mem = existing
	case hasExisting && existing.Status == MembershipStatusRemoved:
		return Detail{}, ErrNotMember
	case hasExisting:
		// Re-activate as member, but only if capacity allows.
		if activeCount >= team.MemberLimit {
			return Detail{}, ErrTeamFull
		}
		const reactivate = `
			UPDATE team_memberships
			   SET status = 'active', role = 'member', joined_at = $2, left_at = NULL
			 WHERE id = $1
			RETURNING id, team_id, user_id, role, status, ai_consent, timezone, joined_at, left_at
		`
		if err := tx.QueryRow(ctx, reactivate, existing.ID, params.JoinedAt).Scan(
			&mem.ID,
			&mem.TeamID,
			&mem.UserID,
			&mem.Role,
			&mem.Status,
			&mem.AIConsent,
			&mem.Timezone,
			&mem.JoinedAt,
			&mem.LeftAt,
		); err != nil {
			return Detail{}, fmt.Errorf("reactivate membership: %w", err)
		}
		activeCount++
	default:
		if activeCount >= team.MemberLimit {
			return Detail{}, ErrTeamFull
		}
		const insert = `
			INSERT INTO team_memberships
				(team_id, user_id, role, status, ai_consent, timezone, joined_at)
			VALUES ($1, $2, 'member', 'active', FALSE, 'Europe/Moscow', $3)
			RETURNING id, team_id, user_id, role, status, ai_consent, timezone, joined_at, left_at
		`
		if err := tx.QueryRow(ctx, insert, team.ID, params.UserID, params.JoinedAt).Scan(
			&mem.ID,
			&mem.TeamID,
			&mem.UserID,
			&mem.Role,
			&mem.Status,
			&mem.AIConsent,
			&mem.Timezone,
			&mem.JoinedAt,
			&mem.LeftAt,
		); err != nil {
			return Detail{}, translateUniqueViolation(err)
		}
		activeCount++
	}

	if err := tx.Commit(ctx); err != nil {
		return Detail{}, fmt.Errorf("commit join tx: %w", err)
	}
	return Detail{Team: team, MyMembership: mem, MemberCount: activeCount}, nil
}

// ChangeMemberRole performs the UPDATE. Partial unique index uq_team_one_lead
// rejects the case "promote to lead while another active lead exists". We
// translate that into ErrMustHaveLead.
//
// Service layer guards demoting self / leaving team without lead. Repo also
// performs a defensive "must keep at least one lead" check on demotion of an
// existing lead, because the partial unique index only catches the "two leads"
// case, not the "zero leads" case.
func (r *PostgresRepository) ChangeMemberRole(ctx context.Context, params ChangeMemberRoleParams) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin role tx: %w", err)
	}
	defer tx.Rollback(ctx)

	const cur = `
		SELECT role, status FROM team_memberships
		WHERE team_id = $1 AND user_id = $2
	`
	var curRole, curStatus string
	if err := tx.QueryRow(ctx, cur, params.TeamID, params.UserID).Scan(&curRole, &curStatus); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotMember
		}
		return fmt.Errorf("get current membership: %w", err)
	}
	if curStatus != string(MembershipStatusActive) {
		return ErrNotMember
	}

	// Defensive: demoting the only lead would leave the team without a lead.
	if curRole == string(RoleLead) && params.NewRole != RoleLead {
		const otherLeadCount = `
			SELECT COUNT(*) FROM team_memberships
			WHERE team_id = $1 AND user_id <> $2
			  AND role = 'lead' AND status = 'active'
		`
		var n int
		if err := tx.QueryRow(ctx, otherLeadCount, params.TeamID, params.UserID).Scan(&n); err != nil {
			return fmt.Errorf("count other leads: %w", err)
		}
		if n == 0 {
			return ErrMustHaveLead
		}
	}

	const upd = `
		UPDATE team_memberships
		   SET role = $1
		 WHERE team_id = $2 AND user_id = $3 AND status = 'active'
	`
	if _, err := tx.Exec(ctx, upd, string(params.NewRole), params.TeamID, params.UserID); err != nil {
		return translateRoleViolation(err)
	}
	return tx.Commit(ctx)
}

// RemoveMember marks the membership as 'removed'.
func (r *PostgresRepository) RemoveMember(ctx context.Context, params RemoveMemberParams) error {
	const upd = `
		UPDATE team_memberships
		   SET status = 'removed', left_at = $3
		 WHERE team_id = $1 AND user_id = $2 AND status = 'active'
	`
	tag, err := r.pool.Exec(ctx, upd, params.TeamID, params.UserID, params.At)
	if err != nil {
		return fmt.Errorf("remove member: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotMember
	}
	return nil
}

// LeaveTeam refuses if the caller is the only active lead of the team.
// Otherwise marks status='left'.
func (r *PostgresRepository) LeaveTeam(ctx context.Context, teamID, userID int64, at time.Time) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin leave tx: %w", err)
	}
	defer tx.Rollback(ctx)

	const cur = `
		SELECT role FROM team_memberships
		WHERE team_id = $1 AND user_id = $2 AND status = 'active'
	`
	var role string
	if err := tx.QueryRow(ctx, cur, teamID, userID).Scan(&role); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotMember
		}
		return fmt.Errorf("get role for leave: %w", err)
	}

	if role == string(RoleLead) {
		const otherLeads = `
			SELECT COUNT(*) FROM team_memberships
			WHERE team_id = $1 AND user_id <> $2
			  AND role = 'lead' AND status = 'active'
		`
		var n int
		if err := tx.QueryRow(ctx, otherLeads, teamID, userID).Scan(&n); err != nil {
			return fmt.Errorf("count other leads: %w", err)
		}
		if n == 0 {
			return ErrCannotLeaveAsOnlyLead
		}
	}

	const upd = `
		UPDATE team_memberships
		   SET status = 'left', left_at = $3
		 WHERE team_id = $1 AND user_id = $2 AND status = 'active'
	`
	if _, err := tx.Exec(ctx, upd, teamID, userID, at); err != nil {
		return fmt.Errorf("leave team: %w", err)
	}
	return tx.Commit(ctx)
}

// SetAIConsent flips ai_consent on a membership row.
func (r *PostgresRepository) SetAIConsent(ctx context.Context, teamID, userID int64, consent bool) error {
	const upd = `
		UPDATE team_memberships
		   SET ai_consent = $3
		 WHERE team_id = $1 AND user_id = $2 AND status = 'active'
	`
	tag, err := r.pool.Exec(ctx, upd, teamID, userID, consent)
	if err != nil {
		return fmt.Errorf("set ai_consent: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotMember
	}
	return nil
}

// RegenerateInviteCode replaces the team's invite_code.
func (r *PostgresRepository) RegenerateInviteCode(ctx context.Context, teamID int64, newCode string) error {
	const upd = `
		UPDATE teams SET invite_code = $2, updated_at = NOW()
		 WHERE id = $1
	`
	tag, err := r.pool.Exec(ctx, upd, teamID, newCode)
	if err != nil {
		return translateUniqueViolation(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrTeamNotFound
	}
	return nil
}

// ArchiveTeam stamps archived_at.
func (r *PostgresRepository) ArchiveTeam(ctx context.Context, teamID int64, at time.Time) error {
	const upd = `UPDATE teams SET archived_at = $2, updated_at = $2 WHERE id = $1 AND archived_at IS NULL`
	tag, err := r.pool.Exec(ctx, upd, teamID, at)
	if err != nil {
		return fmt.Errorf("archive team: %w", err)
	}
	if tag.RowsAffected() == 0 {
		// Either team doesn't exist or already archived — both caller-safe.
		return nil
	}
	return nil
}

// GetMembership returns the active membership or ErrNotMember.
func (r *PostgresRepository) GetMembership(ctx context.Context, teamID, userID int64) (Membership, error) {
	const query = `
		SELECT id, team_id, user_id, role, status, ai_consent, timezone, joined_at, left_at
		FROM team_memberships
		WHERE team_id = $1 AND user_id = $2 AND status = 'active'
	`
	var mem Membership
	err := r.pool.QueryRow(ctx, query, teamID, userID).Scan(
		&mem.ID,
		&mem.TeamID,
		&mem.UserID,
		&mem.Role,
		&mem.Status,
		&mem.AIConsent,
		&mem.Timezone,
		&mem.JoinedAt,
		&mem.LeftAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Membership{}, ErrNotMember
	}
	if err != nil {
		return Membership{}, fmt.Errorf("get membership: %w", err)
	}
	return mem, nil
}

// ──────────────────────────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────────────────────────

// scanDetail scans a row that contains a flat (team, membership, member_count)
// triple in the column order used by ListMyTeams / GetTeamForUser.
func scanDetail(scan func(...any) error) (Detail, error) {
	var d Detail
	err := scan(
		&d.Team.ID,
		&d.Team.LeadUserID,
		&d.Team.Name,
		&d.Team.InviteCode,
		&d.Team.MemberLimit,
		&d.Team.AIMode,
		&d.Team.CreatedAt,
		&d.Team.UpdatedAt,
		&d.Team.ArchivedAt,
		&d.MyMembership.ID,
		&d.MyMembership.TeamID,
		&d.MyMembership.UserID,
		&d.MyMembership.Role,
		&d.MyMembership.Status,
		&d.MyMembership.AIConsent,
		&d.MyMembership.Timezone,
		&d.MyMembership.JoinedAt,
		&d.MyMembership.LeftAt,
		&d.MemberCount,
	)
	if err != nil {
		return Detail{}, err
	}
	return d, nil
}

// translateUniqueViolation maps pg unique_violation on team_memberships
// to a meaningful domain error.
func translateUniqueViolation(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != pgUniqueViolation {
		return fmt.Errorf("insert membership: %w", err)
	}
	switch pgErr.ConstraintName {
	case "uq_team_one_lead":
		return ErrMustHaveLead
	case "team_memberships_team_id_user_id_key":
		// Two concurrent inserts for the same (team, user) — one wins, the
		// other surfaces as already-member which is the safe meaning here.
		return ErrAlreadyMember
	case "teams_invite_code_key":
		// Used by RegenerateInviteCode if a freshly-generated code happens
		// to collide with an existing team — caller should retry.
		return fmt.Errorf("invite code collision; retry")
	default:
		return fmt.Errorf("unique violation %s: %w", pgErr.ConstraintName, err)
	}
}

// translateRoleViolation handles partial unique idx hit during ChangeMemberRole.
func translateRoleViolation(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation && pgErr.ConstraintName == "uq_team_one_lead" {
		return ErrMustHaveLead
	}
	return fmt.Errorf("change role: %w", err)
}
