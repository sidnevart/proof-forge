package telegram

import (
	"errors"
	"time"
)

var (
	ErrTokenNotFound  = errors.New("link token not found or expired")
	ErrAlreadyLinked  = errors.New("telegram already linked for this user")
	ErrLinkNotFound   = errors.New("telegram link not found")
)

// TelegramLink is the association between a ProofForge user and a Telegram chat.
type TelegramLink struct {
	ID               int64     `json:"id"`
	UserID           int64     `json:"user_id"`
	TelegramChatID   int64     `json:"telegram_chat_id"`
	TelegramUsername string    `json:"telegram_username,omitempty"`
	LinkedAt         time.Time `json:"linked_at"`
	Status           string    `json:"status"`
}

// PendingToken is a short-lived link token stored during the linking flow.
type PendingToken struct {
	Token     string
	UserID    int64
	ExpiresAt time.Time
}
