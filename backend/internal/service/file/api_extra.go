package file

import (
	"archive/zip"
	"context"
	"io"
	"path"
	"strings"

	"github.com/google/uuid"

	"sapphirebroking.com/sftp_service/internal/db/sftpdb"
	models "sapphirebroking.com/sftp_service/internal/models/file"
)

// WriteFolderZip streams a folder (recursively) as a zip archive to w and
// returns the folder name (for the download filename). Decrypts transparently.
func (s *Service) WriteFolderZip(ctx context.Context, owner, folderID uuid.UUID, w io.Writer) (string, error) {
	root, err := s.ownedFolder(ctx, owner, folderID)
	if err != nil {
		return "", err
	}
	zw := zip.NewWriter(w)
	defer zw.Close()

	type node struct {
		id   uuid.UUID
		path string
	}
	queue := []node{{folderID, root.Name}}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		files, err := s.q.ListFilesInFolder(ctx, sftpdb.ListFilesInFolderParams{OwnerID: owner, FolderID: &cur.id})
		if err != nil {
			return root.Name, err
		}
		for _, f := range files {
			rc, err := s.store.Open(f.StorageKey)
			if err != nil {
				s.log.Warn("zip: open failed", "file", f.ID, "err", err)
				continue
			}
			fw, err := zw.Create(cur.path + "/" + f.Name)
			if err == nil {
				_, _ = io.Copy(fw, rc)
			}
			rc.Close()
		}

		children, err := s.q.ListFoldersByParent(ctx, sftpdb.ListFoldersByParentParams{OwnerID: owner, ParentID: &cur.id})
		if err != nil {
			return root.Name, err
		}
		for _, c := range children {
			queue = append(queue, node{c.ID, cur.path + "/" + c.Name})
		}
	}
	return root.Name, nil
}

// CopyFile duplicates a file in place ("name (copy).ext"), respecting quota.
func (s *Service) CopyFile(ctx context.Context, owner, id uuid.UUID) (*models.FileResponse, error) {
	f, err := s.ownedFile(ctx, owner, id)
	if err != nil {
		return nil, err
	}
	rc, err := s.store.Open(f.StorageKey)
	if err != nil {
		return nil, err
	}
	res, err := s.store.Save(rc)
	rc.Close()
	if err != nil {
		return nil, err
	}
	if err := s.checkQuota(ctx, owner, res.Size); err != nil {
		_ = s.store.Delete(res.Key)
		return nil, err
	}
	nf, _, err := s.commitContent(ctx, owner, f.FolderID, copyName(f.Name), res.Key, res.Checksum, res.Size)
	if err != nil {
		_ = s.store.Delete(res.Key)
		return nil, err
	}
	s.indexAsync(nf.ID)
	return toFileResponse(nf), nil
}

// copyName inserts " (copy)" before the extension.
func copyName(name string) string {
	ext := path.Ext(name)
	base := strings.TrimSuffix(name, ext)
	return base + " (copy)" + ext
}
