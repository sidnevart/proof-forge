import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { CheckInScreen } from "./checkin-screen";

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: vi.fn() }),
}));

const BASE_CHECK_IN = {
  id: 1,
  goal_id: 10,
  owner_user_id: 1,
  status: "draft",
  created_at: "2026-05-01T10:00:00Z",
  updated_at: "2026-05-01T10:00:00Z",
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

describe("CheckInScreen", () => {
  const fetchMock = vi.fn<typeof fetch>();

  beforeEach(() => {
    vi.stubGlobal("fetch", fetchMock);
  });

  afterEach(() => {
    cleanup();
    vi.unstubAllGlobals();
    fetchMock.mockReset();
  });

  it("shows Start button when no draft exists", async () => {
    fetchMock.mockResolvedValueOnce(json({ check_ins: [] }));

    render(<CheckInScreen goalID={10} />);

    expect(await screen.findByRole("button", { name: "Подготовить подтверждение" })).toBeInTheDocument();
  });

  it("creates draft check-in on Start click", async () => {
    fetchMock
      .mockResolvedValueOnce(json({ check_ins: [] }))
      .mockResolvedValueOnce(json({ check_in: BASE_CHECK_IN }, 201));

    render(<CheckInScreen goalID={10} />);

    const startBtn = await screen.findByRole("button", { name: "Подготовить подтверждение" });
    fireEvent.click(startBtn);

    await waitFor(() => {
      expect(screen.getByRole("button", { name: "Отправить на проверку" })).toBeInTheDocument();
    });
  });

  it("loads existing draft without creating a new one", async () => {
    fetchMock.mockResolvedValueOnce(json({ check_ins: [BASE_CHECK_IN] }));

    render(<CheckInScreen goalID={10} />);

    expect(await screen.findByRole("button", { name: "Отправить на проверку" })).toBeInTheDocument();
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it("shows unauthenticated state on 401", async () => {
    fetchMock.mockResolvedValueOnce(errorJson("auth_required", "Authentication required", 401));

    render(<CheckInScreen goalID={10} />);

    expect(await screen.findByText("Сессия не найдена")).toBeInTheDocument();
  });

  it("adds text evidence and shows it in the list", async () => {
    fetchMock
      .mockResolvedValueOnce(json({ check_ins: [BASE_CHECK_IN] }))
      .mockResolvedValueOnce(
        json(
          {
            evidence: {
              id: 5,
              check_in_id: 1,
              kind: "text",
              text_content: "Shipped the feature",
              created_at: "2026-05-01T10:00:00Z",
            },
          },
          201,
        ),
      );

    render(<CheckInScreen goalID={10} />);
    await screen.findByRole("button", { name: "Отправить на проверку" });

    const textarea = screen.getByPlaceholderText("Опишите конкретный результат или сделанный шаг");
    fireEvent.change(textarea, { target: { value: "Shipped the feature" } });
    fireEvent.submit(screen.getByRole("button", { name: "Добавить текст" }).closest("form")!);

    await waitFor(() => {
      expect(screen.getByText("Shipped the feature")).toBeInTheDocument();
    });
  });

  it("adds link evidence and shows it in the list", async () => {
    fetchMock
      .mockResolvedValueOnce(json({ check_ins: [BASE_CHECK_IN] }))
      .mockResolvedValueOnce(
        json(
          {
            evidence: {
              id: 6,
              check_in_id: 1,
              kind: "link",
              external_url: "https://github.com/pr/123",
              created_at: "2026-05-01T10:00:00Z",
            },
          },
          201,
        ),
      );

    render(<CheckInScreen goalID={10} />);
    await screen.findByRole("button", { name: "Отправить на проверку" });

    const input = screen.getByPlaceholderText("https://github.com/...");
    fireEvent.change(input, { target: { value: "https://github.com/pr/123" } });
    fireEvent.submit(screen.getByRole("button", { name: "Добавить ссылку" }).closest("form")!);

    await waitFor(() => {
      expect(screen.getByText("https://github.com/pr/123")).toBeInTheDocument();
    });
  });

  it("disables submit when no evidence items", async () => {
    fetchMock.mockResolvedValueOnce(json({ check_ins: [BASE_CHECK_IN] }));

    render(<CheckInScreen goalID={10} />);

    const submitBtn = await screen.findByRole("button", { name: "Отправить на проверку" });
    expect(submitBtn).toBeDisabled();
  });

  it("transitions to submitted state on success", async () => {
    fetchMock
      .mockResolvedValueOnce(
        json({
          check_ins: [
            {
              ...BASE_CHECK_IN,
            },
          ],
        }),
      )
      .mockResolvedValueOnce(
        json({
          evidence: { id: 5, check_in_id: 1, kind: "text", text_content: "Done", created_at: "2026-05-01T10:00:00Z" },
        }, 201),
      )
      .mockResolvedValueOnce(json({ submitted: true }));

    render(<CheckInScreen goalID={10} />);
    await screen.findByRole("button", { name: "Отправить на проверку" });

    const textarea = screen.getByPlaceholderText("Опишите конкретный результат или сделанный шаг");
    fireEvent.change(textarea, { target: { value: "Done" } });
    fireEvent.submit(screen.getByRole("button", { name: "Добавить текст" }).closest("form")!);

    const submitBtn = await screen.findByRole("button", { name: "Отправить на проверку" });
    fireEvent.click(submitBtn);

    await waitFor(() => {
      expect(screen.getByText("Подтверждение отправлено на проверку")).toBeInTheDocument();
    });
  });

  it("shows changes_requested banner for that status", async () => {
    fetchMock.mockResolvedValueOnce(
      json({ check_ins: [{ ...BASE_CHECK_IN, status: "changes_requested" }] }),
    );

    render(<CheckInScreen goalID={10} />);

    expect(await screen.findByText("Нужно дополнить подтверждение")).toBeInTheDocument();
  });
});
