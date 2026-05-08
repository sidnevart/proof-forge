// Package buddy provides buddy dashboard metrics.
package buddy

import "time"

// QueueItem represents a check-in waiting for buddy review.
type QueueItem struct {
	CheckInID    int64     `json:"check_in_id"`
	GoalID       int64     `json:"goal_id"`
	GoalTitle    string    `json:"goal_title"`
	UserID       int64     `json:"user_id"`
	DisplayName  string    `json:"display_name"`
	SubmittedAt  time.Time `json:"submitted_at"`
	WaitingHours float64   `json:"waiting_hours"`
	Preview      string    `json:"preview,omitempty"`
	Status       string    `json:"status"`
}

// NeedsAttentionItem represents a mentee who hasn't submitted a proof recently.
type NeedsAttentionItem struct {
	UserID             int64      `json:"user_id"`
	DisplayName        string     `json:"display_name"`
	GoalTitle          string     `json:"goal_title"`
	DaysSinceLastProof float64    `json:"days_since_last_proof"`
	LastCheckInAt      *time.Time `json:"last_check_in_at,omitempty"`
	HasBrokenContract  bool       `json:"has_broken_contract"`
}

// BuddyStats holds effectiveness metrics for the buddy.
type BuddyStats struct {
	ActiveBuddiesCount   int     `json:"active_buddies_count"`
	TotalReviewsGiven    int     `json:"total_reviews_given"`
	AvgResponseHours     float64 `json:"avg_response_hours"`
	HelpfulReviewsCount  int     `json:"helpful_reviews_count"`
	PeopleSupportedCount int     `json:"people_supported_count"`
	ReviewsThisWeek      int     `json:"reviews_this_week"`
	ReviewsLastWeek      int     `json:"reviews_last_week"`
}
