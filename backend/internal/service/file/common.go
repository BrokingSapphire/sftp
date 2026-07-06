package file

import (
	"context"
	"io"

	"github.com/google/uuid"

	"sapphirebroking.com/sftp_service/internal/apperrors"
	"sapphirebroking.com/sftp_service/internal/db/sftpdb"
	models "sapphirebroking.com/sftp_service/internal/models/file"
)

// ListInherited returns files assigned to the caller from a deleted user.
func (s *Service) ListInherited(ctx context.Context, owner uuid.UUID) ([]models.FileResponse, error) {
	rows, err := s.q.ListInheritedFiles(ctx, owner)
	if err != nil {
		return nil, err
	}
	return mapFiles(rows), nil
}

// KeepInherited clears the pending flag (the heir chooses to keep the file).
func (s *Service) KeepInherited(ctx context.Context, owner, id uuid.UUID) error {
	return s.q.ClearFilePending(ctx, sftpdb.ClearFilePendingParams{ID: id, OwnerID: owner})
}

// ListCommon returns the organisation-wide Common files. can_delete is true
// for the uploader and for admins.
func (s *Service) ListCommon(ctx context.Context, caller uuid.UUID, isAdmin bool, limit, offset int) ([]models.CommonFileResponse, int64, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	rows, err := s.q.ListCommonFiles(ctx, sftpdb.ListCommonFilesParams{Limit: int32(limit), Offset: int32(offset)})
	if err != nil {
		return nil, 0, err
	}
	total, err := s.q.CountCommonFiles(ctx)
	if err != nil {
		return nil, 0, err
	}
	out := make([]models.CommonFileResponse, 0, len(rows))
	for _, r := range rows {
		name := r.UploaderName
		if name == "" {
			name = r.UploaderUsername
		}
		item := models.CommonFileResponse{
			ID: r.ID.String(), Name: r.Name, Extension: r.Extension, MimeType: r.MimeType,
			SizeBytes: r.SizeBytes, IsStarred: r.IsStarred, UploaderID: r.OwnerID.String(),
			UploaderName: name, CanDelete: isAdmin || r.OwnerID == caller,
			VersionNo: r.VersionNo, DownloadCount: r.DownloadCount,
			CreatedAt: fmtTS(r.CreatedAt), UpdatedAt: fmtTS(r.UpdatedAt),
		}
		if r.ChecksumSha256 != nil {
			item.Checksum = *r.ChecksumSha256
		}
		out = append(out, item)
	}
	return out, total, nil
}

// MakeCommon shares one of the caller's own files into the Common area.
func (s *Service) MakeCommon(ctx context.Context, owner, fileID uuid.UUID) error {
	if _, err := s.ownedFile(ctx, owner, fileID); err != nil {
		return err
	}
	return s.q.SetFileCommon(ctx, sftpdb.SetFileCommonParams{ID: fileID, IsCommon: true})
}

// UploadCommon stores a file directly into the Common area (any user may add).
func (s *Service) UploadCommon(ctx context.Context, owner uuid.UUID, filename string, r io.Reader) (*models.FileResponse, error) {
	f, err := s.SimpleUpload(ctx, owner, nil, filename, r)
	if err != nil {
		return nil, err
	}
	id, _ := uuid.Parse(f.ID)
	if err := s.q.SetFileCommon(ctx, sftpdb.SetFileCommonParams{ID: id, IsCommon: true}); err != nil {
		return nil, err
	}
	return f, nil
}

// DeleteCommon permanently removes a Common file. Only the uploader or an
// admin may delete it.
func (s *Service) DeleteCommon(ctx context.Context, caller uuid.UUID, isAdmin bool, fileID uuid.UUID) error {
	f, err := s.q.GetFileByIDIncludingTrashed(ctx, fileID)
	if err != nil || !f.IsCommon {
		return apperrors.ErrFileNotFound
	}
	if !isAdmin && f.OwnerID != caller {
		return apperrors.ErrForbidden
	}
	key, err := s.q.HardDeleteFile(ctx, fileID)
	if err != nil {
		return err
	}
	if err := s.store.Delete(key); err != nil {
		s.log.Error("delete common storage object failed", "key", key, "err", err)
	}
	if err := s.q.AddStorageUsed(ctx, sftpdb.AddStorageUsedParams{ID: f.OwnerID, StorageUsed: -f.SizeBytes}); err != nil {
		s.log.Error("decrement uploader storage failed", "err", err)
	}
	return nil
}
