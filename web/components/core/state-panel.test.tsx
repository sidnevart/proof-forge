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
});
