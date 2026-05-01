import type { PropsWithChildren } from "react";

import { cn } from "@/lib/cn";

import styles from "./section-shell.module.css";

type Props = PropsWithChildren<{
  title?: string;
  eyebrow?: string;
  className?: string;
}>;

export function SectionShell({ title, eyebrow, className, children }: Props) {
  return (
    <section className={cn(styles.section, className)}>
      {(eyebrow || title) && (
        <header className={styles.header}>
          {eyebrow ? <span className="eyebrow">{eyebrow}</span> : null}
          {title ? <h2 className={styles.title}>{title}</h2> : null}
        </header>
      )}
      {children}
    </section>
  );
}
