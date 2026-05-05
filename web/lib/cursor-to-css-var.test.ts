import { describe, expect, it } from "vitest";

import { cursorToCssVars } from "./cursor-to-css-var";

const rect = { left: 100, top: 200, width: 400, height: 200 };

describe("cursorToCssVars", () => {
  it("returns 0%/0% at the top-left corner", () => {
    expect(cursorToCssVars({ clientX: 100, clientY: 200 }, rect)).toEqual({
      x: "0%",
      y: "0%",
    });
  });

  it("returns 100%/100% at the bottom-right corner", () => {
    expect(cursorToCssVars({ clientX: 500, clientY: 400 }, rect)).toEqual({
      x: "100%",
      y: "100%",
    });
  });

  it("returns the centre at the geometric mid-point", () => {
    expect(cursorToCssVars({ clientX: 300, clientY: 300 }, rect)).toEqual({
      x: "50%",
      y: "50%",
    });
  });

  it("clamps cursor positions outside the rectangle", () => {
    expect(cursorToCssVars({ clientX: -200, clientY: 1000 }, rect)).toEqual({
      x: "0%",
      y: "100%",
    });
  });

  it("falls back to 50%/50% when the rectangle is degenerate", () => {
    expect(
      cursorToCssVars({ clientX: 50, clientY: 50 }, { left: 0, top: 0, width: 0, height: 100 })
    ).toEqual({ x: "50%", y: "50%" });
  });

  it("formats fractional positions with up to two decimals", () => {
    const result = cursorToCssVars({ clientX: 233, clientY: 200 }, rect);
    // (233 - 100) / 400 = 0.3325 → 33.25%
    expect(result.x).toBe("33.25%");
  });
});
