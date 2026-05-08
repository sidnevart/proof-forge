import styles from "./health-score-badge.module.css";

export function HealthScoreBadge({ score }: { score: number }) {
  const color = score >= 70 ? "var(--win)" : score >= 40 ? "var(--warn, #f6c90e)" : "var(--danger, #e53e3e)";
  return (
    <div className={styles.badge}>
      <div className={styles.bar}>
        <div className={styles.fill} style={{ width: `${score}%`, background: color }} />
      </div>
      <span className={styles.value} style={{ color }}>{score}</span>
    </div>
  );
}
