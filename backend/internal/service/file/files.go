package file

import (
	"context"

	"github.com/google/uuid"

	"sapphirebroking.com/sftp_service/internal/apperrors"
	"sapphirebroking.com/sftp_service/internal/db/sftpdb"
	models "sapphirebroking.com/sftp_service/internal/models/file"
	"sapphirebroking.com/sftp_service/internal/utils"
)

// GetFile returns a file's metadata.
func (s *Service) GetFile(ctx context.Context, owner, id uuid.UUID) (*models.FileResponse, error) {
	f, err := s.ownedFile(ctx, owner, id)
	if err != nil {
		return nil, err
	}
	return toFileResponse(f), nil
}

// RenameFile renames a file, preserving/deriving its extension.
func (s *Service) RenameFile(ctx context.Context, owner, id uuid.UUID, newName string) error {
	name, err := utils.SanitizeName(newName)
	if err != nil {
		return err
	}
	f, err := s.ownedFile(ctx, owner, id)
	if err != nil {
		return err
	}
	if err := mutationBlocked(f, false); err != nil {
		return err
	}
	return s.q.RenameFile(ctx, sftpdb.RenameFileParams{ID: id, Name: name, Extension: utils.FileExtension(name)})
}

// MoveFile moves a file into targetFolder (nil = root).
func (s *Service) MoveFile(ctx context.Context, owner, id uuid.UUID, targetFolder *uuid.UUID) error {
	f, err := s.ownedFile(ctx, owner, id)
	if err != nil {
		return err
	}
	if err := mutationBlocked(f, false); err != nil {
		return err
	}
	if targetFolder != nil {
		if _, err := s.ownedFolder(ctx, owner, *targetFolder); err != nil {
			return err
		}
	}
	return s.q.MoveFile(ctx, sftpdb.MoveFileParams{ID: id, FolderID: targetFolder})
}

// StarFile toggles the starred flag.
func (s *Service) StarFile(ctx context.Context, owner, id uuid.UUID, starred bool) error {
	if _, err := s.ownedFile(ctx, owner, id); err != nil {
		return err
	}
	return s.q.SetFileStar(ctx, sftpdb.SetFileStarParams{ID: id, IsStarred: starred})
}

// TrashFile moves a file to the recycle bin (soft delete).
func (s *Service) TrashFile(ctx context.Context, owner, id uuid.UUID) error {
	f, err := s.ownedFile(ctx, owner, id)
	if err != nil {
		return err
	}
	if err := mutationBlocked(f, true); err != nil {
		return err
	}
	return s.q.SoftDeleteFile(ctx, id)
}

// RestoreFile restores a file from the recycle bin.
func (s *Service) RestoreFile(ctx context.Context, owner, id uuid.UUID) error {
	f, err := s.q.GetFileByIDIncludingTrashed(ctx, id)
	if err != nil {
		return apperrors.ErrFileNotFound
	}
	if f.OwnerID != owner {
		return apperrors.ErrForbidden
	}
	return s.q.RestoreFile(ctx, id)
}

// DeletePermanent removes a file for good and frees its storage + quota.
func (s *Service) DeletePermanent(ctx context.Context, owner, id uuid.UUID) error {
	f, err := s.q.GetFileByIDIncludingTrashed(ctx, id)
	if err != nil {
		return apperrors.ErrFileNotFound
	}
	if f.OwnerID != owner {
		return apperrors.ErrForbidden
	}
	if err := mutationBlocked(f, true); err != nil {
		return err
	}
	key, err := s.q.HardDeleteFile(ctx, id)
	if err != nil {
		return err
	}
	if err := s.store.Delete(key); err != nil {
		s.log.Error("delete storage object failed", "key", key, "err", err)
	}
	if err := s.q.AddStorageUsed(ctx, sftpdb.AddStorageUsedParams{ID: owner, StorageUsed: -f.SizeBytes}); err != nil {
		s.log.Error("decrement storage used failed", "err", err)
	}
	return nil
}

// ListTrash returns soft-deleted files.
func (s *Service) ListTrash(ctx context.Context, owner uuid.UUID, limit, offset int) ([]models.FileResponse, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.q.ListTrash(ctx, sftpdb.ListTrashParams{OwnerID: owner, Limit: int32(limit), Offset: int32(offset)})
	if err != nil {
		return nil, err
	}
	return mapFiles(rows), nil
}

// ListRecent returns recently created files.
func (s *Service) ListRecent(ctx context.Context, owner uuid.UUID, limit int) ([]models.FileResponse, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := s.q.ListRecentFiles(ctx, sftpdb.ListRecentFilesParams{OwnerID: owner, Limit: int32(limit)})
	if err != nil {
		return nil, err
	}
	return mapFiles(rows), nil
}

// ListStarred returns starred files.
func (s *Service) ListStarred(ctx context.Context, owner uuid.UUID) ([]models.FileResponse, error) {
	rows, err := s.q.ListStarredFiles(ctx, owner)
	if err != nil {
		return nil, err
	}
	return mapFiles(rows), nil
}

// Search finds files by fuzzy name match.
func (s *Service) Search(ctx context.Context, owner uuid.UUID, query string, limit, offset int) ([]models.FileResponse, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.q.SearchFiles(ctx, sftpdb.SearchFilesParams{
		OwnerID: owner, Column2: &query, Limit: int32(limit), Offset: int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return mapFiles(rows), nil
}

// SearchContent runs full-text search over extracted file text, returning hits
// with a highlighted snippet, ranked by relevance.
func (s *Service) SearchContent(ctx context.Context, owner uuid.UUID, query string, limit int) ([]models.SearchHit, error) {
	if len(query) == 0 {
		return []models.SearchHit{}, nil
	}
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	rows, err := s.q.SearchFileContent(ctx, sftpdb.SearchFileContentParams{
		Query: query, OwnerID: owner, RowLimit: int32(limit),
	})
	if err != nil {
		return nil, err
	}
	out := make([]models.SearchHit, 0, len(rows))
	for _, r := range rows {
		hit := models.SearchHit{
			ID: r.ID.String(), Name: r.Name, Extension: r.Extension, MimeType: r.MimeType,
			SizeBytes: r.SizeBytes, IsStarred: r.IsStarred, VersionNo: r.VersionNo,
			DownloadCount: r.DownloadCount, CreatedAt: fmtTS(r.CreatedAt), UpdatedAt: fmtTS(r.UpdatedAt),
			Snippet: string(r.Snippet), Rank: float64(r.Rank),
		}
		if r.FolderID != nil {
			s := r.FolderID.String()
			hit.FolderID = &s
		}
		out = append(out, hit)
	}
	return out, nil
}

func mapFiles(rows []sftpdb.File) []models.FileResponse {
	out := make([]models.FileResponse, 0, len(rows))
	for _, f := range rows {
		out = append(out, *toFileResponse(f))
	}
	return out
}
