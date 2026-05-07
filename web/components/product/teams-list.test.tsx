import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import type { TeamDetail } from "@/lib/types";

import { TeamsList } from "./teams-list";

function makeTeam(over: Partial<TeamDetail> = {}): TeamDetail {
  return {
    team: {
      id: 1,
      lead_user_id: 7,
      name: "ML",
      invite_code: "AAAA",
      member_limit: 25,
      ai_mode: "metadata-only",
      created_at: "2026-05-07T00:00:00Z",
      archived_at: null,
      ...(over.team ?? {}),
    },
    my_membership: {
      team_id: 1,
      user_id: 7,
      role: "lead",
      status: "active",
      ai_consent: false,
      timezone: "Europe/Moscow",
      joined_at: "2026-05-07T00:00:00Z",
      ...(over.my_membership ?? {}),
    },
    member_count: over.member_count ?? 1,
  };
}

describe("TeamsList", () => {
  it("renders empty state with default CTA", () => {
    render(<TeamsList teams={[]} />);
    expect(screen.getByText("КОМАНД ПОКА НЕТ.")).toBeInTheDocument();
    const cta = screen.getByRole("link", { name: /СОЗДАТЬ КОМАНДУ/ });
    expect(cta).toHaveAttribute("href", "/teams/new");
  });

  it("renders team rows with my_role badge", () => {
    const teams = [
      makeTeam({ team: { id: 1, name: "ML", lead_user_id: 7 } as TeamDetail["team"] }),
      makeTeam({
        team: { id: 2, name: "Frontend", lead_user_id: 7 } as TeamDetail["team"],
        my_membership: { role: "trusted_approver" } as TeamDetail["my_membership"],
        member_count: 4,
      }),
      makeTeam({
        team: { id: 3, name: "Design", lead_user_id: 7 } as TeamDetail["team"],
        my_membership: { role: "member" } as TeamDetail["my_membership"],
        member_count: 8,
      }),
    ];
    render(<TeamsList teams={teams} />);

    expect(screen.getByText("ML")).toBeInTheDocument();
    expect(screen.getByText("FRONTEND")).toBeInTheDocument();
    expect(screen.getByText("DESIGN")).toBeInTheDocument();
    expect(screen.getByText("ТИМЛИД")).toBeInTheDocument();
    expect(screen.getByText("ДОВЕРЕННЫЙ")).toBeInTheDocument();
    expect(screen.getByText("УЧАСТНИК")).toBeInTheDocument();
  });

  it("links each row to /teams/:id", () => {
    const teams = [
      makeTeam({ team: { id: 42, name: "X", lead_user_id: 7 } as TeamDetail["team"] }),
    ];
    render(<TeamsList teams={teams} />);
    const link = screen.getByRole("link", { name: /X/ });
    expect(link).toHaveAttribute("href", "/teams/42");
  });

  it("marks archived teams visibly", () => {
    const teams = [
      makeTeam({
        team: {
          id: 9,
          name: "Old",
          lead_user_id: 7,
          archived_at: "2026-04-01T00:00:00Z",
        } as TeamDetail["team"],
      }),
    ];
    render(<TeamsList teams={teams} />);
    expect(screen.getByText(/В АРХИВЕ/)).toBeInTheDocument();
  });
});
