package notifications

import (
	"context"
	"fmt"
	"time"
)

const maxNudgesPerDay = 3

// Policy enforces throttling and opt-out rules for notifications.
type Policy struct {
	repo *PostgresRepository
}

func NewPolicy(repo *PostgresRepository) *Policy {
	return &Policy{repo: repo}
}

// CanSendNudge returns true when the user is under the daily nudge limit and not in quiet mode.
func (p *Policy) CanSendNudge(ctx context.Context, userID int64) (bool, error) {
	quiet, err := p.repo.IsQuietToday(ctx, userID)
	if err != nil {
		return false, err
	}
	if quiet {
		return false, nil
	}

	count, err := p.repo.CountTodayNotifications(ctx, userID)
	if err != nil {
		return false, err
	}
	return count < maxNudgesPerDay, nil
}

// NudgeDedupKey returns the dedup key for a nudge notification on the given day.
func NudgeDedupKey(userID int64, kind string) string {
	today := time.Now().UTC().Format("2006-01-02")
	return fmt.Sprintf("%s_%d_%s", kind, userID, today)
}

// DigestDedupKey returns the dedup key for a daily digest for a circle.
func DigestDedupKey(circleID int64, date string) string {
	return fmt.Sprintf("digest_%d_%s", circleID, date)
}
