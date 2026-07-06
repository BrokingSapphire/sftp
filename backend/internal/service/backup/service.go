// Package backup creates and restores encrypted, binary backups of every user's
// drive to a super-admin-selected directory (e.g. a mounted removable disk).
//
// It is incremental: the target directory holds a state manifest recording the
// checksum of every file already backed up. A run scans the live database and
// only archives files that are new or changed since the last run — a full backup
// happens automatically the first time (empty/missing manifest).
//
// Each archive is an AES-256-CTR–encrypted tar (magic-prefixed) containing a
// per-archive manifest plus the file contents (decrypted from at-rest storage,
// then re-encrypted with the backup key).
package backup

import (
	"archive/tar"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"sapphirebroking.com/sftp_service/internal/db/sftpdb"
	"sapphirebroking.com/sftp_service/internal/storage"
	"sapphirebroking.com/sftp_service/pkg/filecrypt"
	"sapphirebroking.com/sftp_service/pkg/logger"
)

const (
	stateFile  = ".sapphire-backup.json"
	magic      = "SPHRBKP1"
	dirPerm    = 0o750
	filePerm   = 0o640
)

// Service performs backups and restores.
type Service struct {
	q      *sftpdb.Queries
	store  *storage.Engine
	cipher *filecrypt.Cipher // required — backups are always encrypted
	log    logger.Logger
}

// New builds the backup service. cipher may be nil (backup then returns an error
// prompting the operator to configure an encryption key).
func New(q *sftpdb.Queries, store *storage.Engine, cipher *filecrypt.Cipher, log logger.Logger) *Service {
	return &Service{q: q, store: store, cipher: cipher, log: log.Named("service.backup")}
}

// Enabled reports whether an encryption key is configured (required for backups).
func (s *Service) Enabled() bool { return s.cipher != nil }

// fileEntry is a file recorded in the state / archive manifest.
type fileEntry struct {
	Owner      string `json:"owner"`
	OwnerName  string `json:"owner_name"`
	FolderPath string `json:"folder_path"`
	Name       string `json:"name"`
	Checksum   string `json:"checksum"`
	Size       int64  `json:"size"`
	IsCommon   bool   `json:"is_common"`
	Archive    string `json:"archive"` // which archive holds the latest copy
}

// archiveRecord summarises one written archive.
type archiveRecord struct {
	Name  string    `json:"name"`
	Mode  string    `json:"mode"` // full | incremental
	At    time.Time `json:"at"`
	Count int       `json:"count"`
	Bytes int64     `json:"bytes"`
}

// state is the manifest kept in the target directory.
type state struct {
	Version   int                  `json:"version"`
	CreatedAt time.Time            `json:"created_at"`
	UpdatedAt time.Time            `json:"updated_at"`
	Files     map[string]fileEntry `json:"files"`
	Archives  []archiveRecord      `json:"archives"`
}

// Result is returned to the caller after a run.
type Result struct {
	Mode        string `json:"mode"`         // full | incremental | none
	Archive     string `json:"archive"`      // file written (empty if none)
	FilesBacked int    `json:"files_backed"` // files in this run
	Bytes       int64  `json:"bytes"`
	TotalFiles  int    `json:"total_files"` // total tracked after this run
}

// Status describes the backups already present on a target.
type Status struct {
	Exists       bool            `json:"exists"`
	TotalFiles   int             `json:"total_files"`
	Archives     []archiveRecord `json:"archives"`
	LastBackupAt *time.Time      `json:"last_backup_at,omitempty"`
	NextMode     string          `json:"next_mode"` // what a run now would do: full | incremental
}

// Run performs a backup to targetDir, choosing full or incremental automatically.
func (s *Service) Run(ctx context.Context, targetDir string) (*Result, error) {
	if s.cipher == nil {
		return nil, fmt.Errorf("backups require an encryption key (set STORAGE_ENCRYPTION_KEY)")
	}
	if err := ensureWritableDir(targetDir); err != nil {
		return nil, err
	}
	st, err := loadState(targetDir)
	if err != nil {
		return nil, err
	}
	full := len(st.Files) == 0

	rows, err := s.q.ListAllFilesForBackup(ctx)
	if err != nil {
		return nil, err
	}
	// Select files that are new or changed since the last run.
	var todo []sftpdb.ListAllFilesForBackupRow
	for _, r := range rows {
		sum := deref(r.ChecksumSha256)
		if prev, ok := st.Files[r.ID.String()]; ok && prev.Checksum == sum {
			continue
		}
		todo = append(todo, r)
	}
	if len(todo) == 0 {
		return &Result{Mode: "none", TotalFiles: len(st.Files)}, nil
	}

	mode := "incremental"
	if full {
		mode = "full"
	}
	now := time.Now().UTC()
	archiveName := fmt.Sprintf("%s-%s.bin", mode, now.Format("20060102-150405"))
	bytes, entries, err := s.writeArchive(ctx, filepath.Join(targetDir, archiveName), todo)
	if err != nil {
		return nil, err
	}

	// Update state.
	if st.CreatedAt.IsZero() {
		st.CreatedAt = now
	}
	st.UpdatedAt = now
	for id, e := range entries {
		e.Archive = archiveName
		st.Files[id] = e
	}
	st.Archives = append(st.Archives, archiveRecord{Name: archiveName, Mode: mode, At: now, Count: len(entries), Bytes: bytes})
	if err := saveState(targetDir, st); err != nil {
		return nil, err
	}

	s.log.Info("backup complete", "mode", mode, "files", len(entries), "bytes", bytes, "archive", archiveName)
	return &Result{Mode: mode, Archive: archiveName, FilesBacked: len(entries), Bytes: bytes, TotalFiles: len(st.Files)}, nil
}

// writeArchive streams an encrypted tar of the given files to path.
func (s *Service) writeArchive(ctx context.Context, path string, rows []sftpdb.ListAllFilesForBackupRow) (int64, map[string]fileEntry, error) {
	out, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, filePerm)
	if err != nil {
		return 0, nil, err
	}
	defer out.Close()

	if _, err := out.Write([]byte(magic)); err != nil {
		return 0, nil, err
	}
	encW, err := s.cipher.EncryptWriter(out)
	if err != nil {
		return 0, nil, err
	}
	tw := tar.NewWriter(encW)

	entries := make(map[string]fileEntry, len(rows))
	for _, r := range rows {
		entries[r.ID.String()] = fileEntry{
			Owner: r.OwnerID.String(), OwnerName: r.OwnerUsername, FolderPath: r.FolderPath,
			Name: r.Name, Checksum: deref(r.ChecksumSha256), Size: r.SizeBytes, IsCommon: r.IsCommon,
		}
	}
	// Archive manifest first.
	mj, _ := json.Marshal(entries)
	if err := writeTar(tw, "manifest.json", int64(len(mj)), func(w io.Writer) error {
		_, e := w.Write(mj)
		return e
	}); err != nil {
		return 0, nil, err
	}

	var total int64
	for _, r := range rows {
		rc, err := s.store.Open(r.StorageKey)
		if err != nil {
			s.log.Warn("backup: open failed", "file", r.ID, "err", err)
			continue
		}
		err = writeTar(tw, "data/"+r.ID.String(), r.SizeBytes, func(w io.Writer) error {
			_, e := io.Copy(w, rc)
			return e
		})
		rc.Close()
		if err != nil {
			return 0, nil, err
		}
		total += r.SizeBytes
	}
	if err := tw.Close(); err != nil {
		return 0, nil, err
	}
	return total, entries, nil
}

func writeTar(tw *tar.Writer, name string, size int64, write func(io.Writer) error) error {
	if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o600, Size: size}); err != nil {
		return err
	}
	return write(tw)
}

// Status scans a target directory and reports what a run would do.
func (s *Service) Status(_ context.Context, targetDir string) (*Status, error) {
	st, err := loadState(targetDir)
	if err != nil {
		return nil, err
	}
	out := &Status{TotalFiles: len(st.Files), Archives: st.Archives}
	if len(st.Archives) > 0 {
		out.Exists = true
		last := st.Archives[len(st.Archives)-1].At
		out.LastBackupAt = &last
		out.NextMode = "incremental"
	} else {
		out.NextMode = "full"
	}
	return out, nil
}

// RestoreResult summarises a restore.
type RestoreResult struct {
	Restored int `json:"restored"`
	Skipped  int `json:"skipped"`
}

// Restore rebuilds files (and their folder trees) from every archive in the
// target directory. It is idempotent: a file already present at the same
// owner/folder/name with a matching checksum is skipped.
func (s *Service) Restore(ctx context.Context, targetDir string) (*RestoreResult, error) {
	if s.cipher == nil {
		return nil, fmt.Errorf("restore requires the same encryption key used for the backup")
	}
	st, err := loadState(targetDir)
	if err != nil {
		return nil, err
	}
	res := &RestoreResult{}
	for _, ar := range st.Archives {
		if err := s.restoreArchive(ctx, filepath.Join(targetDir, ar.Name), res); err != nil {
			s.log.Error("restore archive failed", "archive", ar.Name, "err", err)
			return res, err
		}
	}
	s.log.Info("restore complete", "restored", res.Restored, "skipped", res.Skipped)
	return res, nil
}

func (s *Service) restoreArchive(ctx context.Context, path string, res *RestoreResult) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	hdr := make([]byte, len(magic))
	if _, err := io.ReadFull(f, hdr); err != nil || string(hdr) != magic {
		return fmt.Errorf("not a valid backup archive: %s", filepath.Base(path))
	}
	dr, err := s.cipher.DecryptReader(f)
	if err != nil {
		return err
	}
	tr := tar.NewReader(dr)

	var manifest map[string]fileEntry
	for {
		th, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if th.Name == "manifest.json" {
			raw, _ := io.ReadAll(tr)
			_ = json.Unmarshal(raw, &manifest)
			continue
		}
		if len(th.Name) <= len("data/") || th.Name[:5] != "data/" {
			continue
		}
		id := th.Name[5:]
		e, ok := manifest[id]
		if !ok {
			continue
		}
		if err := s.restoreFile(ctx, e, tr); err != nil {
			s.log.Warn("restore file failed", "file", id, "err", err)
			res.Skipped++
			continue
		}
		res.Restored++
	}
	return nil
}

func (s *Service) restoreFile(ctx context.Context, e fileEntry, content io.Reader) error {
	owner, err := uuid.Parse(e.Owner)
	if err != nil {
		return err
	}
	if _, err := s.q.GetUserByID(ctx, owner); err != nil {
		return fmt.Errorf("owner no longer exists")
	}
	folderID, err := s.ensureFolderPath(ctx, owner, e.FolderPath)
	if err != nil {
		return err
	}
	// Idempotent: skip if an identical file already exists.
	if existing, err := s.q.GetFileByOwnerFolderName(ctx, sftpdb.GetFileByOwnerFolderNameParams{
		OwnerID: owner, Name: e.Name, FolderID: folderID,
	}); err == nil && deref(existing.ChecksumSha256) == e.Checksum {
		return nil
	}

	res, err := s.store.Save(io.LimitReader(content, e.Size))
	if err != nil {
		return err
	}
	checksum := res.Checksum
	nf, err := s.q.CreateFile(ctx, sftpdb.CreateFileParams{
		OwnerID: owner, FolderID: folderID, Name: e.Name,
		Extension: extOf(e.Name), MimeType: mimeOf(e.Name),
		SizeBytes: res.Size, ChecksumSha256: &checksum, StorageKey: res.Key,
	})
	if err != nil {
		_ = s.store.Delete(res.Key)
		return err
	}
	if !e.IsCommon {
		_ = s.q.AddStorageUsed(ctx, sftpdb.AddStorageUsedParams{ID: owner, StorageUsed: res.Size})
	} else {
		_ = s.q.SetFileCommon(ctx, sftpdb.SetFileCommonParams{ID: nf.ID, IsCommon: true})
	}
	return nil
}

func extOf(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	return strings.TrimPrefix(ext, ".")
}

func mimeOf(name string) string {
	if t := mime.TypeByExtension(filepath.Ext(name)); t != "" {
		return t
	}
	return "application/octet-stream"
}

// ensureFolderPath recreates a "/a/b/c" folder tree under owner, returning the
// deepest folder id (nil for root).
func (s *Service) ensureFolderPath(ctx context.Context, owner uuid.UUID, path string) (*uuid.UUID, error) {
	path = strings.Trim(path, "/")
	if path == "" {
		return nil, nil
	}
	var parent *uuid.UUID
	accum := ""
	depth := int32(0)
	for _, seg := range strings.Split(path, "/") {
		if seg == "" {
			continue
		}
		accum += "/" + seg
		depth++
		// Find an existing child with this name.
		children, err := s.q.ListFoldersByParent(ctx, sftpdb.ListFoldersByParentParams{OwnerID: owner, ParentID: parent})
		if err != nil {
			return nil, err
		}
		var found *uuid.UUID
		for _, c := range children {
			if c.Name == seg {
				id := c.ID
				found = &id
				break
			}
		}
		if found != nil {
			parent = found
			continue
		}
		nf, err := s.q.CreateFolder(ctx, sftpdb.CreateFolderParams{
			OwnerID: owner, Name: seg, Path: accum, Depth: depth, ParentID: parent,
		})
		if err != nil {
			return nil, err
		}
		id := nf.ID
		parent = &id
	}
	return parent, nil
}

func ensureWritableDir(dir string) error {
	if dir == "" {
		return fmt.Errorf("a target directory is required")
	}
	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("target directory not accessible: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("target is not a directory")
	}
	// Probe writability.
	probe := filepath.Join(dir, ".sphr-write-test")
	if err := os.WriteFile(probe, []byte("ok"), filePerm); err != nil {
		return fmt.Errorf("target directory is not writable: %w", err)
	}
	_ = os.Remove(probe)
	return nil
}

func loadState(dir string) (*state, error) {
	raw, err := os.ReadFile(filepath.Join(dir, stateFile))
	if err != nil {
		if os.IsNotExist(err) {
			return &state{Version: 1, Files: map[string]fileEntry{}}, nil
		}
		return nil, err
	}
	var st state
	if err := json.Unmarshal(raw, &st); err != nil {
		return nil, fmt.Errorf("corrupt backup manifest: %w", err)
	}
	if st.Files == nil {
		st.Files = map[string]fileEntry{}
	}
	return &st, nil
}

func saveState(dir string, st *state) error {
	raw, _ := json.MarshalIndent(st, "", "  ")
	return os.WriteFile(filepath.Join(dir, stateFile), raw, filePerm)
}

func deref(p *string) string {
	if p != nil {
		return *p
	}
	return ""
}
