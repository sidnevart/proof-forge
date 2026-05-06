"use client";

import styles from "./circle-standings.module.css";
import type { StandingEntry } from "@/lib/types";

type Props = {
  standings: StandingEntry[];
  currentUserId?: number;
};

export function CircleStandings({ standings, currentUserId }: Props) {
  return (
    <div className={styles.root}>
      <div className={styles.header}>
        <span className={styles.title}>ТАБЛИЦА</span>
        <span className={styles.meta}>{standings.length} участников</span>
      </div>
      <ol className={styles.list}>
        {standings.map((entry) => {
          const isMe = entry.user_id === currentUserId;
          const isFrozen = entry.weekly_status === "dropped";
          const isAtRisk = entry.weekly_status === "at_risk";

          return (
            <li
              key={entry.user_id}
              className={`${styles.row} ${isMe ? styles.me : ""} ${isFrozen ? styles.frozen : ""}`}
            >
              <div className={`${styles.ribbon} ${getRibbonClass(entry.rank)}`}>
                {entry.rank}
              </div>

              <div className={`${styles.avatar} ${isFrozen ? styles.avatarFrozen : ""}`}>
                {getInitials(entry.display_name)}
                {isFrozen && <div className={styles.frozenCross} />}
              </div>

              <div className={styles.info}>
                <span className={`${styles.name} ${isFrozen ? styles.strikethrough : ""}`}>
                  {entry.display_name.toUpperCase()}
                  {isMe && " (ТЫ)"}
                </span>
                <span className={styles.streak}>
                  {entry.current_streak > 7 ? "🔥" : entry.current_streak > 0 ? "⚡" : ""}
                  {entry.current_streak > 0 ? ` ${entry.current_streak}` : ""}
                </span>
              </div>

              <div className={`${styles.statusPill} ${getStatusClass(entry.weekly_status)}`}>
                {formatStatus(entry.weekly_status)}
              </div>

              <div className={styles.score}>{entry.score}</div>

              {isAtRisk && !isFrozen && (
                <div className={styles.riskIndicator} title="В шаге от заморозки" />
              )}
            </li>
          );
        })}
      </ol>
    </div>
  );
}

function getInitials(name: string): string {
  return name
    .split(" ")
    .slice(0, 2)
    .map((w) => w[0]?.toUpperCase() ?? "")
    .join("");
}

function getRibbonClass(rank: number): string {
  if (rank === 1) return styles.ribbonGold;
  if (rank <= 3) return styles.ribbonSilver;
  return styles.ribbonGray;
}

function getStatusClass(status: StandingEntry["weekly_status"]): string {
  switch (status) {
    case "approved":
    case "comeback":
      return styles.statusWin;
    case "waiting_review":
      return styles.statusPending;
    case "at_risk":
      return styles.statusWarn;
    case "dropped":
      return styles.statusDanger;
    default:
      return styles.statusNeutral;
  }
}

function formatStatus(status: StandingEntry["weekly_status"]): string {
  switch (status) {
    case "approved": return "СДАЛ";
    case "comeback": return "КАМБЭК";
    case "waiting_review": return "ЖДЁТ";
    case "at_risk": return "НЕ СДАЛ";
    case "dropped": return "🥶 ЗАМОРОЖЕН";
    default: return "—";
  }
}
