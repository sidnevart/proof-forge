package teamproof

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository is the production storage for team proofs.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository constructs the production repo.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// CreateProof inserts a check_in row with team_id, evidence items, and consumes daily log entries.
func (r *PostgresRepository) CreateProof(ctx context.Context, p *TeamProof, evidence []EvidenceInput, consumedEntryIDs []int64) (*TeamProof, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin proof tx: %w", err)
	}
	defer tx.Rollback(ctx)

	const insertProof = `
		INSERT INTO check_ins (goal_id, owner_user_id, status, team_id, submitted_at, created_at, updated_at)
		VALUES ($1, $2, 'submitted', $3, $4, $4, $4)
		RETURNING id, goal_id, owner_user_id, status, team_id, submitted_at, approved_at, rejected_at, created_at, updated_at
	`
	var proof TeamProof
	if err := tx.QueryRow(ctx, insertProof, p.GoalID, p.OwnerUserID, p.TeamID, p.SubmittedAt).Scan(
		&proof.ID, &proof.GoalID, &proof.OwnerUserID, &proof.Status, &proof.TeamID,
		&proof.SubmittedAt, &proof.ApprovedAt, &proof.RejectedAt, &proof.CreatedAt, &proof.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("insert proof: %w", err)
	}

	// Insert evidence.
	for _, ev := range evidence {
		const ins = `
			INSERT INTO evidence_items (check_in_id, kind, external_url)
			VALUES ($1, $2, $3)
		`
		if _, err := tx.Exec(ctx, ins, proof.ID, ev.Kind, ev.ExternalURL); err != nil {
			return nil, fmt.Errorf("insert evidence: %w", err)
		}
	}

	// Consume daily log entries.
	if len(consumedEntryIDs) > 0 {
		const consume = `
			UPDATE daily_log_entries
			   SET consumed_in_check_in_id = $1,
			       consumed_at = $2
			 WHERE id = ANY($3)
			   AND user_id = $4
			   AND team_id = $5
			   AND consumed_in_check_in_id IS NULL
		`
		tag, err := tx.Exec(ctx, consume, proof.ID, p.SubmittedAt, consumedEntryIDs, p.OwnerUserID, p.TeamID)
		if err != nil {
			return nil, fmt.Errorf("consume entries: %w", err)
		}
		if tag.RowsAffected() != int64(len(consumedEntryIDs)) {
			return nil, ErrEntryAlreadyUsed
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit proof tx: %w", err)
	}
	return &proof, nil
}

// UpdateProofStatus sets the terminal status and approver role.
func (r *PostgresRepository) UpdateProofStatus(ctx context.Context, proofID int64, status string, approverRole string, now time.Time) error {
	var sql string
	switch status {
	case "approved":
		sql = `UPDATE check_ins SET status = 'approved', approved_at = $2, updated_at = $2, approver_role = $3 WHERE id = $1`
	case "rejected":
		sql = `UPDATE check_ins SET status = 'rejected', rejected_at = $2, updated_at = $2, approver_role = $3 WHERE id = $1`
	default:
		return fmt.Errorf("unsupported status transition: %s", status)
	}
	_, err := r.pool.Exec(ctx, sql, proofID, now, approverRole)
	return err
}

// GetProof fetches a proof by id.
func (r *PostgresRepository) GetProof(ctx context.Context, proofID int64) (*TeamProof, error) {
	const sql = `
		SELECT id, goal_id, owner_user_id, status, team_id,
		       submitted_at, approved_at, rejected_at, approver_role, created_at, updated_at
		FROM check_ins
		WHERE id = $1
	`
	var p TeamProof
	var role *string
	err := r.pool.QueryRow(ctx, sql, proofID).Scan(
		&p.ID, &p.GoalID, &p.OwnerUserID, &p.Status, &p.TeamID,
		&p.SubmittedAt, &p.ApprovedAt, &p.RejectedAt, &role, &p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrProofNotFound
	}
	if err != nil {
		return nil, err
	}
	p.ApproverRole = role
	return &p, nil
}

// ListFeed returns privacy-aware feed items for a team.
func (r *PostgresRepository) ListFeed(ctx context.Context, teamID int64, viewerUserID int64, role string, cursor int64, limit int) ([]FeedItem, error) {
	// Privacy: members see only approved. Author/lead/trusted see pending/rejected too.
	statusFilter := "ci.status = 'approved'"
	if role == "lead" || role == "trusted_approver" {
		statusFilter = "ci.status IN ('submitted', 'approved', 'rejected')"
	}

	sql := fmt.Sprintf(`
		SELECT
			ci.id,
			ci.owner_user_id,
			u.display_name,
			g.title,
			ci.approver_role,
			ci.status,
			ci.submitted_at,
			COALESCE(c.count, 0) AS comments_count
		FROM check_ins ci
		JOIN users u ON u.id = ci.owner_user_id
		JOIN goals g ON g.id = ci.goal_id
		LEFT JOIN (
			SELECT proof_id, COUNT(*) AS count FROM proof_comments GROUP BY proof_id
		) c ON c.proof_id = ci.id
		WHERE ci.team_id = $1 AND %s AND ci.id < $2
		ORDER BY ci.submitted_at DESC, ci.id DESC
		LIMIT $3
	`, statusFilter)
	rows, err := r.pool.Query(ctx, sql, teamID, cursor, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]FeedItem, 0)
	for rows.Next() {
		var f FeedItem
		var alias, goalTitle, status, approverRole string
		var submittedAt time.Time
		err := rows.Scan(
			&f.ID, &f.OwnerUserID, &alias, &goalTitle, &approverRole, &status, &submittedAt, &f.CommentsCount,
		)
		if err != nil {
			return nil, err
		}
		f.OwnerAlias = alias
		f.GoalTitle = goalTitle
		f.Status = status
		f.SubmittedAt = submittedAt
		f.CanApprove = (role == "lead" || role == "trusted_approver") && status == "submitted" && f.OwnerUserID != viewerUserID
		out = append(out, f)
	}
	return out, rows.Err()
}

// AddComment inserts a proof comment.
func (r *PostgresRepository) AddComment(ctx context.Context, proofID, userID int64, text string, now time.Time) (*Comment, error) {
	const sql = `
		INSERT INTO proof_comments (proof_id, user_id, text_content, created_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, proof_id, user_id, text_content, created_at
	`
	var c Comment
	err := r.pool.QueryRow(ctx, sql, proofID, userID, text, now).Scan(
		&c.ID, &c.ProofID, &c.UserID, &c.Text, &c.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// ListComments returns comments ordered by created_at desc.
func (r *PostgresRepository) ListComments(ctx context.Context, proofID int64) ([]Comment, error) {
	const sql = `
		SELECT pc.id, pc.proof_id, pc.user_id, u.display_name, pc.text_content, pc.created_at
		FROM proof_comments pc
		JOIN users u ON u.id = pc.user_id
		WHERE pc.proof_id = $1
		ORDER BY pc.created_at DESC
	`
	rows, err := r.pool.Query(ctx, sql, proofID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Comment, 0)
	for rows.Next() {
		var c Comment
		err := rows.Scan(&c.ID, &c.ProofID, &c.UserID, &c.UserAlias, &c.Text, &c.CreatedAt)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// GetMembership delegates to team_memberships.
func (r *PostgresRepository) GetMembership(ctx context.Context, teamID, userID int64) (MembershipInfo, error) {
	const sql = `
		SELECT role, status, timezone
		FROM team_memberships
		WHERE team_id = $1 AND user_id = $2 AND status = 'active'
	`
	var m MembershipInfo
	err := r.pool.QueryRow(ctx, sql, teamID, userID).Scan(&m.Role, &m.Status, &m.Timezone)
	if errors.Is(err, pgx.ErrNoRows) {
		return MembershipInfo{}, fmt.Errorf("not a member")
	}
	return m, err
}

// GetGoalTeamOwner validates that a goal belongs to a team and returns its owner.
func (r *PostgresRepository) GetGoalTeamOwner(ctx context.Context, goalID int64) (int64, int64, error) {
	const sql = `
		SELECT team_id, owner_user_id
		FROM goals
		WHERE id = $1 AND team_id IS NOT NULL
	`
	var teamID, ownerID int64
	err := r.pool.QueryRow(ctx, sql, goalID).Scan(&teamID, &ownerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, 0, ErrInvalidGoal
	}
	return teamID, ownerID, err
}

// GetDailyLogEntries verifies ownership and consumption status.
func (r *PostgresRepository) GetDailyLogEntries(ctx context.Context, entryIDs []int64, userID, teamID int64) ([]DailyLogEntryInfo, error) {
	const sql = `
		SELECT id, log_date, status, has_artifact, consumed_in_check_in_id
		FROM daily_log_entries
		WHERE id = ANY($1) AND user_id = $2 AND team_id = $3
	`
	rows, err := r.pool.Query(ctx, sql, entryIDs, userID, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]DailyLogEntryInfo, 0)
	for rows.Next() {
		var e DailyLogEntryInfo
		var consumed *int64
		err := rows.Scan(&e.ID, &e.LogDate, &e.Status, &e.HasArtifact, &consumed)
		if err != nil {
			return nil, err
		}
		e.ConsumedInCheckInID = consumed
		out = append(out, e)
	}
	return out, rows.Err()
}

// MarkEntriesConsumed is a no-op here because consumption happens inside CreateProof tx.
func (r *PostgresRepository) MarkEntriesConsumed(ctx context.Context, entryIDs []int64, proofID int64, now time.Time) error {
	return nil
}
