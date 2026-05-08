// Package contracts owns the ProofContract aggregate — a concrete,
// time-bound promise of what the user will prove and how.
package contracts

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Status enumerates the lifecycle states of a proof contract.
//
//	pending   — created but not yet accepted/activated
//	active    — buddy accepted; user is working on it
//	fulfilled — user submitted a proof that was accepted
//	broken    — due_at passed with no accepted proof
//	cancelled — user or buddy cancelled before fulfillment
type Status string

const (
	StatusPending   Status = "pending"
	StatusActive    Status = "active"
	StatusFulfilled Status = "fulfilled"
	StatusBroken    Status = "broken"
	StatusCancelled Status = "cancelled"
)

// Urgency classifies how close a contract is to its due date.
type Urgency string

const (
	UrgencyOverdue  Urgency = "overdue"
	UrgencyToday    Urgency = "today"
	UrgencyTomorrow Urgency = "tomorrow"
	UrgencyThisWeek Urgency = "this_week"
	UrgencyLater    Urgency = "later"
)

var (
	ErrContractNotFound    = errors.New("proof contract not found")
	ErrInvalidContract     = errors.New("invalid proof contract")
	ErrInvalidTransition   = errors.New("invalid status transition")
	ErrForbidden           = errors.New("forbidden")
)

// ProofContract is the persisted aggregate row.
type ProofContract struct {
	ID           int64
	GoalID       int64
	UserID       int64
	BuddyUserID  *int64
	WhatToProve  string
	HowToProve   string
	DueAt        time.Time
	Status       Status
	FulfilledAt  *time.Time
	BrokenAt     *time.Time
	CancelledAt  *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Urgency returns how time-critical this contract is relative to now.
func (c *ProofContract) Urgency() Urgency {
	now := time.Now()
	if c.DueAt.Before(now) {
		return UrgencyOverdue
	}
	diff := c.DueAt.Sub(now)
	switch {
	case diff < 24*time.Hour:
		return UrgencyToday
	case diff < 48*time.Hour:
		return UrgencyTomorrow
	case diff < 7*24*time.Hour:
		return UrgencyThisWeek
	default:
		return UrgencyLater
	}
}

// CanTransitionTo reports whether the contract may move to the given status.
func (c *ProofContract) CanTransitionTo(next Status) bool {
	switch c.Status {
	case StatusPending:
		return next == StatusActive || next == StatusCancelled
	case StatusActive:
		return next == StatusFulfilled || next == StatusBroken || next == StatusCancelled
	default:
		return false
	}
}

// ValidateWhatToProve checks that the proof description is not empty.
func ValidateWhatToProve(s string) error {
	if strings.TrimSpace(s) == "" {
		return fmt.Errorf("%w: what_to_prove is required", ErrInvalidContract)
	}
	if len(s) > 2000 {
		return fmt.Errorf("%w: what_to_prove exceeds 2000 chars", ErrInvalidContract)
	}
	return nil
}
