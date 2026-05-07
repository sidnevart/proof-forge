package analytics

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PartitionManager creates and drops analytics event partitions.
type PartitionManager struct {
	pool *pgxpool.Pool
	log  *slog.Logger
}

// NewPartitionManager constructs a manager.
func NewPartitionManager(pool *pgxpool.Pool, log *slog.Logger) *PartitionManager {
	return &PartitionManager{pool: pool, log: log}
}

// CreateFuturePartitions ensures the next N monthly partitions exist.
func (m *PartitionManager) CreateFuturePartitions(ctx context.Context, monthsAhead int) error {
	now := time.Now().UTC()
	for i := 0; i < monthsAhead; i++ {
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, i, 0)
		end := start.AddDate(0, 1, 0)
		name := fmt.Sprintf("analytics_events_%04d_%02d", start.Year(), start.Month())
		sql := fmt.Sprintf(
			`CREATE TABLE IF NOT EXISTS %s PARTITION OF analytics_events
			 FOR VALUES FROM ('%s') TO ('%s')`,
			name, start.Format("2006-01-02"), end.Format("2006-01-02"),
		)
		if _, err := m.pool.Exec(ctx, sql); err != nil {
			return fmt.Errorf("create partition %s: %w", name, err)
		}
		m.log.Info("analytics partition created", "partition", name, "start", start.Format("2006-01-02"), "end", end.Format("2006-01-02"))
	}
	return nil
}

// DropExpiredPartitions drops partitions older than retention.
func (m *PartitionManager) DropExpiredPartitions(ctx context.Context, retentionMonths int, dryRun bool) ([]string, error) {
	cutoff := time.Now().UTC().AddDate(0, -retentionMonths, 0)
	const sql = `
		SELECT tablename FROM pg_tables
		WHERE schemaname = 'public'
		  AND tablename ~ '^analytics_events_\d{4}_\d{2}$'
	`
	rows, err := m.pool.Query(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("list partitions: %w", err)
	}
	defer rows.Close()

	var dropped []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}
		var y, mo int
		if _, err := fmt.Sscanf(name, "analytics_events_%4d_%2d", &y, &mo); err != nil {
			continue
		}
		partitionStart := time.Date(y, time.Month(mo), 1, 0, 0, 0, 0, time.UTC)
		if partitionStart.Before(cutoff) {
			if dryRun {
				m.log.Info("analytics partition drop candidate (dry-run)", "partition", name)
			} else {
				dropSQL := fmt.Sprintf("DROP TABLE IF EXISTS %s", name)
				if _, err := m.pool.Exec(ctx, dropSQL); err != nil {
					m.log.Warn("failed to drop partition", "partition", name, "err", err)
					continue
				}
				m.log.Info("analytics partition dropped", "partition", name)
			}
			dropped = append(dropped, name)
		}
	}
	return dropped, rows.Err()
}
