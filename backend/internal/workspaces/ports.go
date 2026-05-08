package workspaces

import "context"

// Repository is the storage abstraction for Workspace persistence.
type Repository interface {
	Create(ctx context.Context, w *Workspace) error
	GetByID(ctx context.Context, id int64) (*Workspace, error)
	GetBySlug(ctx context.Context, slug string) (*Workspace, error)
	ListByOwner(ctx context.Context, ownerUserID int64) ([]*Workspace, error)
	SlugExists(ctx context.Context, slug string) (bool, error)
	Freeze(ctx context.Context, id int64, reason string) error
	Unfreeze(ctx context.Context, id int64) error
	// ListAll is used by platform_admin panel.
	ListAll(ctx context.Context, limit, offset int) ([]*Workspace, error)
	Count(ctx context.Context) (int64, error)
}
