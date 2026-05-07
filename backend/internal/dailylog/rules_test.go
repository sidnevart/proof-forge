package dailylog

import (
	"testing"
	"time"
)

func TestIsWorkingDay(t *testing.T) {
	hc := RussianHolidayChecker{}

	cases := []struct {
		date string
		want bool
	}{
		{"2026-05-04", false}, // Mon — transferred holiday (1 May)
		{"2026-05-05", true},  // Tue
		{"2026-05-08", true},  // Fri
		{"2026-05-09", false}, // Sat
		{"2026-05-10", false}, // Sun
		{"2026-01-01", false}, // holiday
		{"2026-06-12", false}, // holiday
	}

	for _, c := range cases {
		d, _ := time.Parse("2006-01-02", c.date)
		got := IsWorkingDay(d, hc)
		if got != c.want {
			t.Errorf("IsWorkingDay(%s) = %v, want %v", c.date, got, c.want)
		}
	}
}

func TestComputeStreakUpdate(t *testing.T) {
	mk := func(c, b int, last string) Streak {
		var d *time.Time
		if last != "" {
			pd, _ := time.Parse("2006-01-02", last)
			d = &pd
		}
		return Streak{Current: c, Best: b, LastActiveDate: d}
	}

	cases := []struct {
		name   string
		old    Streak
		status EntryStatus
		wantC  int
		wantB  int
		record bool
	}{
		{"logged extends", mk(4, 4, "2026-05-06"), StatusLogged, 5, 5, true},
		{"logged no record", mk(2, 5, "2026-05-06"), StatusLogged, 3, 5, false},
		{"frozen preserves", mk(3, 3, "2026-05-06"), StatusFrozen, 3, 3, false},
		{"skipped breaks", mk(5, 5, "2026-05-06"), StatusSkipped, 0, 5, false},
		{"missed breaks", mk(5, 5, "2026-05-06"), StatusMissed, 0, 5, false},
		{"logged from zero", mk(0, 0, ""), StatusLogged, 1, 1, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			date, _ := time.Parse("2006-01-02", "2026-05-07")
			su := ComputeStreakUpdate(c.old, c.status, date)
			if su.NewCurrent != c.wantC {
				t.Errorf("current = %d, want %d", su.NewCurrent, c.wantC)
			}
			if su.NewBest != c.wantB {
				t.Errorf("best = %d, want %d", su.NewBest, c.wantB)
			}
			if su.IsNewRecord != c.record {
				t.Errorf("record = %v, want %v", su.IsNewRecord, c.record)
			}
		})
	}
}

func TestConsecutiveWorkingDays(t *testing.T) {
	hc := RussianHolidayChecker{}

	entries := []Entry{
		{LogDate: mustDate("2026-05-07"), Status: StatusLogged},
		{LogDate: mustDate("2026-05-06"), Status: StatusLogged},
		{LogDate: mustDate("2026-05-05"), Status: StatusFrozen},
		// 2026-05-04 is Monday working day but missed → breaks
	}

	today := mustDate("2026-05-07")
	got := ConsecutiveWorkingDays(today, entries, hc)
	if got != 3 {
		t.Errorf("ConsecutiveWorkingDays = %d, want 3", got)
	}
}

func TestFreezeAllowed(t *testing.T) {
	if FreezeAllowed(0) != nil {
		t.Error("freeze 0 should be allowed")
	}
	if FreezeAllowed(1) != nil {
		t.Error("freeze 1 should be allowed")
	}
	if FreezeAllowed(2) == nil {
		t.Error("freeze 2 should NOT be allowed")
	}
}

func TestMonthReset(t *testing.T) {
	feb := time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC)
	mar := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	if !MonthReset(feb, mar) {
		t.Error("Feb -> Mar should reset")
	}
	if MonthReset(mar, mar) {
		t.Error("same month should NOT reset")
	}
}

func mustDate(s string) time.Time {
	d, _ := time.Parse("2006-01-02", s)
	return d
}
