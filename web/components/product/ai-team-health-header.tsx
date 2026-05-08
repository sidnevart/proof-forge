"use client";

import { useEffect, useState } from "react";
import { getTeamAIHealth } from "@/lib/api";
import type { TeamAIHealth } from "@/lib/types";
import styles from "./ai-team-health-header.module.css";

export function AITeamHealthHeader({ teamID }: { teamID: number }) {
  const [health, setHealth] = useState<TeamAIHealth | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    getTeamAIHealth(teamID)
      .then((res) => {
        if (cancelled) return;
        setHealth(res);
      })
      .catch(() => {
        if (cancelled) return;
        setHealth(null);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [teamID]);

  if (loading) {
    return (
      <div className={styles.header}>
        <div className={styles.skeletonPulse} />
      </div>
    );
  }

  if (!health) return null;

  const scoreClass =
    health.team_health_score >= 80
      ? styles.green
      : health.team_health_score >= 50
        ? styles.yellow
        : styles.red;

  return (
    <section className={styles.header} aria-label="AI здоровье команды">
      <div className={styles.topRow}>
        <span className={styles.title}>AI ЗДОРОВЬЕ КОМАНДЫ</span>
        <span className={`${styles.scoreBadge} ${scoreClass}`}>
          {health.team_health_score} / 100
        </span>
      </div>
      <div className={styles.metrics}>
        <div className={styles.metric}>
          <span className={styles.metricLabel}>Ожидают approve</span>
          <span className={styles.metricValue}>{health.pending_approvals}</span>
        </div>
        <div className={styles.metric}>
          <span className={styles.metricLabel}>Fair play</span>
          <span className={styles.metricValue}>{health.fair_play_status}</span>
        </div>
      </div>
      {health.alerts && health.alerts.length > 0 && (
        <div className={styles.alerts}>
          {health.alerts.map((a, i) => (
            <div key={i} className={styles.alert}>
              {a.message}
            </div>
          ))}
        </div>
      )}
    </section>
  );
}
