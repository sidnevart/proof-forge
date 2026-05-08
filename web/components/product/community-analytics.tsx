"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { StatCard } from "./analytics-stat-card";
import styles from "./community-analytics.module.css";

interface CircleEngagement {
  circle_id: number;
  circle_name: string;
  member_count: number;
  completion_pct: number;
  proofs_count: number;
  status: string;
}

interface BestProof {
  check_in_id: number;
  goal_title: string;
  owner_display_name: string;
  approved_at: string;
}

interface CommunityData {
  community_name: string;
  summary: {
    total_members: number;
    active_members: number;
    retention_pct: number;
    completion_rate_pct: number;
    total_proofs_this_season: number;
    avg_proofs_per_member: number;
  };
  circle_engagement: CircleEngagement[];
  best_proofs_week: BestProof[];
}

interface Props {
  communityId: number;
}

export function CommunityAnalytics({ communityId }: Props) {
  const [data, setData] = useState<CommunityData | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetch(`/v1/community-spaces/${communityId}/analytics`, { credentials: "include" })
      .then((r) => r.json())
      .then((json) => setData(json.data ?? json))
      .finally(() => setLoading(false));
  }, [communityId]);

  return (
    <div className={styles.screen}>
      {loading && <p className={styles.hint}>Загрузка…</p>}

      {!loading && data && (
        <>
          <section>
            <div className={styles.eyebrow}>Обзор</div>
            <div className={styles.statGrid}>
              <StatCard label="Участников" value={data.summary.total_members} />
              <StatCard label="Retention" value={`${data.summary.retention_pct.toFixed(0)}%`} accent="win" />
              <StatCard label="Пруфов" value={data.summary.total_proofs_this_season} />
              <StatCard label="Завершили" value={`${data.summary.completion_rate_pct.toFixed(0)}%`} />
            </div>
          </section>

          {data.circle_engagement.length > 0 && (
            <section>
              <div className={styles.eyebrow}>Круги</div>
              <div className={styles.circleTable}>
                {data.circle_engagement.map((c) => (
                  <div key={c.circle_id} className={styles.circleRow}>
                    <span className={styles.circleName}>{c.circle_name}</span>
                    <span className={styles.circleMembers}>{c.member_count} уч.</span>
                    <div className={styles.circleBarWrap}>
                      <div
                        className={styles.circleBar}
                        style={{ width: `${c.completion_pct}%` }}
                      />
                    </div>
                    <span className={styles.circlePct}>{c.completion_pct.toFixed(0)}%</span>
                  </div>
                ))}
              </div>
            </section>
          )}

          {data.best_proofs_week.length > 0 && (
            <section>
              <div className={styles.eyebrow}>Лучшие пруфы недели</div>
              <div className={styles.proofsList}>
                {data.best_proofs_week.map((p) => (
                  <Link key={p.check_in_id} href={`/check-ins/${p.check_in_id}`} className={styles.proofCard}>
                    <span className={styles.proofTitle}>{p.goal_title}</span>
                    <span className={styles.proofAuthor}>{p.owner_display_name}</span>
                  </Link>
                ))}
              </div>
            </section>
          )}
        </>
      )}
    </div>
  );
}
