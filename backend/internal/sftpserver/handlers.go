package sftpserver

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/sftp"

	filesvc "sapphirebroking.com/sftp_service/internal/service/file"
	"sapphirebroking.com/sftp_service/pkg/logger"
)

// newHandlers builds the per-user SFTP request handlers.
func (s *Server) newHandlers(owner uuid.UUID) sftp.Handlers {
	h := &vfsHandler{files: s.files, owner: owner, log: s.log}
	return sftp.Handlers{FileGet: h, FilePut: h, FileCmd: h, FileList: h}
}

type vfsHandler struct {
	files *filesvc.Service
	owner uuid.UUID
	log   logger.Logger
}

func (h *vfsHandler) Fileread(r *sftp.Request) (io.ReaderAt, error) {
	f, _, err := h.files.OpenRead(context.Background(), h.owner, r.Filepath)
	if err != nil {
		return nil, err
	}
	return f, nil // *os.File is io.ReaderAt + io.Closer
}

func (h *vfsHandler) Filewrite(r *sftp.Request) (io.WriterAt, error) {
	tmp, err := os.CreateTemp("", "sftp-in-*")
	if err != nil {
		return nil, err
	}
	return &uploadWriter{tmp: tmp, svc: h.files, owner: h.owner, path: r.Filepath, log: h.log}, nil
}

func (h *vfsHandler) Filecmd(r *sftp.Request) error {
	ctx := context.Background()
	switch r.Method {
	case "Mkdir":
		return h.files.Mkdir(ctx, h.owner, r.Filepath)
	case "Rmdir", "Remove":
		return h.files.RemovePath(ctx, h.owner, r.Filepath)
	case "Rename":
		return h.files.Rename(ctx, h.owner, r.Filepath, r.Target)
	case "Setstat":
		return nil // chmod/chown/truncate metadata is not applicable
	default:
		return fmt.Errorf("unsupported operation: %s", r.Method)
	}
}

func (h *vfsHandler) Filelist(r *sftp.Request) (sftp.ListerAt, error) {
	ctx := context.Background()
	switch r.Method {
	case "List":
		entries, err := h.files.ListDir(ctx, h.owner, r.Filepath)
		if err != nil {
			return nil, err
		}
		infos := make([]os.FileInfo, 0, len(entries))
		for _, e := range entries {
			infos = append(infos, toFileInfo(e))
		}
		return listerAt(infos), nil
	case "Stat":
		e, err := h.files.StatPath(ctx, h.owner, r.Filepath)
		if err != nil {
			return nil, err
		}
		return listerAt{toFileInfo(e)}, nil
	default:
		return nil, fmt.Errorf("unsupported list operation: %s", r.Method)
	}
}

// ── upload writer: buffers to temp, commits on Close ──────

type uploadWriter struct {
	tmp   *os.File
	svc   *filesvc.Service
	owner uuid.UUID
	path  string
	log   logger.Logger
}

func (w *uploadWriter) WriteAt(p []byte, off int64) (int, error) {
	return w.tmp.WriteAt(p, off)
}

func (w *uploadWriter) Close() error {
	name := w.tmp.Name()
	defer os.Remove(name)
	if _, err := w.tmp.Seek(0, io.SeekStart); err != nil {
		w.tmp.Close()
		return err
	}
	err := w.svc.WriteFile(context.Background(), w.owner, w.path, w.tmp)
	_ = w.tmp.Close()
	if err != nil {
		w.log.Warn("sftp upload commit failed", "path", w.path, "err", err)
	}
	return err
}

// ── os.FileInfo + ListerAt ────────────────────────────────

type fileInfo struct {
	name    string
	size    int64
	isDir   bool
	modTime time.Time
}

func toFileInfo(e filesvc.Entry) fileInfo {
	return fileInfo{name: e.Name, size: e.Size, isDir: e.IsDir, modTime: e.ModTime}
}

func (fi fileInfo) Name() string { return fi.name }
func (fi fileInfo) Size() int64  { return fi.size }
func (fi fileInfo) Mode() os.FileMode {
	if fi.isDir {
		return os.ModeDir | 0o755
	}
	return 0o644
}
func (fi fileInfo) ModTime() time.Time { return fi.modTime }
func (fi fileInfo) IsDir() bool        { return fi.isDir }
func (fi fileInfo) Sys() any           { return nil }

type listerAt []os.FileInfo

func (l listerAt) ListAt(ls []os.FileInfo, offset int64) (int, error) {
	if offset >= int64(len(l)) {
		return 0, io.EOF
	}
	n := copy(ls, l[offset:])
	if n < len(ls) {
		return n, io.EOF
	}
	return n, nil
}
