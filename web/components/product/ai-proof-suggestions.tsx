"use client";

import { useState } from "react";
import styles from "./ai-proof-suggestions.module.css";

interface ProofSuggestion {
  what_to_prove: string;
  how_to_prove: string;
  movement_mode: string;
  due_days: number;
}

interface Props {
  goalText: string | (() => string);
  onSelect?: (s: ProofSuggestion) => void;
}

const MODE_LABEL: Record<string, string> = {
  single_proof: "Разовый",
  regular_rhythm: "Регулярный",
  challenge: "Вызов",
  work_initiative: "Рабочая инициатива",
  free_goal: "Свободный",
};

export function AIProofSuggestions({ goalText, onSelect }: Props) {
  const [loading, setLoading] = useState(false);
  const [suggestions, setSuggestions] = useState<ProofSuggestion[] | null>(null);
  const [error, setError] = useState("");

  async function load() {
    const resolvedText = typeof goalText === "function" ? goalText() : goalText;
    if (!resolvedText.trim()) return;
    setLoading(true);
    setError("");
    try {
      const r = await fetch("/v1/ai/goal-to-proofs", {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ goal_text: resolvedText }),
      });
      if (!r.ok) throw new Error();
      const data = await r.json();
      setSuggestions(data.suggestions ?? []);
    } catch {
      setError("Не удалось получить предложения. Попробуй ещё раз.");
    } finally {
      setLoading(false);
    }
  }

  if (suggestions === null) {
    return (
      <button
        type="button"
        className={styles.triggerBtn}
        onClick={load}
        disabled={loading}
      >
        {loading ? "AI думает…" : "✦ Предложить 3 пруфа"}
      </button>
    );
  }

  return (
    <div className={styles.sheet}>
      <div className={styles.sheetHeader}>
        <span className={styles.sheetTitle}>AI-предложения пруфов</span>
        <button type="button" className={styles.closeBtn} onClick={() => setSuggestions(null)}>×</button>
      </div>
      {error && <p className={styles.error}>{error}</p>}
      <div className={styles.list}>
        {suggestions.map((s, i) => (
          <div key={i} className={styles.card}>
            <div className={styles.cardBody}>
              <p className={styles.what}>{s.what_to_prove}</p>
              <p className={styles.how}>{s.how_to_prove}</p>
              <div className={styles.meta}>
                <span className={styles.modeBadge}>{MODE_LABEL[s.movement_mode] ?? s.movement_mode}</span>
                <span className={styles.days}>{s.due_days} дн.</span>
              </div>
            </div>
            {onSelect && (
              <button
                type="button"
                className={styles.useBtn}
                onClick={() => { onSelect(s); setSuggestions(null); }}
              >
                Выбрать
              </button>
            )}
          </div>
        ))}
      </div>
      <button type="button" className={styles.reloadBtn} onClick={load} disabled={loading}>
        {loading ? "…" : "Ещё варианты"}
      </button>
    </div>
  );
}
