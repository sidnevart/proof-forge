import { cleanup, fireEvent, render } from "@testing-library/react";
import { useRef } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { useMagneticHover } from "../use-magnetic-hover";

interface MediaSettings {
  reduce?: boolean;
  hoverable?: boolean;
}

function mockMatchMedia({ reduce = false, hoverable = true }: MediaSettings = {}) {
  Object.defineProperty(window, "matchMedia", {
    writable: true,
    configurable: true,
    value: vi.fn().mockImplementation((query: string) => {
      if (query.includes("prefers-reduced-motion: reduce")) {
        return media(reduce, query);
      }
      if (query.includes("hover: hover") && query.includes("pointer: fine")) {
        return media(hoverable, query);
      }
      return media(false, query);
    }),
  });
}

function media(matches: boolean, query: string) {
  return {
    matches,
    media: query,
    onchange: null,
    addEventListener: () => {},
    removeEventListener: () => {},
    addListener: () => {},
    removeListener: () => {},
    dispatchEvent: () => false,
  };
}

function Probe() {
  const ref = useRef<HTMLDivElement>(null);
  const { x, y } = useMagneticHover(ref, { radius: 200, strength: 0.5, maxOffset: 20 });
  return (
    <div ref={ref} data-testid="target" data-offset={`${x},${y}`}>
      magnet
    </div>
  );
}

describe("useMagneticHover", () => {
  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
  });

  it("does not pull the element when the device cannot hover", () => {
    mockMatchMedia({ hoverable: false });
    const rafSpy = vi
      .spyOn(window, "requestAnimationFrame")
      .mockImplementation((cb: FrameRequestCallback) => {
        cb(0);
        return 1;
      });

    const { getByTestId } = render(<Probe />);
    fireEvent.mouseMove(document, { clientX: 100, clientY: 100 });

    expect(getByTestId("target").dataset.offset).toBe("0,0");
    rafSpy.mockRestore();
  });

  it("does not register listeners when reduced motion is preferred", () => {
    mockMatchMedia({ reduce: true });
    const addSpy = vi.spyOn(document, "addEventListener");

    render(<Probe />);
    expect(addSpy).not.toHaveBeenCalledWith(
      "mousemove",
      expect.any(Function),
      expect.anything()
    );
    addSpy.mockRestore();
  });

  describe("with hover-capable, full-motion device", () => {
    beforeEach(() => {
      mockMatchMedia({ hoverable: true, reduce: false });
    });

    it("returns a non-zero offset when the cursor is inside the radius", () => {
      const rafSpy = vi
        .spyOn(window, "requestAnimationFrame")
        .mockImplementation((cb: FrameRequestCallback) => {
          cb(0);
          return 1;
        });

      const { getByTestId } = render(<Probe />);
      const target = getByTestId("target");
      target.getBoundingClientRect = () => ({
        x: 0,
        y: 0,
        left: 0,
        top: 0,
        right: 100,
        bottom: 50,
        width: 100,
        height: 50,
        toJSON: () => ({}),
      });

      // Cursor 80px right of centre, 25 below — within radius=200
      fireEvent.mouseMove(document, { clientX: 130, clientY: 50 });

      // Compute expected: dx=130-50=80, dy=50-25=25, strength 0.5 → x=40 (clamp 20), y=12.5
      const [x, y] = target.dataset.offset!.split(",").map(Number);
      expect(x).toBe(20); // clamped to maxOffset
      expect(Math.round(y)).toBe(13);

      rafSpy.mockRestore();
    });

    it("snaps back to zero when cursor exits the radius", () => {
      const rafSpy = vi
        .spyOn(window, "requestAnimationFrame")
        .mockImplementation((cb: FrameRequestCallback) => {
          cb(0);
          return 1;
        });

      const { getByTestId } = render(<Probe />);
      const target = getByTestId("target");
      target.getBoundingClientRect = () => ({
        x: 0,
        y: 0,
        left: 0,
        top: 0,
        right: 100,
        bottom: 50,
        width: 100,
        height: 50,
        toJSON: () => ({}),
      });

      fireEvent.mouseMove(document, { clientX: 1000, clientY: 1000 });

      expect(target.dataset.offset).toBe("0,0");
      rafSpy.mockRestore();
    });
  });
});
