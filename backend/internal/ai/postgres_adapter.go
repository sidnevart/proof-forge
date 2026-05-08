package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresGoalReader implements GoalReader using a direct pgxpool connection.
type PostgresGoalReader struct {
	pool *pgxpool.Pool
}

func NewPostgresGoalReader(pool *pgxpool.Pool) *PostgresGoalReader {
	return &PostgresGoalReader{pool: pool}
}

// GetGoalText returns the goal's title and description as a single string.
func (r *PostgresGoalReader) GetGoalText(ctx context.Context, goalID int64, userID int64) (string, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT title, description FROM goals WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`,
		goalID, userID,
	)
	var title, desc string
	if err := row.Scan(&title, &desc); err != nil {
		return "", err
	}
	if desc == "" {
		return title, nil
	}
	return fmt.Sprintf("%s. %s", title, desc), nil
}

// GetRecentActivityText returns a short text summary of the user's last 5 approved check-ins.
func (r *PostgresGoalReader) GetRecentActivityText(ctx context.Context, userID int64) (string, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT g.title, e.text_content
		FROM check_ins ci
		JOIN goals g ON g.id = ci.goal_id
		LEFT JOIN evidence_items e ON e.check_in_id = ci.id AND e.kind = 'text'
		WHERE ci.user_id = $1 AND ci.status = 'approved'
		ORDER BY ci.approved_at DESC
		LIMIT 5
	`, userID)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var parts []string
	for rows.Next() {
		var goalTitle, text string
		if err := rows.Scan(&goalTitle, &text); err != nil {
			continue
		}
		if text != "" {
			parts = append(parts, fmt.Sprintf("[%s] %s", goalTitle, text))
		}
	}
	return strings.Join(parts, "\n"), nil
}

