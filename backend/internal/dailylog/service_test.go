package dailylog

import (
	"context"
	"errors"
	"testing"
	"time"
)

// mockRepo is a test double that satisfies Repository.
type mockRepo struct {
	membership MembershipInfo
	membershipErr error

	entry *Entry
	entryErr error

	entries []Entry
	entriesErr error

	streak *Streak
	streakErr error

	upsertEntry *Entry
	upsertErr   error
}

func (m *mockRepo) UpsertEntry(ctx context.Context, e *Entry) (*Entry, error) {
	return m.upsertEntry, m.upsertErr
}
func (m *mockRepo) GetEntry(ctx context.Context, userID, teamID int64, date time.Time) (*Entry, error) {
	return m.entry, m.entryErr
}
func (m *mockRepo) ListEntries(ctx context.Context, filter ListFilter) ([]Entry, error) {
	return m.entries, m.entriesErr
}
func (m *mockRepo) GetStreak(ctx context.Context, userID, teamID int64) (*Streak, error) {
	return m.streak, m.streakErr
}
func (m *mockRepo) UpdateStreak(ctx context.Context, s *Streak) error {
	return nil
}
func (m *mockRepo) UpsertEntryAndStreak(ctx context.Context, e *Entry, s *Streak) (*Entry, error) {
	return m.upsertEntry, m.upsertErr
}
func (m *mockRepo) GetMembership(ctx context.Context, teamID, userID int64) (MembershipInfo, error) {
	return m.membership, m.membershipErr
}

func fixedClock(t time.Time) func() time.Time {
	return func() time.Time { return t }
}

func TestSubmitLogValidation(t *testing.T) {
	moscow := "Europe/Moscow"
	now := time.Date(2026, 5, 7, 14, 0, 0, 0, time.UTC)
	repo := &mockRepo{
		membership: MembershipInfo{Role: "member", Status: "active", Timezone: moscow},
		streak:     &Streak{UserID: 1, TeamID: 2, Current: 0, Best: 0, FreezesUsedThisMonth: 0},
		upsertEntry: &Entry{ID: 101},
	}
	svc := NewService(repo, WithClock(fixedClock(now)))

	cases := []struct {
		name string
		in   SubmitInput
		want error
	}{
		{
			name: "text too short",
			in: SubmitInput{
				UserID: 1, TeamID: 2, LogDate: now,
				TextContent: "short",
			},
			want: ErrTextTooShort,
		},
		{
			name: "future date",
			in: SubmitInput{
				UserID: 1, TeamID: 2,
				LogDate:     now.AddDate(0, 0, 1),
				TextContent: "Something learned today",
			},
			want: ErrFutureDate,
		},
		{
			name: "two days ago rejected",
			in: SubmitInput{
				UserID: 1, TeamID: 2,
				LogDate:     now.AddDate(0, 0, -2),
				TextContent: "Something learned today",
			},
			want: ErrInvalidLogDate,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, _, err := svc.SubmitLog(context.Background(), c.in)
			if !errors.Is(err, c.want) {
				t.Fatalf("expected %v, got %v", c.want, err)
			}
		})
	}
}

func TestSubmitLogHappyPath(t *testing.T) {
	moscow := "Europe/Moscow"
	now := time.Date(2026, 5, 7, 14, 0, 0, 0, time.UTC)
	repo := &mockRepo{
		membership:  MembershipInfo{Role: "member", Status: "active", Timezone: moscow},
		streak:      &Streak{UserID: 1, TeamID: 2, Current: 3, Best: 3},
		upsertEntry: &Entry{ID: 101, Status: StatusLogged},
	}
	svc := NewService(repo, WithClock(fixedClock(now)))

	entry, su, err := svc.SubmitLog(context.Background(), SubmitInput{
		UserID:      1,
		TeamID:      2,
		LogDate:     now,
		TextContent: "Learned something interesting today",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry.ID != 101 {
		t.Errorf("entry.ID = %d, want 101", entry.ID)
	}
	if su.NewCurrent != 4 {
		t.Errorf("streak current = %d, want 4", su.NewCurrent)
	}
	if su.NewBest != 4 || !su.IsNewRecord {
		t.Errorf("expected new record best=4, got best=%d record=%v", su.NewBest, su.IsNewRecord)
	}
}

func TestFreezeDayLimit(t *testing.T) {
	moscow := "Europe/Moscow"
	now := time.Date(2026, 5, 7, 14, 0, 0, 0, time.UTC)
	repo := &mockRepo{
		membership:  MembershipInfo{Role: "member", Status: "active", Timezone: moscow},
		streak:      &Streak{UserID: 1, TeamID: 2, Current: 5, Best: 5, FreezesUsedThisMonth: 2, UpdatedAt: now},
		upsertEntry: &Entry{ID: 102},
	}
	svc := NewService(repo, WithClock(fixedClock(now)))

	_, err := svc.FreezeDay(context.Background(), FreezeInput{
		UserID: 1, TeamID: 2, LogDate: now,
	})
	if !errors.Is(err, ErrFreezeLimitReached) {
		t.Fatalf("expected ErrFreezeLimitReached, got %v", err)
	}
}

func TestFreezeDayHappyPath(t *testing.T) {
	moscow := "Europe/Moscow"
	now := time.Date(2026, 5, 7, 14, 0, 0, 0, time.UTC)
	repo := &mockRepo{
		membership:  MembershipInfo{Role: "member", Status: "active", Timezone: moscow},
		streak:      &Streak{UserID: 1, TeamID: 2, Current: 5, Best: 5, FreezesUsedThisMonth: 0},
		upsertEntry: &Entry{ID: 103, Status: StatusFrozen},
	}
	svc := NewService(repo, WithClock(fixedClock(now)))

	entry, err := svc.FreezeDay(context.Background(), FreezeInput{
		UserID: 1, TeamID: 2, LogDate: now,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry.Status != StatusFrozen {
		t.Errorf("status = %s, want frozen", entry.Status)
	}
}
