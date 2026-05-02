package email

import "context"

type NoopSender struct{}

func (NoopSender) SendBuddyInvite(_ context.Context, _ BuddyInviteParams) error { return nil }
