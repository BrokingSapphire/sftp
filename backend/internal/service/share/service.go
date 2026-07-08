// Package share implements share-link creation and public access.
package share

import (
	"archive/zip"
	"context"
	"crypto/rand"
	"encoding/base64"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"sapphirebroking.com/sftp_service/internal/apperrors"
	"sapphirebroking.com/sftp_service/internal/db/sftpdb"
	models "sapphirebroking.com/sftp_service/internal/models/share"
	"sapphirebroking.com/sftp_service/internal/storage"
	"sapphirebroking.com/sftp_service/pkg/argon2"
	"sapphirebroking.com/sftp_service/pkg/logger"
	"sapphirebroking.com/sftp_service/pkg/mailer"
)

// Deps are the share service dependencies.
type Deps struct {
	Queries    *sftpdb.Queries
	Storage    *storage.Engine
	BaseURL    string
	Mailer     *mailer.Mailer
	OrgDomains []string
	Logger     logger.Logger
}

// Service manages share links.
type Service struct {
	q          *sftpdb.Queries
	store      *storage.Engine
	baseURL    string
	mail       *mailer.Mailer
	orgDomains []string
	log        logger.Logger
}

// New builds the share Service.
func New(d Deps) *Service {
	return &Service{
		q: d.Queries, store: d.Storage, baseURL: d.BaseURL,
		mail: d.Mailer, orgDomains: d.OrgDomains, log: d.Logger.Named("service.share"),
	}
}

// Create makes a share link for a file or a folder the caller owns. Exactly one
// of FileID / FolderID must be set.
func (s *Service) Create(ctx context.Context, owner uuid.UUID, req models.CreateRequest) (*models.CreateResponse, error) {
	if (req.FileID == "") == (req.FolderID == "") {
		return nil, apperrors.ErrInvalidRequest // need exactly one of file/folder
	}

	// Resolve the shared resource and confirm ownership. "kind" and "name" drive
	// the DB row, the recipient email, and DLP handling below.
	var (
		kind   string
		name   string
		fileID *uuid.UUID
		folder *uuid.UUID
	)
	if req.FileID != "" {
		id, err := uuid.Parse(req.FileID)
		if err != nil {
			return nil, apperrors.ErrInvalidRequest
		}
		file, err := s.q.GetFileByID(ctx, id)
		if err != nil {
			return nil, apperrors.ErrFileNotFound
		}
		if file.OwnerID != owner {
			return nil, apperrors.ErrForbidden
		}
		// DLP: a public link exposes the file to anyone with the URL. Block it for
		// files classified as restricted (PAN/Aadhaar/card etc.) — those must be
		// shared with specific internal people instead.
		if file.Sensitivity == "restricted" {
			s.log.Warn("dlp: blocked public link for restricted file", "file", file.Name, "owner", owner)
			return nil, apperrors.ErrDLPBlocked
		}
		kind, name, fileID = "file", file.Name, &id
	} else {
		id, err := uuid.Parse(req.FolderID)
		if err != nil {
			return nil, apperrors.ErrInvalidRequest
		}
		fol, err := s.q.GetFolderByID(ctx, id)
		if err != nil {
			return nil, apperrors.ErrFolderNotFound
		}
		if fol.OwnerID != owner {
			return nil, apperrors.ErrForbidden
		}
		// DLP: refuse to publish a folder that contains any restricted file — a
		// public link would otherwise expose it inside the zip.
		if s.folderHasRestricted(ctx, owner, id) {
			s.log.Warn("dlp: blocked public link for folder with restricted files", "folder", fol.Name, "owner", owner)
			return nil, apperrors.ErrDLPBlocked
		}
		kind, name, folder = "folder", fol.Name, &id
	}

	token, err := randomToken()
	if err != nil {
		return nil, err
	}
	permission := req.Permission
	if permission == "" {
		permission = "read"
	}

	var pwHash *string
	if req.Password != "" {
		h, err := argon2.Hash(req.Password, argon2.DefaultParams())
		if err != nil {
			return nil, err
		}
		pwHash = &h
	}
	var limit *int32
	if req.DownloadLimit != nil {
		v := int32(*req.DownloadLimit)
		limit = &v
	}
	var expires pgtype.Timestamptz
	if req.ExpiresInDays != nil && *req.ExpiresInDays > 0 {
		expires = pgtype.Timestamptz{Time: time.Now().AddDate(0, 0, *req.ExpiresInDays), Valid: true}
	}

	sh, err := s.q.CreateShare(ctx, sftpdb.CreateShareParams{
		Token: token, OwnerID: owner, FileID: fileID, FolderID: folder, Permission: permission,
		PasswordHash: pwHash, DownloadLimit: limit, ExpiresAt: expires,
	})
	if err != nil {
		return nil, err
	}

	shareURL := s.baseURL + "/share/" + sh.Token
	resp := &models.CreateResponse{
		ID: sh.ID.String(), Token: sh.Token, URL: shareURL, Kind: kind,
		HasPassword: pwHash != nil, DownloadLimit: limit, CreatedAt: fmtTS(sh.CreatedAt),
	}
	if sh.ExpiresAt.Valid {
		resp.ExpiresAt = sh.ExpiresAt.Time.Format(time.RFC3339)
	}

	// Optionally email the recipient. Flag external (outside-org) recipients.
	if email := strings.TrimSpace(req.RecipientEmail); email != "" {
		resp.External = s.isExternal(email)
		if s.mail != nil && s.mail.Enabled() {
			subject := "A file has been shared with you"
			if kind == "folder" {
				subject = "A folder has been shared with you"
			}
			if err := s.mail.Send(email, subject, shareEmailHTML(kind, name, shareURL, pwHash != nil)); err != nil {
				s.log.Error("share email failed", "to", email, "err", err)
			} else {
				resp.Emailed = true
			}
		}
		s.log.Info("share created", "kind", kind, "name", name, "recipient", email, "external", resp.External)
	}
	return resp, nil
}

// isExternal reports whether an email is outside the configured org domains.
func (s *Service) isExternal(email string) bool {
	if len(s.orgDomains) == 0 {
		return false
	}
	at := strings.LastIndex(email, "@")
	if at < 0 {
		return true
	}
	domain := strings.ToLower(email[at+1:])
	for _, d := range s.orgDomains {
		if strings.EqualFold(strings.TrimSpace(d), domain) {
			return false
		}
	}
	return true
}

func shareEmailHTML(kind, name, url string, hasPassword bool) string {
	pw := ""
	if hasPassword {
		pw = `<p style="color:#b45309;font-size:13px">This link is password-protected — the sender will share the password separately.</p>`
	}
	heading, cta := "A file has been shared with you", "Open the file"
	if kind == "folder" {
		heading, cta = "A folder has been shared with you", "Open the folder"
	}
	return `<div style="font-family:-apple-system,Segoe UI,Roboto,sans-serif;max-width:480px;margin:0 auto">
  <h2 style="color:#064D51">` + heading + `</h2>
  <p style="color:#333">You have been given access to <strong>` + name + `</strong> on Sapphire SFTP.</p>
  <p><a href="` + url + `" style="display:inline-block;background:#064D51;color:#fff;text-decoration:none;padding:10px 20px;border-radius:8px">` + cta + `</a></p>
  ` + pw + `
  <p style="color:#999;font-size:12px">If you were not expecting this, you can ignore this email.</p>
</div>`
}

// List returns the caller's shares.
func (s *Service) List(ctx context.Context, owner uuid.UUID) ([]models.Response, error) {
	rows, err := s.q.ListSharesByOwner(ctx, owner)
	if err != nil {
		return nil, err
	}
	out := make([]models.Response, 0, len(rows))
	for _, sh := range rows {
		r := models.Response{
			ID: sh.ID.String(), Token: sh.Token, Kind: "file", Permission: sh.Permission,
			HasPassword: sh.PasswordHash != nil, DownloadLimit: sh.DownloadLimit,
			DownloadCount: sh.DownloadCount, IsActive: sh.IsActive, CreatedAt: fmtTS(sh.CreatedAt),
		}
		if sh.FileID != nil {
			r.FileID = sh.FileID.String()
		}
		if sh.FolderID != nil {
			r.Kind = "folder"
			r.FolderID = sh.FolderID.String()
		}
		if sh.ExpiresAt.Valid {
			r.ExpiresAt = sh.ExpiresAt.Time.Format(time.RFC3339)
		}
		out = append(out, r)
	}
	return out, nil
}

// Revoke deactivates a share.
func (s *Service) Revoke(ctx context.Context, owner, id uuid.UUID) error {
	return s.q.RevokeShare(ctx, sftpdb.RevokeShareParams{ID: id, OwnerID: owner})
}

// Info returns the public metadata of a share (no download).
func (s *Service) Info(ctx context.Context, token string) (*models.PublicInfo, error) {
	sh, err := s.resolveShare(ctx, token)
	if err != nil {
		return nil, err
	}
	if sh.FolderID != nil {
		fol, err := s.q.GetFolderByID(ctx, *sh.FolderID)
		if err != nil {
			return nil, apperrors.ErrFolderNotFound
		}
		return &models.PublicInfo{
			Token: sh.Token, Kind: "folder", FileName: fol.Name,
			ItemCount:   s.countFolderFiles(ctx, sh.OwnerID, *sh.FolderID),
			HasPassword: sh.PasswordHash != nil, Permission: sh.Permission,
		}, nil
	}
	if sh.FileID == nil {
		return nil, apperrors.ErrShareNotFound
	}
	file, err := s.q.GetFileByID(ctx, *sh.FileID)
	if err != nil {
		return nil, apperrors.ErrFileNotFound
	}
	return &models.PublicInfo{
		Token: sh.Token, Kind: "file", FileName: file.Name, SizeBytes: file.SizeBytes,
		MimeType: file.MimeType, HasPassword: sh.PasswordHash != nil, Permission: sh.Permission,
	}, nil
}

// ShareKind reports whether an active share targets a "file" or a "folder".
func (s *Service) ShareKind(ctx context.Context, token string) (string, error) {
	sh, err := s.resolveShare(ctx, token)
	if err != nil {
		return "", err
	}
	if sh.FolderID != nil {
		return "folder", nil
	}
	if sh.FileID != nil {
		return "file", nil
	}
	return "", apperrors.ErrShareNotFound
}

// OpenHandle is the public download target of a share.
type OpenHandle struct {
	File     io.ReadSeekCloser
	Name     string
	MimeType string
	Size     int64
	ModTime  time.Time
	ShareID  uuid.UUID
	FileID   uuid.UUID
}

// Access validates a file share (and its password) and opens the file for download.
func (s *Service) Access(ctx context.Context, token, password string) (*OpenHandle, error) {
	sh, err := s.resolveShare(ctx, token)
	if err != nil {
		return nil, err
	}
	if sh.FileID == nil {
		return nil, apperrors.ErrShareNotFound
	}
	if err := checkGate(sh, password); err != nil {
		return nil, err
	}
	file, err := s.q.GetFileByID(ctx, *sh.FileID)
	if err != nil {
		return nil, apperrors.ErrFileNotFound
	}
	fh, err := s.store.Open(file.StorageKey)
	if err != nil {
		return nil, err
	}
	_ = s.q.IncrementShareDownload(ctx, sh.ID)
	_ = s.q.IncrementDownloadCount(ctx, file.ID)

	modTime := time.Now()
	if file.UpdatedAt.Valid {
		modTime = file.UpdatedAt.Time
	}
	return &OpenHandle{
		File: fh, Name: file.Name, MimeType: file.MimeType, Size: file.SizeBytes,
		ModTime: modTime, ShareID: sh.ID, FileID: file.ID,
	}, nil
}

// FolderZipTarget is a validated folder share ready to be streamed as a zip.
type FolderZipTarget struct {
	Owner    uuid.UUID
	FolderID uuid.UUID
	Name     string
}

// AccessFolder validates a folder share (password + limit), records the download,
// and returns the target the caller should zip via WriteFolderZip.
func (s *Service) AccessFolder(ctx context.Context, token, password string) (*FolderZipTarget, error) {
	sh, err := s.resolveShare(ctx, token)
	if err != nil {
		return nil, err
	}
	if sh.FolderID == nil {
		return nil, apperrors.ErrShareNotFound
	}
	if err := checkGate(sh, password); err != nil {
		return nil, err
	}
	fol, err := s.q.GetFolderByID(ctx, *sh.FolderID)
	if err != nil {
		return nil, apperrors.ErrFolderNotFound
	}
	_ = s.q.IncrementShareDownload(ctx, sh.ID)
	return &FolderZipTarget{Owner: sh.OwnerID, FolderID: *sh.FolderID, Name: fol.Name}, nil
}

// WriteFolderZip streams the folder (recursively) as a zip archive to w and
// returns the root folder name. Mirrors the authenticated folder-download walk.
func (s *Service) WriteFolderZip(ctx context.Context, owner, folderID uuid.UUID, rootName string, w io.Writer) (string, error) {
	zw := zip.NewWriter(w)
	defer zw.Close()

	type node struct {
		id   uuid.UUID
		path string
	}
	queue := []node{{folderID, rootName}}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		files, err := s.q.ListFilesInFolder(ctx, sftpdb.ListFilesInFolderParams{OwnerID: owner, FolderID: &cur.id})
		if err != nil {
			return rootName, err
		}
		for _, f := range files {
			rc, err := s.store.Open(f.StorageKey)
			if err != nil {
				s.log.Warn("share zip: open failed", "file", f.ID, "err", err)
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
			return rootName, err
		}
		for _, c := range children {
			queue = append(queue, node{c.ID, cur.path + "/" + c.Name})
		}
	}
	return rootName, nil
}

// countFolderFiles returns the number of files under a folder (recursive).
func (s *Service) countFolderFiles(ctx context.Context, owner, folderID uuid.UUID) int {
	total := 0
	queue := []uuid.UUID{folderID}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		files, err := s.q.ListFilesInFolder(ctx, sftpdb.ListFilesInFolderParams{OwnerID: owner, FolderID: &cur})
		if err != nil {
			return total
		}
		total += len(files)
		children, err := s.q.ListFoldersByParent(ctx, sftpdb.ListFoldersByParentParams{OwnerID: owner, ParentID: &cur})
		if err != nil {
			return total
		}
		for _, c := range children {
			queue = append(queue, c.ID)
		}
	}
	return total
}

// folderHasRestricted reports whether any file under the folder (recursive) is
// classified as restricted by DLP — such folders may not be shared publicly.
func (s *Service) folderHasRestricted(ctx context.Context, owner, folderID uuid.UUID) bool {
	queue := []uuid.UUID{folderID}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		files, err := s.q.ListFilesInFolder(ctx, sftpdb.ListFilesInFolderParams{OwnerID: owner, FolderID: &cur})
		if err != nil {
			return false
		}
		for _, f := range files {
			full, err := s.q.GetFileByID(ctx, f.ID)
			if err == nil && full.Sensitivity == "restricted" {
				return true
			}
		}
		children, err := s.q.ListFoldersByParent(ctx, sftpdb.ListFoldersByParentParams{OwnerID: owner, ParentID: &cur})
		if err != nil {
			return false
		}
		for _, c := range children {
			queue = append(queue, c.ID)
		}
	}
	return false
}

// resolveShare loads an active, unexpired share by token.
func (s *Service) resolveShare(ctx context.Context, token string) (sftpdb.Share, error) {
	sh, err := s.q.GetShareByToken(ctx, token)
	if err != nil {
		return sftpdb.Share{}, apperrors.ErrShareNotFound
	}
	if sh.ExpiresAt.Valid && sh.ExpiresAt.Time.Before(time.Now()) {
		return sftpdb.Share{}, apperrors.ErrShareExpired
	}
	return sh, nil
}

// checkGate validates a share's password and download limit.
func checkGate(sh sftpdb.Share, password string) error {
	if sh.PasswordHash != nil {
		if password == "" {
			return apperrors.ErrSharePasswordNeeded
		}
		ok, err := argon2.Verify(password, *sh.PasswordHash)
		if err != nil || !ok {
			return apperrors.ErrSharePasswordNeeded
		}
	}
	if sh.DownloadLimit != nil && sh.DownloadCount >= *sh.DownloadLimit {
		return apperrors.ErrShareLimitReached
	}
	return nil
}

func randomToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func fmtTS(t pgtype.Timestamptz) string {
	if !t.Valid {
		return ""
	}
	return t.Time.Format(time.RFC3339)
}
