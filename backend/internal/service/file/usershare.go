package file

import (
	"context"

	"github.com/google/uuid"

	"sapphirebroking.com/sftp_service/internal/apperrors"
	"sapphirebroking.com/sftp_service/internal/db/sftpdb"
	models "sapphirebroking.com/sftp_service/internal/models/file"
)

// ShareWithUser grants a specific internal user access to one of the caller's
// files (viewer or editor) and notifies them. Idempotent — re-sharing updates
// the role.
func (s *Service) ShareWithUser(ctx context.Context, owner, fileID uuid.UUID, recipientEmail string, canWrite bool) (*models.FileGrantResponse, error) {
	f, err := s.ownedFile(ctx, owner, fileID)
	if err != nil {
		return nil, err
	}
	recipient, err := s.q.GetUserByEmail(ctx, recipientEmail)
	if err != nil {
		return nil, apperrors.ErrUserNotFound
	}
	if recipient.ID == owner {
		return nil, apperrors.ErrInvalidRequest
	}
	if _, err := s.q.GrantFileToUser(ctx, sftpdb.GrantFileToUserParams{
		FileID: &fileID, GranteeUserID: &recipient.ID, CanWrite: canWrite, CreatedBy: &owner,
	}); err != nil {
		return nil, err
	}

	role := "view"
	if canWrite {
		role = "edit"
	}
	link := "/shared"
	_ = s.q.CreateNotification(ctx, sftpdb.CreateNotificationParams{
		UserID: recipient.ID, Type: "share",
		Title: "A file was shared with you",
		Body:  f.Name + " — you can now " + role + " it.",
		Link:  &link,
	})
	s.log.Info("file shared with user", "file", f.Name, "recipient", recipient.Email, "can_write", canWrite)

	return &models.FileGrantResponse{
		UserID: recipient.ID.String(), Name: displayName(recipient.FullName, recipient.Username),
		Email: recipient.Email, HasAvatar: recipient.AvatarPath != nil && *recipient.AvatarPath != "",
		CanWrite: canWrite,
	}, nil
}

// RevokeUserShare removes a user's access to a file (owner only).
func (s *Service) RevokeUserShare(ctx context.Context, owner, fileID, recipientID uuid.UUID) error {
	if _, err := s.ownedFile(ctx, owner, fileID); err != nil {
		return err
	}
	return s.q.RevokeFileGrant(ctx, sftpdb.RevokeFileGrantParams{FileID: &fileID, GranteeUserID: &recipientID})
}

// ListFileGrants lists the internal recipients of a file (owner only).
func (s *Service) ListFileGrants(ctx context.Context, owner, fileID uuid.UUID) ([]models.FileGrantResponse, error) {
	if _, err := s.ownedFile(ctx, owner, fileID); err != nil {
		return nil, err
	}
	rows, err := s.q.ListFileGrants(ctx, &fileID)
	if err != nil {
		return nil, err
	}
	out := make([]models.FileGrantResponse, 0, len(rows))
	for _, r := range rows {
		var uid string
		if r.GranteeUserID != nil {
			uid = r.GranteeUserID.String()
		}
		out = append(out, models.FileGrantResponse{
			UserID: uid, Name: displayName(r.FullName, r.Username), Email: r.Email,
			HasAvatar: derefBool(r.HasAvatar), CanWrite: r.CanWrite,
		})
	}
	return out, nil
}

// ListSharedWithMe returns files other users have shared with the caller.
func (s *Service) ListSharedWithMe(ctx context.Context, uid uuid.UUID) ([]models.SharedFileResponse, error) {
	rows, err := s.q.ListSharedWithMe(ctx, &uid)
	if err != nil {
		return nil, err
	}
	out := make([]models.SharedFileResponse, 0, len(rows))
	for _, r := range rows {
		item := models.SharedFileResponse{
			ID: r.ID.String(), Name: r.Name, Extension: r.Extension, MimeType: r.MimeType,
			SizeBytes: r.SizeBytes, IsStarred: r.IsStarred, VersionNo: r.VersionNo, DownloadCount: r.DownloadCount,
			CreatedAt: fmtTS(r.CreatedAt), UpdatedAt: fmtTS(r.UpdatedAt),
			OwnerID: r.OwnerID.String(), OwnerName: displayName(r.OwnerName, r.OwnerUsername),
			OwnerHasAvatar: derefBool(r.OwnerHasAvatar), CanWrite: r.CanWrite, SharedAt: fmtTS(r.SharedAt),
		}
		out = append(out, item)
	}
	return out, nil
}

// canAccessFile reports whether caller may read/download a file: owner, a
// Common file, or an explicit per-user grant.
func (s *Service) canAccessFile(ctx context.Context, caller uuid.UUID, f sftpdb.File) bool {
	if f.OwnerID == caller || f.IsCommon {
		return true
	}
	grant, err := s.q.GetFileGrant(ctx, sftpdb.GetFileGrantParams{FileID: &f.ID, GranteeUserID: &caller})
	return err == nil && grant.CanDownload
}

func displayName(full, username string) string {
	if full != "" {
		return full
	}
	return username
}

func derefBool(b *bool) bool { return b != nil && *b }
