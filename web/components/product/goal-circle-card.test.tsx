import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { GoalCircleCard } from "./goal-circle-card";

describe("GoalCircleCard", () => {
  it("frames goal progress around streaks instead of seasons", () => {
    render(
      <GoalCircleCard
        circleId={1}
        goalId={10}
        goalTitle="Собрать investor update"
        goalStatus="active"
        buddyStatus="active"
        proofStreak={5}
        membersCount={2}
        viewerRole="owner"
      />,
    );

    expect(screen.getByLabelText("СЕРИЯ 5")).toBeInTheDocument();
    expect(screen.getByText(/СЛЕДУЮЩИЙ ПРУФ ДЕРЖИТ СЕРИЮ ЖИВОЙ/)).toBeInTheDocument();
    expect(screen.queryByText(/День/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/сезон/i)).not.toBeInTheDocument();
  });
});
