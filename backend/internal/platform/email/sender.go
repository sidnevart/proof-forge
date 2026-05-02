package email

import "context"

type BuddyInviteParams struct {
	To        string
	OwnerName string
	GoalTitle string
	InviteURL string
}

type Sender interface {
	SendBuddyInvite(ctx context.Context, p BuddyInviteParams) error
}
