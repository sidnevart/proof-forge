import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Button } from "./button";

describe("Button", () => {
  it("renders children", () => {
    render(<Button>Open loop</Button>);
    expect(screen.getByRole("button", { name: "Open loop" })).toBeInTheDocument();
  });
});
