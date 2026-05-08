"use client";

import { useState } from "react";
import styles from "./ai-stuck-helper.module.css";

interface AntiProofResult {
  summary: string;
  what_blocked: string;
  next_attempt: string;
}

interface Props {
  goalText: string;
  onResult?: (r: AntiProofResult) => void;
}

export function AIStuckHelper({ goalText, onResult }: Props) {
  const [open, setOpen] = useState(false);
  const [description, setDescription] = useState("");
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<AntiProofResult | null>(null);
  const [error, setError] = useState("");

  async function submit() {
    if (!description.trim()) return;
    setLoading(true);
    setError("");
    try {
      const r = await fetch("/v1/ai/anti-proof", {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ goal_text: goalText, stuck_description: description }),
      });
      if (!r.ok) throw new Error();
      const data = await r.json();
      setResult(data);
      onResult?.(data);
    } catch {
      setError("Не удалось сформировать анти-пруф. Попробуй ещё раз.");
    } finally {
      setLoading(false);
    }
  }

  if (!open) {
    return (
      <button type="button" className={styles.triggerBtn} onClick={() => setOpen(true)}>
        Я застрял
      </button>
    );
  }

  return (
    <div className={styles.panel}>
      <div className={styles.header}>
        <h4 className={styles.title}>Зафиксируем попытку</h4>
        <button type="button" className={styles.closeBtn} onClick={() => setOpen(false)}>×</button>
      </div>
      <p className={styles.hint}>
        Анти-пруф — это не провал, это данные. Опиши что пробовал и где застрял.
      </p>

      {!result ? (
        <>
          <textarea
            className={styles.textarea}
            placeholder="Что пробовал? Где застрял? Что мешает продолжить?"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            rows={4}
            autoFocus
          />
          {error && <p className={styles.error}>{error}</p>}
          <div className={styles.actions}>
            <button
              type="button"
              className={styles.submitBtn}
              disabled={loading || !description.trim()}
              onClick={submit}
            >
              {loading ? "Формирую…" : "Сформировать анти-пруф"}
            </button>
          </div>
        </>
      ) : (
        <div className={styles.result}>
          <div className={styles.resultBlock}>
            <span className={styles.resultLabel}>Попытка</span>
            <p className={styles.resultText}>{result.summary}</p>
          </div>
          <div className={styles.resultBlock}>
            <span className={styles.resultLabel}>Блокер</span>
            <p className={styles.resultText}>{result.what_blocked}</p>
          </div>
          <div className={styles.resultBlock}>
            <span className={styles.resultLabel}>Следующий шаг</span>
            <p className={styles.resultText}>{result.next_attempt}</p>
          </div>
          <button
            type="button"
            className={styles.resetBtn}
            onClick={() => { setResult(null); setDescription(""); }}
          >
            Изменить
          </button>
        </div>
      )}
    </div>
  );
}
