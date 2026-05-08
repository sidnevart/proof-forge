"use client";

import Link from "next/link";
import { useEffect, useRef, useState } from "react";
import { ApiError } from "@/lib/api";
import styles from "./buddy-dashboard.module.css";

// ── Types ─────────────────────────────────────────────────────────────────────

interface QueueItem {
  check_in_id: number;
  goal_id: number;
  goal_title: string;
  user_id: number;
  display_name: string;
  submitted_at: string;
  waiting_hours: number;
  preview?: string;
}

interface AttentionItem {
  user_id: number;
  display_name: string;
  goal_title: string;
  days_since_last_proof: number;
  has_broken_contract: boolean;
}

interface BuddyStats {
  active_buddies_count: number;
  total_reviews_given: number;
  avg_response_hours: number;
  people_supported_count: number;
  reviews_this_week: number;
}

// ── Helpers ───────────────────────────────────────────────────────────────────

function humanizeHours(h: number): string {
  if (h < 1) return "<1ч";
  if (h < 24) return `${Math.round(h)}ч`;
  return `${Math.round(h / 24)}д`;
}

function pluralDays(n: number): string {
  const r = Math.round(n);
  if (r % 10 === 1 && r % 100 !== 11) return `${r} день`;
  if (r % 10 >= 2 && r % 10 <= 4 && (r % 100 < 10 || r % 100 >= 20)) return `${r} дня`;
  return `${r} дней`;
}

// ── Sub-components ─────────────────────────────────────────────────────────────

function QueueItemCard({ item }: { item: QueueItem }) {
  const urgency = item.waiting_hours > 24;
  const borderColor = urgency ? "var(--warn, #f6c90e)" : "var(--border)";
  return (
    <div className={styles.queueItem} style={{ borderLeftColor: borderColor }}>
      <div className={styles.queueHeader}>
        <span className={styles.userName}>{item.display_name}</span>
        {item.waiting_hours > 0 && (
          <span className={styles.waitBadge} style={{ color: urgency ? "var(--warn, #f6c90e)" : "var(--ink-mono)" }}>
            {humanizeHours(item.waiting_hours)} ⏳
          </span>
        )}
      </div>
      <div className={styles.goalTitle}>{item.goal_title}</div>
      {item.preview && (
        <div className={styles.preview}>«{item.preview}…»</div>
      )}
      <Link
        href={`/goals/${item.goal_id}/check-in?id=${item.check_in_id}`}
        className={styles.reviewBtn}
      >
        Просмотреть →
      </Link>
    </div>
  );
}

function AttentionRow({ item }: { item: AttentionItem }) {
  return (
    <div className={styles.attentionRow}>
      <span className={styles.attentionIcon}>⚠</span>
      <div className={styles.attentionBody}>
        <span className={styles.attentionText}>
          {item.display_name} — {pluralDays(item.days_since_last_proof)} без пруфа
          {item.has_broken_contract && " · контракт нарушен"}
        </span>
      </div>
    </div>
  );
}

function BuddyEmptyState() {
  return (
    <div className={styles.emptyState}>
      <div className={styles.emptyTitle}>У тебя пока нет подопечных</div>
      <div className={styles.emptyDesc}>
        Стань buddy для кого-то — помоги не слиться с целью
      </div>
      <Link href="/goals" className={styles.emptyBtn}>
        Посмотреть цели где нужен buddy →
      </Link>
    </div>
  );
}

// ── Main ──────────────────────────────────────────────────────────────────────

async function apiFetch<T>(path: string): Promise<T> {
  const res = await fetch(path, { credentials: "include" });
  if (!res.ok) throw new ApiError(res.status, "fetch error");
  const json = await res.json();
  return (json.data ?? json) as T;
}

export function BuddyDashboard() {
  const [queue, setQueue] = useState<QueueItem[]>([]);
  const [attention, setAttention] = useState<AttentionItem[]>([]);
  const [stats, setStats] = useState<BuddyStats | null>(null);
  const [loading, setLoading] = useState(true);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  async function loadData() {
    try {
      const [q, a, s] = await Promise.all([
        apiFetch<QueueItem[]>("/v1/buddy/queue"),
        apiFetch<AttentionItem[]>("/v1/buddy/needs-attention"),
        apiFetch<BuddyStats>("/v1/buddy/stats").catch(() => null),
      ]);
      setQueue(q);
      setAttention(a);
      setStats(s);
    } catch {
      // non-fatal
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void loadData();
    intervalRef.current = setInterval(loadData, 60_000);
    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current);
    };
  }, []);

  if (loading) return <div className={styles.hint}>Загрузка…</div>;

  if (queue.length === 0 && attention.length === 0) {
    return <BuddyEmptyState />;
  }

  return (
    <div className={styles.screen}>
      {queue.length > 0 && (
        <section>
          <div className={styles.eyebrow}>Очередь ревью</div>
          <div className={styles.queue}>
            {queue.map((item) => (
              <QueueItemCard key={item.check_in_id} item={item} />
            ))}
          </div>
        </section>
      )}

      {attention.length > 0 && (
        <section className={styles.attentionSection}>
          <div className={styles.eyebrow}>Нужна помощь</div>
          {attention.map((item) => (
            <AttentionRow key={item.user_id} item={item} />
          ))}
        </section>
      )}

      {stats && (stats.total_reviews_given > 0) && (
        <section className={styles.statsSection}>
          <div className={styles.eyebrow}>Моя статистика</div>
          <div className={styles.statsList}>
            <div className={styles.statItem}>
              <span className={styles.statValue}>{stats.people_supported_count}</span>
              <span className={styles.statLabel}> подопечных</span>
            </div>
            {stats.avg_response_hours > 0 && (
              <div className={styles.statItem}>
                <span className={styles.statLabel}>Среднее время ответа: </span>
                <span className={styles.statValue}>{stats.avg_response_hours.toFixed(1)}ч</span>
              </div>
            )}
            <div className={styles.statItem}>
              <span className={styles.statLabel}>Дано ревью: </span>
              <span className={styles.statValue}>{stats.total_reviews_given}</span>
            </div>
          </div>
        </section>
      )}
    </div>
  );
}
