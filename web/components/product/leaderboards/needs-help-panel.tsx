"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import styles from "./needs-help-panel.module.css";

interface StuckUser {
  user_id: number;
  display_name: string;
  goal_title: string;
  days_stuck: number;
  last_activity_at: string;
}

interface PendingReviewItem {
  checkin_id: number;
  user_display_name: string;
  goal_title: string;
  submitted_at: string;
  waiting_hours: number;
}

interface FadingCircle {
  circle_id: number;
  circle_name: string;
  last_proof_days_ago: number;
  active_members: number;
  total_members: number;
}

interface NeedsHelpData {
  stuck: StuckUser[];
  pending_review: PendingReviewItem[];
  fading_circles: FadingCircle[];
}

function apiFetch<T>(url: string): Promise<T> {
  return fetch(url, { credentials: "include" }).then((r) => {
    if (!r.ok) throw new Error(r.statusText);
    return r.json() as Promise<T>;
  });
}

interface Props {
  teamspaceId: number;
}

export function NeedsHelpPanel({ teamspaceId }: Props) {
  const [data, setData] = useState<NeedsHelpData | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    apiFetch<NeedsHelpData>(`/v1/teamspaces/${teamspaceId}/needs-help`)
      .then(setData)
      .catch(() => {})
      .finally(() => setLoading(false));
  }, [teamspaceId]);

  const totalIssues = data
    ? data.stuck.length + data.pending_review.length + data.fading_circles.length
    : 0;

  return (
    <div className={styles.root}>
      <div className={styles.headerRow}>
        <h3 className={styles.heading}>Нужна помощь</h3>
        {!loading && totalIssues > 0 && (
          <span className={styles.badge}>{totalIssues}</span>
        )}
      </div>
      <p className={styles.hint}>Видно только тимлиду — данные агрегированы</p>

      {loading ? (
        <p className={styles.empty}>Загрузка…</p>
      ) : totalIssues === 0 ? (
        <p className={styles.allGood}>Всё идёт хорошо 🎯</p>
      ) : (
        <>
          {data!.stuck.length > 0 && (
            <section className={styles.section}>
              <p className={styles.sectionLabel}>Застряли</p>
              {data!.stuck.map((u) => (
                <div key={u.user_id} className={styles.item}>
                  <div className={styles.itemBody}>
                    <span className={styles.itemName}>{u.display_name}</span>
                    <span className={styles.itemSub}>{u.goal_title}</span>
                  </div>
                  <span
                    className={styles.daysTag}
                    style={{ color: u.days_stuck >= 7 ? "var(--danger, #e53e3e)" : "var(--warn, #f6c90e)" }}
                  >
                    {u.days_stuck} дн.
                  </span>
                </div>
              ))}
            </section>
          )}

          {data!.pending_review.length > 0 && (
            <section className={styles.section}>
              <p className={styles.sectionLabel}>Ждут ревью</p>
              {data!.pending_review.map((r) => (
                <Link key={r.checkin_id} href={`/buddy?highlight=${r.checkin_id}`} className={styles.item}>
                  <div className={styles.itemBody}>
                    <span className={styles.itemName}>{r.user_display_name}</span>
                    <span className={styles.itemSub}>{r.goal_title}</span>
                  </div>
                  <span
                    className={styles.hoursTag}
                    style={{ color: r.waiting_hours >= 24 ? "var(--danger, #e53e3e)" : "var(--ink-mono)" }}
                  >
                    {r.waiting_hours >= 24
                      ? `${Math.floor(r.waiting_hours / 24)} дн.`
                      : `${r.waiting_hours} ч.`}
                  </span>
                </Link>
              ))}
            </section>
          )}

          {data!.fading_circles.length > 0 && (
            <section className={styles.section}>
              <p className={styles.sectionLabel}>Затухающие круги</p>
              {data!.fading_circles.map((c) => (
                <Link key={c.circle_id} href={`/circles/${c.circle_id}`} className={styles.item}>
                  <div className={styles.itemBody}>
                    <span className={styles.itemName}>{c.circle_name}</span>
                    <span className={styles.itemSub}>
                      {c.active_members}/{c.total_members} актив.
                    </span>
                  </div>
                  <span className={styles.daysTag} style={{ color: "var(--warn, #f6c90e)" }}>
                    {c.last_proof_days_ago} дн. без пруфов
                  </span>
                </Link>
              ))}
            </section>
          )}
        </>
      )}
    </div>
  );
}
