"use client";

import { type RefObject, useEffect } from "react";

interface TiltOptions {
  /** Maximum tilt in degrees. Default 6. */
  maxDeg?: number;
}

/**
 * 3D-tilt-on-hover effect.
 *
 * Writes four CSS custom properties on the element (and its descendants
 * via inheritance):
 *   --tilt-x   — rotateX in deg
 *   --tilt-y   — rotateY in deg
 *   --tilt-cx  — cursor X in % (used by glow ::after)
 *   --tilt-cy  — cursor Y in %
 *
 * On `mouseleave`, resets all to 0 / 50%.
 *
 * Disabled on touch devices and when reduced-motion is preferred.
 */
export function useTiltOnHover(
  ref: RefObject<HTMLElement | null>,
  options: TiltOptions = {}
): void {
  const { maxDeg = 6 } = options;

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
      if (rect.width === 0 || rect.height === 0) return;
      const px = (lastEvent.clientX - rect.left) / rect.width;
      const py = (lastEvent.clientY - rect.top) / rect.height;
      const rotY = (px - 0.5) * maxDeg * 2;
      const rotX = (py - 0.5) * maxDeg * -2;
      node.style.setProperty("--tilt-x", `${rotX.toFixed(2)}deg`);
      node.style.setProperty("--tilt-y", `${rotY.toFixed(2)}deg`);
      node.style.setProperty("--tilt-cx", `${(px * 100).toFixed(2)}%`);
      node.style.setProperty("--tilt-cy", `${(py * 100).toFixed(2)}%`);
    };

    const onMove = (e: MouseEvent) => {
      lastEvent = e;
      if (rafId === 0) rafId = requestAnimationFrame(flush);
    };

    const onLeave = () => {
      lastEvent = null;
      node.style.setProperty("--tilt-x", "0deg");
      node.style.setProperty("--tilt-y", "0deg");
      node.style.setProperty("--tilt-cx", "50%");
      node.style.setProperty("--tilt-cy", "50%");
    };

    node.addEventListener("mousemove", onMove, { passive: true });
    node.addEventListener("mouseleave", onLeave);

    return () => {
      node.removeEventListener("mousemove", onMove);
      node.removeEventListener("mouseleave", onLeave);
      if (rafId !== 0) cancelAnimationFrame(rafId);
    };
  }, [ref, maxDeg]);
}
