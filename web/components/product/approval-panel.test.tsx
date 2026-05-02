import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { ApprovalPanel } from "./approval-panel";

const BASE_CHECK_IN = {
  id: 1,
  goal_id: 10,
  owner_user_id: 1,
  status: "submitted",
  submitted_at: "2026-05-01T10:00:00Z",
  created_at: "2026-05-01T09:00:00Z",
  updated_at: "2026-05-01T10:00:00Z",
};

const TEXT_EVIDENCE = {
  id: 5,
  check_in_id: 1,
  kind: "text",
  text_content: "Merged the PR",
  created_at: "2026-05-01T09:30:00Z",
};

function json(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

function errorJson(code: string, message: string, status: number) {
  return json({ error: { code, message } }, status);
}

describe("ApprovalPanel", () => {
  const fetchMock = vi.fn<typeof fetch>();

  beforeEach(() => {
    vi.stubGlobal("fetch", fetchMock);
  });

  afterEach(() => {
    cleanup();
    vi.unstubAllGlobals();
    fetchMock.mockReset();
  });

  it("shows evidence from the submitted check-in", async () => {
    fetchMock.mockResolvedValueOnce(
      json({ check_in: BASE_CHECK_IN, evidence: [TEXT_EVIDENCE] }),
    );

    render(<ApprovalPanel checkInID={1} />);

    expect(await screen.findByText("Merged the PR")).toBeInTheDocument();
  });

  it("shows decision buttons when check-in is submitted", async () => {
    fetchMock.mockResolvedValueOnce(json({ check_in: BASE_CHECK_IN, evidence: [] }));

    render(<ApprovalPanel checkInID={1} />);

    expect(await screen.findByRole("button", { name: "Подтвердить" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Отклонить" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Вернуть на доработку" })).toBeInTheDocument();
  });

  it("shows approved state after approve click", async () => {
    fetchMock
      .mockResolvedValueOnce(json({ check_in: BASE_CHECK_IN, evidence: [] }))
      .mockResolvedValueOnce(
        json({ review: { id: 1, check_in_id: 1, reviewer_user_id: 2, decision: "approved", created_at: "2026-05-01T11:00:00Z" } }),
      );

    render(<ApprovalPanel checkInID={1} />);
    const approveBtn = await screen.findByRole("button", { name: "Подтвердить" });
    fireEvent.click(approveBtn);

    await waitFor(() => {
      expect(screen.getByText("Подтверждение принято")).toBeInTheDocument();
    });
  });

  it("shows rejected state after reject click", async () => {
    fetchMock
      .mockResolvedValueOnce(json({ check_in: BASE_CHECK_IN, evidence: [] }))
      .mockResolvedValueOnce(
        json({ review: { id: 2, check_in_id: 1, reviewer_user_id: 2, decision: "rejected", created_at: "2026-05-01T11:00:00Z" } }),
      );

    render(<ApprovalPanel checkInID={1} />);
    const rejectBtn = await screen.findByRole("button", { name: "Отклонить" });
    fireEvent.click(rejectBtn);

    await waitFor(() => {
      expect(screen.getByText("Подтверждение отклонено")).toBeInTheDocument();
    });
  });

  it("shows changes_requested state after request-changes click", async () => {
    fetchMock
      .mockResolvedValueOnce(json({ check_in: BASE_CHECK_IN, evidence: [] }))
      .mockResolvedValueOnce(
        json({ review: { id: 3, check_in_id: 1, reviewer_user_id: 2, decision: "changes_requested", created_at: "2026-05-01T11:00:00Z" } }),
      );

    render(<ApprovalPanel checkInID={1} />);
    const reqBtn = await screen.findByRole("button", { name: "Вернуть на доработку" });
    fireEvent.click(reqBtn);

    await waitFor(() => {
      expect(screen.getByText("Запрошена доработка")).toBeInTheDocument();
    });
  });

  it("shows not_submitted state when check-in is not submitted", async () => {
    fetchMock.mockResolvedValueOnce(
      json({ check_in: { ...BASE_CHECK_IN, status: "draft" }, evidence: [] }),
    );

    render(<ApprovalPanel checkInID={1} />);

    expect(await screen.findByText("Подтверждение ещё не отправлено")).toBeInTheDocument();
  });

  it("shows unauthenticated state on 401", async () => {
    fetchMock.mockResolvedValueOnce(errorJson("auth_required", "Authentication required", 401));

    render(<ApprovalPanel checkInID={1} />);

    expect(await screen.findByText("Сессия не найдена")).toBeInTheDocument();
  });

  it("shows error state on network failure", async () => {
    fetchMock.mockResolvedValueOnce(errorJson("not_found", "Not found", 404));

    render(<ApprovalPanel checkInID={999} />);

    expect((await screen.findAllByText("Ошибка")).length).toBeGreaterThan(0);
  });

  it("passes comment to approve request", async () => {
    fetchMock
      .mockResolvedValueOnce(json({ check_in: BASE_CHECK_IN, evidence: [] }))
      .mockResolvedValueOnce(
        json({ review: { id: 1, check_in_id: 1, reviewer_user_id: 2, decision: "approved", created_at: "2026-05-01T11:00:00Z" } }),
      );

    render(<ApprovalPanel checkInID={1} />);
    await screen.findByRole("button", { name: "Подтвердить" });

    const textarea = screen.getByPlaceholderText("Поясните решение для владельца");
    fireEvent.change(textarea, { target: { value: "Excellent work" } });
    fireEvent.click(screen.getByRole("button", { name: "Подтвердить" }));

    await waitFor(() => {
      const body = JSON.parse(fetchMock.mock.calls[1][1]?.body as string);
      expect(body.comment).toBe("Excellent work");
    });
  });
});
