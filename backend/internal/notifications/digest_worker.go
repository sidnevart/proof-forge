package notifications

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// DigestWorker sends morning digests to circle members.
type DigestWorker struct {
	repo       *PostgresRepository
	dispatcher *Dispatcher
	log        *slog.Logger
}

func NewDigestWorker(repo *PostgresRepository, dispatcher *Dispatcher, log *slog.Logger) *DigestWorker {
	return &DigestWorker{repo: repo, dispatcher: dispatcher, log: log}
}

// Run checks for digest targets every 5 minutes.
func (w *DigestWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.tick(ctx); err != nil {
				w.log.Error("digest worker tick", "err", err)
			}
		}
	}
}

func (w *DigestWorker) tick(ctx context.Context) error {
	targets, err := w.repo.GetDigestTargets(ctx)
	if err != nil {
		return fmt.Errorf("digest worker: get targets: %w", err)
	}

	// Group targets by circle so we only query members once per circle.
	byCircle := map[int64][]DigestTargetRow{}
	for _, t := range targets {
		byCircle[t.CircleID] = append(byCircle[t.CircleID], t)
	}

	for circleID, rows := range byCircle {
		members, err := w.repo.GetCircleDigestMembers(ctx, circleID)
		if err != nil {
			w.log.Warn("digest worker: get members", "circle", circleID, "err", err)
			continue
		}

		for _, target := range rows {
			text := buildDigestText(target, members)
			dedupKey := DigestDedupKey(circleID, time.Now().UTC().Format("2006-01-02"))

			if err := w.dispatcher.SendDigest(ctx, target.UserID, target.ChatID, dedupKey, text); err != nil {
				w.log.Warn("digest worker: send", "user", target.UserID, "circle", circleID, "err", err)
			}
		}
	}
	return nil
}

func buildDigestText(target DigestTargetRow, members []DigestMembersRow) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(
		"📋 <b>СВОДКА. КРУГ «%s». ДЕНЬ %d / %d.</b>\n\n",
		target.CircleName, target.DayOfSeason, target.TotalDays,
	))

	var done, waiting []string
	var leader DigestMembersRow
	for i, m := range members {
		if i == 0 {
			leader = m
		}
		if m.HasCheckin {
			done = append(done, m.DisplayName)
		} else {
			waiting = append(waiting, m.DisplayName)
		}
	}

	if len(done) > 0 {
		sb.WriteString("🟢 <b>СДАЛИ:</b> " + strings.Join(done, ", ") + "\n")
	}
	if len(waiting) > 0 {
		sb.WriteString("❌ <b>НЕ СДАЛИ:</b> " + strings.Join(waiting, ", ") + "\n")
	}

	if leader.UserID != 0 {
		sb.WriteString(fmt.Sprintf("🔥 <b>ЛИДЕР:</b> %s, серия %d\n", leader.DisplayName, leader.Streak))
	}

	for _, m := range members {
		if m.UserID == target.UserID {
			sb.WriteString(fmt.Sprintf("\n🏷 <b>ТЫ:</b> ранг %d из %d, серия %d", m.Rank, len(members), m.Streak))
			break
		}
	}

	return sb.String()
}
