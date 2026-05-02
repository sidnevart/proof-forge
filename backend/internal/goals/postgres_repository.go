package goals

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) CreateGoalWithInvite(ctx context.Context, params CreateGoalParams) (GoalView, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return GoalView{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	buddy, err := r.findOrCreateBuddy(ctx, tx, params.BuddyEmail, params.BuddyName)
	if err != nil {
		return GoalView{}, err
	}

	goal, err := r.insertGoal(ctx, tx, params, buddy.ID)
	if err != nil {
		return GoalView{}, err
	}

	pact, err := r.insertPact(ctx, tx, params, goal.ID, buddy.ID)
	if err != nil {
		return GoalView{}, err
	}

	invite, err := r.insertInvite(ctx, tx, params, goal.ID, pact.ID, buddy.ID)
	if err != nil {
		return GoalView{}, err
	}

	owner, err := r.findUserByID(ctx, tx, params.OwnerID)
	if err != nil {
		return GoalView{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return GoalView{}, fmt.Errorf("commit goal tx: %w", err)
	}

	return GoalView{
		Goal:  goal,
		Owner: owner,
		Buddy: buddy,
		Pact:  pact,
		Invite: Invite{
			ID:        invite.ID,
			Status:    invite.Status,
			ExpiresAt: invite.ExpiresAt,
		},
		Role: RoleOwner,
	}, nil
}

func (r *PostgresRepository) findUserByID(ctx context.Context, tx pgx.Tx, userID int64) (Buddy, error) {
	const query = `SELECT id, email, display_name FROM users WHERE id = $1`
	var u Buddy
	if err := tx.QueryRow(ctx, query, userID).Scan(&u.ID, &u.Email, &u.DisplayName); err != nil {
		return Buddy{}, fmt.Errorf("find user by id: %w", err)
	}
	return u, nil
}

// goalViewSelect is the shared projection for both list and single-goal queries.
// Includes owner data, buddy data, pact, invite, and a computed role column.
const goalViewSelect = `
	SELECT
		g.id, g.title, g.description, g.status,
		g.current_progress_health, g.current_streak_count,
		g.created_at, g.updated_at,
		o.id, o.email, o.display_name,
		b.id, b.email, b.display_name,
		p.id, p.status, p.accepted_at,
		i.id, i.status, i.expires_at,
		CASE WHEN g.owner_user_id = $1 THEN 'owner' ELSE 'buddy' END AS role
	FROM goals g
	JOIN users o ON o.id = g.owner_user_id
	JOIN users b ON b.id = g.buddy_user_id
	JOIN pacts p ON p.goal_id = g.id
	JOIN invites i ON i.goal_id = g.id
`

func scanGoalView(rows interface {
	Scan(...any) error
}) (GoalView, error) {
	var item GoalView
	var acceptedAt sql.NullTime
	if err := rows.Scan(
		&item.Goal.ID, &item.Goal.Title, &item.Goal.Description, &item.Goal.Status,
		&item.Goal.CurrentProgressHealth, &item.Goal.CurrentStreakCount,
		&item.Goal.CreatedAt, &item.Goal.UpdatedAt,
		&item.Owner.ID, &item.Owner.Email, &item.Owner.DisplayName,
		&item.Buddy.ID, &item.Buddy.Email, &item.Buddy.DisplayName,
		&item.Pact.ID, &item.Pact.Status, &acceptedAt,
		&item.Invite.ID, &item.Invite.Status, &item.Invite.ExpiresAt,
		&item.Role,
	); err != nil {
		return GoalView{}, err
	}
	if acceptedAt.Valid {
		value := acceptedAt.Time
		item.Pact.AcceptedAt = &value
	}
	return item, nil
}

func (r *PostgresRepository) ListGoalsForUser(ctx context.Context, userID int64) ([]GoalView, error) {
	query := goalViewSelect + `
		WHERE g.owner_user_id = $1 OR g.buddy_user_id = $1
		ORDER BY g.created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("query user goals: %w", err)
	}
	defer rows.Close()

	goals := make([]GoalView, 0)
	for rows.Next() {
		item, err := scanGoalView(rows)
		if err != nil {
			return nil, fmt.Errorf("scan user goal: %w", err)
		}
		goals = append(goals, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user goals: %w", err)
	}

	return goals, nil
}

func (r *PostgresRepository) FindGoalForUser(ctx context.Context, goalID, userID int64) (GoalView, error) {
	query := goalViewSelect + `
		WHERE g.id = $2 AND (g.owner_user_id = $1 OR g.buddy_user_id = $1)
	`

	row := r.pool.QueryRow(ctx, query, userID, goalID)
	item, err := scanGoalView(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return GoalView{}, ErrGoalNotFound
		}
		return GoalView{}, fmt.Errorf("find goal for user: %w", err)
	}
	return item, nil
}

func (r *PostgresRepository) FindInviteByToken(ctx context.Context, tokenHash string) (InviteRecord, error) {
	const query = `
		SELECT
			i.id,
			i.status,
			i.expires_at,
			i.goal_id,
			i.pact_id,
			i.inviter_user_id,
			i.invitee_user_id,
			invitee.email,
			g.title,
			g.status,
			owner.display_name
		FROM invites i
		JOIN users invitee ON invitee.id = i.invitee_user_id
		JOIN goals g ON g.id = i.goal_id
		JOIN users owner ON owner.id = i.inviter_user_id
		WHERE i.token_hash = $1
	`

	var rec InviteRecord
	err := r.pool.QueryRow(ctx, query, tokenHash).Scan(
		&rec.InviteID,
		&rec.InviteStatus,
		&rec.ExpiresAt,
		&rec.GoalID,
		&rec.PactID,
		&rec.InviterID,
		&rec.InviteeID,
		&rec.InviteeEmail,
		&rec.GoalTitle,
		&rec.GoalStatus,
		&rec.OwnerName,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return InviteRecord{}, ErrInviteNotFound
		}
		return InviteRecord{}, fmt.Errorf("find invite by token: %w", err)
	}

	return rec, nil
}

func (r *PostgresRepository) AcceptInvite(ctx context.Context, params AcceptInviteParams) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin accept tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Guard on status='pending' makes the accept atomic: concurrent requests that
	// race past the service-layer check will affect 0 rows and be rejected.
	tag, err := tx.Exec(ctx,
		`UPDATE invites SET status='accepted', accepted_at=$2 WHERE id=$1 AND status='pending'`,
		params.InviteID, params.AcceptedAt,
	)
	if err != nil {
		return fmt.Errorf("update invite status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrInviteAlreadyAccepted
	}

	if _, err := tx.Exec(ctx,
		`UPDATE pacts SET status='active', accepted_at=$2, updated_at=$2 WHERE id=$1`,
		params.PactID, params.AcceptedAt,
	); err != nil {
		return fmt.Errorf("update pact status: %w", err)
	}

	if _, err := tx.Exec(ctx,
		`UPDATE goals SET status='active', updated_at=$2 WHERE id=$1`,
		params.GoalID, params.AcceptedAt,
	); err != nil {
		return fmt.Errorf("update goal status: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit accept tx: %w", err)
	}

	return nil
}

func (r *PostgresRepository) findOrCreateBuddy(ctx context.Context, tx pgx.Tx, email string, displayName string) (Buddy, error) {
	const selectQuery = `
		SELECT id, email, display_name
		FROM users
		WHERE email = $1
	`

	var buddy Buddy
	err := tx.QueryRow(ctx, selectQuery, email).Scan(&buddy.ID, &buddy.Email, &buddy.DisplayName)
	switch {
	case err == nil:
		return buddy, nil
	case err != nil && !errors.Is(err, pgx.ErrNoRows):
		return Buddy{}, fmt.Errorf("select buddy: %w", err)
	}

	const insertQuery = `
		INSERT INTO users (email, display_name)
		VALUES ($1, $2)
		RETURNING id, email, display_name
	`
	if err := tx.QueryRow(ctx, insertQuery, email, displayName).Scan(&buddy.ID, &buddy.Email, &buddy.DisplayName); err != nil {
		return Buddy{}, fmt.Errorf("insert buddy: %w", err)
	}

	return buddy, nil
}

func (r *PostgresRepository) insertGoal(ctx context.Context, tx pgx.Tx, params CreateGoalParams, buddyID int64) (Goal, error) {
	const query = `
		INSERT INTO goals (
			owner_user_id,
			buddy_user_id,
			title,
			description,
			status,
			current_progress_health,
			current_streak_count
		)
		VALUES ($1, $2, $3, $4, $5, $6, 0)
		RETURNING id, title, description, status, current_progress_health, current_streak_count, created_at, updated_at
	`

	var goal Goal
	if err := tx.QueryRow(
		ctx,
		query,
		params.OwnerID,
		buddyID,
		params.Title,
		params.Description,
		params.GoalStatus,
		params.ProgressHealth,
	).Scan(
		&goal.ID,
		&goal.Title,
		&goal.Description,
		&goal.Status,
		&goal.CurrentProgressHealth,
		&goal.CurrentStreakCount,
		&goal.CreatedAt,
		&goal.UpdatedAt,
	); err != nil {
		return Goal{}, fmt.Errorf("insert goal: %w", err)
	}

	return goal, nil
}

func (r *PostgresRepository) insertPact(ctx context.Context, tx pgx.Tx, params CreateGoalParams, goalID int64, buddyID int64) (Pact, error) {
	const query = `
		INSERT INTO pacts (goal_id, owner_user_id, buddy_user_id, status)
		VALUES ($1, $2, $3, $4)
		RETURNING id, status, accepted_at
	`

	var pact Pact
	var acceptedAt sql.NullTime
	if err := tx.QueryRow(ctx, query, goalID, params.OwnerID, buddyID, params.PactStatus).Scan(
		&pact.ID,
		&pact.Status,
		&acceptedAt,
	); err != nil {
		return Pact{}, fmt.Errorf("insert pact: %w", err)
	}

	if acceptedAt.Valid {
		value := acceptedAt.Time
		pact.AcceptedAt = &value
	}

	return pact, nil
}

func (r *PostgresRepository) insertInvite(ctx context.Context, tx pgx.Tx, params CreateGoalParams, goalID int64, pactID int64, buddyID int64) (Invite, error) {
	const query = `
		INSERT INTO invites (goal_id, pact_id, inviter_user_id, invitee_user_id, token_hash, status, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, status, expires_at
	`

	var invite Invite
	if err := tx.QueryRow(
		ctx,
		query,
		goalID,
		pactID,
		params.OwnerID,
		buddyID,
		params.InviteTokenHash,
		params.InviteStatus,
		params.InviteExpiresAt,
	).Scan(
		&invite.ID,
		&invite.Status,
		&invite.ExpiresAt,
	); err != nil {
		return Invite{}, fmt.Errorf("insert invite: %w", err)
	}

	return invite, nil
}
