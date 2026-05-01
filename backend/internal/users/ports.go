package users

import "context"

type UserRepository interface {
	FindByEmail(context.Context, string) (User, error)
	FindByID(context.Context, int64) (User, error)
	Create(context.Context, RegisterInput) (User, error)
}

type SessionRepository interface {
	CreateSession(context.Context, Session) error
	FindUserBySessionTokenHash(context.Context, string) (User, error)
}
