"use client";

import styles from "./goal-circle-card.module.css";

export interface GoalCircleCardProps {
  circleId: number;
  goalId: number;
  goalTitle: string;
  goalStatus: "pending_buddy_acceptance" | "active";
  buddyStatus: "invited" | "active";
  proofStreak: number;
  membersCount: number;
  lastCheckInAt?: string | null;
  onCheckIn?: () => void;
  // viewerRole drives the «вы — автор» / «вы — партнёр» eyebrow so a user who
  // joined a buddy's круг can immediately tell it isn't their own.
  viewerRole?: "owner" | "buddy";
}

export function GoalCircleCard({
  goalId,
  goalTitle,
  goalStatus,
  buddyStatus,
  proofStreak,
  onCheckIn,
  viewerRole,
}: GoalCircleCardProps) {
  const canCheckIn = goalStatus === "active";
  const streakLabel = proofStreak > 0 ? `СЕРИЯ ${proofStreak}` : "СЕРИЯ ЕЩЁ НЕ НАЧАТА";

  return (
    <article className={styles.card} data-goal-id={goalId}>
      {/* Top row: streak + title */}
      <div className={styles.topRow}>
        <div className={styles.streakMark} aria-label={streakLabel}>
          <span className={styles.streakNumber}>{proofStreak}</span>
          <span className={styles.streakUnit}>streak</span>
        </div>
        <div className={styles.titleColumn}>
          {viewerRole ? (
            <span className={styles.viewerRole} data-role={viewerRole}>
              {viewerRole === "owner" ? "ВЫ — АВТОР" : "ВЫ — ПАРТНЁР"}
            </span>
          ) : null}
          <h2 className={styles.goalTitle}>{goalTitle}</h2>
        </div>
      </div>

      <p className={styles.progressMeta}>
        {streakLabel} · СЛЕДУЮЩИЙ ПРУФ ДЕРЖИТ СЕРИЮ ЖИВОЙ
      </p>

      {/* Status pill: pending = waiting for buddy; active = pact accepted */}
      {goalStatus === "pending_buddy_acceptance" && (
        <span className={styles.statusPill} data-status="pending">
          ЖДЁМ ПАРТНЁРА
        </span>
      )}
      {buddyStatus === "active" && goalStatus === "active" && (
        <span className={styles.statusPill} data-status="active">
          ПАРТНЁР АКТИВЕН
        </span>
      )}

      {/* Check-in button */}
      {canCheckIn && (
        <button
          type="button"
          className={styles.checkInBtn}
          onClick={onCheckIn}
        >
          СДАТЬ ПРУФ →
        </button>
      )}
    </article>
  );
}
