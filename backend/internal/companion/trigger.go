package companion

import (
	"context"
	"fmt"
)

// ShouldFire determines whether a trigger may execute for a user now.
// It checks idempotency (already fired) and rate limits (per day / per week).
func (s *Service) ShouldFire(ctx context.Context, userID int64, feature Feature, triggerID string) (bool, error) {
	meta, ok := FeatureMeta[feature]
	if !ok {
		return false, fmt.Errorf("unknown companion feature: %s", feature)
	}

	// 1. Idempotency: already fired for this trigger_id?
	fired, err := s.repo.HasFired(ctx, userID, triggerID)
	if err != nil {
		return false, fmt.Errorf("companion should fire: check fired: %w", err)
	}
	if fired {
		return false, ErrAlreadyFired
	}

	// 2. Rate limits.
	if meta.MaxPerDay > 0 {
		count, err := s.repo.CountFiredToday(ctx, userID, feature)
		if err != nil {
			return false, fmt.Errorf("companion should fire: count today: %w", err)
		}
		if count >= meta.MaxPerDay {
			return false, ErrRateLimitExceeded
		}
	}
	if meta.MaxPerWeek > 0 {
		count, err := s.repo.CountFiredThisWeek(ctx, userID, feature)
		if err != nil {
			return false, fmt.Errorf("companion should fire: count week: %w", err)
		}
		if count >= meta.MaxPerWeek {
			return false, ErrRateLimitExceeded
		}
	}

	return true, nil
}

// RecordFired marks a trigger as executed.
func (s *Service) RecordFired(ctx context.Context, userID int64, feature Feature, triggerID string) error {
	return s.repo.RecordFired(ctx, userID, feature, triggerID)
}

// BuildTriggerID creates a deterministic trigger identifier.
func BuildTriggerID(feature Feature, userID int64, date string) string {
	return fmt.Sprintf("%s:%d:%s", feature, userID, date)
}
