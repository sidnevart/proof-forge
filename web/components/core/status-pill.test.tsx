import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { StatusPill } from "./status-pill";

describe("StatusPill", () => {
  it("uses russian default labels", () => {
    render(
      <div>
        <StatusPill status="active" />
        <StatusPill status="approved" />
        <StatusPill status="pending" />
        <StatusPill status="changes_requested" />
        <StatusPill status="rejected" />
      </div>,
    );

    expect(screen.getByText("Активно")).toBeInTheDocument();
    expect(screen.getByText("Подтверждено")).toBeInTheDocument();
    expect(screen.getByText("Ожидает")).toBeInTheDocument();
    expect(screen.getByText("Нужна доработка")).toBeInTheDocument();
    expect(screen.getByText("Отклонено")).toBeInTheDocument();
  });
});
