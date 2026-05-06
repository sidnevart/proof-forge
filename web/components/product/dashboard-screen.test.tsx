import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { DashboardScreen } from "./dashboard-screen";

// Next.js useRouter is referenced inside the component; the App Router test
// environment doesn't mount a real router, so we stub it.
vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: vi.fn(), replace: vi.fn(), prefetch: vi.fn(), back: vi.fn() }),
}));

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

    expect(await screen.findByRole("button", { name: "СОЗДАТЬ АККАУНТ" })).toBeInTheDocument();
    expect(screen.getByText("ВОЙДИТЕ, ЧТОБЫ ДЕРЖАТЬ ЦЕЛЬ ПОД КОНТРОЛЕМ")).toBeInTheDocument();
  });

  it("renders empty-state card with circle context when authenticated and goals are empty", async () => {
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
          summary: { total_goals: 0, pending_buddy_acceptance: 0, active_goals: 0 },
          goals: [],
          circles: [{ id: 7, name: "Утренний круг", member_count: 1 }],
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      ),
    );

    render(<DashboardScreen />);

    // Eyebrow with circle name + correctly pluralized member count.
    expect(
      await screen.findByText("КРУГ «УТРЕННИЙ КРУГ» · 1 участник"),
    ).toBeInTheDocument();
    // Headline + sub.
    expect(screen.getByRole("heading", { name: "ЧТО БУДЕШЬ ДОКАЗЫВАТЬ?" })).toBeInTheDocument();
    expect(screen.getByText(/Объяви цель/)).toBeInTheDocument();
    // Inline white CTA.
    expect(screen.getByRole("link", { name: "ОБЪЯВИТЬ ЦЕЛЬ" })).toBeInTheDocument();
    // No identity header (display_name + email) anywhere.
    expect(screen.queryByText("АРТЁМ")).toBeNull();
    expect(screen.queryByText("owner@example.com")).toBeNull();
  });

  it("falls back to НОВЫЙ КРУГ when the user has no circles yet", async () => {
    fetchMock.mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          user: {
            id: 1,
            email: "owner@example.com",
            display_name: "Owner",
            created_at: "2026-05-01T10:00:00Z",
            updated_at: "2026-05-01T10:00:00Z",
          },
          summary: { total_goals: 0, pending_buddy_acceptance: 0, active_goals: 0 },
          goals: [],
          circles: [],
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      ),
    );

    render(<DashboardScreen />);

    expect(await screen.findByText("НОВЫЙ КРУГ")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "ОБЪЯВИТЬ ЦЕЛЬ" })).toBeInTheDocument();
  });

  it("registration completes and reveals empty-state card", async () => {
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
            summary: { total_goals: 0, pending_buddy_acceptance: 0, active_goals: 0 },
            goals: [],
            circles: [],
          }),
          { status: 200, headers: { "Content-Type": "application/json" } },
        ),
      );

    render(<DashboardScreen />);

    expect(await screen.findByRole("button", { name: "СОЗДАТЬ АККАУНТ" })).toBeInTheDocument();

    fireEvent.change(screen.getByPlaceholderText("Например, Артём"), {
      target: { value: "Owner" },
    });
    fireEvent.change(screen.getByPlaceholderText("you@example.com"), {
      target: { value: "owner@example.com" },
    });
    fireEvent.submit(screen.getByRole("button", { name: "СОЗДАТЬ АККАУНТ" }).closest("form")!);

    expect(await screen.findByRole("heading", { name: "ЧТО БУДЕШЬ ДОКАЗЫВАТЬ?" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "ОБЪЯВИТЬ ЦЕЛЬ" })).toBeInTheDocument();
  });
});
