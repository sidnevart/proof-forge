import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { LandingPage } from "./landing-page";

describe("LandingPage", () => {
  it("explains the russian core loop and scenarios", () => {
    render(<LandingPage />);

    expect(
      screen.getByText("Внешняя система ответственности для серьёзных целей"),
    ).toBeInTheDocument();
    expect(screen.getByText("Как работает цикл")).toBeInTheDocument();
    expect(screen.getByText("Где это особенно полезно")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Начать первую цель" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Посмотреть рабочий экран" }),
    ).toBeInTheDocument();
  });
});
