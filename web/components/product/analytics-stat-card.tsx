import styles from "./analytics-stat-card.module.css";

interface StatCardProps {
  label: string;
  value: string | number;
  sublabel?: string;
  accent?: "win" | "warn" | "danger" | "neutral";
}

export function StatCard({ label, value, sublabel, accent = "neutral" }: StatCardProps) {
  return (
    <div className={`${styles.card} ${styles[accent]}`}>
      <div className={styles.value}>{value}</div>
      <div className={styles.label}>{label}</div>
      {sublabel && <div className={styles.sublabel}>{sublabel}</div>}
    </div>
  );
}
