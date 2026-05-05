package goals

import (
	"context"
	"database/sql"
	"encoding/json"
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

func (r *PostgresRepository) CreateGoalWithInvite(ctx context.Context, params CreateGoalParams) (GoalView, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return GoalView{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// If the caller did not pick an existing circle, create one in the same tx.
	// Goal -> circle binding is non-nullable in the schema, so we MUST end up
	// with a valid circle_id for every goal.
	if params.CircleID == nil {
		if params.AutoCircle == nil {
			return GoalView{}, fmt.Errorf("create goal: circle_id is required (AutoCircle params missing)")
		}
		circleID, err := r.insertAutoCircle(ctx, tx, params.OwnerID, *params.AutoCircle)
		if err != nil {
			return GoalView{}, err
		}
		params.CircleID = &circleID
	}

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

	if err := tx.Commit(ctx); err != nil {
		return GoalView{}, fmt.Errorf("commit goal tx: %w", err)
	}

	return GoalView{
		Goal:  goal,
		Buddy: buddy,
		Pact:  pact,
		Invite: Invite{
			ID:        invite.ID,
			Status:    invite.Status,
			ExpiresAt: invite.ExpiresAt,
		},
	}, nil
}

func (r *PostgresRepository) ListGoalsByOwner(ctx context.Context, ownerID int64) ([]GoalView, error) {
	const query = `
		SELECT
			g.id,
			g.circle_id,
			g.title,
			g.description,
			g.proof_examples,
			g.category,
			g.status,
			g.current_progress_health,
			g.current_streak_count,
			g.created_at,
			g.updated_at,
			b.id,
			b.email,
			b.display_name,
			p.id,
			p.status,
			p.accepted_at,
			i.id,
			i.status,
			i.expires_at
		FROM goals g
		JOIN users b ON b.id = g.buddy_user_id
		JOIN pacts p ON p.goal_id = g.id
		JOIN invites i ON i.goal_id = g.id
		WHERE g.owner_user_id = $1
		ORDER BY g.created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, ownerID)
	if err != nil {
		return nil, fmt.Errorf("query owner goals: %w", err)
	}
	defer rows.Close()

	goals := make([]GoalView, 0)
	for rows.Next() {
		var item GoalView
		var acceptedAt sql.NullTime
		var proofExamples, category sql.NullString
		if err := rows.Scan(
			&item.Goal.ID,
			&item.Goal.CircleID,
			&item.Goal.Title,
			&item.Goal.Description,
			&proofExamples,
			&category,
			&item.Goal.Status,
			&item.Goal.CurrentProgressHealth,
			&item.Goal.CurrentStreakCount,
			&item.Goal.CreatedAt,
			&item.Goal.UpdatedAt,
			&item.Buddy.ID,
			&item.Buddy.Email,
			&item.Buddy.DisplayName,
			&item.Pact.ID,
			&item.Pact.Status,
			&acceptedAt,
			&item.Invite.ID,
			&item.Invite.Status,
			&item.Invite.ExpiresAt,
		); err != nil {
			return nil, fmt.Errorf("scan owner goal: %w", err)
		}
		item.Goal.ProofExamples = proofExamples.String
		item.Goal.Category = category.String
		if acceptedAt.Valid {
			value := acceptedAt.Time
			item.Pact.AcceptedAt = &value
		}
		goals = append(goals, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate owner goals: %w", err)
	}

	return goals, nil
}

func (r *PostgresRepository) FindGoalRefineCache(ctx context.Context, draftHash string, minCreatedAt time.Time) (GoalRefineResponse, bool, error) {
	const query = `
		SELECT response
		FROM goal_refine_cache
		WHERE hash = $1
		  AND created_at >= $2
	`

	var raw json.RawMessage
	if err := r.pool.QueryRow(ctx, query, draftHash, minCreatedAt).Scan(&raw); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return GoalRefineResponse{}, false, nil
		}
		return GoalRefineResponse{}, false, fmt.Errorf("query refine cache: %w", err)
	}

	var response GoalRefineResponse
	if err := json.Unmarshal(raw, &response); err != nil {
		return GoalRefineResponse{}, false, fmt.Errorf("decode refine cache: %w", err)
	}

	return response, true, nil
}

func (r *PostgresRepository) SaveGoalRefineCache(ctx context.Context, draftHash string, response GoalRefineResponse) error {
	const query = `
		INSERT INTO goal_refine_cache (hash, response, created_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (hash) DO UPDATE
		SET response = EXCLUDED.response,
		    created_at = EXCLUDED.created_at
	`

	payload, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("encode refine cache response: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, draftHash, payload); err != nil {
		return fmt.Errorf("upsert refine cache: %w", err)
	}
	return nil
}

func (r *PostgresRepository) CountGoalRefineRequestsSince(ctx context.Context, userID int64, since time.Time) (int, error) {
	const query = `
		SELECT COUNT(*)
		FROM goal_refine_requests
		WHERE user_id = $1
		  AND created_at >= $2
	`

	var count int
	if err := r.pool.QueryRow(ctx, query, userID, since).Scan(&count); err != nil {
		return 0, fmt.Errorf("count refine requests: %w", err)
	}
	return count, nil
}

func (r *PostgresRepository) InsertGoalRefineRequest(ctx context.Context, params GoalRefineRequestLogParams) error {
	const query = `
		INSERT INTO goal_refine_requests (user_id, draft_hash)
		VALUES ($1, $2)
	`

	if _, err := r.pool.Exec(ctx, query, params.UserID, params.DraftHash); err != nil {
		return fmt.Errorf("insert refine request: %w", err)
	}
	return nil
}

func (r *PostgresRepository) IsCircleMember(ctx context.Context, circleID int64, userID int64) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM circle_memberships WHERE circle_id = $1 AND user_id = $2 AND status = 'active')`,
		circleID, userID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check circle membership: %w", err)
	}
	return exists, nil
}

// HasActiveGoalInCircle reports whether the given owner already has an active
// or pending goal inside the circle. The schema enforces this via a partial
// UNIQUE index, but the service uses this to fail with a friendly domain error
// before hitting the index violation.
func (r *PostgresRepository) HasActiveGoalInCircle(ctx context.Context, circleID int64, ownerID int64) (bool, error) {
	const query = `
		SELECT EXISTS(
			SELECT 1 FROM goals
			WHERE circle_id = $1
			  AND owner_user_id = $2
			  AND status IN ('pending_buddy_acceptance', 'active')
		)
	`
	var exists bool
	if err := r.pool.QueryRow(ctx, query, circleID, ownerID).Scan(&exists); err != nil {
		return false, fmt.Errorf("check active goal in circle: %w", err)
	}
	return exists, nil
}

func (r *PostgresRepository) IsCircleMemberByEmail(ctx context.Context, circleID int64, email string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM circle_memberships m
			JOIN users u ON u.id = m.user_id
			WHERE m.circle_id = $1
			  AND m.status = 'active'
			  AND u.email = $2
		)
	`, circleID, email).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check circle membership by email: %w", err)
	}
	return exists, nil
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
			owner.display_name,
			owner.email
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
		&rec.OwnerEmail,
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

// insertAutoCircle creates a new circle, its first season (7 days from now),
// and the owner membership inside the goal-creation transaction. This keeps
// «goal + circle + season + membership» atomic — if any later step fails the
// whole tx rolls back and we don't leave orphan circles behind.
func (r *PostgresRepository) insertAutoCircle(ctx context.Context, tx pgx.Tx, ownerID int64, params AutoCircleParams) (int64, error) {
	const insertCircle = `
		INSERT INTO circles (owner_user_id, name, invite_code, member_limit)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`
	var circleID int64
	if err := tx.QueryRow(ctx, insertCircle, ownerID, params.Name, params.InviteCode, params.MemberLimit).Scan(&circleID); err != nil {
		return 0, fmt.Errorf("insert auto circle: %w", err)
	}

	const insertSeason = `
		INSERT INTO circle_seasons (circle_id, status, starts_at, ends_at)
		VALUES ($1, 'active', $2, $3)
	`
	if _, err := tx.Exec(ctx, insertSeason, circleID, params.StartsAt, params.EndsAt); err != nil {
		return 0, fmt.Errorf("insert auto season: %w", err)
	}

	const insertMembership = `
		INSERT INTO circle_memberships (circle_id, user_id, status, role)
		VALUES ($1, $2, 'active', 'owner')
	`
	if _, err := tx.Exec(ctx, insertMembership, circleID, ownerID); err != nil {
		return 0, fmt.Errorf("insert owner membership: %w", err)
	}

	return circleID, nil
}

func (r *PostgresRepository) insertGoal(ctx context.Context, tx pgx.Tx, params CreateGoalParams, buddyID int64) (Goal, error) {
	const query = `
		INSERT INTO goals (
			owner_user_id,
			buddy_user_id,
			circle_id,
			title,
			description,
			proof_examples,
			category,
			status,
			current_progress_health,
			current_streak_count
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 0)
		RETURNING id, circle_id, title, description, proof_examples, category, status, current_progress_health, current_streak_count, created_at, updated_at
	`

	var goal Goal
	var proofExamplesOut, categoryOut sql.NullString
	if err := tx.QueryRow(
		ctx,
		query,
		params.OwnerID,
		buddyID,
		params.CircleID,
		params.Title,
		params.Description,
		params.ProofExamples,
		params.Category,
		params.GoalStatus,
		params.ProgressHealth,
	).Scan(
		&goal.ID,
		&goal.CircleID,
		&goal.Title,
		&goal.Description,
		&proofExamplesOut,
		&categoryOut,
		&goal.Status,
		&goal.CurrentProgressHealth,
		&goal.CurrentStreakCount,
		&goal.CreatedAt,
		&goal.UpdatedAt,
	); err != nil {
		return Goal{}, fmt.Errorf("insert goal: %w", err)
	}
	goal.ProofExamples = proofExamplesOut.String
	goal.Category = categoryOut.String

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
