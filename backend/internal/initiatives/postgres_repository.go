package initiatives

import (
	"context"
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

func (r *PostgresRepository) Create(ctx context.Context, input CreateInput, creatorID int64) (Initiative, error) {
	var teamspaceID, communityID *int64
	if input.SpaceType == "teamspace" {
		teamspaceID = &input.SpaceID
	} else {
		communityID = &input.SpaceID
	}

	var out Initiative
	err := r.pool.QueryRow(ctx, `
		INSERT INTO initiatives (space_type, teamspace_id, community_space_id, creator_id, title, description, proof_criteria)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, space_type, teamspace_id, community_space_id, creator_id, title, description, proof_criteria, status, created_at`,
		input.SpaceType, teamspaceID, communityID, creatorID,
		input.Title, input.Description, input.ProofCriteria,
	).Scan(
		&out.ID, &out.SpaceType, &out.TeamspaceID, &out.CommunitySpaceID,
		&out.CreatorID, &out.Title, &out.Description, &out.ProofCriteria,
		&out.Status, &out.CreatedAt,
	)
	if err != nil {
		return Initiative{}, fmt.Errorf("create initiative: %w", err)
	}
	return out, nil
}

func (r *PostgresRepository) FindByID(ctx context.Context, id int64) (Initiative, error) {
	var out Initiative
	err := r.pool.QueryRow(ctx, `
		SELECT i.id, i.space_type, i.teamspace_id, i.community_space_id,
		       i.creator_id, i.title, i.description, i.proof_criteria, i.status, i.created_at,
		       COUNT(DISTINCT g.owner_user_id) AS participant_count,
		       COUNT(DISTINCT CASE WHEN ci.created_at::date = CURRENT_DATE THEN ci.id END) AS active_today,
		       COUNT(DISTINCT CASE WHEN ci.status = 'submitted' THEN ci.id END) AS pending_proof_count
		FROM initiatives i
		LEFT JOIN goals g ON g.initiative_id = i.id
		LEFT JOIN check_ins ci ON ci.goal_id = g.id
		WHERE i.id = $1
		GROUP BY i.id`, id,
	).Scan(
		&out.ID, &out.SpaceType, &out.TeamspaceID, &out.CommunitySpaceID,
		&out.CreatorID, &out.Title, &out.Description, &out.ProofCriteria,
		&out.Status, &out.CreatedAt,
		&out.ParticipantCount, &out.ActiveToday, &out.PendingProofCount,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Initiative{}, ErrInitiativeNotFound
	}
	if err != nil {
		return Initiative{}, fmt.Errorf("find initiative: %w", err)
	}
	return out, nil
}

func (r *PostgresRepository) ListBySpace(ctx context.Context, spaceType string, spaceID int64) ([]Initiative, error) {
	var col string
	if spaceType == "teamspace" {
		col = "teamspace_id"
	} else {
		col = "community_space_id"
	}

	rows, err := r.pool.Query(ctx, fmt.Sprintf(`
		SELECT i.id, i.space_type, i.teamspace_id, i.community_space_id,
		       i.creator_id, i.title, i.description, i.proof_criteria, i.status, i.created_at,
		       COUNT(DISTINCT g.owner_user_id) AS participant_count,
		       COUNT(DISTINCT CASE WHEN ci.created_at::date = CURRENT_DATE THEN ci.id END) AS active_today,
		       COUNT(DISTINCT CASE WHEN ci.status = 'submitted' THEN ci.id END) AS pending_proof_count
		FROM initiatives i
		LEFT JOIN goals g ON g.initiative_id = i.id
		LEFT JOIN check_ins ci ON ci.goal_id = g.id
		WHERE i.%s = $1 AND i.status = 'active'
		GROUP BY i.id
		ORDER BY i.created_at DESC`, col), spaceID)
	if err != nil {
		return nil, fmt.Errorf("list initiatives: %w", err)
	}
	defer rows.Close()

	var out []Initiative
	for rows.Next() {
		var ini Initiative
		if err := rows.Scan(
			&ini.ID, &ini.SpaceType, &ini.TeamspaceID, &ini.CommunitySpaceID,
			&ini.CreatorID, &ini.Title, &ini.Description, &ini.ProofCriteria,
			&ini.Status, &ini.CreatedAt,
			&ini.ParticipantCount, &ini.ActiveToday, &ini.PendingProofCount,
		); err != nil {
			return nil, fmt.Errorf("scan initiative: %w", err)
		}
		out = append(out, ini)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) JoinOrGet(ctx context.Context, initiativeID, userID int64) (JoinResult, error) {
	// Fetch initiative title + criteria for goal creation.
	var title, criteria string
	err := r.pool.QueryRow(ctx,
		`SELECT title, proof_criteria FROM initiatives WHERE id = $1`, initiativeID,
	).Scan(&title, &criteria)
	if err != nil {
		return JoinResult{}, fmt.Errorf("join fetch initiative: %w", err)
	}

	// Idempotent insert: if goal already exists for this user+initiative, return it.
	var goalID int64
	err = r.pool.QueryRow(ctx, `
		WITH ins AS (
			INSERT INTO goals (owner_user_id, title, description, status, initiative_id)
			VALUES ($1, $2, $3, 'active', $4)
			ON CONFLICT DO NOTHING
			RETURNING id
		)
		SELECT id FROM ins
		UNION ALL
		SELECT id FROM goals WHERE owner_user_id = $1 AND initiative_id = $4
		LIMIT 1`,
		userID, title, criteria, initiativeID,
	).Scan(&goalID)
	if err != nil {
		return JoinResult{}, fmt.Errorf("join or get goal: %w", err)
	}

	// Determine if we just created it by checking if it was very recently inserted.
	var created bool
	_ = r.pool.QueryRow(ctx,
		`SELECT created_at > NOW() - INTERVAL '3 seconds' FROM goals WHERE id = $1`, goalID,
	).Scan(&created)

	return JoinResult{GoalID: goalID, Created: created}, nil
}

func (r *PostgresRepository) PendingProofs(ctx context.Context, initiativeID, viewerID int64) ([]PendingProof, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT ci.id, ci.owner_user_id, COALESCE(u.display_name, u.email), ci.text_evidence, ci.created_at
		FROM check_ins ci
		JOIN goals g ON g.id = ci.goal_id
		JOIN users u ON u.id = ci.owner_user_id
		WHERE g.initiative_id = $1
		  AND ci.status = 'submitted'
		  AND ci.owner_user_id <> $2
		ORDER BY ci.created_at ASC`, initiativeID, viewerID)
	if err != nil {
		return nil, fmt.Errorf("pending proofs: %w", err)
	}
	defer rows.Close()

	var out []PendingProof
	for rows.Next() {
		var p PendingProof
		if err := rows.Scan(&p.CheckinID, &p.AuthorID, &p.AuthorName, &p.Content, &p.SubmittedAt); err != nil {
			return nil, fmt.Errorf("scan pending proof: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) CheckinAuthorID(ctx context.Context, checkinID int64) (int64, error) {
	var authorID int64
	err := r.pool.QueryRow(ctx,
		`SELECT owner_user_id FROM check_ins WHERE id = $1`, checkinID,
	).Scan(&authorID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrCheckinNotFound
	}
	return authorID, err
}

func (r *PostgresRepository) IsParticipant(ctx context.Context, initiativeID, userID int64) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM goals WHERE initiative_id = $1 AND owner_user_id = $2)`,
		initiativeID, userID,
	).Scan(&exists)
	return exists, err
}

func (r *PostgresRepository) Approve(ctx context.Context, checkinID, approverID int64, comment string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin approve tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	_, err = tx.Exec(ctx,
		`INSERT INTO initiative_approvals (checkin_id, approver_id, comment) VALUES ($1, $2, $3)`,
		checkinID, approverID, comment,
	)
	if err != nil {
		return fmt.Errorf("insert approval: %w", err)
	}

	tag, err := tx.Exec(ctx,
		`UPDATE check_ins SET status = 'approved', updated_at = NOW() WHERE id = $1 AND status = 'submitted'`,
		checkinID,
	)
	if err != nil {
		return fmt.Errorf("update checkin status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrCheckinNotFound
	}

	return tx.Commit(ctx)
}

func (r *PostgresRepository) Participants(ctx context.Context, initiativeID int64) ([]ParticipantProgress, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT u.id, COALESCE(u.display_name, u.email),
		       COUNT(ci.id) FILTER (WHERE ci.status = 'approved') AS proof_count
		FROM goals g
		JOIN users u ON u.id = g.owner_user_id
		LEFT JOIN check_ins ci ON ci.goal_id = g.id
		WHERE g.initiative_id = $1
		GROUP BY u.id, u.display_name, u.email
		ORDER BY proof_count DESC, u.id`, initiativeID)
	if err != nil {
		return nil, fmt.Errorf("participants: %w", err)
	}
	defer rows.Close()

	var out []ParticipantProgress
	for rows.Next() {
		var p ParticipantProgress
		if err := rows.Scan(&p.UserID, &p.DisplayName, &p.ProofCount); err != nil {
			return nil, fmt.Errorf("scan participant: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) Archive(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE initiatives SET status = 'archived', updated_at = NOW() WHERE id = $1`, id,
	)
	if err != nil {
		return fmt.Errorf("archive initiative: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrInitiativeNotFound
	}
	return nil
}
