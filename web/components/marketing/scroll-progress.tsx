"use client";

import { useEffect, useRef, useState } from "react";

import styles from "./scroll-progress.module.css";

/**
 * Thin fixed bar at the top of the viewport that fills as the user scrolls
 * the document. Single rAF-throttled scroll listener — cheap and steady.
 *
 * Renders even under reduced-motion: it communicates information, not motion.
 */
export function ScrollProgress() {
  const [progress, setProgress] = useState(0);
  const rafRef = useRef<number | null>(null);

  useEffect(() => {
    if (typeof window === "undefined") return;

    const compute = () => {
      rafRef.current = null;
      const doc = document.documentElement;
      const max = doc.scrollHeight - window.innerHeight;
      if (max <= 0) {
        setProgress(0);
        return;
      }
      const ratio = Math.min(1, Math.max(0, window.scrollY / max));
      setProgress(ratio);
    };

    const onScroll = () => {
      if (rafRef.current === null) {
        rafRef.current = requestAnimationFrame(compute);
      }
    };

    compute();
    window.addEventListener("scroll", onScroll, { passive: true });
    window.addEventListener("resize", onScroll, { passive: true });
    return () => {
      window.removeEventListener("scroll", onScroll);
      window.removeEventListener("resize", onScroll);
      if (rafRef.current !== null) cancelAnimationFrame(rafRef.current);
    };
  }, []);

  return (
    <div
      className={styles.bar}
      data-testid="scroll-progress"
      role="progressbar"
      aria-label="Прогресс прокрутки"
      aria-valuemin={0}
      aria-valuemax={100}
      aria-valuenow={Math.round(progress * 100)}
      style={{ width: `${(progress * 100).toFixed(2)}%` }}
    />
  );
}
