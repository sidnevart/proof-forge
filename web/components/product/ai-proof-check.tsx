"use client";

import { useState } from "react";
import styles from "./ai-proof-check.module.css";

interface ProofCheckResult {
  has_artifact: boolean;
  has_conclusion: boolean;
  has_next_step: boolean;
  score: number;
  feedback: string;
  suggestion: string;
}

interface Props {
  getDraft: () => string;
}

function ScoreBar({ score }: { score: number }) {
  const color = score >= 80 ? "var(--win)" : score >= 50 ? "var(--warn, #f6c90e)" : "var(--danger, #e53e3e)";
  return (
    <div className={styles.scoreWrap}>
      <div className={styles.scoreBar}>
        <div className={styles.scoreFill} style={{ width: `${score}%`, background: color }} />
      </div>
      <span className={styles.scoreNum} style={{ color }}>{score}/100</span>
    </div>
  );
}

function Check({ ok, label }: { ok: boolean; label: string }) {
  return (
    <div className={styles.checkRow}>
      <span className={ok ? styles.checkOk : styles.checkNo}>{ok ? "✓" : "○"}</span>
      <span className={ok ? styles.checkLabelOk : styles.checkLabelNo}>{label}</span>
    </div>
  );
}

export function AIProofCheck({ getDraft }: Props) {
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<ProofCheckResult | null>(null);
  const [error, setError] = useState("");
  const [open, setOpen] = useState(false);

  async function check() {
    const draft = getDraft();
    if (!draft.trim()) return;
    setLoading(true);
    setError("");
    setOpen(true);
    try {
      const r = await fetch("/v1/ai/proof-check", {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ draft }),
      });
      if (!r.ok) throw new Error();
      setResult(await r.json());
    } catch {
      setError("Не удалось проверить. Попробуй ещё раз.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className={styles.root}>
      <button
        type="button"
        className={styles.checkBtn}
        onClick={check}
        disabled={loading}
      >
        {loading ? "Проверяю…" : "✦ Проверить перед отправкой"}
      </button>

      {open && (
        <div className={styles.panel}>
          <div className={styles.panelHeader}>
            <span className={styles.panelTitle}>AI-проверка пруфа</span>
            <button type="button" className={styles.closeBtn} onClick={() => setOpen(false)}>×</button>
          </div>

          {error && <p className={styles.error}>{error}</p>}

          {result && (
            <>
              <ScoreBar score={result.score} />
              <div className={styles.checks}>
                <Check ok={result.has_artifact} label="Артефакт (ссылка / скриншот / файл)" />
                <Check ok={result.has_conclusion} label="Вывод — что удалось понять или сделать" />
                <Check ok={result.has_next_step} label="Следующий шаг" />
              </div>
              {result.feedback && (
                <p className={styles.feedback}>{result.feedback}</p>
              )}
              {result.suggestion && (
                <p className={styles.suggestion}>{result.suggestion}</p>
              )}
            </>
          )}
        </div>
      )}
    </div>
  );
}
