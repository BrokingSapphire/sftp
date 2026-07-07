package file

import (
	"context"
	"io"

	"github.com/google/uuid"

	"sapphirebroking.com/sftp_service/internal/apperrors"
	"sapphirebroking.com/sftp_service/internal/db/sftpdb"
	models "sapphirebroking.com/sftp_service/internal/models/file"
	"sapphirebroking.com/sftp_service/internal/utils"
)

// ListInherited returns files assigned to the caller from a deleted user.
func (s *Service) ListInherited(ctx context.Context, owner uuid.UUID) ([]models.FileResponse, error) {
	rows, err := s.q.ListInheritedFiles(ctx, owner)
	if err != nil {
		return nil, err
	}
	return mapFiles(rows), nil
}

// ListInheritedGrouped returns inherited files grouped by the (deleted) user
// they were transferred from — so the heir sees one section per source user.
func (s *Service) ListInheritedGrouped(ctx context.Context, owner uuid.UUID) ([]models.InheritedGroup, error) {
	rows, err := s.q.ListInheritedWithSource(ctx, owner)
	if err != nil {
		return nil, err
	}
	order := make([]string, 0)
	groups := make(map[string]*models.InheritedGroup)
	for _, r := range rows {
		key := ""
		if r.TransferFrom != nil {
			key = r.TransferFrom.String()
		}
		g, ok := groups[key]
		if !ok {
			name := derefStr(r.FromName)
			if name == "" {
				name = derefStr(r.FromUsername)
			}
			if name == "" {
				name = "Unknown user"
			}
			g = &models.InheritedGroup{FromID: key, FromName: name, FromEmail: derefStr(r.FromEmail)}
			groups[key] = g
			order = append(order, key)
		}
		g.Files = append(g.Files, *toFileResponse(inheritedRowToFile(r)))
	}
	out := make([]models.InheritedGroup, 0, len(order))
	for _, k := range order {
		out = append(out, *groups[k])
	}
	return out, nil
}

func inheritedRowToFile(r sftpdb.ListInheritedWithSourceRow) sftpdb.File {
	return sftpdb.File{
		ID: r.ID, OwnerID: r.OwnerID, FolderID: r.FolderID, Name: r.Name, Extension: r.Extension,
		MimeType: r.MimeType, SizeBytes: r.SizeBytes, ChecksumSha256: r.ChecksumSha256, StorageKey: r.StorageKey,
		ThumbnailKey: r.ThumbnailKey, IsStarred: r.IsStarred, VersionNo: r.VersionNo, DownloadCount: r.DownloadCount,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, DeletedAt: r.DeletedAt, IsCommon: r.IsCommon,
		TransferPending: r.TransferPending, TransferDeadline: r.TransferDeadline, TransferFrom: r.TransferFrom,
		LegalHold: r.LegalHold, RetainUntil: r.RetainUntil, Sensitivity: r.Sensitivity, PiiTypes: r.PiiTypes,
	}
}

func derefStr(p *string) string {
	if p != nil {
		return *p
	}
	return ""
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
			UploaderName: name, UploaderHasAvatar: r.UploaderHasAvatar != nil && *r.UploaderHasAvatar, CanDelete: isAdmin || r.OwnerID == caller,
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

// MakeCommon shares one of the caller's own files into the Common area. The
// Common area is unlimited, so the file's size is freed from the owner's quota.
func (s *Service) MakeCommon(ctx context.Context, owner, fileID uuid.UUID) error {
	f, err := s.ownedFile(ctx, owner, fileID)
	if err != nil {
		return err
	}
	if f.IsCommon {
		return nil
	}
	if err := s.q.SetFileCommon(ctx, sftpdb.SetFileCommonParams{ID: fileID, IsCommon: true}); err != nil {
		return err
	}
	// Common files don't count against personal storage — free the space.
	if err := s.q.AddStorageUsed(ctx, sftpdb.AddStorageUsedParams{ID: owner, StorageUsed: -f.SizeBytes}); err != nil {
		s.log.Error("make-common storage accounting failed", "err", err)
	}
	return nil
}

// UploadCommon stores a file directly into the Common area (any user may add).
// The Common area is UNLIMITED — this bypasses the per-user quota and does not
// count toward the uploader's storage usage.
func (s *Service) UploadCommon(ctx context.Context, owner uuid.UUID, filename string, r io.Reader) (*models.FileResponse, error) {
	name, err := utils.SanitizeName(filename)
	if err != nil {
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
	checksum := res.Checksum
	file, err := s.q.CreateFile(ctx, sftpdb.CreateFileParams{
		OwnerID: owner, FolderID: nil, Name: name,
		Extension: utils.FileExtension(name), MimeType: mimeByName(name),
		SizeBytes: res.Size, ChecksumSha256: &checksum, StorageKey: res.Key,
	})
	if err != nil {
		_ = s.store.Delete(res.Key)
		return nil, mapConflict(err)
	}
	// No AddStorageUsed — Common is free/unlimited.
	if err := s.q.SetFileCommon(ctx, sftpdb.SetFileCommonParams{ID: file.ID, IsCommon: true}); err != nil {
		return nil, err
	}
	s.indexAsync(file.ID)
	return toFileResponse(file), nil
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
	// Common files are not counted against the owner's quota, so nothing to free.
	return nil
}

// CreateCommonFolder creates a navigable folder inside the Common area.
func (s *Service) CreateCommonFolder(ctx context.Context, owner uuid.UUID, parentID *string, name string) (*models.FolderResponse, error) {
	n, err := utils.SanitizeName(name)
	if err != nil {
		return nil, err
	}
	var pid *uuid.UUID
	parentPath := ""
	depth := int32(0)
	if parentID != nil && *parentID != "" {
		id, err := uuid.Parse(*parentID)
		if err != nil {
			return nil, apperrors.ErrInvalidRequest
		}
		parent, err := s.q.GetFolderByID(ctx, id)
		if err != nil {
			return nil, apperrors.ErrFolderNotFound
		}
		if !parent.IsCommon {
			return nil, apperrors.ErrForbidden
		}
		pid = &id
		parentPath = parent.Path
		depth = parent.Depth + 1
	}
	f, err := s.q.CreateCommonFolder(ctx, sftpdb.CreateCommonFolderParams{
		OwnerID: owner, ParentID: pid, Name: n, Path: parentPath + "/" + n, Depth: depth,
	})
	if err != nil {
		return nil, mapConflict(err)
	}
	return toFolderResponse(f), nil
}

// ListCommonAt lists the Common folders and files at a level (nil = root).
func (s *Service) ListCommonAt(ctx context.Context, caller uuid.UUID, isAdmin bool, parentID *string) ([]models.FolderResponse, []models.CommonFileResponse, error) {
	var pid *uuid.UUID
	if parentID != nil && *parentID != "" {
		id, err := uuid.Parse(*parentID)
		if err != nil {
			return nil, nil, apperrors.ErrInvalidRequest
		}
		pid = &id
	}
	fdrs, err := s.q.ListCommonFolders(ctx, pid)
	if err != nil {
		return nil, nil, err
	}
	folders := make([]models.FolderResponse, 0, len(fdrs))
	for _, f := range fdrs {
		folders = append(folders, *toFolderResponse(f))
	}
	rows, err := s.q.ListCommonFilesByFolder(ctx, pid)
	if err != nil {
		return nil, nil, err
	}
	files := make([]models.CommonFileResponse, 0, len(rows))
	for _, r := range rows {
		name := r.UploaderName
		if name == "" {
			name = r.UploaderUsername
		}
		item := models.CommonFileResponse{
			ID: r.ID.String(), Name: r.Name, Extension: r.Extension, MimeType: r.MimeType,
			SizeBytes: r.SizeBytes, IsStarred: r.IsStarred, UploaderID: r.OwnerID.String(),
			UploaderName: name, UploaderHasAvatar: r.UploaderHasAvatar != nil && *r.UploaderHasAvatar,
			CanDelete: isAdmin || r.OwnerID == caller, VersionNo: r.VersionNo, DownloadCount: r.DownloadCount,
			CreatedAt: fmtTS(r.CreatedAt), UpdatedAt: fmtTS(r.UpdatedAt),
		}
		if r.ChecksumSha256 != nil {
			item.Checksum = *r.ChecksumSha256
		}
		files = append(files, item)
	}
	return folders, files, nil
}

// UploadCommonTo uploads a file into a Common folder (nil = Common root).
func (s *Service) UploadCommonTo(ctx context.Context, owner uuid.UUID, folderID *string, filename string, r io.Reader) (*models.FileResponse, error) {
	var fid *uuid.UUID
	if folderID != nil && *folderID != "" {
		id, err := uuid.Parse(*folderID)
		if err != nil {
			return nil, apperrors.ErrInvalidRequest
		}
		parent, err := s.q.GetFolderByID(ctx, id)
		if err != nil {
			return nil, apperrors.ErrFolderNotFound
		}
		if !parent.IsCommon {
			return nil, apperrors.ErrForbidden
		}
		fid = &id
	}
	name, err := utils.SanitizeName(filename)
	if err != nil {
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
	checksum := res.Checksum
	file, err := s.q.CreateFile(ctx, sftpdb.CreateFileParams{
		OwnerID: owner, FolderID: fid, Name: name,
		Extension: utils.FileExtension(name), MimeType: mimeByName(name),
		SizeBytes: res.Size, ChecksumSha256: &checksum, StorageKey: res.Key,
	})
	if err != nil {
		_ = s.store.Delete(res.Key)
		return nil, mapConflict(err)
	}
	if err := s.q.SetFileCommon(ctx, sftpdb.SetFileCommonParams{ID: file.ID, IsCommon: true}); err != nil {
		return nil, err
	}
	s.indexAsync(file.ID)
	return toFileResponse(file), nil
}
