export type User = {
  id: number;
  email: string;
  display_name: string;
  created_at: string;
  updated_at: string;
};

export type GoalStatus = "pending_buddy_acceptance" | "active";
export type PactStatus = "invited" | "active";
export type InviteStatus = "pending" | "accepted";
export type ProgressHealth = "unknown" | "stable" | "at_risk";

export type GoalView = {
  goal: {
    id: number;
    title: string;
    description: string;
    status: GoalStatus;
    current_progress_health: ProgressHealth;
    current_streak_count: number;
    created_at: string;
    updated_at: string;
  };
  buddy: {
    id: number;
    email: string;
    display_name: string;
  };
  pact: {
    id: number;
    status: PactStatus;
    accepted_at?: string | null;
  };
  invite: {
    id: number;
    status: InviteStatus;
    expires_at: string;
    acceptance_token?: string;
  };
};

export type CheckInStatus =
  | "draft"
  | "submitted"
  | "changes_requested"
  | "approved"
  | "rejected";

export type EvidenceKind = "text" | "link" | "file" | "image";

export type CheckIn = {
  id: number;
  goal_id: number;
  owner_user_id: number;
  status: CheckInStatus;
  submitted_at?: string | null;
  approved_at?: string | null;
  rejected_at?: string | null;
  changes_requested_at?: string | null;
  created_at: string;
  updated_at: string;
};

export type EvidenceItem = {
  id: number;
  check_in_id: number;
  kind: EvidenceKind;
  text_content?: string;
  external_url?: string;
  storage_key?: string;
  mime_type?: string;
  file_size_bytes?: number;
  created_at: string;
};

export type ReviewDecision = "approved" | "rejected" | "changes_requested";

export type ReviewRecord = {
  id: number;
  check_in_id: number;
  reviewer_user_id: number;
  decision: ReviewDecision;
  comment?: string;
  created_at: string;
};

export type CheckInDetail = {
  check_in: CheckIn;
  evidence: EvidenceItem[];
};

export type InvitePreview = {
  goal_title: string;
  owner_name: string;
  invitee_email: string;
  status: InviteStatus;
  expires_at: string;
};

export type RecapStatus = "pending" | "generating" | "done" | "failed";

export type WeeklyRecap = {
  id: number;
  goal_id: number;
  owner_user_id: number;
  period_start: string;
  period_end: string;
  status: RecapStatus;
  summary_text: string;
  model_name?: string;
  generated_at?: string | null;
  created_at: string;
};

export type DashboardSummary = {
  total_goals: number;
  pending_buddy_acceptance: number;
  active_goals: number;
};

export type DashboardResponse = {
  user: User;
  summary: DashboardSummary;
  goals: GoalView[];
};
