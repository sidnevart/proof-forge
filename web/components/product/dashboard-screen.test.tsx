import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { DashboardScreen } from "./dashboard-screen";

describe("DashboardScreen", () => {
  const fetchMock = vi.fn<typeof fetch>();

  beforeEach(() => {
    vi.stubGlobal("fetch", fetchMock);
  });

  afterEach(() => {
    cleanup();
    vi.unstubAllGlobals();
    fetchMock.mockReset();
  });

  it("shows registration form when session is missing", async () => {
    fetchMock.mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          error: { code: "auth_required", message: "Authentication required" },
        }),
        { status: 401, headers: { "Content-Type": "application/json" } },
      ),
    );

    render(<DashboardScreen />);

    expect(await screen.findByRole("button", { name: "Создать аккаунт" })).toBeInTheDocument();
    expect(screen.getByText("Войдите, чтобы держать цель под контролем")).toBeInTheDocument();
  });

  it("renders real dashboard surfaces for authenticated user", async () => {
    fetchMock.mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          user: {
            id: 1,
            email: "owner@example.com",
            display_name: "Артём",
            created_at: "2026-05-01T10:00:00Z",
            updated_at: "2026-05-01T10:00:00Z",
          },
          summary: {
            total_goals: 1,
            pending_buddy_acceptance: 1,
            active_goals: 0,
          },
          goals: [
            {
              goal: {
                id: 9,
                title: "Ship MVP vertical slice",
                description: "Registration + goals dashboard",
                status: "pending_buddy_acceptance",
                current_progress_health: "unknown",
                current_streak_count: 0,
                created_at: "2026-05-01T10:00:00Z",
                updated_at: "2026-05-01T10:00:00Z",
              },
              buddy: {
                id: 2,
                email: "peer@example.com",
                display_name: "Peer",
              },
              pact: {
                id: 3,
                status: "invited",
              },
              invite: {
                id: 4,
                status: "pending",
                expires_at: "2026-05-08T10:00:00Z",
              },
            },
          ],
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      ),
    );

    render(<DashboardScreen />);

    expect((await screen.findAllByText("Ship MVP vertical slice")).length).toBeGreaterThan(0);
    expect(screen.getByText("Следующий шаг")).toBeInTheDocument();
    expect(screen.getAllByText("Главная цель").length).toBeGreaterThan(0);
    expect(screen.getByText("История подтверждений")).toBeInTheDocument();
    expect(screen.getByText("Недельная сводка")).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Создать goal" }),
    ).not.toBeInTheDocument();
  });

  it("completes registration and opens the dedicated goal creation flow", async () => {
    fetchMock
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            error: { code: "auth_required", message: "Authentication required" },
          }),
          { status: 401, headers: { "Content-Type": "application/json" } },
        ),
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            user: {
              id: 1,
              email: "owner@example.com",
              display_name: "Owner",
              created_at: "2026-05-01T10:00:00Z",
              updated_at: "2026-05-01T10:00:00Z",
            },
          }),
          { status: 201, headers: { "Content-Type": "application/json" } },
        ),
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            user: {
              id: 1,
              email: "owner@example.com",
              display_name: "Owner",
              created_at: "2026-05-01T10:00:00Z",
              updated_at: "2026-05-01T10:00:00Z",
            },
            summary: {
              total_goals: 0,
              pending_buddy_acceptance: 0,
              active_goals: 0,
            },
            goals: [],
          }),
          { status: 200, headers: { "Content-Type": "application/json" } },
        ),
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            goal: {
              goal: {
                id: 9,
                title: "Ship MVP vertical slice",
                description: "Registration + goals dashboard",
                status: "pending_buddy_acceptance",
                current_progress_health: "unknown",
                current_streak_count: 0,
                created_at: "2026-05-01T10:00:00Z",
                updated_at: "2026-05-01T10:00:00Z",
              },
              buddy: {
                id: 2,
                email: "peer@example.com",
                display_name: "Peer",
              },
              pact: {
                id: 3,
                status: "invited",
              },
              invite: {
                id: 4,
                status: "pending",
                expires_at: "2026-05-08T10:00:00Z",
              },
            },
          }),
          { status: 201, headers: { "Content-Type": "application/json" } },
        ),
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            user: {
              id: 1,
              email: "owner@example.com",
              display_name: "Owner",
              created_at: "2026-05-01T10:00:00Z",
              updated_at: "2026-05-01T10:00:00Z",
            },
            summary: {
              total_goals: 1,
              pending_buddy_acceptance: 1,
              active_goals: 0,
            },
            goals: [
              {
                goal: {
                  id: 9,
                  title: "Ship MVP vertical slice",
                  description: "Registration + goals dashboard",
                  status: "pending_buddy_acceptance",
                  current_progress_health: "unknown",
                  current_streak_count: 0,
                  created_at: "2026-05-01T10:00:00Z",
                  updated_at: "2026-05-01T10:00:00Z",
                },
                buddy: {
                  id: 2,
                  email: "peer@example.com",
                  display_name: "Peer",
                },
                pact: {
                  id: 3,
                  status: "invited",
                },
                invite: {
                  id: 4,
                  status: "pending",
                  expires_at: "2026-05-08T10:00:00Z",
                },
              },
            ],
          }),
          { status: 200, headers: { "Content-Type": "application/json" } },
        ),
      );

    render(<DashboardScreen />);

    expect(await screen.findByRole("button", { name: "Создать аккаунт" })).toBeInTheDocument();

    fireEvent.change(screen.getByPlaceholderText("Например, Артём"), {
      target: { value: "Owner" },
    });
    fireEvent.change(screen.getByPlaceholderText("you@example.com"), {
      target: { value: "owner@example.com" },
    });
    fireEvent.submit(screen.getByRole("button", { name: "Создать аккаунт" }).closest("form")!);

    expect(
      await screen.findByRole("link", { name: "Создать первую цель" }),
    ).toHaveAttribute("href", "/goals/new");
  });
});
