import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { getPersonalLeaderboard } from "./api";

describe("api client", () => {
  const fetchMock = vi.fn<typeof fetch>();

  beforeEach(() => {
    vi.stubGlobal("fetch", fetchMock);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    fetchMock.mockReset();
  });

  it("unwraps the personal leaderboard response envelope", async () => {
    fetchMock.mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          data: {
            current_week: {
              proofs_count: 3,
              vs_last_week: "+1",
              trend: "better",
            },
            current_season: {
              proofs_count: 12,
              vs_last_season: "+4",
              trend: "better",
            },
            streak: {
              current_weeks: 2,
              personal_record_weeks: 5,
              is_personal_record: false,
            },
            weekly_history: [
              { week: "2026-W19", proofs_count: 1 },
              { week: "2026-W20", proofs_count: 3 },
            ],
          },
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      ),
    );

    const leaderboard = await getPersonalLeaderboard();

    expect(leaderboard.current_week.proofs_count).toBe(3);
    expect(leaderboard.streak.current_weeks).toBe(2);
    expect(leaderboard.weekly_history[1].proofs_count).toBe(3);
  });
});
