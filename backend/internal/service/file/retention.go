package file

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"sapphirebroking.com/sftp_service/internal/apperrors"
	"sapphirebroking.com/sftp_service/internal/db/sftpdb"
)

// mutationBlocked reports whether a compliance control prevents mutating a file.
// Legal hold blocks everything; retention blocks only destructive changes
// (delete, content overwrite) until the retain-until timestamp passes.
func mutationBlocked(f sftpdb.File, destructive bool) error {
	if f.LegalHold {
		return apperrors.ErrLegalHold
	}
	if destructive && f.RetainUntil.Valid && f.RetainUntil.Time.After(time.Now()) {
		return apperrors.ErrUnderRetention
	}
	return nil
}

// SetLegalHold places or releases a legal hold on a file (admin action).
func (s *Service) SetLegalHold(ctx context.Context, fileID uuid.UUID, hold bool) error {
	if _, err := s.q.GetFileByID(ctx, fileID); err != nil {
		return apperrors.ErrFileNotFound
	}
	return s.q.SetFileLegalHold(ctx, sftpdb.SetFileLegalHoldParams{ID: fileID, Hold: hold})
}

// SetRetention sets (or clears, when until is nil) a WORM retention lock on a
// file. Retention can be extended but not shortened below the current value.
func (s *Service) SetRetention(ctx context.Context, fileID uuid.UUID, until *time.Time) error {
	f, err := s.q.GetFileByID(ctx, fileID)
	if err != nil {
		return apperrors.ErrFileNotFound
	}
	// Never allow shortening an active retention (compliance requirement).
	if f.RetainUntil.Valid && f.RetainUntil.Time.After(time.Now()) {
		if until == nil || until.Before(f.RetainUntil.Time) {
			return apperrors.ErrUnderRetention
		}
	}
	var val pgtype.Timestamptz
	if until != nil {
		val = pgtype.Timestamptz{Time: *until, Valid: true}
	}
	return s.q.SetFileRetention(ctx, sftpdb.SetFileRetentionParams{ID: fileID, RetainUntil: val})
}
