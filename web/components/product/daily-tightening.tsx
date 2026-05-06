"use client";

import { useEffect, useState } from "react";

import styles from "./daily-tightening.module.css";

type Props = {
  deadlineUTC?: string | null;
  submittedCount: number;
  totalCount: number;
  mySubmitted: boolean;
};

export function DailyTightening({ deadlineUTC, submittedCount, totalCount, mySubmitted }: Props) {
  const [msLeft, setMsLeft] = useState(() => computeMsLeft(deadlineUTC));

  useEffect(() => {
    const id = setInterval(() => setMsLeft(computeMsLeft(deadlineUTC)), 1000);
    return () => clearInterval(id);
  }, [deadlineUTC]);

  const urgency = getUrgency(msLeft);

  return (
    <div className={`${styles.root} ${styles[urgency]}`} data-testid="daily-tightening">
      <div className={styles.countdown}>{formatMs(msLeft)}</div>
      <div className={styles.caption}>ДО ЗАМОРОЗКИ</div>
      <div className={styles.label}>
        {mySubmitted ? "ТЫ СДАЛ. ОТДЫХ." : `${submittedCount} ИЗ ${totalCount} СДАЛИ. ТЫ НЕТ.`}
      </div>
    </div>
  );
}

function computeMsLeft(deadlineUTC?: string | null): number {
  const now = Date.now();
  if (deadlineUTC) {
    return Math.max(0, new Date(deadlineUTC).getTime() - now);
  }
  const target = new Date();
  target.setHours(23, 59, 0, 0);
  if (target.getTime() <= now) {
    target.setDate(target.getDate() + 1);
  }
  return Math.max(0, target.getTime() - now);
}

function getUrgency(ms: number): "normal" | "warn" | "danger" | "critical" {
  const hours = ms / 3_600_000;
  if (hours > 4) return "normal";
  if (hours > 2) return "warn";
  if (ms > 30 * 60_000) return "danger";
  return "critical";
}

function formatMs(ms: number): string {
  const total = Math.floor(ms / 1000);
  const h = Math.floor(total / 3600);
  const m = Math.floor((total % 3600) / 60);
  const s = total % 60;
  return `${String(h).padStart(2, "0")}:${String(m).padStart(2, "0")}:${String(s).padStart(2, "0")}`;
}
