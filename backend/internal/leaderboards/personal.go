package leaderboards

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PersonalLeaderboard contains self-comparison data for a single user.
type PersonalLeaderboard struct {
	CurrentWeek   WeekComparison   `json:"current_week"`
	CurrentSeason SeasonComparison `json:"current_season"`
	Streak        StreakData        `json:"streak"`
	WeeklyHistory []WeekCount      `json:"weekly_history"`
}

type WeekComparison struct {
	ProofsCount int    `json:"proofs_count"`
	VsLastWeek  string `json:"vs_last_week"`
	Trend       string `json:"trend"`
}

type SeasonComparison struct {
	ProofsCount   int    `json:"proofs_count"`
	VsLastSeason  string `json:"vs_last_season"`
	Trend         string `json:"trend"`
}

type StreakData struct {
	CurrentWeeks       int  `json:"current_weeks"`
	PersonalRecordWeeks int  `json:"personal_record_weeks"`
	IsPersonalRecord   bool `json:"is_personal_record"`
}

type WeekCount struct {
	Week        string `json:"week"`
	ProofsCount int    `json:"proofs_count"`
}

func getPersonalLeaderboard(ctx context.Context, pool *pgxpool.Pool, userID int64) (*PersonalLeaderboard, error) {
	lb := &PersonalLeaderboard{}

	thisWeekStart := isoWeekStart(time.Now())
	lastWeekStart := thisWeekStart.AddDate(0, 0, -7)

	var thisWeek, lastWeek int
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM check_ins
		WHERE owner_user_id = $1 AND status = 'approved'
		  AND approved_at >= $2
	`, userID, thisWeekStart).Scan(&thisWeek); err != nil {
		return nil, fmt.Errorf("personal lb: this week: %w", err)
	}
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM check_ins
		WHERE owner_user_id = $1 AND status = 'approved'
		  AND approved_at >= $2 AND approved_at < $3
	`, userID, lastWeekStart, thisWeekStart).Scan(&lastWeek); err != nil {
		return nil, fmt.Errorf("personal lb: last week: %w", err)
	}

	lb.CurrentWeek = buildWeekComparison(thisWeek, lastWeek)

	// Season: last 90 days vs prior 90 days
	seasonStart := time.Now().AddDate(0, 0, -90)
	priorSeasonStart := time.Now().AddDate(0, 0, -180)
	var thisSeason, lastSeason int
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM check_ins
		WHERE owner_user_id = $1 AND status = 'approved' AND approved_at >= $2
	`, userID, seasonStart).Scan(&thisSeason); err != nil {
		return nil, fmt.Errorf("personal lb: season: %w", err)
	}
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM check_ins
		WHERE owner_user_id = $1 AND status = 'approved'
		  AND approved_at >= $2 AND approved_at < $3
	`, userID, priorSeasonStart, seasonStart).Scan(&lastSeason); err != nil {
		return nil, fmt.Errorf("personal lb: prior season: %w", err)
	}
	lb.CurrentSeason = buildSeasonComparison(thisSeason, lastSeason)

	// Streak and personal record
	streak, record, err := computeStreakAndRecord(ctx, pool, userID)
	if err != nil {
		return nil, err
	}
	lb.Streak = StreakData{
		CurrentWeeks:       streak,
		PersonalRecordWeeks: record,
		IsPersonalRecord:   streak > 0 && streak >= record,
	}

	// Weekly history: last 8 ISO weeks
	history, err := getWeeklyHistory(ctx, pool, userID, 8)
	if err != nil {
		return nil, err
	}
	lb.WeeklyHistory = history

	return lb, nil
}

func buildWeekComparison(thisWeek, lastWeek int) WeekComparison {
	delta := thisWeek - lastWeek
	var vs, trend string
	if lastWeek == 0 {
		vs = fmt.Sprintf("+%d", thisWeek)
		trend = "first_week"
	} else if delta > 0 {
		vs = fmt.Sprintf("+%d", delta)
		trend = "better"
	} else if delta < 0 {
		vs = fmt.Sprintf("%d", delta)
		trend = "worse"
	} else {
		vs = "0"
		trend = "same"
	}
	return WeekComparison{ProofsCount: thisWeek, VsLastWeek: vs, Trend: trend}
}

func buildSeasonComparison(thisSeason, lastSeason int) SeasonComparison {
	delta := thisSeason - lastSeason
	var vs, trend string
	if lastSeason == 0 {
		vs = fmt.Sprintf("+%d", thisSeason)
		trend = "first_season"
	} else if delta > 0 {
		vs = fmt.Sprintf("+%d", delta)
		trend = "better"
	} else if delta < 0 {
		vs = fmt.Sprintf("%d", delta)
		trend = "worse"
	} else {
		vs = "0"
		trend = "same"
	}
	return SeasonComparison{ProofsCount: thisSeason, VsLastSeason: vs, Trend: trend}
}

func computeStreakAndRecord(ctx context.Context, pool *pgxpool.Pool, userID int64) (streak, record int, err error) {
	// Get distinct ISO weeks with approved proofs, ordered descending.
	rows, err := pool.Query(ctx, `
		SELECT DISTINCT TO_CHAR(DATE_TRUNC('week', approved_at), 'IYYY-IW') as week
		FROM check_ins
		WHERE owner_user_id = $1 AND status = 'approved' AND approved_at IS NOT NULL
		ORDER BY week DESC
	`, userID)
	if err != nil {
		return 0, 0, fmt.Errorf("streak weeks: %w", err)
	}
	defer rows.Close()

	var weeks []string
	for rows.Next() {
		var w string
		if err := rows.Scan(&w); err != nil {
			return 0, 0, err
		}
		weeks = append(weeks, w)
	}
	if len(weeks) == 0 {
		return 0, 0, nil
	}

	currentISO := isoWeekLabel(time.Now())
	lastISO := isoWeekLabel(time.Now().AddDate(0, 0, -7))

	// Current streak: consecutive weeks starting from current or last week.
	streak = 0
	if weeks[0] == currentISO || weeks[0] == lastISO {
		for i, w := range weeks {
			expected := isoWeekLabel(isoWeekStart(time.Now()).AddDate(0, 0, -i*7))
			if w != expected {
				break
			}
			streak++
		}
	}

	// Personal record: longest consecutive run in all history.
	record = computeLongestStreak(weeks)
	return streak, record, nil
}

func computeLongestStreak(weeks []string) int {
	if len(weeks) == 0 {
		return 0
	}
	best, cur := 1, 1
	for i := 1; i < len(weeks); i++ {
		prev := isoWeekStart(time.Now()) // recalculate from actual week strings via parsing
		_ = prev
		// weeks are sorted desc; check if consecutive
		t1, _ := isoWeekStringToTime(weeks[i-1])
		t2, _ := isoWeekStringToTime(weeks[i])
		if t1.Sub(t2) == 7*24*time.Hour {
			cur++
			if cur > best {
				best = cur
			}
		} else {
			cur = 1
		}
	}
	return best
}

func getWeeklyHistory(ctx context.Context, pool *pgxpool.Pool, userID int64, numWeeks int) ([]WeekCount, error) {
	history := make([]WeekCount, numWeeks)
	now := isoWeekStart(time.Now())
	for i := 0; i < numWeeks; i++ {
		weekStart := now.AddDate(0, 0, -i*7)
		weekEnd := weekStart.AddDate(0, 0, 7)
		history[numWeeks-1-i] = WeekCount{
			Week: isoWeekLabel(weekStart),
		}
		_ = weekEnd
	}

	rows, err := pool.Query(ctx, `
		SELECT
			TO_CHAR(DATE_TRUNC('week', approved_at), 'IYYY-"W"IW') as week,
			COUNT(*) as cnt
		FROM check_ins
		WHERE owner_user_id = $1 AND status = 'approved'
		  AND approved_at >= $2
		GROUP BY week
	`, userID, now.AddDate(0, 0, -numWeeks*7))
	if err != nil {
		return nil, fmt.Errorf("weekly history: %w", err)
	}
	defer rows.Close()

	counts := map[string]int{}
	for rows.Next() {
		var week string
		var cnt int
		if err := rows.Scan(&week, &cnt); err != nil {
			return nil, err
		}
		counts[week] = cnt
	}

	for i := range history {
		if c, ok := counts[history[i].Week]; ok {
			history[i].ProofsCount = c
		}
	}
	return history, nil
}

func isoWeekStart(t time.Time) time.Time {
	wd := int(t.Weekday())
	if wd == 0 {
		wd = 7
	}
	start := t.AddDate(0, 0, -(wd - 1))
	y, m, d := start.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

func isoWeekLabel(t time.Time) string {
	y, w := t.ISOWeek()
	return fmt.Sprintf("%04d-W%02d", y, w)
}

func isoWeekStringToTime(s string) (time.Time, error) {
	var y, w int
	_, err := fmt.Sscanf(s, "%04d-%02d", &y, &w)
	if err != nil {
		return time.Time{}, err
	}
	// Jan 4 is always in week 1 of its year (ISO 8601)
	jan4 := time.Date(y, 1, 4, 0, 0, 0, 0, time.UTC)
	_, jan4Week := jan4.ISOWeek()
	weekStart := jan4.AddDate(0, 0, (w-jan4Week)*7)
	return isoWeekStart(weekStart), nil
}
