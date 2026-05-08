"use client";

import type { PersonalLeaderboard, WeekTrend } from "@/lib/types";
import styles from "./personal-progress-bar.module.css";

// ── Sub-components ────────────────────────────────────────────────────────────

function TrendBadge({ trend, delta }: { trend: WeekTrend; delta: number }) {
  if (trend === "first_week") return null;

  const text =
    trend === "better"
      ? `+${delta} лучше чем неделю назад`
      : trend === "same"
      ? "как неделю назад"
      : `${Math.abs(delta)} меньше чем неделю назад`;

  const color =
    trend === "better"
      ? "var(--win)"
      : trend === "worse"
      ? "var(--warn, #f6c90e)"
      : "var(--ink-mono)";

  return (
    <span className={styles.trend} style={{ color }}>
      {text}
    </span>
  );
}

function SparkChart({ weeks, maxValue }: { weeks: { week: string; count: number }[]; maxValue: number }) {
  const safeMax = maxValue > 0 ? maxValue : 1;
  return (
    <div className={styles.sparkChart}>
      {weeks.map((w) => (
        <div key={w.week} className={styles.sparkBar}>
          <div
            className={styles.sparkFill}
            style={{ height: `${Math.max((w.count / safeMax) * 100, w.count > 0 ? 4 : 0)}%` }}
            title={`${w.count} пруфов`}
          />
          <div className={styles.sparkWeekLabel}>
            {w.week.slice(-2)}
          </div>
        </div>
      ))}
    </div>
  );
}

function StreakBar({ current, record }: { current: number; record: number }) {
  if (current === 0 && record === 0) return null;
  return (
    <div className={styles.streakRow}>
      <span className={styles.streakLabel}>Streak:</span>
      <span className={styles.streakValue}>{current} {pluralWeeks(current)}</span>
      {record > current && (
        <span className={styles.streakRecord}>· рекорд {record} {pluralWeeks(record)}</span>
      )}
    </div>
  );
}

function pluralWeeks(n: number): string {
  if (n % 10 === 1 && n % 100 !== 11) return "неделя";
  if (n % 10 >= 2 && n % 10 <= 4 && (n % 100 < 10 || n % 100 >= 20)) return "недели";
  return "недель";
}

// ── Main component ────────────────────────────────────────────────────────────

interface Props {
  data: PersonalLeaderboard;
}

export function PersonalProgressBar({ data }: Props) {
  const proofsThisWeek = data.current_week.proofs_count;
  const trend = data.current_week.trend as WeekTrend;
  const maxProofs = Math.max(...data.weekly_history.map((w) => w.count), 1);

  return (
    <section className={styles.section}>
      <div className={styles.eyebrow}>МОЙ ПРОГРЕСС</div>

      <div className={styles.thisWeek}>
        <div className={styles.weekCount}>
          <span className={styles.count}>{proofsThisWeek}</span>
          <span className={styles.countLabel}>{" "}пруфов на этой неделе</span>
        </div>
        <TrendBadge trend={trend} delta={parseInt(data.current_week.vs_last_week, 10) || 0} />
      </div>

      {data.weekly_history.length > 0 && (
        <SparkChart weeks={data.weekly_history} maxValue={maxProofs} />
      )}

      <StreakBar
        current={data.streak.current_streak}
        record={data.streak.longest_streak}
      />
    </section>
  );
}
