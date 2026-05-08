"use client";

import { useEffect, useState } from "react";
import styles from "./stability-board.module.css";

interface StabilityEntry {
  rank: number;
  display_name: string;
  weeks_active: number;
  proof_streak: number;
  consistency_pct: number;
}

function apiFetch<T>(url: string): Promise<T> {
  return fetch(url, { credentials: "include" }).then((r) => {
    if (!r.ok) throw new Error(r.statusText);
    return r.json() as Promise<T>;
  });
}

interface Props {
  circleId: number;
}

export function StabilityBoard({ circleId }: Props) {
  const [entries, setEntries] = useState<StabilityEntry[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    apiFetch<{ entries: StabilityEntry[] }>(`/v1/circles/${circleId}/stability-board`)
      .then((d) => setEntries(d.entries ?? []))
      .catch(() => {})
      .finally(() => setLoading(false));
  }, [circleId]);

  if (loading) return <div className={styles.root}><p className={styles.empty}>Загрузка…</p></div>;

  return (
    <div className={styles.root}>
      <h3 className={styles.heading}>Стабильность круга</h3>
      <p className={styles.hint}>Кто регулярно движется — без публичного ранжирования отстающих</p>

      {entries.length === 0 ? (
        <p className={styles.empty}>Ещё нет данных</p>
      ) : (
        <div className={styles.list}>
          {entries.map((e) => (
            <div key={e.rank} className={styles.row}>
              <span className={styles.rank}>{e.rank}</span>
              <div className={styles.info}>
                <span className={styles.name}>{e.display_name}</span>
                <div className={styles.meta}>
                  <span>{e.weeks_active} нед. активна</span>
                  <span>·</span>
                  <span>серия {e.proof_streak}</span>
                </div>
              </div>
              <div className={styles.consistencyWrap}>
                <div className={styles.consistencyBar}>
                  <div
                    className={styles.consistencyFill}
                    style={{ width: `${e.consistency_pct}%` }}
                  />
                </div>
                <span className={styles.consistencyNum}>{e.consistency_pct}%</span>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
