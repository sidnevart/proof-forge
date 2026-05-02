import type { CheckIn, CheckInDetail, DashboardResponse, EvidenceItem, GoalView, InvitePreview, Milestone, ReviewRecord, StakeView, User, WeeklyRecap } from "@/lib/types";

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
};

async function request<T>(path: string, init?: RequestInit): Promise<T> {
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

export async function getGoal(goalID: number): Promise<{ goal: GoalView }> {
  return request<{ goal: GoalView }>(`/v1/goals/${goalID}`);
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

export async function acceptGoalInvite(goalID: number): Promise<{ accepted: boolean }> {
  return request<{ accepted: boolean }>(`/v1/goals/${goalID}/accept-invite`, {
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

export async function createStake(goalID: number, description: string): Promise<{ stake: StakeView }> {
  return request<{ stake: StakeView }>(`/v1/goals/${goalID}/stakes`, {
    method: "POST",
    body: JSON.stringify({ description }),
  });
}

export async function listStakes(goalID: number): Promise<{ stakes: StakeView[] | null }> {
  return request<{ stakes: StakeView[] | null }>(`/v1/goals/${goalID}/stakes`);
}

export async function cancelStake(stakeID: number): Promise<void> {
  return request<void>(`/v1/stakes/${stakeID}`, { method: "DELETE" });
}

export async function forfeitStake(stakeID: number, reason: string): Promise<{ stake: StakeView }> {
  return request<{ stake: StakeView }>(`/v1/stakes/${stakeID}/forfeit`, {
    method: "POST",
    body: JSON.stringify({ reason }),
  });
}

export async function listMilestones(goalID: number): Promise<{ milestones: Milestone[] | null }> {
  return request<{ milestones: Milestone[] | null }>(`/v1/goals/${goalID}/milestones`);
}

export async function createMilestone(goalID: number, title: string, description: string): Promise<{ milestone: Milestone }> {
  return request<{ milestone: Milestone }>(`/v1/goals/${goalID}/milestones`, {
    method: "POST",
    body: JSON.stringify({ title, description }),
  });
}

export async function updateMilestone(
  milestoneID: number,
  input: { title?: string; description?: string; sort_order?: number },
): Promise<{ milestone: Milestone }> {
  return request<{ milestone: Milestone }>(`/v1/milestones/${milestoneID}`, {
    method: "PATCH",
    body: JSON.stringify(input),
  });
}

export async function deleteMilestone(milestoneID: number): Promise<void> {
  return request<void>(`/v1/milestones/${milestoneID}`, { method: "DELETE" });
}

export async function completeMilestone(milestoneID: number): Promise<{ milestone: Milestone }> {
  return request<{ milestone: Milestone }>(`/v1/milestones/${milestoneID}/complete`, { method: "POST" });
}

export async function reopenMilestone(milestoneID: number): Promise<{ milestone: Milestone }> {
  return request<{ milestone: Milestone }>(`/v1/milestones/${milestoneID}/reopen`, { method: "POST" });
}
