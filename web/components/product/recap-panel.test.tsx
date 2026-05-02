import { cleanup, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { RecapPanel } from "./recap-panel";

const BASE_RECAP = {
  id: 1,
  goal_id: 10,
  owner_user_id: 1,
  period_start: "2026-04-27T00:00:00Z",
  period_end: "2026-05-04T00:00:00Z",
  status: "done" as const,
  summary_text: "Great progress this week. The owner merged two PRs and ran 3 sessions.",
  model_name: "gpt-4o-mini",
  created_at: "2026-04-27T00:00:00Z",
};

function json(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

describe("RecapPanel", () => {
  const fetchMock = vi.fn<typeof fetch>();

  beforeEach(() => {
    vi.stubGlobal("fetch", fetchMock);
  });

  afterEach(() => {
    cleanup();
    vi.unstubAllGlobals();
    fetchMock.mockReset();
  });

  it("shows recap summary when status is done", async () => {
    fetchMock.mockResolvedValueOnce(json({ recaps: [BASE_RECAP] }));

    render(<RecapPanel goalID={10} />);

    expect(await screen.findByText("Great progress this week. The owner merged two PRs and ran 3 sessions.")).toBeInTheDocument();
  });

  it("shows the period range", async () => {
    fetchMock.mockResolvedValueOnce(json({ recaps: [BASE_RECAP] }));

    render(<RecapPanel goalID={10} />);

    // period is rendered as formatted dates
    await waitFor(() => {
      expect(screen.getByText(/апр/i)).toBeInTheDocument();
    });
  });

  it("shows empty state when no recaps", async () => {
    fetchMock.mockResolvedValueOnce(json({ recaps: null }));

    render(<RecapPanel goalID={10} />);

    expect(await screen.findByText("Пока нет недельных сводок")).toBeInTheDocument();
  });

  it("shows generating state for in-progress recap", async () => {
    fetchMock.mockResolvedValueOnce(
      json({ recaps: [{ ...BASE_RECAP, status: "generating", summary_text: "" }] }),
    );

    render(<RecapPanel goalID={10} />);

    expect(await screen.findByText(/Генерируем/i)).toBeInTheDocument();
  });

  it("shows failed state for failed recap", async () => {
    fetchMock.mockResolvedValueOnce(
      json({ recaps: [{ ...BASE_RECAP, status: "failed", summary_text: "" }] }),
    );

    render(<RecapPanel goalID={10} />);

    expect(await screen.findByText(/Не удалось сгенерировать/i)).toBeInTheDocument();
  });

  it("shows error state on fetch failure", async () => {
    fetchMock.mockResolvedValueOnce(
      new Response(JSON.stringify({ error: { code: "not_found", message: "Not found" } }), { status: 404 }),
    );

    render(<RecapPanel goalID={10} />);

    expect((await screen.findAllByText("Ошибка")).length).toBeGreaterThan(0);
  });

  it("shows multiple recaps", async () => {
    const second = { ...BASE_RECAP, id: 2, summary_text: "Second week summary." };
    fetchMock.mockResolvedValueOnce(json({ recaps: [BASE_RECAP, second] }));

    render(<RecapPanel goalID={10} />);

    await waitFor(() => {
      expect(screen.getByText("Недельные сводки (2)")).toBeInTheDocument();
    });
    expect(screen.getByText("Second week summary.")).toBeInTheDocument();
  });
});
