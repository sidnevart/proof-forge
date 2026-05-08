package ai

import "context"

// ProofSuggestion is one AI-generated proof contract variant.
type ProofSuggestion struct {
	WhatToProve  string `json:"what_to_prove"`
	HowToProve   string `json:"how_to_prove"`
	MovementMode string `json:"movement_mode"`
	DueDays      int    `json:"due_days"`
}

// GoalToProofsResult is the response from GoalToProofs.
type GoalToProofsResult struct {
	Suggestions []ProofSuggestion `json:"suggestions"`
}

// NextStepResult is the response from NextStep.
type NextStepResult struct {
	Step        string `json:"step"`
	Rationale   string `json:"rationale"`
	DueDays     int    `json:"due_days"`
}

// ProofCheckResult is the response from ProofCheck.
type ProofCheckResult struct {
	HasArtifact  bool   `json:"has_artifact"`
	HasConclusion bool  `json:"has_conclusion"`
	HasNextStep  bool   `json:"has_next_step"`
	Score        int    `json:"score"` // 0-100
	Feedback     string `json:"feedback"`
	Suggestion   string `json:"suggestion"`
}

// AntiProofResult helps user articulate a stuck check-in.
type AntiProofResult struct {
	Summary     string `json:"summary"`
	WhatBlocked string `json:"what_blocked"`
	NextAttempt string `json:"next_attempt"`
}

// BuddyMatchSuggestion is one buddy candidate.
type BuddyMatchSuggestion struct {
	UserID      int64  `json:"user_id"`
	DisplayName string `json:"display_name"`
	MatchReason string `json:"match_reason"`
	SharedGoals int    `json:"shared_goals"`
}

// BuddyMatchResult is the response from BuddyMatch.
type BuddyMatchResult struct {
	Suggestions []BuddyMatchSuggestion `json:"suggestions"`
}

// AssistantProvider is the interface for all v2 AI assistant operations.
type AssistantProvider interface {
	GoalToProofs(ctx context.Context, goalText string) (GoalToProofsResult, error)
	NextStep(ctx context.Context, goalText string, recentActivity string) (NextStepResult, error)
	ProofCheck(ctx context.Context, draft string) (ProofCheckResult, error)
	AntiProof(ctx context.Context, goalText, stuckDescription string) (AntiProofResult, error)
}
