/**
 * Convert a cursor position into CSS custom-property values (percentage strings)
 * relative to a rectangle. Used by the marketing landing hero spotlight.
 *
 * Pure function — no DOM access — so it can be unit-tested without a browser.
 *
 * Behavior:
 *   - Output is clamped to [0%, 100%]
 *   - Zero-sized rectangles return "50%" / "50%" (sensible centred fallback)
 *   - Numbers are formatted with up to two decimal places (no trailing zeros)
 */
export interface CursorPoint {
  clientX: number;
  clientY: number;
}

export interface CursorRect {
  left: number;
  top: number;
  width: number;
  height: number;
}

export interface CursorCssVars {
  x: string;
  y: string;
}

const CENTER: CursorCssVars = { x: "50%", y: "50%" };

function clamp01(value: number): number {
  if (Number.isNaN(value)) return 0.5;
  if (value < 0) return 0;
  if (value > 1) return 1;
  return value;
}

function formatPercent(ratio: number): string {
  const pct = clamp01(ratio) * 100;
  // Trim trailing zeros: 50.00 → 50, 33.33 → 33.33
  const rounded = Math.round(pct * 100) / 100;
  return `${rounded}%`;
}

export function cursorToCssVars(point: CursorPoint, rect: CursorRect): CursorCssVars {
  if (rect.width <= 0 || rect.height <= 0) return CENTER;

  const x = (point.clientX - rect.left) / rect.width;
  const y = (point.clientY - rect.top) / rect.height;

  return {
    x: formatPercent(x),
    y: formatPercent(y),
  };
}
