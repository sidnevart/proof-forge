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
export type MemberRole = "owner" | "buddy" | "observer";
export type SeasonEndAction = "extend" | "start_new";

export type SeasonEndResult = {
  action: SeasonEndAction;
  new_season_id?: number;
  redirect_to_goals?: boolean;
};

// ViewerRole tells the UI whether the requesting user is the goal's owner
// (creator) or the invited buddy (partner). Computed in the list query, not
// stored. Powers the «вы — автор» / «вы — партнёр» card label.
export type ViewerRole = "owner" | "buddy";

export type GoalView = {
  goal: {
    id: number;
    circle_id: number;
    title: string;
    description: string;
    status: GoalStatus;
    current_progress_health: ProgressHealth;
    current_streak_count: number;
    category?: string | null;
    proof_examples?: string | null;
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
  viewer_role: ViewerRole;
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
  goal_proof_examples?: string;
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

export type CircleSummary = {
  id: number;
  name: string;
  member_count: number;
};

export type DashboardResponse = {
  user: User;
  summary: DashboardSummary;
  goals: GoalView[] | null;
  circles: CircleSummary[] | null;
};

export type Circle = {
  id: number;
  owner_user_id: number;
  name: string;
  invite_code: string;
  member_limit: number;
  created_at: string;
  updated_at: string;
};

export type CircleMember = {
  user_id: number;
  email: string;
  display_name: string;
  status: "active";
  role: MemberRole;
  joined_at: string;
};

export type CircleSeason = {
  id: number;
  circle_id: number;
  status: "active" | "completed";
  starts_at: string;
  ends_at: string;
};

export type CircleDetail = {
  circle: Circle;
  members: CircleMember[];
  active_season: CircleSeason;
};

export type CircleWeeklyStatus =
  | "no_goal"
  | "approved"
  | "waiting_review"
  | "at_risk"
  | "dropped"
  | "comeback";

export type StandingEntry = {
  user_id: number;
  user_email: string;
  display_name: string;
  rank: number;
  weekly_status: CircleWeeklyStatus;
  approved_weeks: number;
  missed_weeks: number;
  current_streak: number;
  score: number;
  goals_count: number;
  has_activity_this_week: boolean;
};

export type CircleEvent = {
  kind: "approved" | "dropped" | "comeback" | "at_risk";
  user_id: number;
  user_email: string;
  message: string;
  occurred_at: string;
};

export type WeeklyAssembly = {
  circle: Circle;
  season: CircleSeason;
  current_week: number;
  standings: StandingEntry[];
  events: CircleEvent[];
};

export type CircleInvitationStatus = "pending" | "accepted" | "declined";

export type CircleInvitation = {
  id: number;
  circle_id: number;
  circle_name: string;
  inviter_id: number;
  target_email: string;
  status: CircleInvitationStatus;
  message: string;
  created_at: string;
};

export type TelegramLinkToken = {
  token: string;
  deeplink: string;
};

export type TelegramLinkStatus = {
  linked: boolean;
  username?: string;
};

export type GoalRefineVariant = {
  title: string;
  smart: string;
  proof_examples: string[];
};

export type GoalRefineResponse = {
  category: string;
  variants: GoalRefineVariant[];
};

export type PublicGoal = {
  id: number;
  title: string;
  proof_examples: string;
  category: string;
  author_alias: string;
  created_at: string;
};

export type PublicProof = {
  id: number;
  goal_title: string;
  category: string;
  author_alias: string;
  streak: number;
  text_content?: string;
  external_url?: string;
  created_at: string;
};

export type CircleFeedItem = {
  id: number;
  goal_id: number;
  goal_title: string;
  category: string;
  circle_id: number;
  circle_name: string;
  author_alias: string;
  text_content?: string;
  external_url?: string;
  submitted_at: string;
  status: "submitted" | "approved" | "rejected";
  can_approve: boolean;
  streak: number;
};

// GoalRole identifies whether the requesting user is the goal's owner (creator)
// or its buddy (invited reviewer). Equivalent to ViewerRole on GoalView, kept
// as a separate alias because the stake/milestone panels accept it as an
// explicit prop.
export type GoalRole = "owner" | "buddy";

export type MilestoneStatus = "pending" | "completed";

export type Milestone = {
  id: number;
  goal_id: number;
  title: string;
  description: string;
  status: MilestoneStatus;
  sort_order: number;
  completed_at?: string | null;
  completed_by_user_id?: number | null;
  created_at: string;
  updated_at: string;
};

export type StakeStatus = "active" | "forfeited" | "completed" | "cancelled";

export type Stake = {
  id: number;
  goal_id: number;
  owner_user_id: number;
  description: string;
  status: StakeStatus;
  forfeited_at?: string | null;
  completed_at?: string | null;
  cancelled_at?: string | null;
  created_at: string;
  updated_at: string;
};

export type StakeForfeiture = {
  id: number;
  stake_id: number;
  declared_by_user_id: number;
  reason: string;
  created_at: string;
};

export type StakeView = {
  stake: Stake;
  forfeiture?: StakeForfeiture | null;
};

// ── Teams (Phase 0) ─────────────────────────────────────────────────────────
//
// Mirror the JSON shapes from backend/internal/teams/http_handler.go.
// Field names match the backend DTOs exactly.

export type TeamRole = "lead" | "trusted_approver" | "member";
export type TeamAIMode = "off" | "metadata-only" | "full";
export type TeamMembershipStatus = "active" | "left" | "removed";

export type Team = {
  id: number;
  lead_user_id: number;
  name: string;
  invite_code?: string;
  member_limit: number;
  ai_mode: TeamAIMode;
  created_at: string;
  archived_at: string | null;
};

export type TeamMembership = {
  team_id: number;
  user_id: number;
  role: TeamRole;
  status: TeamMembershipStatus;
  ai_consent: boolean;
  timezone: string;
  joined_at: string;
};

export type TeamDetail = {
  team: Team;
  my_membership: TeamMembership;
  member_count: number;
};

// ── Daily Log (Phase 1) ─────────────────────────────────────────────────────

export type DailyLogEntry = {
  id: number;
  log_date: string;
  status: "logged" | "skipped" | "frozen" | "missed";
  text_content?: string;
  has_artifact: boolean;
  external_url?: string | null;
  submitted_at: string;
};

export type DailyLogStreak = {
  current: number;
  best: number;
  is_new_record: boolean;
  freezes_used_this_month: number;
};

export type ProofComment = {
  id: number;
  user_id: number;
  user_alias: string;
  text: string;
  created_at: string;
};

export type ProofCandidate = {
  goal_id: number;
  note_ids: number[];
  rationale: string;
  confidence: string;
};

export type AssembleProofResult = {
  candidates: ProofCandidate[];
  provider: string;
};

export type LeadBriefing = {
  text: string;
  provider: string;
};

export type NowUrgency = "danger" | "fire" | "warn" | "win" | "neutral";

export type NowCardData = {
  type: string;
  title: string;
  subtitle: string;
  urgency: NowUrgency;
  action: { label: string; url: string } | null;
};

export type WeekTrend = "better" | "same" | "worse" | "first_week";

export type WeeklyHistoryEntry = {
  week: string;
  proofs_count: number;
};

export type UserStats = {
  proof_streak: number;
  proofs_this_week: number;
  proofs_last_week: number;
  week_vs_last_week: number;
  week_trend: WeekTrend;
  active_weeks: number;
  personal_record_week: number;
  season_completion_pct: number;
  active_goals_count: number;
  pending_contracts_count: number;
};

export type WeekCount = {
  week: string;
  proofs_count: number;
};

export type PersonalLeaderboard = {
  current_week: {
    proofs_count: number;
    vs_last_week: string;
    trend: WeekTrend;
  };
  current_season: {
    proofs_count: number;
    vs_last_season: string;
    trend: string;
  };
  streak: {
    current_weeks: number;
    personal_record_weeks: number;
    is_personal_record: boolean;
  };
  weekly_history: WeekCount[];
};

export type MovementMode = "single_proof" | "regular_rhythm" | "challenge" | "work_initiative" | "free_goal";
export type RhythmCadence = "daily" | "weekly" | "biweekly" | "custom";
export type ContractStatus = "active" | "fulfilled" | "broken" | "cancelled";

export type ProofContract = {
  id: number;
  goal_id: number;
  user_id: number;
  buddy_user_id?: number;
  what_to_prove: string;
  how_to_prove: string;
  due_at: string;
  status: ContractStatus;
  urgency: string;
  fulfilled_at?: string;
  broken_at?: string;
  cancelled_at?: string;
};

export type CreateContractInput = {
  what_to_prove: string;
  how_to_prove: string;
  due_at: string;
  buddy_user_id?: number;
};

export type WorkspaceType = "organization" | "community";

export type Workspace = {
  id: number;
  owner_user_id: number;
  name: string;
  slug: string;
  type: WorkspaceType;
  is_active: boolean;
  frozen_at?: string;
  created_at: string;
};

export type CreateWorkspaceInput = {
  name: string;
  slug: string;
  type: WorkspaceType;
};

// ── Space Initiatives ────────────────────────────────────────────────────────

export type InitiativeStatus = "active" | "archived";
export type InitiativeSpaceType = "teamspace" | "community";

export type Initiative = {
  id: number;
  space_type: InitiativeSpaceType;
  teamspace_id?: number | null;
  community_space_id?: number | null;
  creator_id: number;
  title: string;
  description: string;
  proof_criteria: string;
  status: InitiativeStatus;
  participant_count: number;
  active_today: number;
  pending_proof_count: number;
  created_at: string;
};

export type JoinInitiativeResult = {
  goal_id: number;
  created: boolean;
};

export type PendingProof = {
  checkin_id: number;
  author_id: number;
  author_name: string;
  content: string;
  submitted_at: string;
};

export type ParticipantProgress = {
  user_id: number;
  display_name: string;
  proof_count: number;
};

export type InitiativeDetail = Initiative & {
  participants: ParticipantProgress[];
};

export type CreateInitiativeInput = {
  title: string;
  description?: string;
  proof_criteria: string;
};

// ── AI Companion ───────────────────────────────────────────────────────────────

export type AICompanionFeature =
  | "evening_ping"
  | "weekly_recap"
  | "lead_weekly_brief"
  | "streak_reminder"
  | "proof_draft"
  | "buddy_stalled"
  | "goal_risk"
  | "streak_milestone"
  | "leader_fair_play"
  | "team_health";

export type NotificationAction = {
  label: string;
  action: string;
  url?: string;
};

export type AICompanionNotification = {
  id: string;
  feature: AICompanionFeature;
  title: string;
  body: string;
  actions: NotificationAction[];
  created_at: string;
};

export type AIProofDraft = {
  id: string;
  team_id: number;
  goal_id?: number | null;
  note_ids: number[];
  rationale: string;
  confidence: string;
  created_at: string;
};

export type AICompanionAlert = {
  type: string;
  message: string;
};

export type TeamAIHealth = {
  team_health_score: number;
  fair_play_status: string;
  pending_approvals: number;
  alerts: AICompanionAlert[];
};

export type MyMembership = {
  space_type: "teamspace" | "community";
  space_id: number;
  space_name: string;
  role: string;
};

export type CircleBoardEntry = {
  rank: number;
  circle_id: number;
  circle_name: string;
  members_count: number;
  proofs_submitted: number;
  proofs_approved: number;
  completion_pct: number;
  active_members: number;
  circle_score: number;
  is_my_circle: boolean;
};

export type CirclePosition = {
  rank: number;
  circle_score: number;
  total_circles: number;
};

export type CirclesBoard = {
  board_type: string;
  period: string;
  entries: CircleBoardEntry[];
  my_circle_position?: CirclePosition;
};
