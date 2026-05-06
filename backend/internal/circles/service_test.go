package circles

import "testing"

func TestBuildStandingsRanksApprovedAheadOfAtRisk(t *testing.T) {
	entries := buildStandings([]StandingSnapshot{
		{
			UserID:               2,
			UserEmail:            "peer@example.com",
			DisplayName:          "Peer",
			GoalsCount:           0,
			ApprovedWeeks:        0,
			MissedWeeks:          1,
			CurrentStreak:        0,
			HasApprovedThisWeek:  false,
			HasSubmittedThisWeek: false,
		},
		{
			UserID:               1,
			UserEmail:            "owner@example.com",
			DisplayName:          "Owner",
			GoalsCount:           1,
			ApprovedWeeks:        1,
			MissedWeeks:          0,
			CurrentStreak:        1,
			HasApprovedThisWeek:  true,
			HasSubmittedThisWeek: false,
		},
	}, 1)

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].UserEmail != "owner@example.com" {
		t.Fatalf("expected owner to lead standings, got %s", entries[0].UserEmail)
	}
	if entries[0].WeeklyStatus != WeeklyStatusApproved {
		t.Fatalf("expected owner status approved, got %s", entries[0].WeeklyStatus)
	}
	if entries[1].WeeklyStatus != WeeklyStatusAtRisk {
		t.Fatalf("expected peer status at_risk, got %s", entries[1].WeeklyStatus)
	}
}

func TestBuildStandingsMarksComebackAfterMissedWeek(t *testing.T) {
	entries := buildStandings([]StandingSnapshot{
		{
			UserID:               1,
			UserEmail:            "owner@example.com",
			DisplayName:          "Owner",
			GoalsCount:           1,
			ApprovedWeeks:        1,
			CurrentStreak:        1,
			HasApprovedThisWeek:  true,
			HasSubmittedThisWeek: false,
			HasApprovedPrevWeek:  false,
		},
	}, 2)

	if entries[0].WeeklyStatus != WeeklyStatusComeback {
		t.Fatalf("expected comeback status, got %s", entries[0].WeeklyStatus)
	}
}
