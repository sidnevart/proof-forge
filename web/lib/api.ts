import type {
  CheckIn,
  CheckInDetail,
  CircleDetail,
  CircleFeedItem,
  CircleInvitation,
  DashboardResponse,
  EvidenceItem,
  GoalRefineResponse,
  GoalView,
  InvitePreview,
  Milestone,
  PublicGoal,
  PublicProof,
  ReviewRecord,
  SeasonEndResult,
  StakeView,
  TelegramLinkToken,
  User,
  WeeklyAssembly,
  WeeklyRecap,
} from "@/lib/types";

// Production: empty string = same-origin requests through nginx.
// Development: set NEXT_PUBLIC_API_BASE_URL=http://localhost:8080 in .env.local or compose.
const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL ?? "";

type ApiErrorPayload = {
  error?: {
    code?: string;
    message?: string;
  };
};

export class ApiError extends Error {
  status: number;
  code?: string;

  constructor(status: number, message: string, code?: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
  }
}

export type RegisterInput = {
  email: string;
  display_name: string;
};

export type CreateGoalInput = {
  title: string;
  description: string;
  buddy_name: string;
  buddy_email: string;
  circle_id?: number;
  proof_examples?: string;
  category?: string;
};

export type CreateCircleInput = {
  name: string;
};

// Auto-refresh dedup: while one /v1/auth/refresh call is in flight, every
// other request that 401s waits on the same Promise instead of spawning its
// own refresh. Once the refresh resolves, all queued requests retry against
// the freshly minted access cookie.
let inflightRefresh: Promise<boolean> | null = null;

async function performRefresh(): Promise<boolean> {
  try {
    const response = await fetch(`${API_BASE_URL}/v1/auth/refresh`, {
      method: "POST",
      credentials: "include",
    });
    return response.ok;
  } catch {
    // Network error during refresh — treat as failed refresh, the caller
    // will surface the original 401 to the UI.
    return false;
  }
}

function refreshAccessToken(): Promise<boolean> {
  if (!inflightRefresh) {
    inflightRefresh = performRefresh().finally(() => {
      inflightRefresh = null;
    });
  }
  return inflightRefresh;
}

// rawFetch is the underlying fetch+ApiError wrapper. The `request` function
// wraps it with one auto-retry on 401 (after a refresh) so the UI never
// observes the brief gap between an expiring access cookie and its
// replacement.
async function rawFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const headers = new Headers(init?.headers);
  if (init?.body && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  const response = await fetch(`${API_BASE_URL}${path}`, {
    credentials: "include",
    headers,
    ...init,
  });

  if (!response.ok) {
    const payload = (await safeJSON(response)) as ApiErrorPayload | null;
    throw new ApiError(
      response.status,
      payload?.error?.message ?? `Request failed with status ${response.status}`,
      payload?.error?.code,
    );
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return (await response.json()) as T;
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  // Don't auto-refresh the auth endpoints themselves — their 401 IS the
  // signal we'd otherwise try to recover from, and refreshing inside a
  // refresh call would loop.
  const isAuthCall = path.startsWith("/v1/auth/") || path === "/v1/login" || path === "/v1/register";

  try {
    return await rawFetch<T>(path, init);
  } catch (err) {
    // We only auto-refresh on `invalid_session` — the backend uses that code
    // for "had a session, server says it's no good" (expired access cookie,
    // typically). When the backend says `auth_required` it means there was
    // no cookie at all, so the user is anonymous and refreshing makes no
    // sense; that case should fall through to the UI and show the login form.
    if (
      !isAuthCall &&
      err instanceof ApiError &&
      err.status === 401 &&
      err.code === "invalid_session"
    ) {
      const refreshed = await refreshAccessToken();
      if (refreshed) {
        return rawFetch<T>(path, init);
      }
    }
    throw err;
  }
}

// logoutUser hits the server endpoint that revokes both cookies. The fetch is
// best-effort: even if the server can't be reached, the SPA should still
// drop the user from local state.
export async function logoutUser(): Promise<void> {
  try {
    await rawFetch<void>("/v1/auth/logout", { method: "POST" });
  } catch {
    // ignore
  }
}

async function safeJSON(response: Response): Promise<unknown | null> {
  try {
    return await response.json();
  } catch {
    return null;
  }
}

export async function getDashboard(): Promise<DashboardResponse> {
  return request<DashboardResponse>("/v1/dashboard");
}

export async function registerUser(input: RegisterInput): Promise<{ user: User }> {
  return request<{ user: User }>("/v1/register", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export async function loginUser(email: string): Promise<{ user: User }> {
  return request<{ user: User }>("/v1/login", {
    method: "POST",
    body: JSON.stringify({ email }),
  });
}

export async function createGoal(input: CreateGoalInput): Promise<{ goal: GoalView }> {
  return request<{ goal: GoalView }>("/v1/goals", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export async function createCircle(input: CreateCircleInput): Promise<CircleDetail> {
  return request<CircleDetail>("/v1/circles", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export async function listCircles(): Promise<{ circles: CircleDetail[] | null }> {
  return request<{ circles: CircleDetail[] | null }>("/v1/circles");
}

export async function joinCircle(inviteCode: string): Promise<CircleDetail> {
  return request<CircleDetail>("/v1/circles/join", {
    method: "POST",
    body: JSON.stringify({ invite_code: inviteCode }),
  });
}

export async function getWeeklyAssembly(circleID: number): Promise<WeeklyAssembly> {
  return request<WeeklyAssembly>(`/v1/circles/${circleID}/weekly-assembly`);
}

export async function createCheckIn(goalID: number): Promise<{ check_in: CheckIn }> {
  return request<{ check_in: CheckIn }>(`/v1/goals/${goalID}/check-ins`, { method: "POST" });
}

export async function listCheckIns(goalID: number): Promise<{ check_ins: CheckIn[] | null }> {
  return request<{ check_ins: CheckIn[] | null }>(`/v1/goals/${goalID}/check-ins`);
}

export async function submitCheckIn(checkInID: number): Promise<{ submitted: boolean }> {
  return request<{ submitted: boolean }>(`/v1/check-ins/${checkInID}/submit`, { method: "POST" });
}

export async function addTextEvidence(checkInID: number, content: string): Promise<{ evidence: EvidenceItem }> {
  return request<{ evidence: EvidenceItem }>(`/v1/check-ins/${checkInID}/evidence/text`, {
    method: "POST",
    body: JSON.stringify({ content }),
  });
}

export async function addLinkEvidence(checkInID: number, url: string): Promise<{ evidence: EvidenceItem }> {
  return request<{ evidence: EvidenceItem }>(`/v1/check-ins/${checkInID}/evidence/link`, {
    method: "POST",
    body: JSON.stringify({ url }),
  });
}

export async function addFileEvidence(checkInID: number, file: File): Promise<{ evidence: EvidenceItem }> {
  const form = new FormData();
  form.append("file", file);
  const response = await fetch(`${API_BASE_URL}/v1/check-ins/${checkInID}/evidence/file`, {
    method: "POST",
    credentials: "include",
    body: form,
  });
  if (!response.ok) {
    const payload = (await safeJSON(response)) as { error?: { code?: string; message?: string } } | null;
    throw new ApiError(
      response.status,
      payload?.error?.message ?? `Upload failed with status ${response.status}`,
      payload?.error?.code,
    );
  }
  return (await response.json()) as { evidence: EvidenceItem };
}

export async function getInvite(token: string): Promise<{ invite: InvitePreview }> {
  return request<{ invite: InvitePreview }>(`/v1/invites/${encodeURIComponent(token)}`);
}

export async function acceptInvite(token: string): Promise<{ accepted: boolean }> {
  return request<{ accepted: boolean }>(`/v1/invites/${encodeURIComponent(token)}/accept`, {
    method: "POST",
  });
}

export async function getCheckIn(checkInID: number): Promise<CheckInDetail> {
  return request<CheckInDetail>(`/v1/check-ins/${checkInID}`);
}

export async function approveCheckIn(checkInID: number, comment?: string): Promise<{ review: ReviewRecord }> {
  return request<{ review: ReviewRecord }>(`/v1/check-ins/${checkInID}/approve`, {
    method: "POST",
    body: JSON.stringify({ comment: comment ?? "" }),
  });
}

export async function rejectCheckIn(checkInID: number, comment?: string): Promise<{ review: ReviewRecord }> {
  return request<{ review: ReviewRecord }>(`/v1/check-ins/${checkInID}/reject`, {
    method: "POST",
    body: JSON.stringify({ comment: comment ?? "" }),
  });
}

export async function requestChanges(checkInID: number, comment?: string): Promise<{ review: ReviewRecord }> {
  return request<{ review: ReviewRecord }>(`/v1/check-ins/${checkInID}/request-changes`, {
    method: "POST",
    body: JSON.stringify({ comment: comment ?? "" }),
  });
}

export async function getRecaps(goalID: number): Promise<{ recaps: WeeklyRecap[] | null }> {
  return request<{ recaps: WeeklyRecap[] | null }>(`/v1/goals/${goalID}/recaps`);
}

export async function getRecap(recapID: number): Promise<{ recap: WeeklyRecap }> {
  return request<{ recap: WeeklyRecap }>(`/v1/recaps/${recapID}`);
}

export async function createTelegramLinkToken(): Promise<TelegramLinkToken> {
  return request<TelegramLinkToken>("/v1/telegram/link-token", { method: "POST" });
}

export async function refineGoal(draftText: string): Promise<GoalRefineResponse> {
  return request<GoalRefineResponse>("/v1/goals/refine", {
    method: "POST",
    body: JSON.stringify({ draft_text: draftText }),
  });
}

export async function getCirclesFeed(params: {
  cursor?: number;
  limit?: number;
} = {}): Promise<{ items: CircleFeedItem[]; next_cursor: number | null }> {
  const qs = new URLSearchParams();
  if (params.cursor) qs.set("cursor", String(params.cursor));
  if (params.limit) qs.set("limit", String(params.limit));
  const url = `/v1/feed/circles${qs.toString() ? "?" + qs.toString() : ""}`;
  return request<{ items: CircleFeedItem[]; next_cursor: number | null }>(url);
}

export async function getPublicProofs(params: {
  q?: string;
  category?: string;
  similar_to_goal_id?: number;
  cursor?: number;
  limit?: number;
} = {}): Promise<{ proofs: PublicProof[] }> {
  const qs = new URLSearchParams();
  if (params.q) qs.set("q", params.q);
  if (params.category) qs.set("category", params.category);
  if (params.similar_to_goal_id) qs.set("similar_to_goal_id", String(params.similar_to_goal_id));
  if (params.cursor) qs.set("cursor", String(params.cursor));
  if (params.limit) qs.set("limit", String(params.limit));
  const url = `/v1/inspiration/proofs${qs.toString() ? "?" + qs.toString() : ""}`;
  return request<{ proofs: PublicProof[] }>(url);
}

export async function getPublicTemplates(params: {
  q?: string;
  category?: string;
  cursor?: number;
  limit?: number;
} = {}): Promise<{ templates: PublicGoal[] }> {
  const qs = new URLSearchParams();
  if (params.q) qs.set("q", params.q);
  if (params.category) qs.set("category", params.category);
  if (params.cursor) qs.set("cursor", String(params.cursor));
  if (params.limit) qs.set("limit", String(params.limit));
  const url = `/v1/library/templates${qs.toString() ? "?" + qs.toString() : ""}`;
  return request<{ templates: PublicGoal[] }>(url);
}

export async function setGoalVisibility(goalID: number, isPublic: boolean): Promise<void> {
  await request<{ ok: boolean }>(`/v1/goals/${goalID}/visibility`, {
    method: "POST",
    body: JSON.stringify({ is_public: isPublic }),
  });
}

export async function setCheckInVisibility(
  checkInID: number,
  isPublic: boolean,
  publicAttachmentIds?: number[]
): Promise<void> {
  await request<{ ok: boolean }>(`/v1/checkins/${checkInID}/visibility`, {
    method: "POST",
    body: JSON.stringify({ is_public: isPublic, public_attachment_ids: publicAttachmentIds ?? [] }),
  });
}

export async function updateSharingPrefs(prefs: {
  share_default: boolean;
  is_anonymous: boolean;
  alias: string;
}): Promise<void> {
  await request<{ ok: boolean }>("/v1/me/sharing", {
    method: "POST",
    body: JSON.stringify(prefs),
  });
}

export async function reportContent(kind: "goal" | "checkin", id: number, reason: string): Promise<void> {
  await request<{ ok: boolean }>("/v1/inspiration/report", {
    method: "POST",
    body: JSON.stringify({ kind, id, reason }),
  });
}

export async function listMyInvitations(): Promise<{ invitations: CircleInvitation[] | null }> {
  return request<{ invitations: CircleInvitation[] | null }>("/v1/me/invitations");
}

export async function acceptCircleInvitation(id: number): Promise<{ accepted: boolean }> {
  return request<{ accepted: boolean }>(`/v1/me/invitations/${id}/accept`, { method: "POST" });
}

export async function declineCircleInvitation(id: number): Promise<{ declined: boolean }> {
  return request<{ declined: boolean }>(`/v1/me/invitations/${id}/decline`, { method: "POST" });
}

export async function inviteToCircle(
  circleId: number,
  input: { target_email: string; message: string },
): Promise<{ invitation: CircleInvitation }> {
  return request<{ invitation: CircleInvitation }>(`/v1/circles/${circleId}/invitations`, {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export async function endSeason(
  circleId: number,
  seasonId: number,
  action: "extend" | "start_new",
  goalTitle?: string,
): Promise<{ result: SeasonEndResult }> {
  return request<{ result: SeasonEndResult }>(
    `/v1/circles/${circleId}/seasons/${seasonId}/end`,
    {
      method: "POST",
      body: JSON.stringify({ action, ...(goalTitle ? { goal_title: goalTitle } : {}) }),
    },
  );
}

// ── Goal detail ──────────────────────────────────────────────────────────────
// Single-goal lookup powering the goal-detail-screen. Backend `GetGoal`
// scopes by viewer (owner OR buddy) and returns ErrGoalNotFound otherwise.
export async function getGoal(goalID: number): Promise<{ goal: GoalView }> {
  return request<{ goal: GoalView }>(`/v1/goals/${goalID}`);
}

// ── Milestones ───────────────────────────────────────────────────────────────
// Goal-scoped milestone CRUD. The backend exposes:
//   POST   /v1/goals/{goalID}/milestones        — create
//   GET    /v1/goals/{goalID}/milestones        — list
//   POST   /v1/milestones/{id}/complete         — buddy marks complete
//   POST   /v1/milestones/{id}/reopen           — owner reopens
//   DELETE /v1/milestones/{id}                  — owner deletes
export async function listMilestones(goalID: number): Promise<{ milestones: Milestone[] | null }> {
  return request<{ milestones: Milestone[] | null }>(`/v1/goals/${goalID}/milestones`);
}

export async function createMilestone(
  goalID: number,
  title: string,
  description: string,
): Promise<{ milestone: Milestone }> {
  return request<{ milestone: Milestone }>(`/v1/goals/${goalID}/milestones`, {
    method: "POST",
    body: JSON.stringify({ title, description }),
  });
}

export async function completeMilestone(milestoneID: number): Promise<{ milestone: Milestone }> {
  return request<{ milestone: Milestone }>(`/v1/milestones/${milestoneID}/complete`, {
    method: "POST",
  });
}

export async function reopenMilestone(milestoneID: number): Promise<{ milestone: Milestone }> {
  return request<{ milestone: Milestone }>(`/v1/milestones/${milestoneID}/reopen`, {
    method: "POST",
  });
}

export async function deleteMilestone(milestoneID: number): Promise<void> {
  await request<void>(`/v1/milestones/${milestoneID}`, { method: "DELETE" });
}

// ── Stakes ───────────────────────────────────────────────────────────────────
//   POST   /v1/goals/{goalID}/stakes      — owner declares a stake
//   GET    /v1/goals/{goalID}/stakes      — list
//   DELETE /v1/stakes/{id}                — owner cancels active stake
//   POST   /v1/stakes/{id}/forfeit        — buddy declares forfeit (with reason)
export async function listStakes(goalID: number): Promise<{ stakes: StakeView[] | null }> {
  return request<{ stakes: StakeView[] | null }>(`/v1/goals/${goalID}/stakes`);
}

export async function createStake(
  goalID: number,
  description: string,
): Promise<{ stake: StakeView }> {
  return request<{ stake: StakeView }>(`/v1/goals/${goalID}/stakes`, {
    method: "POST",
    body: JSON.stringify({ description }),
  });
}

export async function cancelStake(stakeID: number): Promise<void> {
  await request<void>(`/v1/stakes/${stakeID}`, { method: "DELETE" });
}

export async function forfeitStake(stakeID: number, reason: string): Promise<{ stake: StakeView }> {
  return request<{ stake: StakeView }>(`/v1/stakes/${stakeID}/forfeit`, {
    method: "POST",
    body: JSON.stringify({ reason }),
  });
}
