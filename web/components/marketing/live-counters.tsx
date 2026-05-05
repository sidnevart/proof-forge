"use client";

import { useEffect, useRef, useState } from "react";

import { useCountUp } from "@/lib/use-count-up";

import styles from "./live-counters.module.css";

interface CounterDef {
  label: string;
  initial: number;
}

const COUNTERS: CounterDef[] = [
  { label: "АКТИВНЫХ КРУГОВ", initial: 142 },
  { label: "ПРУФОВ СЕГОДНЯ", initial: 1847 },
  { label: "ЗАМОРОЖЕНО ЗА НЕДЕЛЮ", initial: 23 },
];

/**
 * Strip of 3 live-ish counters shown in the hero.
 *
 * On mount: count-up animation from 0 to target over 1200ms (via useCountUp).
 * Ongoing: every 5–8s, one random counter ticks up by 1 or 2 with a brief
 * highlight flash. Under prefers-reduced-motion, no animation and no ticking.
 */
export function LiveCounters() {
  const [values, setValues] = useState<number[]>(COUNTERS.map((c) => c.initial));
  const [flashIdx, setFlashIdx] = useState<number | null>(null);
  const reduceRef = useRef(false);

  // Detect reduced-motion once on mount (client-only)
  useEffect(() => {
    if (typeof window === "undefined") return;
    reduceRef.current = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
  }, []);

  // Random tick — +1 or +2 to a random counter every 5–8s
  useEffect(() => {
    if (typeof window === "undefined") return;

    const schedule = () => {
      const delay = 5000 + Math.random() * 3000;
      return setTimeout(() => {
        if (reduceRef.current) return;
        const idx = Math.floor(Math.random() * COUNTERS.length);
        const delta = Math.random() < 0.5 ? 1 : 2;
        setValues((prev) => {
          const next = [...prev];
          next[idx] = prev[idx] + delta;
          return next;
        });
        setFlashIdx(idx);
        // Clear flash after 200ms
        setTimeout(() => setFlashIdx(null), 200);
        timerRef.current = schedule();
      }, delay);
    };

    const timerRef = { current: schedule() };
    return () => clearTimeout(timerRef.current);
  }, []);

  return (
    <div className={styles.strip}>
      {COUNTERS.map((counter, i) => (
        <CounterCell
          key={counter.label}
          label={counter.label}
          target={values[i]}
          flash={flashIdx === i}
        />
      ))}
    </div>
  );
}

interface CounterCellProps {
  label: string;
  target: number;
  flash: boolean;
}

function CounterCell({ label, target, flash }: CounterCellProps) {
  const value = useCountUp(target, 1200);

  return (
    <div className={`${styles.cell} ${flash ? styles.cellFlash : ""}`}>
      <span className={styles.number}>{value.toLocaleString("ru-RU")}</span>
      <span className={styles.label}>{label}</span>
    </div>
  );
}
