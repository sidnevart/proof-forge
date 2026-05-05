"use client";

import { useState } from "react";
import { createTelegramLinkToken } from "@/lib/api";
import styles from "./page.module.css";

export default function TelegramSettingsPage() {
  const [deeplink, setDeeplink] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [copied, setCopied] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleConnect() {
    setLoading(true);
    setError(null);
    try {
      const result = await createTelegramLinkToken();
      setDeeplink(result.deeplink);
    } catch {
      setError("Не удалось создать ссылку. Попробуй ещё раз.");
    } finally {
      setLoading(false);
    }
  }

  async function handleCopy() {
    if (!deeplink) return;
    await navigator.clipboard.writeText(deeplink);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }

  return (
    <main className={styles.page}>
      <h1 className={styles.heading}>TELEGRAM</h1>
      <p className={styles.sub}>
        Подключи бота — получай сводку утром, нуджи в нужный момент, одобряй пруфы без открытия сайта.
      </p>

      {!deeplink ? (
        <button
          className={styles.ctaButton}
          onClick={handleConnect}
          disabled={loading}
        >
          {loading ? "ГЕНЕРИРУЮ…" : "ПОДКЛЮЧИТЬ TELEGRAM"}
        </button>
      ) : (
        <div className={styles.linkBlock}>
          <p className={styles.instruction}>
            Ссылка действует <strong>10 минут</strong>. Открой в Telegram или скопируй.
          </p>
          <a
            href={deeplink}
            target="_blank"
            rel="noopener noreferrer"
            className={styles.deeplinkButton}
          >
            ⚡ ОТКРЫТЬ В TELEGRAM
          </a>
          <button className={styles.copyButton} onClick={handleCopy}>
            {copied ? "✓ СКОПИРОВАНО" : "СКОПИРОВАТЬ ССЫЛКУ"}
          </button>
          <button className={styles.refreshLink} onClick={handleConnect}>
            Обновить ссылку
          </button>
        </div>
      )}

      {error && <p className={styles.error}>{error}</p>}
    </main>
  );
}
