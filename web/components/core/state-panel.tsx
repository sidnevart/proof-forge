import type { ReactNode } from "react";

import { cn } from "@/lib/cn";
import { TONE_LABELS } from "@/lib/ui-copy";

import styles from "./state-panel.module.css";

type Tone = "loading" | "error" | "empty" | "success" | "pending";

type Props = {
  title: string;
  description: string;
  tone: Tone;
  meta?: ReactNode;
  className?: string;
};

export function StatePanel({ title, description, tone, meta, className }: Props) {
  return (
    <div className={cn(styles.panel, styles[tone], className)}>
      <div className={styles.header}>
        <span className="eyebrow">{TONE_LABELS[tone]}</span>
        <h3>{title}</h3>
      </div>
      <p>{description}</p>
      {meta ? <div className={styles.meta}>{meta}</div> : null}
    </div>
  );
}
