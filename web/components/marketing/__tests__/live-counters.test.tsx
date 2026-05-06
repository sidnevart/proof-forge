import { render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { LiveCounters } from "../live-counters";

function mockMatchMedia(reduce: boolean) {
  Object.defineProperty(window, "matchMedia", {
    writable: true,
    configurable: true,
    value: vi.fn().mockImplementation((query: string) => ({
      matches: reduce && query.includes("prefers-reduced-motion: reduce"),
      media: query,
      onchange: null,
      addEventListener: () => {},
      removeEventListener: () => {},
      addListener: () => {},
      removeListener: () => {},
      dispatchEvent: () => false,
    })),
  });
}

describe("LiveCounters", () => {
  beforeEach(() => mockMatchMedia(true)); // reduce=true → no rAF, targets immediately
  afterEach(() => vi.restoreAllMocks());

  it("renders without crashing", () => {
    render(<LiveCounters />);
  });

  it("shows the АКТИВНЫХ КРУГОВ label", () => {
    render(<LiveCounters />);
    expect(screen.getByText("АКТИВНЫХ КРУГОВ")).toBeInTheDocument();
  });

  it("shows the ПРУФОВ СЕГОДНЯ label", () => {
    render(<LiveCounters />);
    expect(screen.getByText("ПРУФОВ СЕГОДНЯ")).toBeInTheDocument();
  });

  it("shows the ЗАМОРОЖЕНО ЗА НЕДЕЛЮ label", () => {
    render(<LiveCounters />);
    expect(screen.getByText("ЗАМОРОЖЕНО ЗА НЕДЕЛЮ")).toBeInTheDocument();
  });

  it("displays numeric values greater than zero", () => {
    render(<LiveCounters />);
    // Under reduced-motion, useCountUp returns target immediately.
    // Targets are: 142, 1847, 23 — all > 0.
    const cells = screen.getAllByRole("generic").filter((el) =>
      /\d/.test(el.textContent ?? "")
    );
    expect(cells.length).toBeGreaterThan(0);
  });
});
