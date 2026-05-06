"use client";

import { useState } from "react";
import { createTelegramLinkToken } from "@/lib/api";
import styles from "./telegram-link-banner.module.css";

interface TelegramLinkBannerProps {
  onDismiss: () => void;
}

export function TelegramLinkBanner({ onDismiss }: TelegramLinkBannerProps) {
  const [loading, setLoading] = useState(false);

  async function handleConnect() {
    setLoading(true);
    try {
      const result = await createTelegramLinkToken();
      window.open(result.deeplink, "_blank", "noopener,noreferrer");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className={styles.banner}>
      <span className={styles.icon}>⚡</span>
      <span className={styles.text}>
        ПОДКЛЮЧИ TELEGRAM. БЕЗ НЕГО ТЫ ПРОПУСТИШЬ ВСЁ.
      </span>
      <button
        className={styles.connectButton}
        onClick={handleConnect}
        disabled={loading}
      >
        {loading ? "…" : "ПОДКЛЮЧИТЬ"}
      </button>
      <button
        className={styles.dismissButton}
        onClick={onDismiss}
        aria-label="Закрыть"
      >
        ×
      </button>
    </div>
  );
}
