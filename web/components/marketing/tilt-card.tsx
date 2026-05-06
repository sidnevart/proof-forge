"use client";

import { useRef } from "react";

import { useTiltOnHover } from "@/lib/use-tilt-on-hover";

interface TiltCardProps {
  children: React.ReactNode;
  className?: string;
  style?: React.CSSProperties;
}

/**
 * Wrapper that applies the useTiltOnHover effect to its outermost div.
 * Drop-in replacement for a plain <div> wherever 3D tilt is wanted.
 */
export function TiltCard({ children, className, style }: TiltCardProps) {
  const ref = useRef<HTMLDivElement | null>(null);
  useTiltOnHover(ref);

  return (
    <div
      ref={ref}
      className={className}
      style={{
        ...style,
        transform: `perspective(1000px) rotateX(var(--tilt-x, 0deg)) rotateY(var(--tilt-y, 0deg))`,
        transition: "transform 200ms ease",
        willChange: "transform",
        position: "relative",
        isolation: "isolate",
      }}
    >
      {children}
    </div>
  );
}
