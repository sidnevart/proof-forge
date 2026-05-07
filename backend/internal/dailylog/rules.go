package dailylog

import (
	"time"
)

// HolidayChecker answers whether a given local date is a non-working holiday.
// The default implementation covers Russian federal holidays.
type HolidayChecker interface {
	IsHoliday(t time.Time) bool
}

// RussianHolidayChecker is the MVP holiday set.
type RussianHolidayChecker struct{}

func (RussianHolidayChecker) IsHoliday(t time.Time) bool {
	// Simple static list for 2025-2026. Production can swap in a proper library.
	switch t.Format("2006-01-02") {
	// 2025
	case "2025-01-01", "2025-01-02", "2025-01-03", "2025-01-06", "2025-01-07", "2025-01-08",
		"2025-05-01", "2025-05-02", "2025-05-08", "2025-05-09",
		"2025-06-12", "2025-06-13",
		"2025-11-03", "2025-11-04",
		"2025-12-31",
	// 2026
		"2026-01-01", "2026-01-02", "2026-01-05", "2026-01-06", "2026-01-07", "2026-01-08",
		"2026-02-23",
		"2026-03-08", "2026-03-09",
		"2026-05-01", "2026-05-04", "2026-05-11",
		"2026-06-12",
		"2026-11-04",
		"2026-12-31":
		return true
	}
	return false
}

// IsWorkingDay returns true for Mon-Fri that are not holidays.
func IsWorkingDay(t time.Time, hc HolidayChecker) bool {
	wd := t.Weekday()
	if wd == time.Saturday || wd == time.Sunday {
		return false
	}
	if hc != nil && hc.IsHoliday(t) {
		return false
	}
	return true
}

// ComputeStreakUpdate calculates the new streak state after recording an entry.
//
// Rules:
//   - Logged  → increment current; update best if current > best.
//   - Frozen  → preserve current; do NOT update best.
//   - Skipped → reset current to 0.
//   - Missed  → reset current to 0.
//
// The caller must already have verified that the day is a working day for
// streak purposes (non-working days never reach this function).
func ComputeStreakUpdate(old Streak, status EntryStatus, date time.Time) StreakUpdate {
	su := StreakUpdate{
		OldCurrent: old.Current,
		OldBest:    old.Best,
		Reason:     string(status),
	}

	switch status {
	case StatusLogged:
		su.NewCurrent = old.Current + 1
		su.NewBest = old.Best
		if su.NewCurrent > su.NewBest {
			su.NewBest = su.NewCurrent
			su.IsNewRecord = true
		}
	case StatusFrozen:
		su.NewCurrent = old.Current
		su.NewBest = old.Best
	case StatusSkipped, StatusMissed:
		su.NewCurrent = 0
		su.NewBest = old.Best
	}

	return su
}

// ShouldBreakStreakOnMiss decides whether a missed working day breaks the streak.
// It always returns true for working days; weekends and holidays are filtered
// by the caller (rollover worker).
func ShouldBreakStreakOnMiss(date time.Time, hc HolidayChecker) bool {
	return IsWorkingDay(date, hc)
}

// FreezeAllowed checks the per-user/per-team monthly limit.
const MaxFreezesPerMonth = 2

func FreezeAllowed(used int) error {
	if used >= MaxFreezesPerMonth {
		return ErrFreezeLimitReached
	}
	return nil
}

// MonthReset checks whether the streak state's last update was in a different
// calendar month than the given date, signalling that freezes_used_this_month
// should be reset.
func MonthReset(lastUpdated time.Time, now time.Time) bool {
	return lastUpdated.Year() != now.Year() || lastUpdated.Month() != now.Month()
}

// ConsecutiveWorkingDays counts how many working days back from `today` have
// active (logged or frozen) streak entries. Used for streak-audit logic and
// potential heatmap generation.
func ConsecutiveWorkingDays(today time.Time, entries []Entry, hc HolidayChecker) int {
	count := 0
	date := today
	for {
		if !IsWorkingDay(date, hc) {
			date = date.AddDate(0, 0, -1)
			continue
		}
		found := false
		for _, e := range entries {
			if e.LogDate.Equal(date) && (e.Status == StatusLogged || e.Status == StatusFrozen) {
				found = true
				break
			}
		}
		if !found {
			break
		}
		count++
		date = date.AddDate(0, 0, -1)
	}
	return count
}
