"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import styles from "./circles-board.module.css";

interface CircleEntry {
  id: number;
  name: string;
  member_count: number;
  proofs_this_week: number;
  completion_rate_pct: number;
  activity_score: number;
  trend: "up" | "stable" | "down";
}

function apiFetch<T>(url: string): Promise<T> {
  return fetch(url, { credentials: "include" }).then((r) => {
    if (!r.ok) throw new Error(r.statusText);
    return r.json() as Promise<T>;
  });
}

const TREND_ICON = { up: "↑", stable: "→", down: "↓" } as const;
const TREND_COLOR = {
  up: "var(--win)",
  stable: "var(--ink-mono)",
  down: "var(--warn, #f6c90e)",
} as const;

interface Props {
  spaceId: number;
  spaceKind: "teamspace" | "community_space";
}

export function CirclesBoard({ spaceId, spaceKind }: Props) {
  const [circles, setCircles] = useState<CircleEntry[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (spaceKind !== "community_space") {
      setCircles([]);
      setLoading(false);
      return;
    }

    apiFetch<{ circles: CircleEntry[] }>(`/v1/community-spaces/${spaceId}/circles-board`)
      .then((d) => setCircles(d.circles ?? []))
      .catch(() => {})
      .finally(() => setLoading(false));
  }, [spaceId, spaceKind]);

  return (
    <div className={styles.root}>
      <h3 className={styles.heading}>Круги пространства</h3>
      <p className={styles.hint}>Сравниваются круги — у каждого своя динамика</p>

      {loading ? (
        <p className={styles.empty}>Загрузка…</p>
      ) : circles.length === 0 ? (
        <p className={styles.empty}>Кругов ещё нет</p>
      ) : (
        <div className={styles.list}>
          {circles.map((c) => (
            <Link key={c.id} href={`/circles/${c.id}`} className={styles.row}>
              <div className={styles.nameWrap}>
                <span className={styles.circleName}>{c.name}</span>
                <span className={styles.memberCount}>{c.member_count} участ.</span>
              </div>

              <div className={styles.stats}>
                <span className={styles.statNum}>{c.proofs_this_week}</span>
                <span className={styles.statLabel}>пруфов</span>
              </div>

              <div className={styles.completionWrap}>
                <div className={styles.completionBar}>
                  <div
                    className={styles.completionFill}
                    style={{
                      width: `${c.completion_rate_pct}%`,
                      background:
                        c.completion_rate_pct >= 70
                          ? "var(--win)"
                          : c.completion_rate_pct >= 40
                          ? "var(--warn, #f6c90e)"
                          : "var(--danger, #e53e3e)",
                    }}
                  />
                </div>
                <span className={styles.completionNum}>{c.completion_rate_pct}%</span>
              </div>

              <span
                className={styles.trend}
                style={{ color: TREND_COLOR[c.trend] }}
                aria-label={c.trend}
              >
                {TREND_ICON[c.trend]}
              </span>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}
