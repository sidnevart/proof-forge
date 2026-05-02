import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { StatePanel } from "./state-panel";

describe("StatePanel", () => {
  it("renders title and description", () => {
    render(
      <StatePanel
        tone="empty"
        title="No proof yet"
        description="Upload the first artifact."
      />,
    );

    expect(screen.getByText("No proof yet")).toBeInTheDocument();
    expect(screen.getByText("Upload the first artifact.")).toBeInTheDocument();
  });

  it("renders russian tone label instead of raw tone key", () => {
    render(
      <StatePanel tone="loading" title="Загружаем экран" description="Проверяем состояние." />,
    );

    expect(screen.getByText("Загрузка")).toBeInTheDocument();
    expect(screen.queryByText("loading")).not.toBeInTheDocument();
  });
});
