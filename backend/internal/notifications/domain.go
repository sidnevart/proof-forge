package notifications

import (
	"encoding/json"
	"time"
)

const (
	KindDigest           = "digest"
	KindNudgeFirst       = "nudge_first_checkin"
	KindNudgeFrozen      = "nudge_member_frozen"
	KindNudge2h          = "nudge_deadline_2h"
	KindNudgeSelf        = "nudge_self_frozen"
	KindNudgeRank        = "nudge_rank_up"
	KindQuiet            = "quiet"
	KindApprovalRequest  = "approval_request"
	KindCheckinApproved  = "nudge_checkin_approved"
	KindCheckinRejected  = "nudge_checkin_rejected"
	KindInviteAccepted   = "nudge_invite_accepted"
)

// DomainEvent is an unprocessed event from the domain_events table.
type DomainEvent struct {
	ID        int64
	Kind      string
	Payload   json.RawMessage
	CreatedAt time.Time
}

// NotificationLog is a logged outbound notification.
type NotificationLog struct {
	ID      int64
	UserID  int64
	Kind    string
	DedupKey string
	SentAt  time.Time
	Status  string
}
