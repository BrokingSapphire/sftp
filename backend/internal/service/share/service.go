// Package share implements share-link creation and public access.
package share

import (
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

// Create makes a share link for a file the caller owns.
func (s *Service) Create(ctx context.Context, owner uuid.UUID, req models.CreateRequest) (*models.CreateResponse, error) {
	fileID, err := uuid.Parse(req.FileID)
	if err != nil {
		return nil, apperrors.ErrInvalidRequest
	}
	file, err := s.q.GetFileByID(ctx, fileID)
	if err != nil {
		return nil, apperrors.ErrFileNotFound
	}
	if file.OwnerID != owner {
		return nil, apperrors.ErrForbidden
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
		Token: token, OwnerID: owner, FileID: &fileID, Permission: permission,
		PasswordHash: pwHash, DownloadLimit: limit, ExpiresAt: expires,
	})
	if err != nil {
		return nil, err
	}

	shareURL := s.baseURL + "/share/" + sh.Token
	resp := &models.CreateResponse{
		ID: sh.ID.String(), Token: sh.Token, URL: shareURL,
		HasPassword: pwHash != nil, DownloadLimit: limit, CreatedAt: fmtTS(sh.CreatedAt),
	}
	if sh.ExpiresAt.Valid {
		resp.ExpiresAt = sh.ExpiresAt.Time.Format(time.RFC3339)
	}

	// Optionally email the recipient. Flag external (outside-org) recipients.
	if email := strings.TrimSpace(req.RecipientEmail); email != "" {
		resp.External = s.isExternal(email)
		if s.mail != nil && s.mail.Enabled() {
			if err := s.mail.Send(email, "A file has been shared with you", shareEmailHTML(file.Name, shareURL, pwHash != nil)); err != nil {
				s.log.Error("share email failed", "to", email, "err", err)
			} else {
				resp.Emailed = true
			}
		}
		s.log.Info("share created", "file", file.Name, "recipient", email, "external", resp.External)
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

func shareEmailHTML(fileName, url string, hasPassword bool) string {
	pw := ""
	if hasPassword {
		pw = `<p style="color:#b45309;font-size:13px">This link is password-protected — the sender will share the password separately.</p>`
	}
	return `<div style="font-family:-apple-system,Segoe UI,Roboto,sans-serif;max-width:480px;margin:0 auto">
  <h2 style="color:#064D51">A file has been shared with you</h2>
  <p style="color:#333">You have been given access to <strong>` + fileName + `</strong> on Sapphire SFTP.</p>
  <p><a href="` + url + `" style="display:inline-block;background:#064D51;color:#fff;text-decoration:none;padding:10px 20px;border-radius:8px">Open the file</a></p>
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
			ID: sh.ID.String(), Token: sh.Token, Permission: sh.Permission,
			HasPassword: sh.PasswordHash != nil, DownloadLimit: sh.DownloadLimit,
			DownloadCount: sh.DownloadCount, IsActive: sh.IsActive, CreatedAt: fmtTS(sh.CreatedAt),
		}
		if sh.FileID != nil {
			r.FileID = sh.FileID.String()
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
	sh, file, err := s.resolve(ctx, token)
	if err != nil {
		return nil, err
	}
	return &models.PublicInfo{
		Token: sh.Token, FileName: file.Name, SizeBytes: file.SizeBytes,
		MimeType: file.MimeType, HasPassword: sh.PasswordHash != nil, Permission: sh.Permission,
	}, nil
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

// Access validates a share (and its password) and opens the file for download.
func (s *Service) Access(ctx context.Context, token, password string) (*OpenHandle, error) {
	sh, file, err := s.resolve(ctx, token)
	if err != nil {
		return nil, err
	}
	if sh.PasswordHash != nil {
		if password == "" {
			return nil, apperrors.ErrSharePasswordNeeded
		}
		ok, err := argon2.Verify(password, *sh.PasswordHash)
		if err != nil || !ok {
			return nil, apperrors.ErrSharePasswordNeeded
		}
	}
	if sh.DownloadLimit != nil && sh.DownloadCount >= *sh.DownloadLimit {
		return nil, apperrors.ErrShareLimitReached
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

func (s *Service) resolve(ctx context.Context, token string) (sftpdb.Share, sftpdb.File, error) {
	sh, err := s.q.GetShareByToken(ctx, token)
	if err != nil {
		return sftpdb.Share{}, sftpdb.File{}, apperrors.ErrShareNotFound
	}
	if sh.ExpiresAt.Valid && sh.ExpiresAt.Time.Before(time.Now()) {
		return sftpdb.Share{}, sftpdb.File{}, apperrors.ErrShareExpired
	}
	if sh.FileID == nil {
		return sftpdb.Share{}, sftpdb.File{}, apperrors.ErrShareNotFound
	}
	file, err := s.q.GetFileByID(ctx, *sh.FileID)
	if err != nil {
		return sftpdb.Share{}, sftpdb.File{}, apperrors.ErrFileNotFound
	}
	return sh, file, nil
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
