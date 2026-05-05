package email

import "context"

type BuddyInviteParams struct {
	To        string
	OwnerName string
	GoalTitle string
	InviteURL string
}

type BuddyAcceptedParams struct {
	To           string
	BuddyName    string
	GoalTitle    string
	DashboardURL string
}

type Sender interface {
	SendBuddyInvite(ctx context.Context, p BuddyInviteParams) error
	SendBuddyAccepted(ctx context.Context, p BuddyAcceptedParams) error
}
