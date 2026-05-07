package dailylog

import (
	"context"
	"time"
)

// Repository is the storage contract for daily-log entries and streak state.
type Repository interface {
	// Entry read/write.
	UpsertEntry(ctx context.Context, e *Entry) (*Entry, error)
	GetEntry(ctx context.Context, userID, teamID int64, date time.Time) (*Entry, error)
	ListEntries(ctx context.Context, filter ListFilter) ([]Entry, error)

	// Streak read/write.
	GetStreak(ctx context.Context, userID, teamID int64) (*Streak, error)
	UpdateStreak(ctx context.Context, s *Streak) error

	// Atomic entry + streak upsert for mutation paths.
	UpsertEntryAndStreak(ctx context.Context, e *Entry, s *Streak) (*Entry, error)

	// Membership pre-flight (delegates to teams repository semantics).
	GetMembership(ctx context.Context, teamID, userID int64) (MembershipInfo, error)
}

// MembershipInfo is the subset of team membership needed for authorization.
type MembershipInfo struct {
	Role      string
	Status    string
	Timezone  string
	AIConsent bool
}
