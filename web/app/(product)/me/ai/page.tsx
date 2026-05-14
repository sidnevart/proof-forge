"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import {
  getAINotifications,
  getAIProofDrafts,
  dismissAINotification,
  acceptAIProofDraft,
  rejectAIProofDraft,
} from "@/lib/api";
import type { AICompanionNotification, AIProofDraft } from "@/lib/types";
import styles from "./page.module.css";

const SHORTCUTS = [
  { label: "НОВЫЙ КРУГ", icon: "🎯", href: "/goals/new" },
  { label: "К ЧЕК-ИНУ", icon: "✅", href: "/dashboard" },
  { label: "ПОМОЩЬ", icon: "❓", href: "/feed" },
];

function getNotificationTone(feature: string): string {
  if (feature.includes("risk")) return styles.notificationCard_warn;
  if (feature.includes("streak")) return styles.notificationCard_good;
  if (feature.includes("fair_play")) return styles.notificationCard_danger;
  return "";
}

function getConfidenceClass(confidence: string): string {
  const c = confidence.toLowerCase();
  if (c.includes("high") || c.includes("высок")) return styles.confidence_high;
  if (c.includes("medium") || c.includes("средн")) return styles.confidence_medium;
  if (c.includes("low") || c.includes("низк")) return styles.confidence_low;
  return "";
}

export default function AICenterPage() {
  const [notifications, setNotifications] = useState<AICompanionNotification[]>([]);
  const [drafts, setDrafts] = useState<AIProofDraft[]>([]);
  const [loadingNotifs, setLoadingNotifs] = useState(true);
  const [loadingDrafts, setLoadingDrafts] = useState(true);

  useEffect(() => {
    getAINotifications()
      .then((res) => setNotifications(res.notifications ?? []))
      .catch(() => setNotifications([]))
      .finally(() => setLoadingNotifs(false));

    getAIProofDrafts()
      .then((res) => setDrafts(res.drafts ?? []))
      .catch(() => setDrafts([]))
      .finally(() => setLoadingDrafts(false));
  }, []);

  async function handleDismiss(id: string) {
    try {
      await dismissAINotification(id);
      setNotifications((prev) => prev.filter((n) => n.id !== id));
    } catch {
      // ignore
    }
  }

  async function handleAcceptDraft(id: string) {
    try {
      await acceptAIProofDraft(id);
      setDrafts((prev) => prev.filter((d) => d.id !== id));
    } catch {
      // ignore
    }
  }

  async function handleRejectDraft(id: string) {
    try {
      await rejectAIProofDraft(id);
      setDrafts((prev) => prev.filter((d) => d.id !== id));
    } catch {
      // ignore
    }
  }

  const activeCount = notifications.length + drafts.length;

  return (
    <div className={`page-shell ${styles.page}`}>
      <header className={styles.header}>
        <Link href="/me" className={styles.back}>
          ← ПРОФИЛЬ
        </Link>
        <h1 className={styles.title}>AI ЦЕНТР</h1>
        {activeCount > 0 && (
          <span className={styles.badge}>{activeCount} активных</span>
        )}
      </header>

      {/* Shortcuts */}
      <nav className={styles.shortcutGrid}>
        {SHORTCUTS.map((s) => (
          <Link
            key={s.href}
            href={s.href}
            className={styles.shortcutBtn}
          >
            <span className={styles.shortcutIcon}>{s.icon}</span>
            <span>{s.label}</span>
          </Link>
        ))}
      </nav>

      {/* Notifications */}
      <section className={styles.section}>
        <div className={styles.sectionHead}>
          <span className={styles.sectionLabel}>УВЕДОМЛЕНИЯ</span>
          <span className={styles.sectionCount}>{notifications.length}</span>
        </div>
        {loadingNotifs ? (
          <div className={styles.empty}>
            <span className={styles.emptyIcon}>⏳</span>
            <span className={styles.emptyText}>Загрузка…</span>
          </div>
        ) : notifications.length === 0 ? (
          <div className={styles.empty}>
            <span className={styles.emptyIcon}>✨</span>
            <span className={styles.emptyText}>Нет активных уведомлений</span>
            <span className={styles.emptySub}>AI пришлёт сигнал, когда придёт время</span>
          </div>
        ) : (
          <div className={styles.notificationList}>
            {notifications.map((n) => (
              <article
                key={n.id}
                className={`${styles.notificationCard} ${getNotificationTone(n.feature)}`}
              >
                <div className={styles.notificationFeature}>
                  {n.feature.replace(/_/g, " ").toUpperCase()}
                </div>
                <div className={styles.notificationTitle}>{n.title}</div>
                <p className={styles.notificationBody}>{n.body}</p>
                <div className={styles.notificationActions}>
                  {n.actions?.map((a) => (
                    <button
                      key={a.action}
                      className={
                        a.action === "dismiss" || a.action === "dismiss_draft"
                          ? styles.ghostBtn
                          : styles.ctaBtn
                      }
                      onClick={() => {
                        if (
                          a.action === "dismiss" ||
                          a.action === "dismiss_draft"
                        ) {
                          handleDismiss(n.id);
                        } else if (a.url) {
                          window.location.href = a.url;
                        }
                      }}
                    >
                      {a.label}
                    </button>
                  ))}
                </div>
              </article>
            ))}
          </div>
        )}
      </section>

      {/* Drafts */}
      <section className={styles.section}>
        <div className={styles.sectionHead}>
          <span className={styles.sectionLabel}>ЧЕРНОВИКИ ПРУФОВ</span>
          <span className={styles.sectionCount}>{drafts.length}</span>
        </div>
        {loadingDrafts ? (
          <div className={styles.empty}>
            <span className={styles.emptyIcon}>⏳</span>
            <span className={styles.emptyText}>Загрузка…</span>
          </div>
        ) : drafts.length === 0 ? (
          <div className={styles.empty}>
            <span className={styles.emptyIcon}>📝</span>
            <span className={styles.emptyText}>Нет черновиков</span>
            <span className={styles.emptySub}>AI предложит текст, когда соберёт контекст</span>
          </div>
        ) : (
          <div className={styles.draftList}>
            {drafts.map((d) => (
              <article key={d.id} className={styles.draftCard}>
                <div className={styles.draftRationale}>{d.rationale}</div>
                <div className={styles.draftMeta}>
                  <span
                    className={`${styles.confidenceBadge} ${getConfidenceClass(d.confidence)}`}
                  >
                    {d.confidence.toUpperCase()}
                  </span>
                  <span className={styles.goalLink}>Цель #{d.goal_id}</span>
                </div>
                <div className={styles.draftActions}>
                  <button
                    className={styles.ctaBtn}
                    onClick={() => handleAcceptDraft(d.id)}
                  >
                    ПРИНЯТЬ
                  </button>
                  <button
                    className={styles.ghostBtn}
                    onClick={() => handleRejectDraft(d.id)}
                  >
                    ОТКЛОНИТЬ
                  </button>
                </div>
              </article>
            ))}
          </div>
        )}
      </section>
    </div>
  );
}
