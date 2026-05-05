import { fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { LandingPage } from "./landing-page";

type MediaQueryList = {
  matches: boolean;
  media: string;
  onchange: null;
  addEventListener: () => void;
  removeEventListener: () => void;
  addListener: () => void;
  removeListener: () => void;
  dispatchEvent: () => boolean;
};

function mockMatchMedia(reduceMotion: boolean) {
  const fn = vi.fn().mockImplementation((query: string): MediaQueryList => ({
    matches: reduceMotion && query.includes("prefers-reduced-motion: reduce"),
    media: query,
    onchange: null,
    addEventListener: () => {},
    removeEventListener: () => {},
    addListener: () => {},
    removeListener: () => {},
    dispatchEvent: () => false,
  }));
  Object.defineProperty(window, "matchMedia", {
    writable: true,
    configurable: true,
    value: fn,
  });
}

describe("LandingPage", () => {
  beforeEach(() => {
    mockMatchMedia(false);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("renders brutal landing with hero and CTA", () => {
    render(<LandingPage />);

    expect(screen.getByText(/ТВОИ ДРУЗЬЯ/)).toBeInTheDocument();
    expect(screen.getByText("КАК ЭТО РАБОТАЕТ")).toBeInTheDocument();
    expect(screen.getByText("ДО ЗАМОРОЗКИ")).toBeInTheDocument();
    // Hero CTA "СОБРАТЬ КРУГ" + final CTA "ОБЪЯВИТЬ ЦЕЛЬ" are the two primary
    // links to /dashboard. The old "ВОЙТИ В КРУГ" copy was confusing because
    // an anonymous user has no circle to enter yet.
    expect(screen.getByRole("link", { name: "СОБРАТЬ КРУГ" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /ОБЪЯВИТЬ ЦЕЛЬ/ })).toBeInTheDocument();
  });

  it("updates --cursor-x / --cursor-y on hero mousemove", () => {
    // Stub rAF: first call always fires at t=10_000ms so useCountUp converges
    // to its target in a single frame and stops re-scheduling. Subsequent calls
    // are deferred via queueMicrotask so they don't block the synchronous
    // mouse-move + assertion below.
    let frameTime = 10_000;
    const rafSpy = vi
      .spyOn(window, "requestAnimationFrame")
      .mockImplementation((cb: FrameRequestCallback) => {
        const t = frameTime;
        frameTime += 10_000; // keep advancing so t>=1 always
        cb(t);
        return 1;
      });

    const { container } = render(<LandingPage />);
    const hero = container.querySelector("section") as HTMLElement;
    expect(hero).toBeTruthy();

    // Force a known bounding rect — jsdom returns zeros otherwise.
    hero.getBoundingClientRect = () => ({
      x: 0,
      y: 0,
      left: 0,
      top: 0,
      right: 200,
      bottom: 100,
      width: 200,
      height: 100,
      toJSON: () => ({}),
    });

    fireEvent.mouseMove(hero, { clientX: 100, clientY: 50 });

    expect(hero.style.getPropertyValue("--cursor-x")).toBe("50%");
    expect(hero.style.getPropertyValue("--cursor-y")).toBe("50%");

    rafSpy.mockRestore();
  });

  it("does not attach the spotlight listener when reduced motion is preferred", () => {
    mockMatchMedia(true);

    const { container } = render(<LandingPage />);
    const hero = container.querySelector("section") as HTMLElement;
    expect(hero).toBeTruthy();

    hero.getBoundingClientRect = () => ({
      x: 0,
      y: 0,
      left: 0,
      top: 0,
      right: 200,
      bottom: 100,
      width: 200,
      height: 100,
      toJSON: () => ({}),
    });

    fireEvent.mouseMove(hero, { clientX: 100, clientY: 50 });

    expect(hero.style.getPropertyValue("--cursor-x")).toBe("");
    expect(hero.style.getPropertyValue("--cursor-y")).toBe("");
  });

  it("renders the ScrollProgress bar", () => {
    render(<LandingPage />);
    expect(screen.getByTestId("scroll-progress")).toBeInTheDocument();
  });

  it("renders live counter labels", () => {
    render(<LandingPage />);
    expect(screen.getByText("АКТИВНЫХ КРУГОВ")).toBeInTheDocument();
    expect(screen.getByText("ПРУФОВ СЕГОДНЯ")).toBeInTheDocument();
    expect(screen.getByText("ЗАМОРОЖЕНО ЗА НЕДЕЛЮ")).toBeInTheDocument();
  });

  it("renders FAQ accordion with first item expanded", () => {
    render(<LandingPage />);
    const firstBtn = screen.getByRole("button", { name: /ЧТО ЕСЛИ Я ВЫПАЛ/ });
    expect(firstBtn).toHaveAttribute("aria-expanded", "true");

    // Other FAQ buttons are collapsed
    const secondBtn = screen.getByRole("button", { name: /ЭТО БЕСПЛАТНО/ });
    expect(secondBtn).toHaveAttribute("aria-expanded", "false");
  });
});
