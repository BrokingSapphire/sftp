package user

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"

	"sapphirebroking.com/sftp_service/internal/apperrors"
)

// mapCreateErr converts a unique-violation into a domain conflict error.
func mapCreateErr(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return apperrors.ErrUserAlreadyExists
	}
	return err
}
