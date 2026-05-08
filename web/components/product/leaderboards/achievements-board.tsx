"use client";

import { useEffect, useState } from "react";
import styles from "./achievements-board.module.css";

interface Achievement {
  id: string;
  title: string;
  description: string;
  icon: string;
  earned_at: string;
  category: "streak" | "volume" | "buddy" | "season" | "special";
}

interface LockedAchievement {
  id: string;
  title: string;
  icon: string;
  hint: string;
}

interface AchievementsData {
  earned: Achievement[];
  locked: LockedAchievement[];
}

function apiFetch<T>(url: string): Promise<T> {
  return fetch(url, { credentials: "include" }).then((r) => {
    if (!r.ok) throw new Error(r.statusText);
    return r.json() as Promise<T>;
  });
}

function formatEarned(iso: string): string {
  return new Date(iso).toLocaleDateString("ru-RU", { day: "numeric", month: "long" });
}

export function AchievementsBoard() {
  const [data, setData] = useState<AchievementsData | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    apiFetch<AchievementsData>("/v1/me/achievements")
      .then(setData)
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  if (loading) return <div className={styles.root}><p className={styles.empty}>Загрузка…</p></div>;
  if (!data) return null;

  return (
    <div className={styles.root}>
      <h3 className={styles.heading}>Достижения</h3>

      {data.earned.length === 0 && data.locked.length === 0 ? (
        <p className={styles.empty}>Достижений пока нет — начни двигаться!</p>
      ) : (
        <>
          {data.earned.length > 0 && (
            <div className={styles.section}>
              <div className={styles.grid}>
                {data.earned.map((a) => (
                  <div key={a.id} className={styles.badge} title={`Получено ${formatEarned(a.earned_at)}`}>
                    <span className={styles.badgeIcon}>{a.icon}</span>
                    <span className={styles.badgeTitle}>{a.title}</span>
                    <span className={styles.badgeDesc}>{a.description}</span>
                  </div>
                ))}
              </div>
            </div>
          )}

          {data.locked.length > 0 && (
            <div className={styles.section}>
              <p className={styles.sectionLabel}>Следующие</p>
              <div className={styles.grid}>
                {data.locked.map((a) => (
                  <div key={a.id} className={`${styles.badge} ${styles.locked}`}>
                    <span className={styles.badgeIcon}>{a.icon}</span>
                    <span className={styles.badgeTitle}>{a.title}</span>
                    <span className={styles.badgeHint}>{a.hint}</span>
                  </div>
                ))}
              </div>
            </div>
          )}
        </>
      )}
    </div>
  );
}
