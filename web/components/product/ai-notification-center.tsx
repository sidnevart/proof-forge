"use client";

import { useEffect, useState } from "react";
import { getAINotifications, dismissAINotification } from "@/lib/api";
import type { AICompanionNotification } from "@/lib/types";
import styles from "./ai-notification-center.module.css";

export function AINotificationCenter() {
  const [notifications, setNotifications] = useState<AICompanionNotification[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    getAINotifications()
      .then((res) => {
        if (cancelled) return;
        setNotifications(res.notifications ?? []);
      })
      .catch(() => {
        if (cancelled) return;
        setNotifications([]);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  async function handleDismiss(id: string) {
    try {
      await dismissAINotification(id);
      setNotifications((prev) => prev.filter((n) => n.id !== id));
    } catch {
      // ignore
    }
  }

  function handleAction(action: string, url?: string) {
    if (url) {
      window.location.href = url;
    }
  }

  return (
    <section className={styles.panel} aria-label="AI уведомления">
      <div className={styles.header}>
        <span className={styles.title}>AI уведомления</span>
        {notifications.length > 0 && (
          <span className={styles.count}>{notifications.length}</span>
        )}
      </div>

      {loading ? (
        <div className={styles.empty}>Загрузка...</div>
      ) : notifications.length === 0 ? (
        <div className={styles.empty}>Нет активных уведомлений</div>
      ) : (
        <div className={styles.list}>
          {notifications.map((n) => (
            <article key={n.id} className={styles.item}>
              <div className={styles.itemHeader}>
                <span className={styles.itemFeature}>{n.feature.replace(/_/g, " ")}</span>
                <span className={styles.itemDate}>{new Date(n.created_at).toLocaleDateString("ru-RU")}</span>
              </div>
              <div className={styles.itemTitle}>{n.title}</div>
              <div className={styles.itemBody}>{n.body}</div>
              {n.actions && n.actions.length > 0 && (
                <div className={styles.itemActions}>
                  {n.actions.map((a) => (
                    <button
                      key={a.action}
                      className={
                        a.action === "dismiss_draft" || a.action === "dismiss"
                          ? styles.ghostBtn
                          : styles.ctaBtn
                      }
                      onClick={() =>
                        a.action === "dismiss_draft" || a.action === "dismiss"
                          ? handleDismiss(n.id)
                          : handleAction(a.action, a.url)
                      }
                    >
                      {a.label}
                    </button>
                  ))}
                </div>
              )}
            </article>
          ))}
        </div>
      )}
    </section>
  );
}
