package resolver

import (
	"context"

	"iam-service/internal/auth"
)

type Resolver interface {
	Resolve(
		ctx context.Context,
		identity *auth.Identity,
	) (userID string, err error)

	GetSessionVersion(
		ctx context.Context,
		userID string,
	) (int, error)

	IncrementSessionVersion(
		ctx context.Context,
		userID string,
	) error

	GetUserStatus(
		ctx context.Context,
		userID string,
	) (string, error)
}
