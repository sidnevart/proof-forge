"use client";

import { type RefObject, useEffect, useState } from "react";

interface MagneticOptions {
  /** Distance (px) within which the element reacts. Default 120. */
  radius?: number;
  /** Strength multiplier of cursor → element vector. Default 0.18. */
  strength?: number;
  /** Maximum offset in pixels. Default 12. */
  maxOffset?: number;
}

interface MagneticOffset {
  x: number;
  y: number;
}

const ZERO: MagneticOffset = { x: 0, y: 0 };

/**
 * Magnetic hover: when the cursor enters a radius around the element,
 * the element follows the cursor by a fraction of the offset (capped).
 *
 * Disabled entirely on touch devices and when reduced-motion is preferred.
 *
 * Listener is on `document` (not the element) — keeps tracking even when
 * the cursor is moving fast across the boundary.
 */
export function useMagneticHover(
  ref: RefObject<HTMLElement | null>,
  options: MagneticOptions = {}
): MagneticOffset {
  const { radius = 120, strength = 0.18, maxOffset = 12 } = options;
  const [offset, setOffset] = useState<MagneticOffset>(ZERO);

  useEffect(() => {
    if (typeof window === "undefined") return;

    const hasFinePointer = window.matchMedia("(hover: hover) and (pointer: fine)").matches;
    const reduce = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    if (!hasFinePointer || reduce) return;

    const node = ref.current;
    if (!node) return;

    let rafId = 0;
    let lastEvent: MouseEvent | null = null;

    const flush = () => {
      rafId = 0;
      if (!lastEvent) return;
      const rect = node.getBoundingClientRect();
      const cx = rect.left + rect.width / 2;
      const cy = rect.top + rect.height / 2;
      const dx = lastEvent.clientX - cx;
      const dy = lastEvent.clientY - cy;
      const dist = Math.hypot(dx, dy);
      if (dist > radius) {
        setOffset((prev) => (prev.x === 0 && prev.y === 0 ? prev : ZERO));
        return;
      }
      const x = clamp(dx * strength, maxOffset);
      const y = clamp(dy * strength, maxOffset);
      setOffset({ x, y });
    };

    const onMove = (e: MouseEvent) => {
      lastEvent = e;
      if (rafId === 0) rafId = requestAnimationFrame(flush);
    };

    const onLeave = () => {
      lastEvent = null;
      setOffset((prev) => (prev.x === 0 && prev.y === 0 ? prev : ZERO));
    };

    document.addEventListener("mousemove", onMove, { passive: true });
    document.addEventListener("mouseleave", onLeave);

    return () => {
      document.removeEventListener("mousemove", onMove);
      document.removeEventListener("mouseleave", onLeave);
      if (rafId !== 0) cancelAnimationFrame(rafId);
    };
  }, [ref, radius, strength, maxOffset]);

  return offset;
}

function clamp(value: number, max: number): number {
  if (value > max) return max;
  if (value < -max) return -max;
  return value;
}
