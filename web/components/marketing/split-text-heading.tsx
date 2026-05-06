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

  // Split text into lines; render each line's chars with a <br> between lines.
  const lines = children.split("\n");
  let charIndex = 0;
  const lineNodes = lines.flatMap((line, lineIdx) => {
    const charSpans = line.split("").map((char) => {
      const idx = charIndex++;
      return (
        <span
          key={`c${idx}`}
          className={char === " " ? styles.space : styles.letter}
          style={{ "--char-index": idx } as React.CSSProperties}
        >
          {char === " " ? " " : char}
        </span>
      );
    });
    if (lineIdx < lines.length - 1) {
      charSpans.push(<br key={`br${lineIdx}`} />);
    }
    return charSpans;
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
