import Link from "next/link";
import type { NowCardData } from "@/lib/types";
import styles from "./now-card.module.css";

const URGENCY_COLORS: Record<string, string> = {
  danger: "var(--danger, #e53e3e)",
  fire: "var(--fire, #ff6b00)",
  warn: "var(--warn, #f6c90e)",
  win: "var(--win)",
  neutral: "var(--border)",
};

export function NowCard({ data }: { data: NowCardData }) {
  const borderColor = URGENCY_COLORS[data.urgency] ?? URGENCY_COLORS.neutral;

  return (
    <div className={styles.card} style={{ borderLeftColor: borderColor }}>
      <div className={styles.eyebrow}>СЕЙЧАС</div>
      <div className={styles.title}>{data.title}</div>
      {data.subtitle && (
        <div className={styles.subtitle}>{data.subtitle}</div>
      )}
      {data.action && (
        <Link href={data.action.url} className={styles.cta}>
          → {data.action.label}
        </Link>
      )}
    </div>
  );
}
