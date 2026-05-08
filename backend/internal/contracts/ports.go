package contracts

import "context"

// Repository is the storage abstraction for ProofContract persistence.
type Repository interface {
	Create(ctx context.Context, c *ProofContract) error
	GetByID(ctx context.Context, id int64) (*ProofContract, error)
	ListByGoal(ctx context.Context, goalID int64) ([]*ProofContract, error)
	ListByUser(ctx context.Context, userID int64, status *Status) ([]*ProofContract, error)
	UpdateStatus(ctx context.Context, id int64, status Status) error
	// MarkBrokenOverdue transitions all overdue active contracts to 'broken'.
	MarkBrokenOverdue(ctx context.Context) (int64, error)
}
