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
}
