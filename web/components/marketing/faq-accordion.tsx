"use client";

import { useLayoutEffect, useRef, useState } from "react";

import styles from "./faq-accordion.module.css";

interface FaqItem {
  q: string;
  a: string;
}

interface FaqAccordionProps {
  items: FaqItem[];
}

/**
 * Accessible FAQ accordion.
 *
 * - First item open by default.
 * - Only one item can be open at a time.
 * - Height animates via `max-height` transition driven by `--faq-answer-h`.
 * - Chevron rotates 180° when open.
 * - Full keyboard support: Tab to focus, Enter/Space to toggle.
 * - Under prefers-reduced-motion: the global CSS collapses transition to 0.01ms.
 */
export function FaqAccordion({ items }: FaqAccordionProps) {
  const [openIndex, setOpenIndex] = useState<number>(0);

  const toggle = (i: number) => {
    setOpenIndex((prev) => (prev === i ? -1 : i));
  };

  return (
    <div className={styles.grid}>
      {items.map((item, i) => (
        <FaqItemRow
          key={item.q}
          item={item}
          isOpen={openIndex === i}
          onToggle={() => toggle(i)}
        />
      ))}
    </div>
  );
}

interface FaqItemRowProps {
  item: FaqItem;
  isOpen: boolean;
  onToggle: () => void;
}

function FaqItemRow({ item, isOpen, onToggle }: FaqItemRowProps) {
  const answerRef = useRef<HTMLDivElement | null>(null);

  // Measure answer scrollHeight and expose it as --faq-answer-h
  // so CSS can animate max-height without magic numbers.
  useLayoutEffect(() => {
    const el = answerRef.current;
    if (!el) return;
    el.style.setProperty("--faq-answer-h", `${el.scrollHeight}px`);
  }, [item.a]);

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" || e.key === " ") {
      e.preventDefault();
      onToggle();
    }
  };

  return (
    <div className={`${styles.item} ${isOpen ? styles.itemOpen : ""}`}>
      <button
        className={styles.question}
        aria-expanded={isOpen}
        onClick={onToggle}
        onKeyDown={handleKeyDown}
        type="button"
      >
        <span>{item.q}</span>
        <span className={styles.chevron} aria-hidden="true">
          ↓
        </span>
      </button>
      <div
        ref={answerRef}
        className={styles.answer}
        aria-hidden={!isOpen}
        role="region"
      >
        <p className={styles.answerText}>{item.a}</p>
      </div>
    </div>
  );
}
