"use client";

import { useEffect, useState } from "react";
import type { PersonalLeaderboard, WeekCount, WeekTrend } from "@/lib/types";
import styles from "./personal-leaderboard.module.css";

function apiFetch<T>(url: string): Promise<T> {
  return fetch(url, { credentials: "include" }).then((r) => {
    if (!r.ok) throw new Error(r.statusText);
    return r.json() as Promise<T>;
  });
}

const TREND_ICON: Record<WeekTrend, string> = {
  better: "↑",
  worse: "↓",
  same: "→",
  first_week: "★",
};
const TREND_COLOR: Record<WeekTrend, string> = {
  better: "var(--win)",
  worse: "var(--danger, #e53e3e)",
  same: "var(--ink-mono)",
  first_week: "var(--warn, #f6c90e)",
};

function SparkBars({ history }: { history: WeekCount[] }) {
  const last8 = history.slice(-8);
  const max = Math.max(...last8.map((w) => w.count), 1);
  return (
    <div className={styles.sparkWrap} aria-hidden>
      {last8.map((w) => (
        <div key={w.week} className={styles.sparkBar}>
          <div
            className={styles.sparkFill}
            style={{ height: `${Math.round((w.count / max) * 100)}%` }}
          />
        </div>
      ))}
    </div>
  );
}

function StatRow({ label, value, sublabel }: { label: string; value: string | number; sublabel?: string }) {
  return (
    <div className={styles.statRow}>
      <span className={styles.statLabel}>{label}</span>
      <span className={styles.statValue}>{value}</span>
      {sublabel && <span className={styles.statSub}>{sublabel}</span>}
    </div>
  );
}

export function PersonalLeaderboardWidget() {
  const [data, setData] = useState<PersonalLeaderboard | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    apiFetch<PersonalLeaderboard>("/v1/me/leaderboard")
      .then(setData)
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  if (loading) return <div className={styles.root}><p className={styles.empty}>Загрузка…</p></div>;
  if (!data) return null;

  const weekDelta = parseInt(data.current_week.vs_last_week, 10);
  const weekTrend = data.current_week.trend;

  return (
    <div className={styles.root}>
      <h3 className={styles.heading}>Мой прогресс</h3>

      <div className={styles.weekBlock}>
        <div className={styles.weekMain}>
          <span className={styles.bigNum}>{data.current_week.proofs_count}</span>
          <span className={styles.weekLabel}>пруфов на этой неделе</span>
        </div>
        <div
          className={styles.trendBadge}
          style={{ color: TREND_COLOR[weekTrend], borderColor: TREND_COLOR[weekTrend] }}
        >
          <span>{TREND_ICON[weekTrend]}</span>
          {weekTrend !== "first_week" && (
            <span>
              {Math.abs(weekDelta)} {weekTrend === "better" ? "лучше" : weekTrend === "worse" ? "меньше" : "как"} чем неделю назад
            </span>
          )}
          {weekTrend === "first_week" && <span>первая неделя</span>}
        </div>
      </div>

      {data.weekly_history.length > 0 && <SparkBars history={data.weekly_history} />}

      <div className={styles.stats}>
        <StatRow
          label="Серия недель"
          value={data.streak.current_streak}
          sublabel={
            data.streak.longest_streak > data.streak.current_streak
              ? `рекорд ${data.streak.longest_streak}`
              : undefined
          }
        />
        <StatRow label="Активных недель" value={data.streak.weeks_active} />
        <StatRow
          label="Этот сезон"
          value={data.current_season.proofs_count}
          sublabel={data.current_season.vs_last_season !== "0" ? `${data.current_season.vs_last_season} vs прошлый` : undefined}
        />
      </div>
    </div>
  );
}
