"use client";

import { useEffect, useState } from "react";
import { getAINotifications, dismissAINotification } from "@/lib/api";
import type { AICompanionNotification } from "@/lib/types";
import styles from "./ai-badge.module.css";

export function AIBadge() {
  const [notification, setNotification] = useState<AICompanionNotification | null>(null);
  const [loading, setLoading] = useState(true);
  const [dismissing, setDismissing] = useState(false);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    getAINotifications()
      .then((res) => {
        if (cancelled) return;
        const n = res.notifications?.[0] ?? null;
        setNotification(n);
      })
      .catch(() => {
        if (cancelled) return;
        setNotification(null);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  async function handleDismiss() {
    if (!notification) return;
    setDismissing(true);
    try {
      await dismissAINotification(notification.id);
      setNotification(null);
    } catch {
      // ignore
    } finally {
      setDismissing(false);
    }
  }

  async function handleAction(action: string, url?: string) {
    if (action === "dismiss_draft" || action === "dismiss") {
      await handleDismiss();
      return;
    }
    if (url) {
      window.location.href = url;
    }
  }

  if (loading) {
    return (
      <div className={styles.skeleton}>
        <div className={styles.skeletonPulse} />
      </div>
    );
  }

  if (!notification) return null;

  return (
    <article className={styles.badge} aria-label="AI инсайт">
      <div className={styles.header}>
        <span className={styles.eyebrow}>AI КОМПАНЬОН</span>
        <button
          className={styles.dismissBtn}
          onClick={handleDismiss}
          disabled={dismissing}
          aria-label="Закрыть"
          title="Закрыть"
        >
          ×
        </button>
      </div>
      <h3 className={styles.title}>{notification.title}</h3>
      <p className={styles.body}>{notification.body}</p>
      {notification.actions && notification.actions.length > 0 && (
        <div className={styles.actions}>
          {notification.actions.map((a) => (
            <button
              key={a.action}
              className={a.action === "dismiss_draft" || a.action === "dismiss" ? styles.ghostBtn : styles.ctaBtn}
              onClick={() => handleAction(a.action, a.url)}
              disabled={dismissing}
            >
              {a.label}
            </button>
          ))}
        </div>
      )}
    </article>
  );
}
