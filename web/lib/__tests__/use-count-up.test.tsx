import { render } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { useCountUp } from "../use-count-up";

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

function Probe({ target, duration }: { target: number; duration?: number }) {
  const value = useCountUp(target, duration);
  return <span data-testid="value">{value}</span>;
}

describe("useCountUp", () => {
  afterEach(() => vi.restoreAllMocks());

  describe("with reduced motion", () => {
    beforeEach(() => mockMatchMedia(true));

    it("returns the target immediately without animating", () => {
      const { getByTestId } = render(<Probe target={1847} />);
      expect(getByTestId("value").textContent).toBe("1847");
    });
  });

  describe("with normal motion", () => {
    beforeEach(() => mockMatchMedia(false));

    it("reaches the target value when the animation completes", () => {
      // Drive each rAF call straight to the duration end so the loop converges in 1 frame.
      const rafSpy = vi
        .spyOn(window, "requestAnimationFrame")
        .mockImplementation((cb: FrameRequestCallback) => {
          // First call sets the start time; second call advances past `duration`.
          // Using a large timestamp (10_000) guarantees `t >= 1` immediately.
          queueMicrotask(() => cb(10_000));
          return 1;
        });

      const { getByTestId } = render(<Probe target={42} duration={300} />);

      // Wait for queued microtasks (which run rAF callbacks).
      return new Promise<void>((resolve) => {
        queueMicrotask(() => {
          // Two rAF passes: first records start time, second eases past completion.
          queueMicrotask(() => {
            expect(getByTestId("value").textContent).toBe("42");
            rafSpy.mockRestore();
            resolve();
          });
        });
      });
    });
  });
});
