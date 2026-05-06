"use client";

import Link from "next/link";

import styles from "./your-card.module.css";

type Props = {
  rank: number;
  streak: number;
  weekScore: number;
  isAtRisk: boolean;
  goalId?: number;
  circleName?: string;
};

export function YourCard({ rank, streak, weekScore, isAtRisk, goalId, circleName }: Props) {
  const checkinHref = goalId ? `/goals/${goalId}/check-in` : "/goals/new";

  return (
    <div className={`${styles.root} ${isAtRisk ? styles.atRisk : ""}`}>
      {isAtRisk && <div className={styles.riskBanner}>ТЫ В ШАГЕ ОТ ЗАМОРОЗКИ.</div>}

      {circleName && <div className={styles.circleName}>{circleName.toUpperCase()}</div>}

      <div className={styles.stats}>
        <div className={styles.stat} data-tip="Место в круге по очкам за неделю">
          <span className={styles.statValue}>{rank}</span>
          <span className={styles.statLabel}>РАНГ</span>
        </div>
        <div className={styles.divider} />
        <div className={styles.stat} data-tip="Дней подряд без пропуска. Пропустил — серия обнуляется">
          <span className={styles.statValue}>{streak > 0 ? `🔥${streak}` : streak}</span>
          <span className={styles.statLabel}>СЕРИЯ</span>
        </div>
        <div className={styles.divider} />
        <div className={styles.stat} data-tip="Очки за одобренные пруфы на этой неделе">
          <span className={styles.statValue}>{weekScore}</span>
          <span className={styles.statLabel}>ОЧКИ</span>
        </div>
      </div>

      <Link href={checkinHref} className={styles.cta}>
        СДАТЬ ПРУФ
      </Link>
    </div>
  );
}
