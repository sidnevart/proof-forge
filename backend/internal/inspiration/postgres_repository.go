package inspiration

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// ListPublicGoals returns public goal templates ordered by created_at desc.
// If similarToGoalID > 0 and that goal has an embedding, it orders by cosine similarity.
func (r *PostgresRepository) ListPublicGoals(ctx context.Context, p FeedParams) ([]PublicGoal, error) {
	orderBy := "g.created_at DESC"
	if p.SimilarToGoalID > 0 {
		orderBy = "(SELECT embedding FROM goals WHERE id = $5) <=> g.embedding"
	}

	query := fmt.Sprintf(`
		SELECT g.id, g.title,
		       COALESCE(g.proof_examples, ''),
		       COALESCE(g.category, ''),
		       COALESCE(CASE WHEN u.is_anonymous_public THEN NULL ELSE u.public_alias END, 'АНОНИМ'),
		       g.created_at
		FROM goals g
		JOIN users u ON u.id = g.owner_user_id
		WHERE g.is_public_template = TRUE
		  AND g.is_hidden = FALSE
		  AND ($1 = '' OR to_tsvector('russian', g.title) @@ plainto_tsquery('russian', $1))
		  AND ($2 = '' OR g.category = $2)
		  AND ($3 = 0 OR g.id < $3)
		ORDER BY %s
		LIMIT $4
	`, orderBy)

	limit := p.Limit
	if limit == 0 || limit > 48 {
		limit = 24
	}

	args := []any{p.Query, p.Category, p.Cursor, limit}
	if p.SimilarToGoalID > 0 {
		args = append(args, p.SimilarToGoalID)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list public goals: %w", err)
	}
	defer rows.Close()

	var results []PublicGoal
	for rows.Next() {
		var g PublicGoal
		if err := rows.Scan(&g.ID, &g.Title, &g.ProofExamples, &g.Category, &g.AuthorAlias, &g.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan public goal: %w", err)
		}
		results = append(results, g)
	}
	return results, rows.Err()
}

// ListPublicProofs returns public proof feed ordered by created_at desc.
func (r *PostgresRepository) ListPublicProofs(ctx context.Context, p FeedParams) ([]PublicProof, error) {
	orderBy := "ci.created_at DESC"
	if p.SimilarToGoalID > 0 {
		orderBy = "(SELECT embedding FROM goals WHERE id = $5) <=> g.embedding"
	}

	query := fmt.Sprintf(`
		SELECT ci.id,
		       g.title,
		       COALESCE(g.category, ''),
		       COALESCE(CASE WHEN u.is_anonymous_public THEN NULL ELSE u.public_alias END, 'АНОНИМ'),
		       g.current_streak_count,
		       COALESCE(ei.text_content, ''),
		       COALESCE(ei.external_url, ''),
		       ci.created_at
		FROM check_ins ci
		JOIN goals g ON g.id = ci.goal_id
		JOIN users u ON u.id = ci.owner_user_id
		LEFT JOIN evidence_items ei ON ei.check_in_id = ci.id
		    AND ei.id = ANY(ci.public_attachment_ids::bigint[])
		WHERE ci.is_public_example = TRUE
		  AND ci.is_hidden = FALSE
		  AND ci.status = 'approved'
		  AND ($1 = '' OR to_tsvector('russian', g.title) @@ plainto_tsquery('russian', $1))
		  AND ($2 = '' OR g.category = $2)
		  AND ($3 = 0 OR ci.id < $3)
		ORDER BY %s
		LIMIT $4
	`, orderBy)

	limit := p.Limit
	if limit == 0 || limit > 48 {
		limit = 24
	}

	args := []any{p.Query, p.Category, p.Cursor, limit}
	if p.SimilarToGoalID > 0 {
		args = append(args, p.SimilarToGoalID)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list public proofs: %w", err)
	}
	defer rows.Close()

	var results []PublicProof
	for rows.Next() {
		var pp PublicProof
		if err := rows.Scan(&pp.ID, &pp.GoalTitle, &pp.Category, &pp.AuthorAlias,
			&pp.Streak, &pp.TextContent, &pp.ExternalURL, &pp.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan public proof: %w", err)
		}
		results = append(results, pp)
	}
	return results, rows.Err()
}

// SetGoalVisibility toggles is_public_template on a goal owned by ownerID.
func (r *PostgresRepository) SetGoalVisibility(ctx context.Context, goalID, ownerID int64, isPublic bool) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE goals SET is_public_template = $1, updated_at = NOW() WHERE id = $2 AND owner_user_id = $3`,
		isPublic, goalID, ownerID)
	if err != nil {
		return fmt.Errorf("set goal visibility: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotAuthorized
	}
	return nil
}

// SetCheckInVisibility toggles is_public_example on a check-in owned by ownerID.
func (r *PostgresRepository) SetCheckInVisibility(ctx context.Context, checkInID, ownerID int64, isPublic bool, attachmentIDs []int64) error {
	idsStr := "{}"
	if len(attachmentIDs) > 0 {
		parts := make([]string, len(attachmentIDs))
		for i, id := range attachmentIDs {
			parts[i] = fmt.Sprintf("%d", id)
		}
		idsStr = "{" + strings.Join(parts, ",") + "}"
	}

	tag, err := r.pool.Exec(ctx,
		`UPDATE check_ins SET is_public_example = $1, public_attachment_ids = $2, updated_at = NOW()
		 WHERE id = $3 AND owner_user_id = $4`,
		isPublic, idsStr, checkInID, ownerID)
	if err != nil {
		return fmt.Errorf("set checkin visibility: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotAuthorized
	}
	return nil
}

// UpdateSharingPrefs updates user-level sharing settings.
func (r *PostgresRepository) UpdateSharingPrefs(ctx context.Context, userID int64, in SharingPrefsInput) error {
	alias := strings.TrimSpace(in.Alias)
	var aliasArg any
	if alias != "" {
		aliasArg = alias
	}
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET share_default = $1, is_anonymous_public = $2, public_alias = $3, updated_at = NOW() WHERE id = $4`,
		in.ShareDefault, in.IsAnonymous, aliasArg, userID)
	return err
}

// CreateReport inserts an inspiration report.
func (r *PostgresRepository) CreateReport(ctx context.Context, reporterID int64, in ReportInput) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO inspiration_reports (reporter_user_id, target_kind, target_id, reason) VALUES ($1, $2, $3, $4)`,
		reporterID, in.Kind, in.ID, in.Reason)
	return err
}

// ListCheckInsForUserCircles returns a page of check-ins from all circles the actor belongs to.
func (r *PostgresRepository) ListCheckInsForUserCircles(ctx context.Context, actorID int64, p CircleFeedParams) ([]CircleFeedItem, error) {
	limit := p.Limit
	if limit == 0 || limit > 48 {
		limit = 24
	}

	rows, err := r.pool.Query(ctx, `
		SELECT ci.id,
		       ci.goal_id,
		       ci.owner_user_id,
		       g.title,
		       COALESCE(g.category, ''),
		       g.circle_id,
		       c.name,
		       COALESCE(CASE WHEN u.is_anonymous_public THEN NULL ELSE u.public_alias END, 'АНОНИМ'),
		       COALESCE(ei.text_content, ''),
		       COALESCE(ei.external_url, ''),
		       COALESCE(ci.submitted_at, ci.created_at),
		       ci.status,
		       g.current_streak_count
		FROM   check_ins ci
		JOIN   goals g ON g.id = ci.goal_id
		JOIN   circles c ON c.id = g.circle_id
		JOIN   circle_memberships cm
		       ON cm.circle_id = g.circle_id
		       AND cm.status = 'active'
		       AND cm.user_id = $1
		JOIN   users u ON u.id = ci.owner_user_id
		LEFT JOIN evidence_items ei ON ei.check_in_id = ci.id
		WHERE  ci.status IN ('submitted', 'approved')
		  AND  ($2 = 0 OR ci.id < $2)
		ORDER  BY COALESCE(ci.submitted_at, ci.created_at) DESC NULLS LAST, ci.id DESC
		LIMIT  $3
	`, actorID, p.Cursor, limit)
	if err != nil {
		return nil, fmt.Errorf("list circle feed: %w", err)
	}
	defer rows.Close()

	var results []CircleFeedItem
	for rows.Next() {
		var item CircleFeedItem
		if err := rows.Scan(
			&item.ID, &item.GoalID, &item.OwnerUserID,
			&item.GoalTitle, &item.Category,
			&item.CircleID, &item.CircleName,
			&item.AuthorAlias,
			&item.TextContent, &item.ExternalURL,
			&item.SubmittedAt, &item.Status,
			&item.Streak,
		); err != nil {
			return nil, fmt.Errorf("scan circle feed item: %w", err)
		}
		// can_approve: actor is not the owner AND proof is still submitted
		item.CanApprove = item.OwnerUserID != actorID && item.Status == "submitted"
		results = append(results, item)
	}
	return results, rows.Err()
}

// GoalNeedsEmbedding returns up to limit goals with is_public_template=true and embedding IS NULL.
type GoalEmbedRow struct {
	ID    int64
	Title string
	Smart string
}

func (r *PostgresRepository) GoalsNeedingEmbedding(ctx context.Context, limit int) ([]GoalEmbedRow, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, title, COALESCE(proof_examples, '') FROM goals
		 WHERE is_public_template = TRUE AND is_hidden = FALSE AND embedding IS NULL
		 LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []GoalEmbedRow
	for rows.Next() {
		var r GoalEmbedRow
		if err := rows.Scan(&r.ID, &r.Title, &r.Smart); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

func (r *PostgresRepository) SetGoalEmbedding(ctx context.Context, goalID int64, vec []float32) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE goals SET embedding = $1::vector WHERE id = $2`,
		float32SliceToSQL(vec), goalID)
	return err
}

// CheckInsNeedingEmbedding returns up to limit check-ins that are public + approved + no embedding.
type CheckInEmbedRow struct {
	ID          int64
	GoalTitle   string
	TextContent string
}

func (r *PostgresRepository) CheckInsNeedingEmbedding(ctx context.Context, limit int) ([]CheckInEmbedRow, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT ci.id, g.title, COALESCE(ei.text_content, '')
		 FROM check_ins ci
		 JOIN goals g ON g.id = ci.goal_id
		 LEFT JOIN evidence_items ei ON ei.check_in_id = ci.id LIMIT 1
		 WHERE ci.is_public_example = TRUE AND ci.is_hidden = FALSE AND ci.status = 'approved'
		   AND NOT EXISTS (SELECT 1 FROM check_ins c2 WHERE c2.id = ci.id AND c2.embedding IS NOT NULL)
		 LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []CheckInEmbedRow
	for rows.Next() {
		var r CheckInEmbedRow
		if err := rows.Scan(&r.ID, &r.GoalTitle, &r.TextContent); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

func (r *PostgresRepository) SetCheckInEmbedding(ctx context.Context, checkInID int64, vec []float32) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE check_ins SET embedding = $1::vector WHERE id = $2`,
		float32SliceToSQL(vec), checkInID)
	return err
}

// float32SliceToSQL formats a float32 slice as a pgvector literal "[f1,f2,...]".
func float32SliceToSQL(v []float32) string {
	if len(v) == 0 {
		return "[]"
	}
	b := strings.Builder{}
	b.WriteByte('[')
	for i, f := range v {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(fmt.Sprintf("%g", f))
	}
	b.WriteByte(']')
	return b.String()
}
