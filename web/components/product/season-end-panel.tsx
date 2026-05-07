"use client";

import { useState } from "react";
import { endSeason } from "@/lib/api";
import styles from "./season-end-panel.module.css";

interface SeasonEndPanelProps {
  circleId: number;
  seasonId: number;
  onEnd: (action: "extend" | "start_new") => void;
}

export function SeasonEndPanel({ circleId, seasonId, onEnd }: SeasonEndPanelProps) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleAction(action: "extend" | "start_new") {
    if (loading) return;
    setLoading(true);
    setError(null);
    try {
      await endSeason(circleId, seasonId, action);
      onEnd(action);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Ошибка. Попробуйте ещё раз.";
      setError(message);
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className={styles.panel}>
      <p className={styles.heading}>Сезон завершён — что дальше?</p>

      {error && <p className={styles.errorText}>{error}</p>}

      <div className={styles.actions}>
        <button
          className={styles.extendBtn}
          onClick={() => handleAction("extend")}
          disabled={loading}
          type="button"
        >
          ПРОДОЛЖИТЬ КРУГ (ЕЩЁ 7 ДНЕЙ)
        </button>
        <button
          className={styles.newGoalBtn}
          onClick={() => handleAction("start_new")}
          disabled={loading}
          type="button"
        >
          НАЧАТЬ НОВЫЙ КРУГ
        </button>
      </div>
    </div>
  );
}
