package dailylog

import (
	"errors"
	"strings"
	"time"
)

// EntryStatus is the daily state for a single (user, team, date) tuple.
type EntryStatus string

const (
	StatusLogged  EntryStatus = "logged"
	StatusSkipped EntryStatus = "skipped"
	StatusFrozen  EntryStatus = "frozen"
	StatusMissed  EntryStatus = "missed"
)

// Entry is the canonical daily-log row.
type Entry struct {
	ID                  int64
	UserID              int64
	TeamID              int64
	LogDate             time.Time // local date, time component zeroed
	Status              EntryStatus
	TextContent         string
	HasArtifact         bool
	ExternalURL         *string
	StorageKey          *string
	MimeType            *string
	FileSizeBytes       *int64
	SubmittedAt         time.Time
	OverwrittenAt       *time.Time
	ConsumedInCheckInID *int64
	ConsumedAt          *time.Time
}

// Streak is the per-user/per-team streak aggregate.
type Streak struct {
	UserID                int64
	TeamID                int64
	Current               int
	Best                  int
	LastActiveDate        *time.Time // local date
	FreezesUsedThisMonth  int
	UpdatedAt             time.Time
}

// StreakUpdate is the result of applying an entry to streak rules.
type StreakUpdate struct {
	OldCurrent  int
	NewCurrent  int
	OldBest     int
	NewBest     int
	IsNewRecord bool
	Reason      string // e.g. "logged", "frozen", "missed", "skipped"
}

// SubmitInput is the use-case payload for logging a day.
type SubmitInput struct {
	UserID       int64
	TeamID       int64
	LogDate      time.Time // local date
	TextContent  string
	ExternalURL  *string
	ClientSource string    // "web", "telegram_bot"
}

// FreezeInput is the use-case payload for freezing a day.
type FreezeInput struct {
	UserID  int64
	TeamID  int64
	LogDate time.Time // local date
	Reason  string
}

// ListFilter controls the read path.
type ListFilter struct {
	UserID  int64
	TeamID  int64
	From    time.Time
	To      time.Time
	WithContent bool // true for author; false for lead/trusted heatmap
}

// Domain errors.
var (
	ErrEntryNotFound        = errors.New("daily log entry not found")
	ErrInvalidLogDate       = errors.New("log date must be today or yesterday in local timezone")
	ErrTextTooShort         = errors.New("text content must be at least 10 characters after trimming")
	ErrFutureDate           = errors.New("cannot log for a future date")
	ErrAlreadyLogged        = errors.New("entry already exists with terminal status")
	ErrFreezeLimitReached   = errors.New("freeze limit reached for this month")
	ErrNotMember            = errors.New("not a member of this team")
	ErrCannotReadContent    = errors.New("not authorized to read raw daily-log content")
	ErrInvalidTimezone      = errors.New("invalid timezone")
)

// ValidateText checks minimum length after trim.
func ValidateText(s string) error {
	if len(strings.TrimSpace(s)) < 10 {
		return ErrTextTooShort
	}
	return nil
}

// NormalizeDate zeroes the time component to enforce DATE semantics.
func NormalizeDate(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// IsValidLogDate accepts today or yesterday in the given location.
func IsValidLogDate(candidate time.Time, loc *time.Location, now time.Time) error {
	cand := NormalizeDate(candidate.In(loc))
	today := NormalizeDate(now.In(loc))
	yesterday := today.AddDate(0, 0, -1)

	if cand.After(today) {
		return ErrFutureDate
	}
	if !cand.Equal(today) && !cand.Equal(yesterday) {
		return ErrInvalidLogDate
	}
	return nil
}

// EntryContentEquals compares two entries for content equality (used in tests
// and overwrite decisions).
func EntryContentEquals(a, b *Entry) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.TextContent == b.TextContent &&
		ptrStrEq(a.ExternalURL, b.ExternalURL) &&
		ptrStrEq(a.StorageKey, b.StorageKey)
}

func ptrStrEq(a, b *string) bool {
	switch {
	case a == nil && b == nil:
		return true
	case a == nil || b == nil:
		return false
	default:
		return *a == *b
	}
}

