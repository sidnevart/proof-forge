package circles

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) CreateCircle(ctx context.Context, params CreateCircleParams) (Detail, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Detail{}, fmt.Errorf("begin circle tx: %w", err)
	}
	defer tx.Rollback(ctx)

	const insertCircle = `
		INSERT INTO circles (owner_user_id, name, invite_code, member_limit)
		VALUES ($1, $2, $3, $4)
		RETURNING id, owner_user_id, name, invite_code, member_limit, created_at, updated_at
	`

	var circle Circle
	if err := tx.QueryRow(ctx, insertCircle, params.OwnerUserID, params.Name, params.InviteCode, params.MemberLimit).Scan(
		&circle.ID,
		&circle.OwnerUserID,
		&circle.Name,
		&circle.InviteCode,
		&circle.MemberLimit,
		&circle.CreatedAt,
		&circle.UpdatedAt,
	); err != nil {
		return Detail{}, fmt.Errorf("insert circle: %w", err)
	}

	const insertMembership = `
		INSERT INTO circle_memberships (circle_id, user_id, status, role)
		VALUES ($1, $2, $3, $4)
	`
	if _, err := tx.Exec(ctx, insertMembership, circle.ID, params.OwnerUserID, MembershipStatusActive, MemberRoleOwner); err != nil {
		return Detail{}, fmt.Errorf("insert owner membership: %w", err)
	}

	const insertSeason = `
		INSERT INTO circle_seasons (circle_id, status, starts_at, ends_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, circle_id, status, starts_at, ends_at
	`
	var season Season
	if err := tx.QueryRow(ctx, insertSeason, circle.ID, SeasonStatusActive, params.StartsAt, params.EndsAt).Scan(
		&season.ID,
		&season.CircleID,
		&season.Status,
		&season.StartsAt,
		&season.EndsAt,
	); err != nil {
		return Detail{}, fmt.Errorf("insert season: %w", err)
	}

	detail, err := r.getCircleForUserTx(ctx, tx, circle.ID, params.OwnerUserID)
	if err != nil {
		return Detail{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Detail{}, fmt.Errorf("commit circle tx: %w", err)
	}
	return detail, nil
}

func (r *PostgresRepository) ListCirclesForUser(ctx context.Context, userID int64) ([]Detail, error) {
	const query = `
		SELECT c.id
		FROM circles c
		JOIN circle_memberships m ON m.circle_id = c.id
		WHERE m.user_id = $1 AND m.status = 'active'
		ORDER BY c.created_at DESC
	`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list circle ids: %w", err)
	}
	defer rows.Close()

	items := make([]Detail, 0)
	for rows.Next() {
		var circleID int64
		if err := rows.Scan(&circleID); err != nil {
			return nil, fmt.Errorf("scan circle id: %w", err)
		}
		item, err := r.GetCircleForUser(ctx, circleID, userID)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate circle ids: %w", err)
	}
	return items, nil
}

func (r *PostgresRepository) GetCircleForUser(ctx context.Context, circleID int64, userID int64) (Detail, error) {
	return r.getCircleForUserTx(ctx, r.pool, circleID, userID)
}

func (r *PostgresRepository) JoinCircle(ctx context.Context, params JoinCircleParams) (Detail, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Detail{}, fmt.Errorf("begin join tx: %w", err)
	}
	defer tx.Rollback(ctx)

	const findCircle = `
		SELECT c.id, c.member_limit, EXISTS(
			SELECT 1 FROM circle_memberships m WHERE m.circle_id = c.id AND m.user_id = $2
		)
		FROM circles c
		WHERE c.invite_code = $1
	`

	var (
		circleID    int64
		memberLimit int
		alreadyIn   bool
	)
	err = tx.QueryRow(ctx, findCircle, params.InviteCode, params.UserID).Scan(&circleID, &memberLimit, &alreadyIn)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Detail{}, ErrCircleNotFound
		}
		return Detail{}, fmt.Errorf("find circle by invite code: %w", err)
	}
	if alreadyIn {
		return Detail{}, ErrAlreadyCircleMember
	}

	var count int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM circle_memberships WHERE circle_id = $1 AND status = 'active'`, circleID).Scan(&count); err != nil {
		return Detail{}, fmt.Errorf("count members: %w", err)
	}
	if count >= memberLimit {
		return Detail{}, ErrCircleIsFull
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO circle_memberships (circle_id, user_id, status, role, joined_at) VALUES ($1, $2, $3, $4, $5)`,
		circleID, params.UserID, MembershipStatusActive, MemberRoleObserver, params.JoinedAt,
	); err != nil {
		return Detail{}, fmt.Errorf("insert membership: %w", err)
	}

	detail, err := r.getCircleForUserTx(ctx, tx, circleID, params.UserID)
	if err != nil {
		return Detail{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Detail{}, fmt.Errorf("commit join tx: %w", err)
	}
	return detail, nil
}

func (r *PostgresRepository) ListStandingSnapshots(
	ctx context.Context,
	circleID int64,
	seasonStart time.Time,
	seasonEnd time.Time,
	weekStart time.Time,
	weekEnd time.Time,
	prevWeekStart time.Time,
	prevWeekEnd time.Time,
) ([]StandingSnapshot, error) {
	const query = `
		SELECT
			u.id,
			u.email,
			u.display_name,
			COUNT(DISTINCT g.id) FILTER (WHERE g.id IS NOT NULL) AS goals_count,
			COUNT(DISTINCT CASE
				WHEN ci.status = 'approved'
				  AND ci.approved_at >= $2
				  AND ci.approved_at < $3
				THEN DATE_TRUNC('week', ci.approved_at)
			END) AS approved_weeks,
			0 AS missed_weeks,
			COUNT(DISTINCT CASE
				WHEN ci.status = 'approved' AND ci.approved_at >= $4 AND ci.approved_at < $5 THEN ci.id
			END) > 0 AS has_approved_this_week,
			COUNT(DISTINCT CASE
				WHEN ci.status = 'submitted' AND ci.submitted_at >= $4 AND ci.submitted_at < $5 THEN ci.id
			END) > 0 AS has_submitted_this_week,
			COUNT(DISTINCT CASE
				WHEN ci.status = 'approved' AND ci.approved_at >= $6 AND ci.approved_at < $7 THEN ci.id
			END) > 0 AS has_approved_prev_week,
			COUNT(DISTINCT CASE
				WHEN ci.status = 'submitted' AND ci.submitted_at >= $6 AND ci.submitted_at < $7 THEN ci.id
			END) > 0 AS has_submitted_prev_week,
			MAX(COALESCE(ci.approved_at, ci.submitted_at, ci.created_at)) AS last_activity_at
		FROM circle_memberships m
		JOIN users u ON u.id = m.user_id
		LEFT JOIN goals g ON g.circle_id = m.circle_id AND g.owner_user_id = m.user_id
		LEFT JOIN check_ins ci ON ci.goal_id = g.id
		WHERE m.circle_id = $1
		  AND m.status = 'active'
		GROUP BY u.id, u.email, u.display_name
		ORDER BY u.email ASC
	`

	rows, err := r.pool.Query(ctx, query, circleID, seasonStart, seasonEnd, weekStart, weekEnd, prevWeekStart, prevWeekEnd)
	if err != nil {
		return nil, fmt.Errorf("query standings: %w", err)
	}
	defer rows.Close()

	items := make([]StandingSnapshot, 0)
	for rows.Next() {
		var (
			item             StandingSnapshot
			hasApproved      bool
			hasSubmitted     bool
			hasApprovedPrev  bool
			hasSubmittedPrev bool
			lastActivity     sql.NullTime
		)
		if err := rows.Scan(
			&item.UserID,
			&item.UserEmail,
			&item.DisplayName,
			&item.GoalsCount,
			&item.ApprovedWeeks,
			&item.MissedWeeks,
			&hasApproved,
			&hasSubmitted,
			&hasApprovedPrev,
			&hasSubmittedPrev,
			&lastActivity,
		); err != nil {
			return nil, fmt.Errorf("scan standing snapshot: %w", err)
		}
		item.HasApprovedThisWeek = hasApproved
		item.HasSubmittedThisWeek = hasSubmitted
		item.HasApprovedPrevWeek = hasApprovedPrev
		item.HasSubmittedPrevWeek = hasSubmittedPrev
		if lastActivity.Valid {
			value := lastActivity.Time
			item.LastActivityAt = &value
		}
		if item.HasApprovedThisWeek {
			item.CurrentStreak = 1
			if item.HasApprovedPrevWeek {
				item.CurrentStreak++
			}
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate standings: %w", err)
	}
	return items, nil
}

func (r *PostgresRepository) InviteToCircle(ctx context.Context, params InviteToCircleParams) (CircleInvitation, error) {
	const query = `
		INSERT INTO circle_invitations (circle_id, inviter_user_id, target_email, message)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (circle_id, target_email) DO UPDATE
			SET status = 'pending',
			    message = EXCLUDED.message,
			    updated_at = NOW()
		RETURNING id, circle_id, inviter_user_id, target_email, status, message, created_at
	`
	var inv CircleInvitation
	err := r.pool.QueryRow(ctx, query,
		params.CircleID, params.InviterUserID, params.TargetEmail, params.Message,
	).Scan(
		&inv.ID,
		&inv.CircleID,
		&inv.InviterID,
		&inv.TargetEmail,
		&inv.Status,
		&inv.Message,
		&inv.CreatedAt,
	)
	if err != nil {
		return CircleInvitation{}, fmt.Errorf("invite to circle: %w", err)
	}

	// fetch circle name
	if err := r.pool.QueryRow(ctx, `SELECT name FROM circles WHERE id = $1`, params.CircleID).Scan(&inv.CircleName); err != nil {
		return CircleInvitation{}, fmt.Errorf("fetch circle name: %w", err)
	}
	return inv, nil
}

func (r *PostgresRepository) ListInvitationsForUser(ctx context.Context, targetEmail string) ([]CircleInvitation, error) {
	const query = `
		SELECT ci.id, ci.circle_id, c.name, ci.inviter_user_id, ci.target_email, ci.status, ci.message, ci.created_at
		FROM circle_invitations ci
		JOIN circles c ON c.id = ci.circle_id
		WHERE ci.target_email = $1 AND ci.status = 'pending'
		ORDER BY ci.created_at DESC
	`
	rows, err := r.pool.Query(ctx, query, targetEmail)
	if err != nil {
		return nil, fmt.Errorf("list invitations: %w", err)
	}
	defer rows.Close()

	items := make([]CircleInvitation, 0)
	for rows.Next() {
		var inv CircleInvitation
		if err := rows.Scan(
			&inv.ID,
			&inv.CircleID,
			&inv.CircleName,
			&inv.InviterID,
			&inv.TargetEmail,
			&inv.Status,
			&inv.Message,
			&inv.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan invitation: %w", err)
		}
		items = append(items, inv)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate invitations: %w", err)
	}
	return items, nil
}

func (r *PostgresRepository) AcceptCircleInvitation(ctx context.Context, invitationID, userID int64) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin accept invitation tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// look up acting user's email
	var userEmail string
	if err := tx.QueryRow(ctx, `SELECT email FROM users WHERE id = $1`, userID).Scan(&userEmail); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrCircleInvitationNotFound
		}
		return fmt.Errorf("fetch user email: %w", err)
	}

	// lock the invitation row and validate
	var (
		circleID int64
		status   CircleInvitationStatus
		email    string
	)
	err = tx.QueryRow(ctx,
		`SELECT circle_id, status, target_email FROM circle_invitations WHERE id = $1 FOR UPDATE`,
		invitationID,
	).Scan(&circleID, &status, &email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrCircleInvitationNotFound
		}
		return fmt.Errorf("lock invitation: %w", err)
	}
	if status != CircleInvitationStatusPending {
		return ErrCircleInvitationAlreadyDone
	}
	if email != userEmail {
		return ErrCircleInvitationNotFound
	}

	// mark accepted
	if _, err := tx.Exec(ctx,
		`UPDATE circle_invitations SET status = 'accepted', updated_at = NOW() WHERE id = $1`,
		invitationID,
	); err != nil {
		return fmt.Errorf("update invitation status: %w", err)
	}

	// insert membership (ignore duplicate)
	if _, err := tx.Exec(ctx,
		`INSERT INTO circle_memberships (circle_id, user_id, status, role, joined_at)
		 VALUES ($1, $2, 'active', 'observer', NOW())
		 ON CONFLICT DO NOTHING`,
		circleID, userID,
	); err != nil {
		return fmt.Errorf("insert membership: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit accept invitation tx: %w", err)
	}
	return nil
}

func (r *PostgresRepository) DeclineCircleInvitation(ctx context.Context, invitationID, userID int64) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE circle_invitations SET status = 'declined', updated_at = NOW() WHERE id = $1`,
		invitationID,
	)
	if err != nil {
		return fmt.Errorf("decline invitation: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrCircleInvitationNotFound
	}
	return nil
}

type queryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func (r *PostgresRepository) getCircleForUserTx(ctx context.Context, q queryer, circleID int64, userID int64) (Detail, error) {
	const circleQuery = `
		SELECT c.id, c.owner_user_id, c.name, c.invite_code, c.member_limit, c.created_at, c.updated_at
		FROM circles c
		JOIN circle_memberships m ON m.circle_id = c.id
		WHERE c.id = $1 AND m.user_id = $2 AND m.status = 'active'
	`

	var detail Detail
	if err := q.QueryRow(ctx, circleQuery, circleID, userID).Scan(
		&detail.Circle.ID,
		&detail.Circle.OwnerUserID,
		&detail.Circle.Name,
		&detail.Circle.InviteCode,
		&detail.Circle.MemberLimit,
		&detail.Circle.CreatedAt,
		&detail.Circle.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Detail{}, ErrNotCircleMember
		}
		return Detail{}, fmt.Errorf("query circle: %w", err)
	}

	const membersQuery = `
		SELECT u.id, u.email, u.display_name, m.status, m.role, m.joined_at
		FROM circle_memberships m
		JOIN users u ON u.id = m.user_id
		WHERE m.circle_id = $1
		ORDER BY m.joined_at ASC, u.email ASC
	`
	memberRows, err := q.Query(ctx, membersQuery, circleID)
	if err != nil {
		return Detail{}, fmt.Errorf("query members: %w", err)
	}
	defer memberRows.Close()
	for memberRows.Next() {
		var member Member
		if err := memberRows.Scan(&member.UserID, &member.Email, &member.DisplayName, &member.Status, &member.Role, &member.JoinedAt); err != nil {
			return Detail{}, fmt.Errorf("scan member: %w", err)
		}
		detail.Members = append(detail.Members, member)
	}
	if err := memberRows.Err(); err != nil {
		return Detail{}, fmt.Errorf("iterate members: %w", err)
	}

	const seasonQuery = `
		SELECT id, circle_id, status, starts_at, ends_at
		FROM circle_seasons
		WHERE circle_id = $1 AND status = 'active'
		ORDER BY starts_at DESC
		LIMIT 1
	`
	if err := q.QueryRow(ctx, seasonQuery, circleID).Scan(
		&detail.ActiveSeason.ID,
		&detail.ActiveSeason.CircleID,
		&detail.ActiveSeason.Status,
		&detail.ActiveSeason.StartsAt,
		&detail.ActiveSeason.EndsAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Detail{}, ErrCircleNotFound
		}
		return Detail{}, fmt.Errorf("query active season: %w", err)
	}

	return detail, nil
}

func (r *PostgresRepository) GetSeason(ctx context.Context, circleID, seasonID int64) (Season, error) {
	var s Season
	err := r.pool.QueryRow(ctx,
		`SELECT id, circle_id, status, starts_at, ends_at FROM circle_seasons WHERE id = $1 AND circle_id = $2`,
		seasonID, circleID,
	).Scan(&s.ID, &s.CircleID, &s.Status, &s.StartsAt, &s.EndsAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Season{}, ErrSeasonNotFound
		}
		return Season{}, fmt.Errorf("get season: %w", err)
	}
	return s, nil
}

func (r *PostgresRepository) MarkSeasonCompleted(ctx context.Context, seasonID int64, action SeasonEndAction) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE circle_seasons SET status = 'completed', ended_action = $2, updated_at = NOW() WHERE id = $1`,
		seasonID, string(action),
	)
	if err != nil {
		return fmt.Errorf("mark season completed: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrSeasonNotFound
	}
	return nil
}

func (r *PostgresRepository) CreateExtendedSeason(ctx context.Context, circleID int64, startsAt, endsAt time.Time) (Season, error) {
	var s Season
	err := r.pool.QueryRow(ctx,
		`INSERT INTO circle_seasons (circle_id, status, starts_at, ends_at) VALUES ($1, 'active', $2, $3)
		 RETURNING id, circle_id, status, starts_at, ends_at`,
		circleID, startsAt, endsAt,
	).Scan(&s.ID, &s.CircleID, &s.Status, &s.StartsAt, &s.EndsAt)
	if err != nil {
		return Season{}, fmt.Errorf("create extended season: %w", err)
	}
	return s, nil
}

func (r *PostgresRepository) LazyCompleteExpiredSeason(ctx context.Context, circleID int64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE circle_seasons SET status = 'completed' WHERE circle_id = $1 AND status = 'active' AND ends_at < NOW()`,
		circleID,
	)
	if err != nil {
		return fmt.Errorf("lazy complete expired season: %w", err)
	}
	return nil
}

// IsActiveMemberOfGoalCircle returns true when userID is an active member of
// the circle that owns the given goal.
func (r *PostgresRepository) IsActiveMemberOfGoalCircle(ctx context.Context, userID, goalID int64) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM circle_memberships cm
			JOIN goals g ON g.circle_id = cm.circle_id
			WHERE cm.user_id = $1
			  AND g.id = $2
			  AND cm.status = 'active'
		)
	`, userID, goalID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("is active member of goal circle: %w", err)
	}
	return exists, nil
}
