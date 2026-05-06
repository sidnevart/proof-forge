"use client";

import { useEffect, useRef, useState } from "react";

/**
 * Smoothly animates a number from 0 (or current display) up to `target`.
 *
 * - Uses `requestAnimationFrame` with an ease-out-cubic curve
 * - Returns the integer currently displayed
 * - Honors `prefers-reduced-motion` — returns `target` immediately, no rAF loop
 * - Re-animates whenever `target` changes (jumps from current displayed value)
 *
 * Default duration: 1200ms.
 */
export function useCountUp(target: number, duration = 1200): number {
  const [value, setValue] = useState(target);
  const startValueRef = useRef(target);
  const startTimeRef = useRef<number | null>(null);
  const rafIdRef = useRef<number | null>(null);

  useEffect(() => {
    if (typeof window === "undefined") {
      setValue(target);
      return;
    }

    const reduce = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    if (reduce) {
      setValue(target);
      return;
    }

    startValueRef.current = value;
    startTimeRef.current = null;

    const step = (now: number) => {
      if (startTimeRef.current === null) startTimeRef.current = now;
      const elapsed = now - startTimeRef.current;
      const t = Math.min(1, elapsed / duration);
      // ease-out-cubic
      const eased = 1 - Math.pow(1 - t, 3);
      const next = Math.round(
        startValueRef.current + (target - startValueRef.current) * eased
      );
      setValue(next);
      if (t < 1) {
        rafIdRef.current = requestAnimationFrame(step);
      } else {
        rafIdRef.current = null;
      }
    };

    rafIdRef.current = requestAnimationFrame(step);

    return () => {
      if (rafIdRef.current !== null) {
        cancelAnimationFrame(rafIdRef.current);
        rafIdRef.current = null;
      }
    };
    // We intentionally do NOT depend on `value` — we only re-run when the target changes.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [target, duration]);

  return value;
}
