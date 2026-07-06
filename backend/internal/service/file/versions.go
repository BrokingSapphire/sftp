package file

import (
	"context"
	"io"
	"time"

	"github.com/google/uuid"

	"sapphirebroking.com/sftp_service/internal/apperrors"
	"sapphirebroking.com/sftp_service/internal/db/sftpdb"
	models "sapphirebroking.com/sftp_service/internal/models/file"
	"sapphirebroking.com/sftp_service/internal/utils"
)

// commitContent creates a new file, or — if one already exists at the same
// owner/folder/name — archives the current content as a version and updates the
// file in place (bumping version_no). Returns the resulting file and whether it
// was a new version.
func (s *Service) commitContent(ctx context.Context, owner uuid.UUID, folder *uuid.UUID, name, key, checksum string, size int64) (sftpdb.File, bool, error) {
	if existing, err := s.q.GetFileByOwnerFolderName(ctx, sftpdb.GetFileByOwnerFolderNameParams{
		OwnerID: owner, Name: name, FolderID: folder,
	}); err == nil {
		// A legal hold or active retention lock blocks overwriting the content.
		if err := mutationBlocked(existing, true); err != nil {
			return sftpdb.File{}, false, err
		}
		f, err := s.versionedReplace(ctx, existing, key, checksum, size, owner)
		return f, true, err
	}

	f, err := s.q.CreateFile(ctx, sftpdb.CreateFileParams{
		OwnerID: owner, FolderID: folder, Name: name,
		Extension: utils.FileExtension(name), MimeType: mimeByName(name),
		SizeBytes: size, ChecksumSha256: &checksum, StorageKey: key,
	})
	if err != nil {
		return sftpdb.File{}, false, mapConflict(err)
	}
	if err := s.q.AddStorageUsed(ctx, sftpdb.AddStorageUsedParams{ID: owner, StorageUsed: size}); err != nil {
		s.log.Error("increment storage used failed", "err", err)
	}
	return f, false, nil
}

// versionedReplace snapshots a file's current content into file_versions, then
// points the file at the new content and bumps its version. Storage is adjusted
// by the size delta (old content is retained as a downloadable version).
func (s *Service) versionedReplace(ctx context.Context, cur sftpdb.File, newKey, newChecksum string, newSize int64, by uuid.UUID) (sftpdb.File, error) {
	if err := s.q.InsertFileVersion(ctx, sftpdb.InsertFileVersionParams{
		FileID: cur.ID, VersionNo: cur.VersionNo, SizeBytes: cur.SizeBytes,
		ChecksumSha256: cur.ChecksumSha256, StorageKey: cur.StorageKey, CreatedBy: &by,
	}); err != nil {
		return sftpdb.File{}, err
	}
	updated, err := s.q.BumpFileContent(ctx, sftpdb.BumpFileContentParams{
		ID: cur.ID, StorageKey: newKey, SizeBytes: newSize, ChecksumSha256: &newChecksum,
	})
	if err != nil {
		return sftpdb.File{}, err
	}
	if delta := newSize - cur.SizeBytes; delta != 0 {
		if err := s.q.AddStorageUsed(ctx, sftpdb.AddStorageUsedParams{ID: cur.OwnerID, StorageUsed: delta}); err != nil {
			s.log.Error("version storage delta failed", "err", err)
		}
	}
	return updated, nil
}

// OverwriteContent replaces a file's content in place (used by the in-app
// editor), archiving the previous content as a version. Respects legal hold and
// retention, and re-indexes/re-classifies the new content.
func (s *Service) OverwriteContent(ctx context.Context, owner, fileID uuid.UUID, r io.Reader) (*models.FileResponse, error) {
	cur, err := s.ownedFile(ctx, owner, fileID)
	if err != nil {
		return nil, err
	}
	if err := mutationBlocked(cur, true); err != nil {
		return nil, err
	}
	res, err := s.store.Save(r)
	if err != nil {
		return nil, err
	}
	if s.maxUploadSize > 0 && res.Size > s.maxUploadSize {
		_ = s.store.Delete(res.Key)
		return nil, apperrors.ErrPayloadTooLarge
	}
	updated, err := s.versionedReplace(ctx, cur, res.Key, res.Checksum, res.Size, owner)
	if err != nil {
		_ = s.store.Delete(res.Key)
		return nil, err
	}
	s.indexAsync(updated.ID)
	return toFileResponse(updated), nil
}

// ListVersions returns a file's archived (previous) versions, newest first. The
// current content is version_no on the file itself.
func (s *Service) ListVersions(ctx context.Context, owner, fileID uuid.UUID) ([]models.FileVersionResponse, error) {
	if _, err := s.ownedFile(ctx, owner, fileID); err != nil {
		return nil, err
	}
	rows, err := s.q.ListFileVersions(ctx, fileID)
	if err != nil {
		return nil, err
	}
	out := make([]models.FileVersionResponse, 0, len(rows))
	for _, r := range rows {
		v := models.FileVersionResponse{VersionNo: r.VersionNo, SizeBytes: r.SizeBytes}
		if r.ChecksumSha256 != nil {
			v.Checksum = *r.ChecksumSha256
		}
		if r.AuthorName != nil && *r.AuthorName != "" {
			v.Author = *r.AuthorName
		} else if r.AuthorUsername != nil {
			v.Author = *r.AuthorUsername
		}
		if r.CreatedAt.Valid {
			v.CreatedAt = r.CreatedAt.Time.Format(time.RFC3339)
		}
		out = append(out, v)
	}
	return out, nil
}

// RestoreVersion makes a previous version the current content (archiving the
// content it replaces, so restore is itself reversible).
func (s *Service) RestoreVersion(ctx context.Context, owner, fileID uuid.UUID, versionNo int32) (*models.FileResponse, error) {
	cur, err := s.ownedFile(ctx, owner, fileID)
	if err != nil {
		return nil, err
	}
	v, err := s.q.GetFileVersion(ctx, sftpdb.GetFileVersionParams{FileID: fileID, VersionNo: versionNo})
	if err != nil {
		return nil, apperrors.ErrNotFound
	}
	checksum := ""
	if v.ChecksumSha256 != nil {
		checksum = *v.ChecksumSha256
	}
	updated, err := s.versionedReplace(ctx, cur, v.StorageKey, checksum, v.SizeBytes, owner)
	if err != nil {
		return nil, err
	}
	s.indexAsync(updated.ID)
	return toFileResponse(updated), nil
}

// OpenVersionForDownload streams a specific archived version.
func (s *Service) OpenVersionForDownload(ctx context.Context, owner, fileID uuid.UUID, versionNo int32) (*DownloadHandle, error) {
	f, err := s.ownedFile(ctx, owner, fileID)
	if err != nil {
		return nil, err
	}
	v, err := s.q.GetFileVersion(ctx, sftpdb.GetFileVersionParams{FileID: fileID, VersionNo: versionNo})
	if err != nil {
		return nil, apperrors.ErrNotFound
	}
	fh, err := s.store.Open(v.StorageKey)
	if err != nil {
		return nil, err
	}
	return &DownloadHandle{
		File: fh, Name: f.Name, MimeType: f.MimeType, Size: v.SizeBytes, ModTime: time.Now(),
	}, nil
}
