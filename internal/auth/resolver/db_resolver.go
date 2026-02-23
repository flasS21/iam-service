package resolver

import (
	"context"
	"database/sql"
	"errors"

	"iam-service/internal/auth"
	"iam-service/internal/db"

	"github.com/google/uuid"
)

/*
db_resolver.Resolve() maps OAuth identity to application user by Keycloak subject.
Queries database for existing user by keycloak_sub. If not found, creates new user
with email and email_verified fields. Returns user ID string or error if database
operation fails.
*/
type DBResolver struct {
	db *db.DB
}

func NewDBResolver(db *db.DB) *DBResolver {
	return &DBResolver{db: db}
}

func (r *DBResolver) Resolve(
	ctx context.Context,
	identity *auth.Identity,
) (string, error) {

	if identity == nil {
		return "", errors.New("identity is nil")
	}

	if identity.KeycloakSub == "" {
		return "", errors.New("keycloak sub is empty")
	}

	var userID uuid.UUID

	// 1️⃣ Try to find existing user by keycloak_sub
	err := r.db.QueryRowContext(
		ctx,
		`
		SELECT id
		FROM public.users
		WHERE keycloak_sub = $1
		`,
		identity.KeycloakSub,
	).Scan(&userID)

	if err == nil {
		return userID.String(), nil
	}

	if err != sql.ErrNoRows {
		return "", err
	}

	// 2️⃣ Create new user
	err = r.db.QueryRowContext(
		ctx,
		`
	INSERT INTO public.users (keycloak_sub, email, email_verified)
	VALUES ($1, $2, $3)
	ON CONFLICT (keycloak_sub)
	DO UPDATE SET email = EXCLUDED.email
	RETURNING id
	`,
		identity.KeycloakSub,
		identity.Email,
		identity.EmailVerified,
	).Scan(&userID)

	if err != nil {
		return "", err
	}

	return userID.String(), nil
}

func (r *DBResolver) GetSessionVersion(
	ctx context.Context,
	userID string,
) (int, error) {

	var version int

	err := r.db.QueryRowContext(
		ctx,
		`
		SELECT session_version
		FROM public.users
		WHERE id = $1
		`,
		userID,
	).Scan(&version)

	return version, err
}

func (r *DBResolver) IncrementSessionVersion(
	ctx context.Context,
	userID string,
) error {

	_, err := r.db.ExecContext(
		ctx,
		`
		UPDATE public.users
		SET session_version = session_version + 1
		WHERE id = $1
		`,
		userID,
	)

	return err
}
