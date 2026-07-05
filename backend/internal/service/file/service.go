// Package file implements folder/file management, resumable uploads and
// streaming downloads on top of the local storage engine.
package file

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"sapphirebroking.com/sftp_service/internal/apperrors"
	"sapphirebroking.com/sftp_service/internal/db/sftpdb"
	models "sapphirebroking.com/sftp_service/internal/models/file"
	"sapphirebroking.com/sftp_service/internal/storage"
	"sapphirebroking.com/sftp_service/internal/utils"
	"sapphirebroking.com/sftp_service/pkg/logger"
)

// Deps are the file service dependencies.
type Deps struct {
	Queries       *sftpdb.Queries
	Storage       *storage.Engine
	Logger        logger.Logger
	ChunkSize     int64
	MaxUploadSize int64
}

// Service provides file and folder operations.
type Service struct {
	q             *sftpdb.Queries
	store         *storage.Engine
	chunkSize     int64
	maxUploadSize int64
	log           logger.Logger
}

// New builds the file Service.
func New(d Deps) *Service {
	return &Service{
		q: d.Queries, store: d.Storage,
		chunkSize: d.ChunkSize, maxUploadSize: d.MaxUploadSize,
		log: d.Logger.Named("service.file"),
	}
}

// ── Folders ───────────────────────────────────────────────

// CreateFolder creates a folder under an optional parent.
func (s *Service) CreateFolder(ctx context.Context, owner uuid.UUID, req models.CreateFolderRequest) (*models.FolderResponse, error) {
	name, err := utils.SanitizeName(req.Name)
	if err != nil {
		return nil, err
	}

	var parentID *uuid.UUID
	parentPath := ""
	depth := int32(0)
	if req.ParentID != nil {
		pid, err := uuid.Parse(*req.ParentID)
		if err != nil {
			return nil, apperrors.ErrInvalidRequest
		}
		parent, err := s.q.GetFolderByID(ctx, pid)
		if err != nil {
			return nil, apperrors.ErrFolderNotFound
		}
		if parent.OwnerID != owner {
			return nil, apperrors.ErrForbidden
		}
		parentID = &pid
		parentPath = parent.Path
		depth = parent.Depth + 1
	}

	folder, err := s.q.CreateFolder(ctx, sftpdb.CreateFolderParams{
		OwnerID:  owner,
		ParentID: parentID,
		Name:     name,
		Path:     parentPath + "/" + name,
		Depth:    depth,
	})
	if err != nil {
		return nil, mapConflict(err)
	}
	return toFolderResponse(folder), nil
}

// ListFolder returns the folders and files directly under folderID (nil=root).
func (s *Service) ListFolder(ctx context.Context, owner uuid.UUID, folderID *uuid.UUID, limit, offset int) (*models.ListingResponse, int64, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	if folderID != nil {
		folder, err := s.q.GetFolderByID(ctx, *folderID)
		if err != nil {
			return nil, 0, apperrors.ErrFolderNotFound
		}
		if folder.OwnerID != owner {
			return nil, 0, apperrors.ErrForbidden
		}
	}

	folders, err := s.q.ListFoldersByParent(ctx, sftpdb.ListFoldersByParentParams{OwnerID: owner, ParentID: folderID})
	if err != nil {
		return nil, 0, err
	}
	files, err := s.q.ListFilesByFolder(ctx, sftpdb.ListFilesByFolderParams{
		OwnerID: owner, FolderID: folderID, Limit: int32(limit), Offset: int32(offset),
	})
	if err != nil {
		return nil, 0, err
	}
	total, err := s.q.CountFilesByFolder(ctx, sftpdb.CountFilesByFolderParams{OwnerID: owner, FolderID: folderID})
	if err != nil {
		return nil, 0, err
	}

	resp := &models.ListingResponse{
		Folders: make([]models.FolderResponse, 0, len(folders)),
		Files:   make([]models.FileResponse, 0, len(files)),
	}
	for _, f := range folders {
		resp.Folders = append(resp.Folders, *toFolderResponse(f))
	}
	for _, f := range files {
		resp.Files = append(resp.Files, *toFileResponse(f))
	}
	return resp, total, nil
}

// RenameFolder renames a folder (and rewrites its path).
func (s *Service) RenameFolder(ctx context.Context, owner uuid.UUID, id uuid.UUID, newName string) error {
	name, err := utils.SanitizeName(newName)
	if err != nil {
		return err
	}
	folder, err := s.ownedFolder(ctx, owner, id)
	if err != nil {
		return err
	}
	parentPath := strings.TrimSuffix(folder.Path, "/"+folder.Name)
	return s.q.RenameFolder(ctx, sftpdb.RenameFolderParams{ID: id, Name: name, Path: parentPath + "/" + name})
}

// MoveFolder reparents a folder.
func (s *Service) MoveFolder(ctx context.Context, owner uuid.UUID, id uuid.UUID, targetID *uuid.UUID) error {
	folder, err := s.ownedFolder(ctx, owner, id)
	if err != nil {
		return err
	}
	newPath := "/" + folder.Name
	depth := int32(0)
	if targetID != nil {
		if *targetID == id {
			return apperrors.ErrInvalidRequest
		}
		target, err := s.ownedFolder(ctx, owner, *targetID)
		if err != nil {
			return err
		}
		if strings.HasPrefix(target.Path+"/", folder.Path+"/") {
			return apperrors.ErrInvalidRequest // cannot move into own subtree
		}
		newPath = target.Path + "/" + folder.Name
		depth = target.Depth + 1
	}
	return s.q.MoveFolder(ctx, sftpdb.MoveFolderParams{ID: id, ParentID: targetID, Path: newPath, Depth: depth})
}

// DeleteFolder soft-deletes a folder (must be empty).
func (s *Service) DeleteFolder(ctx context.Context, owner uuid.UUID, id uuid.UUID) error {
	if _, err := s.ownedFolder(ctx, owner, id); err != nil {
		return err
	}
	n, err := s.q.CountFolderChildren(ctx, &id)
	if err != nil {
		return err
	}
	if n > 0 {
		return apperrors.ErrNotEmpty
	}
	return s.q.SoftDeleteFolder(ctx, id)
}

// StarFolder toggles a folder's starred flag.
func (s *Service) StarFolder(ctx context.Context, owner, id uuid.UUID, starred bool) error {
	if _, err := s.ownedFolder(ctx, owner, id); err != nil {
		return err
	}
	return s.q.SetFolderStar(ctx, sftpdb.SetFolderStarParams{ID: id, IsStarred: starred})
}

// ── shared helpers ────────────────────────────────────────

func (s *Service) ownedFolder(ctx context.Context, owner, id uuid.UUID) (sftpdb.Folder, error) {
	folder, err := s.q.GetFolderByID(ctx, id)
	if err != nil {
		return sftpdb.Folder{}, apperrors.ErrFolderNotFound
	}
	if folder.OwnerID != owner {
		return sftpdb.Folder{}, apperrors.ErrForbidden
	}
	return folder, nil
}

func (s *Service) ownedFile(ctx context.Context, owner, id uuid.UUID) (sftpdb.File, error) {
	f, err := s.q.GetFileByID(ctx, id)
	if err != nil {
		return sftpdb.File{}, apperrors.ErrFileNotFound
	}
	if f.OwnerID != owner {
		return sftpdb.File{}, apperrors.ErrForbidden
	}
	return f, nil
}

func mapConflict(err error) error {
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "23505") || strings.Contains(strings.ToLower(err.Error()), "duplicate") {
		return apperrors.ErrAlreadyExists
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return apperrors.ErrNotFound
	}
	return err
}
