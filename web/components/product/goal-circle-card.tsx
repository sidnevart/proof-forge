"use client";

import { SeasonProgressRing } from "@/components/product/season-progress-ring";
import { SeasonEndPanel } from "@/components/product/season-end-panel";
import styles from "./goal-circle-card.module.css";

export interface GoalCircleCardProps {
  circleId: number;
  goalId: number;
  goalTitle: string;
  goalStatus: "pending_buddy_acceptance" | "active";
  buddyStatus: "invited" | "active";
  seasonDay: number;
  seasonDaysLeft: number;
  seasonStatus: "active" | "completed";
  membersCount: number;
  lastCheckInAt?: string | null;
  onCheckIn?: () => void;
  // Season end panel props
  seasonId?: number;
  onSeasonEnd?: (action: "extend" | "start_new") => void;
}

export function GoalCircleCard({
  circleId,
  goalId,
  goalTitle,
  goalStatus,
  buddyStatus,
  seasonDay,
  seasonDaysLeft,
  seasonStatus,
  onCheckIn,
  seasonId,
  onSeasonEnd,
}: GoalCircleCardProps) {
  const canCheckIn = seasonStatus === "active" && goalStatus === "active";
  const showSeasonEndPanel =
    seasonStatus === "completed" && seasonId !== undefined && onSeasonEnd !== undefined;

  return (
    <article className={styles.card} data-goal-id={goalId}>
      {/* Top row: ring + title */}
      <div className={styles.topRow}>
        <SeasonProgressRing
          day={seasonDay}
          daysLeft={seasonDaysLeft}
          status={seasonStatus}
        />
        <h2 className={styles.goalTitle}>{goalTitle}</h2>
      </div>

      {/* Season meta */}
      <p className={styles.seasonMeta}>
        День {seasonDay} / 7 &middot; осталось {seasonDaysLeft}{" "}
        {pluralDays(seasonDaysLeft)}
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

      {/* Completed badge (only when no end panel) */}
      {seasonStatus === "completed" && !showSeasonEndPanel && (
        <span className={styles.completedBadge}>СЕЗОН ЗАВЕРШЁН</span>
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

      {/* Season end panel — shown inline when season is over */}
      {showSeasonEndPanel && (
        <SeasonEndPanel
          circleId={circleId}
          seasonId={seasonId!}
          onEnd={onSeasonEnd!}
        />
      )}
    </article>
  );
}

function pluralDays(n: number): string {
  const abs = Math.abs(n);
  const mod10 = abs % 10;
  const mod100 = abs % 100;
  if (mod10 === 1 && mod100 !== 11) return "день";
  if (mod10 >= 2 && mod10 <= 4 && (mod100 < 10 || mod100 >= 20)) return "дня";
  return "дней";
}
