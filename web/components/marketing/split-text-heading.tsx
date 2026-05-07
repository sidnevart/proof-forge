"use client";

import { useEffect, useRef } from "react";

import styles from "./split-text-heading.module.css";

interface SplitTextHeadingProps {
  children: string;
  /** Which HTML heading tag to render. Default: "h1". */
  tag?: "h1" | "h2" | "h3";
  className?: string;
}

/**
 * Splits a heading string into per-character spans and triggers a staggered
 * entrance animation on mount (class `is-ready` on the wrapper).
 *
 * Line breaks in `children` (literal `\n`) are converted to <br> elements.
 *
 * Words inside a line are wrapped in a `.word` span with `white-space: nowrap`
 * so the browser will never break a word in the middle when the line is too
 * narrow — it wraps the whole word to the next line instead. Without this we
 * got renders like «НИКТО НЕ ТЕР / ЯЕТСЯ» because each per-character span was
 * an independent break opportunity.
 *
 * Screen-reader fallback: the full text is rendered in a visually-hidden span
 * so assistive tech reads a single string, not many individual characters.
 *
 * Respects `prefers-reduced-motion` — under reduce the global CSS collapses
 * transition duration to near-zero, so letters appear instantly.
 */
export function SplitTextHeading({
  children,
  tag: Tag = "h1",
  className,
}: SplitTextHeadingProps) {
  // Use a callback ref so we work with any heading element regardless of
  // the specific subtype (HTMLHeadingElement vs HTMLElement).
  const wrapperRef = useRef<HTMLElement | null>(null);
  const setRef = (node: HTMLElement | null) => {
    wrapperRef.current = node;
  };

  useEffect(() => {
    const el = wrapperRef.current;
    if (!el) return;
    // rAF so the element is painted before we trigger the class,
    // giving the browser a chance to apply the initial (opacity:0) state first.
    const rafId = requestAnimationFrame(() => {
      el.classList.add(styles["is-ready"]);
    });
    return () => cancelAnimationFrame(rafId);
  }, []);

  // Build per-line markup: each word is its own nowrap unit, with a
  // breakable space between words and a <br> between lines.
  const lines = children.split("\n");
  let charIndex = 0;
  const lineNodes = lines.flatMap((line, lineIdx) => {
    const words = line.split(" ");
    const lineChildren: React.ReactNode[] = [];

    words.forEach((word, wordIdx) => {
      const charSpans = word.split("").map((char) => {
        const idx = charIndex++;
        return (
          <span
            key={`c${idx}`}
            className={styles.letter}
            style={{ "--char-index": idx } as React.CSSProperties}
          >
            {char}
          </span>
        );
      });
      lineChildren.push(
        <span key={`w${lineIdx}-${wordIdx}`} className={styles.word}>
          {charSpans}
        </span>
      );
      if (wordIdx < words.length - 1) {
        const spaceIdx = charIndex++;
        lineChildren.push(
          <span
            key={`s${lineIdx}-${wordIdx}`}
            className={styles.space}
            style={{ "--char-index": spaceIdx } as React.CSSProperties}
          >
            {" "}
          </span>
        );
      }
    });

    if (lineIdx < lines.length - 1) {
      lineChildren.push(<br key={`br${lineIdx}`} />);
    }
    return lineChildren;
  });

  return (
    <Tag
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      ref={setRef as any}
      className={[styles.root, className].filter(Boolean).join(" ")}
    >
      {/* Hidden text node for screen-readers */}
      <span className={styles.srOnly}>{children}</span>

      {/* Per-character spans, aria-hidden so SR sees only the node above */}
      <span aria-hidden="true">{lineNodes}</span>
    </Tag>
  );
}
