"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { AITeamHealthHeader } from "./ai-team-health-header";
import { StatCard } from "./analytics-stat-card";
import styles from "./teamspace-analytics.module.css";

// ── Types ─────────────────────────────────────────────────────────────────────

interface Topic {
  tag: string;
  proof_count: number;
}

interface BestProof {
  check_in_id: number;
  goal_title: string;
  preview: string;
}

interface TeamspaceData {
  period: string;
  teamspace_name: string;
  summary: {
    active_members: number;
    total_members: number;
    total_proofs_submitted: number;
    total_proofs_approved: number;
    avg_approval_time_hours: number;
  };
  popular_topics: Topic[];
  ipr_eligible_proofs: number;
  attention_needed: { count: number; description: string };
  best_proofs_preview: BestProof[];
}

type Period = "last_week" | "last_4_weeks" | "last_12_weeks";

const PERIOD_LABELS: Record<Period, string> = {
  last_week: "Эта неделя",
  last_4_weeks: "4 недели",
  last_12_weeks: "12 недель",
};

// ── Topics chart ──────────────────────────────────────────────────────────────

function TopicsChart({ topics }: { topics: Topic[] }) {
  const max = topics[0]?.proof_count ?? 1;
  return (
    <div className={styles.topics}>
      {topics.slice(0, 5).map((t) => (
        <div key={t.tag} className={styles.topicRow}>
          <div className={styles.topicName}>{t.tag}</div>
          <div className={styles.topicBarWrap}>
            <div
              className={styles.topicBar}
              style={{ width: `${(t.proof_count / max) * 100}%` }}
            />
          </div>
          <div className={styles.topicCount}>{t.proof_count}</div>
        </div>
      ))}
    </div>
  );
}

// ── Main ──────────────────────────────────────────────────────────────────────

interface Props {
  teamspaceId: number;
}

export function TeamspaceAnalytics({ teamspaceId }: Props) {
  const [period, setPeriod] = useState<Period>("last_4_weeks");
  const [data, setData] = useState<TeamspaceData | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    fetch(`/v1/teamspaces/${teamspaceId}/analytics?period=${period}`, { credentials: "include" })
      .then((r) => r.json())
      .then((json) => setData(json.data ?? json))
      .finally(() => setLoading(false));
  }, [teamspaceId, period]);

  return (
    <div className={styles.screen}>
      <div className={styles.toolbar}>
        <select
          className={styles.periodSelect}
          value={period}
          onChange={(e) => setPeriod(e.target.value as Period)}
        >
          {(Object.entries(PERIOD_LABELS) as [Period, string][]).map(([v, l]) => (
            <option key={v} value={v}>{l}</option>
          ))}
        </select>
      </div>

      <AITeamHealthHeader teamID={teamspaceId} />

      {loading && <p className={styles.hint}>Загрузка…</p>}

      {!loading && data && (
        <>
          <section>
            <div className={styles.eyebrow}>Обзор</div>
            <div className={styles.statGrid}>
              <StatCard
                label={`Активны (${data.summary.total_members} всего)`}
                value={data.summary.active_members}
                accent={data.summary.active_members / data.summary.total_members > 0.7 ? "win" : "warn"}
              />
              <StatCard label="Пруфов за период" value={data.summary.total_proofs_submitted} />
              <StatCard
                label="Ср. время ревью"
                value={`${data.summary.avg_approval_time_hours.toFixed(1)}ч`}
              />
              <StatCard label="ИПР артефактов" value={data.ipr_eligible_proofs} />
            </div>
          </section>

          {data.popular_topics.length > 0 && (
            <section>
              <div className={styles.eyebrow}>Популярные темы</div>
              <TopicsChart topics={data.popular_topics} />
            </section>
          )}

          {data.attention_needed.count > 0 && (
            <section>
              <div className={styles.eyebrow}>Нужна помощь</div>
              <div className={styles.attentionBlock}>
                <span className={styles.attentionIcon}>⚠</span>
                <div>
                  <div className={styles.attentionText}>
                    {data.attention_needed.count} участника без пруфа более 2 недель
                  </div>
                  <Link href={`/teamspaces/${teamspaceId}/needs-help`} className={styles.attentionLink}>
                    Посмотреть подробнее →
                  </Link>
                </div>
              </div>
            </section>
          )}

          {data.best_proofs_preview.length > 0 && (
            <section>
              <div className={styles.eyebrow}>Лучшие пруфы</div>
              <div className={styles.bestProofs}>
                {data.best_proofs_preview.map((p) => (
                  <Link
                    key={p.check_in_id}
                    href={`/check-ins/${p.check_in_id}`}
                    className={styles.proofCard}
                  >
                    <div className={styles.proofTitle}>{p.goal_title}</div>
                    <div className={styles.proofPreview}>{p.preview}</div>
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
