package file

import (
	"context"
	"io"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"sapphirebroking.com/sftp_service/internal/apperrors"
	"sapphirebroking.com/sftp_service/internal/db/sftpdb"
	models "sapphirebroking.com/sftp_service/internal/models/file"
	"sapphirebroking.com/sftp_service/internal/utils"
)

// Entry is a virtual-filesystem directory entry (used by the SFTP server).
type Entry struct {
	Name    string
	Size    int64
	IsDir   bool
	ModTime time.Time
}

// cleanPath normalises an SFTP path to an absolute, slash-form path.
func cleanPath(p string) string {
	if p == "" {
		p = "/"
	}
	return path.Clean("/" + strings.TrimPrefix(p, "/"))
}

// FolderByPath resolves a directory path to its folder id (nil = root).
func (s *Service) FolderByPath(ctx context.Context, owner uuid.UUID, dir string) (*uuid.UUID, error) {
	dir = cleanPath(dir)
	if dir == "/" {
		return nil, nil
	}
	folder, err := s.q.GetFolderByOwnerPath(ctx, sftpdb.GetFolderByOwnerPathParams{OwnerID: owner, Path: dir})
	if err != nil {
		return nil, apperrors.ErrFolderNotFound
	}
	return &folder.ID, nil
}

// ListDir lists the entries under a directory path.
func (s *Service) ListDir(ctx context.Context, owner uuid.UUID, dir string) ([]Entry, error) {
	folderID, err := s.FolderByPath(ctx, owner, dir)
	if err != nil {
		return nil, err
	}
	folders, err := s.q.ListFoldersByParent(ctx, sftpdb.ListFoldersByParentParams{OwnerID: owner, ParentID: folderID})
	if err != nil {
		return nil, err
	}
	files, err := s.q.ListFilesByFolder(ctx, sftpdb.ListFilesByFolderParams{OwnerID: owner, FolderID: folderID, Limit: 10000, Offset: 0})
	if err != nil {
		return nil, err
	}
	out := make([]Entry, 0, len(folders)+len(files))
	for _, f := range folders {
		out = append(out, Entry{Name: f.Name, IsDir: true, ModTime: tsTime(f.UpdatedAt)})
	}
	for _, f := range files {
		out = append(out, Entry{Name: f.Name, Size: f.SizeBytes, ModTime: tsTime(f.UpdatedAt)})
	}
	return out, nil
}

// StatPath returns metadata for a path (file or directory).
func (s *Service) StatPath(ctx context.Context, owner uuid.UUID, p string) (Entry, error) {
	p = cleanPath(p)
	if p == "/" {
		return Entry{Name: "/", IsDir: true, ModTime: time.Now()}, nil
	}
	// Try directory first.
	if folder, err := s.q.GetFolderByOwnerPath(ctx, sftpdb.GetFolderByOwnerPathParams{OwnerID: owner, Path: p}); err == nil {
		return Entry{Name: folder.Name, IsDir: true, ModTime: tsTime(folder.UpdatedAt)}, nil
	}
	// Otherwise a file.
	f, err := s.fileByPath(ctx, owner, p)
	if err != nil {
		return Entry{}, err
	}
	return Entry{Name: f.Name, Size: f.SizeBytes, ModTime: tsTime(f.UpdatedAt)}, nil
}

// OpenRead opens a file by path for reading (caller closes).
func (s *Service) OpenRead(ctx context.Context, owner uuid.UUID, p string) (io.ReadSeekCloser, int64, error) {
	f, err := s.fileByPath(ctx, owner, p)
	if err != nil {
		return nil, 0, err
	}
	fh, err := s.store.Open(f.StorageKey)
	if err != nil {
		return nil, 0, err
	}
	_ = s.q.IncrementDownloadCount(ctx, f.ID)
	return fh, f.SizeBytes, nil
}

// WriteFile creates (or replaces) a file at a path from a reader.
func (s *Service) WriteFile(ctx context.Context, owner uuid.UUID, p string, r io.Reader) error {
	p = cleanPath(p)
	dir, name := path.Split(p)
	sane, err := utils.SanitizeName(name)
	if err != nil {
		return err
	}
	folderID, err := s.FolderByPath(ctx, owner, dir)
	if err != nil {
		return err
	}

	res, err := s.store.Save(r)
	if err != nil {
		return err
	}
	if s.maxUploadSize > 0 && res.Size > s.maxUploadSize {
		_ = s.store.Delete(res.Key)
		return apperrors.ErrPayloadTooLarge
	}
	if err := s.checkQuota(ctx, owner, res.Size); err != nil {
		_ = s.store.Delete(res.Key)
		return err
	}

	// Replace an existing file of the same name (SFTP overwrite semantics).
	if existing, err := s.q.GetFileByOwnerFolderName(ctx, sftpdb.GetFileByOwnerFolderNameParams{
		OwnerID: owner, FolderID: folderID, Name: sane,
	}); err == nil {
		if key, derr := s.q.HardDeleteFile(ctx, existing.ID); derr == nil {
			_ = s.store.Delete(key)
			_ = s.q.AddStorageUsed(ctx, sftpdb.AddStorageUsedParams{ID: owner, StorageUsed: -existing.SizeBytes})
		}
	}

	checksum := res.Checksum
	if _, err := s.q.CreateFile(ctx, sftpdb.CreateFileParams{
		OwnerID: owner, FolderID: folderID, Name: sane, Extension: utils.FileExtension(sane),
		MimeType: mimeByName(sane), SizeBytes: res.Size, ChecksumSha256: &checksum, StorageKey: res.Key,
	}); err != nil {
		_ = s.store.Delete(res.Key)
		return mapConflict(err)
	}
	return s.q.AddStorageUsed(ctx, sftpdb.AddStorageUsedParams{ID: owner, StorageUsed: res.Size})
}

// Mkdir creates a directory at a path.
func (s *Service) Mkdir(ctx context.Context, owner uuid.UUID, p string) error {
	p = cleanPath(p)
	dir, name := path.Split(p)
	parentID, err := s.FolderByPath(ctx, owner, dir)
	if err != nil {
		return err
	}
	var parentStr *string
	if parentID != nil {
		v := parentID.String()
		parentStr = &v
	}
	_, err = s.CreateFolder(ctx, owner, models.CreateFolderRequest{Name: name, ParentID: parentStr})
	return err
}

// RemovePath deletes a file (to trash) or an empty folder at a path.
func (s *Service) RemovePath(ctx context.Context, owner uuid.UUID, p string) error {
	p = cleanPath(p)
	if folder, err := s.q.GetFolderByOwnerPath(ctx, sftpdb.GetFolderByOwnerPathParams{OwnerID: owner, Path: p}); err == nil {
		return s.DeleteFolder(ctx, owner, folder.ID)
	}
	f, err := s.fileByPath(ctx, owner, p)
	if err != nil {
		return err
	}
	return s.q.SoftDeleteFile(ctx, f.ID)
}

// Rename renames a file within the same directory (cross-directory rename is
// treated as unsupported to keep the operation atomic and safe).
func (s *Service) Rename(ctx context.Context, owner uuid.UUID, from, to string) error {
	from, to = cleanPath(from), cleanPath(to)
	fromDir, _ := path.Split(from)
	toDir, toName := path.Split(to)
	if cleanPath(fromDir) != cleanPath(toDir) {
		return apperrors.ErrInvalidRequest
	}
	sane, err := utils.SanitizeName(toName)
	if err != nil {
		return err
	}
	if folder, err := s.q.GetFolderByOwnerPath(ctx, sftpdb.GetFolderByOwnerPathParams{OwnerID: owner, Path: from}); err == nil {
		return s.RenameFolder(ctx, owner, folder.ID, sane)
	}
	f, err := s.fileByPath(ctx, owner, from)
	if err != nil {
		return err
	}
	return s.q.RenameFile(ctx, sftpdb.RenameFileParams{ID: f.ID, Name: sane, Extension: utils.FileExtension(sane)})
}

// ── helpers ────────────────────────────────────────────────

func (s *Service) fileByPath(ctx context.Context, owner uuid.UUID, p string) (sftpdb.File, error) {
	p = cleanPath(p)
	dir, name := path.Split(p)
	folderID, err := s.FolderByPath(ctx, owner, dir)
	if err != nil {
		return sftpdb.File{}, err
	}
	f, err := s.q.GetFileByOwnerFolderName(ctx, sftpdb.GetFileByOwnerFolderNameParams{
		OwnerID: owner, FolderID: folderID, Name: name,
	})
	if err != nil {
		return sftpdb.File{}, apperrors.ErrFileNotFound
	}
	return f, nil
}

func tsTime(t pgtype.Timestamptz) time.Time {
	if t.Valid {
		return t.Time
	}
	return time.Now()
}
