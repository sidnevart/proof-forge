package circles

import (
	"errors"
	"strings"
	"time"
)

type CircleInvitationStatus string

const (
	CircleInvitationStatusPending  CircleInvitationStatus = "pending"
	CircleInvitationStatusAccepted CircleInvitationStatus = "accepted"
	CircleInvitationStatusDeclined CircleInvitationStatus = "declined"
)

const (
	DefaultMemberLimit = 8
	// SeasonLengthDays is the duration of a circle season in days.
	// One circle = one season = one owner goal. After SeasonLengthDays the season is
	// completed and the owner can extend (same goal, fresh season) or start a new
	// circle with a new goal.
	SeasonLengthDays = 7
)

type MembershipStatus string
type MemberRole string
type SeasonStatus string
type WeeklyStatus string
type EventKind string

const (
	MembershipStatusActive MembershipStatus = "active"

	// MemberRoleOwner — the user who created the circle and owns the active goal.
	// MemberRoleBuddy — the single member that approves the owner's check-ins.
	// MemberRoleObserver — every other member; can see proofs, cannot approve.
	MemberRoleOwner    MemberRole = "owner"
	MemberRoleBuddy    MemberRole = "buddy"
	MemberRoleObserver MemberRole = "observer"

	SeasonStatusActive    SeasonStatus = "active"
	SeasonStatusCompleted SeasonStatus = "completed"

	WeeklyStatusNoGoal        WeeklyStatus = "no_goal"
	WeeklyStatusApproved      WeeklyStatus = "approved"
	WeeklyStatusWaitingReview WeeklyStatus = "waiting_review"
	WeeklyStatusAtRisk        WeeklyStatus = "at_risk"
	WeeklyStatusDropped       WeeklyStatus = "dropped"
	WeeklyStatusComeback      WeeklyStatus = "comeback"

	EventKindApproved EventKind = "approved"
	EventKindDropped  EventKind = "dropped"
	EventKindComeback EventKind = "comeback"
	EventKindRisk     EventKind = "at_risk"
)

var (
	ErrInvalidCircleInput          = errors.New("invalid circle input")
	ErrCircleNotFound              = errors.New("circle not found")
	ErrNotCircleMember             = errors.New("user is not a circle member")
	ErrAlreadyCircleMember         = errors.New("user is already in the circle")
	ErrCircleIsFull                = errors.New("circle already has the maximum number of members")
	ErrActiveGoalAlreadyExists     = errors.New("circle already has an active goal for this owner")
	ErrCircleInvitationNotFound    = errors.New("circle invitation not found")
	ErrCircleInvitationAlreadyDone = errors.New("circle invitation already accepted or declined")

	ErrSeasonNotFound    = errors.New("season not found")
	ErrSeasonNotEnded    = errors.New("season has not ended yet")
	ErrSeasonAlreadyDone = errors.New("season already completed")
)

type SeasonEndAction string

const (
	SeasonEndActionExtend   SeasonEndAction = "extend"
	SeasonEndActionStartNew SeasonEndAction = "start_new"
)

type SeasonEndInput struct {
	Action    SeasonEndAction `json:"action"`
	GoalTitle string          `json:"goal_title,omitempty"` // only for start_new
}

type SeasonEndResult struct {
	Action          SeasonEndAction `json:"action"`
	NewSeasonID     *int64          `json:"new_season_id,omitempty"`     // set if extend
	RedirectToGoals bool            `json:"redirect_to_goals,omitempty"` // set if start_new
}

type CircleInvitation struct {
	ID          int64                  `json:"id"`
	CircleID    int64                  `json:"circle_id"`
	CircleName  string                 `json:"circle_name"`
	InviterID   int64                  `json:"inviter_id"`
	TargetEmail string                 `json:"target_email"`
	Status      CircleInvitationStatus `json:"status"`
	Message     string                 `json:"message"`
	CreatedAt   time.Time              `json:"created_at"`
}

type InviteToCircleInput struct {
	TargetEmail string `json:"target_email"`
	Message     string `json:"message"`
}

type CreateInput struct {
	Name string `json:"name"`
}

type JoinInput struct {
	InviteCode string `json:"invite_code"`
}

type Circle struct {
	ID          int64     `json:"id"`
	OwnerUserID int64     `json:"owner_user_id"`
	Name        string    `json:"name"`
	InviteCode  string    `json:"invite_code"`
	MemberLimit int       `json:"member_limit"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Member struct {
	UserID      int64            `json:"user_id"`
	Email       string           `json:"email"`
	DisplayName string           `json:"display_name"`
	Status      MembershipStatus `json:"status"`
	Role        MemberRole       `json:"role"`
	JoinedAt    time.Time        `json:"joined_at"`
}

type Season struct {
	ID       int64        `json:"id"`
	CircleID int64        `json:"circle_id"`
	Status   SeasonStatus `json:"status"`
	StartsAt time.Time    `json:"starts_at"`
	EndsAt   time.Time    `json:"ends_at"`
}

type Detail struct {
	Circle       Circle   `json:"circle"`
	Members      []Member `json:"members"`
	ActiveSeason Season   `json:"active_season"`
}

type StandingEntry struct {
	UserID              int64        `json:"user_id"`
	UserEmail           string       `json:"user_email"`
	DisplayName         string       `json:"display_name"`
	Rank                int          `json:"rank"`
	WeeklyStatus        WeeklyStatus `json:"weekly_status"`
	ApprovedWeeks       int          `json:"approved_weeks"`
	MissedWeeks         int          `json:"missed_weeks"`
	CurrentStreak       int          `json:"current_streak"`
	Score               int          `json:"score"`
	GoalsCount          int          `json:"goals_count"`
	HasActivityThisWeek bool         `json:"has_activity_this_week"`
}

type Event struct {
	Kind       EventKind `json:"kind"`
	UserID     int64     `json:"user_id"`
	UserEmail  string    `json:"user_email"`
	Message    string    `json:"message"`
	OccurredAt time.Time `json:"occurred_at"`
}

type WeeklyAssembly struct {
	Circle      Circle          `json:"circle"`
	Season      Season          `json:"season"`
	CurrentWeek int             `json:"current_week"`
	Standings   []StandingEntry `json:"standings"`
	Events      []Event         `json:"events"`
}

type StandingSnapshot struct {
	UserID               int64
	UserEmail            string
	DisplayName          string
	GoalsCount           int
	ApprovedWeeks        int
	MissedWeeks          int
	CurrentStreak        int
	HasApprovedThisWeek  bool
	HasSubmittedThisWeek bool
	HasApprovedPrevWeek  bool
	HasSubmittedPrevWeek bool
	LastActivityAt       *time.Time
}

func (in CreateInput) Validate() error {
	name := strings.TrimSpace(in.Name)
	if len(name) < 3 {
		return errors.Join(ErrInvalidCircleInput, errors.New("name must be at least 3 characters"))
	}
	if len(name) > 80 {
		return errors.Join(ErrInvalidCircleInput, errors.New("name must be 80 characters or fewer"))
	}
	return nil
}

func (in CreateInput) Normalize() CreateInput {
	return CreateInput{Name: strings.TrimSpace(in.Name)}
}

func (in JoinInput) Validate() error {
	if strings.TrimSpace(in.InviteCode) == "" {
		return errors.Join(ErrInvalidCircleInput, errors.New("invite_code is required"))
	}
	return nil
}
