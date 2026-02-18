package resolver

import (
	"context"
	"database/sql"
	"errors"

	"iam-service/internal/auth"
	"iam-service/internal/db"

	"github.com/google/uuid"
)

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

	email := identity.Email

	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
	})
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	var userID uuid.UUID

	// 1️⃣ Try identity lookup first
	err = tx.QueryRowContext(ctx, `
		SELECT user_id
		FROM public.identities
		WHERE provider = $1
		  AND provider_user_id = $2
	`,
		identity.Provider,
		identity.ProviderUserID,
	).Scan(&userID)

	if err == nil {
		if err := tx.Commit(); err != nil {
			return "", err
		}
		return userID.String(), nil
	}

	if err != sql.ErrNoRows {
		return "", err
	}

	// 2️⃣ Lock user row if exists (email-based linking)
	err = tx.QueryRowContext(ctx, `
		SELECT id
		FROM public.users
		WHERE email = $1
		FOR UPDATE
	`,
		email,
	).Scan(&userID)

	if err == nil {
		// Link identity safely
		_, err = tx.ExecContext(ctx, `
			INSERT INTO public.identities (user_id, provider, provider_user_id)
			VALUES ($1, $2, $3)
			ON CONFLICT (provider, provider_user_id) DO NOTHING
		`,
			userID,
			identity.Provider,
			identity.ProviderUserID,
		)
		if err != nil {
			return "", err
		}

		if err := tx.Commit(); err != nil {
			return "", err
		}
		return userID.String(), nil
	}

	if err != sql.ErrNoRows {
		return "", err
	}

	err = tx.QueryRowContext(ctx,
		`
    		INSERT INTO public.users (email, email_verified)
		    VALUES ($1, $2)
		    ON CONFLICT (email)
		    DO UPDATE SET email = EXCLUDED.email
		    RETURNING id
			`,
		email,
		identity.EmailVerified,
	).Scan(&userID)

	if err != nil {
		return "", err
	}

	// 4️⃣ Insert identity safely
	_, err = tx.ExecContext(ctx, `
		INSERT INTO public.identities (user_id, provider, provider_user_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (provider, provider_user_id) DO NOTHING
	`,
		userID,
		identity.Provider,
		identity.ProviderUserID,
	)

	if err != nil {
		return "", err
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}

	return userID.String(), nil
}
