import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { InviteAcceptScreen } from "./invite-accept-screen";

const VALID_INVITE = {
  goal_title: "Ship MVP",
  owner_name: "Alice",
  invitee_email: "buddy@example.com",
  status: "pending",
  expires_at: "2026-05-08T10:00:00Z",
};

function inviteResponse(invite = VALID_INVITE, status = 200) {
  return new Response(JSON.stringify({ invite }), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

function errorResponse(code: string, message: string, status: number) {
  return new Response(JSON.stringify({ error: { code, message } }), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: vi.fn() }),
}));

describe("InviteAcceptScreen", () => {
  const fetchMock = vi.fn<typeof fetch>();

  beforeEach(() => {
    vi.stubGlobal("fetch", fetchMock);
  });

  afterEach(() => {
    cleanup();
    vi.unstubAllGlobals();
    fetchMock.mockReset();
  });

  it("renders invite details for a valid pending invite", async () => {
    fetchMock.mockResolvedValueOnce(inviteResponse());

    render(<InviteAcceptScreen token="valid-token" />);

    expect(await screen.findByText("Ship MVP")).toBeInTheDocument();
    expect(screen.getAllByText("Alice").length).toBeGreaterThan(0);
    expect(screen.getAllByText("buddy@example.com").length).toBeGreaterThan(0);
    expect(screen.getByRole("button", { name: "Принять pact" })).toBeInTheDocument();
  });

  it("shows not found state when invite does not exist", async () => {
    fetchMock.mockResolvedValueOnce(
      errorResponse("invite_not_found", "Invite not found", 404),
    );

    render(<InviteAcceptScreen token="bad-token" />);

    expect(await screen.findByText("Invite не найден")).toBeInTheDocument();
  });

  it("shows expired state when server returns 410", async () => {
    fetchMock.mockResolvedValueOnce(
      errorResponse("invite_expired", "This invite has expired", 410),
    );

    render(<InviteAcceptScreen token="old-token" />);

    expect(await screen.findByText("Invite истёк")).toBeInTheDocument();
  });

  it("transitions to accepted state on successful pact acceptance", async () => {
    fetchMock
      .mockResolvedValueOnce(inviteResponse())
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ accepted: true }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      );

    render(<InviteAcceptScreen token="valid-token" />);

    const button = await screen.findByRole("button", { name: "Принять pact" });
    fireEvent.click(button);

    await waitFor(() => {
      expect(screen.getByText("Pact принят — контур активирован")).toBeInTheDocument();
    });
  });

  it("shows auth form when accept returns 401", async () => {
    fetchMock
      .mockResolvedValueOnce(inviteResponse())
      .mockResolvedValueOnce(
        errorResponse("auth_required", "Authentication required", 401),
      );

    render(<InviteAcceptScreen token="valid-token" />);

    const button = await screen.findByRole("button", { name: "Принять pact" });
    fireEvent.click(button);

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: "Войти и принять pact" }),
      ).toBeInTheDocument();
    });

    expect(
      (screen.getByDisplayValue("buddy@example.com") as HTMLInputElement).value,
    ).toBe("buddy@example.com");
  });

  it("shows already accepted state when server returns 409", async () => {
    fetchMock
      .mockResolvedValueOnce(inviteResponse())
      .mockResolvedValueOnce(
        errorResponse("invite_already_accepted", "Already accepted", 409),
      );

    render(<InviteAcceptScreen token="valid-token" />);

    const button = await screen.findByRole("button", { name: "Принять pact" });
    fireEvent.click(button);

    await waitFor(() => {
      expect(screen.getByText("Pact уже принят")).toBeInTheDocument();
    });
  });
});
