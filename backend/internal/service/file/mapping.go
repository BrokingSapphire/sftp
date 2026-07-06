package file

import (
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"sapphirebroking.com/sftp_service/internal/db/sftpdb"
	models "sapphirebroking.com/sftp_service/internal/models/file"
)

func fmtTS(t pgtype.Timestamptz) string {
	if !t.Valid {
		return ""
	}
	return t.Time.Format(time.RFC3339)
}

func uuidPtrStr(id *uuid.UUID) *string {
	if id == nil {
		return nil
	}
	s := id.String()
	return &s
}

func toFolderResponse(f sftpdb.Folder) *models.FolderResponse {
	return &models.FolderResponse{
		ID:        f.ID.String(),
		Name:      f.Name,
		ParentID:  uuidPtrStr(f.ParentID),
		Path:      f.Path,
		Depth:     f.Depth,
		SizeBytes: f.SizeBytes,
		Color:     f.Color,
		IsStarred: f.IsStarred,
		IsPinned:  f.IsPinned,
		CreatedAt: fmtTS(f.CreatedAt),
		UpdatedAt: fmtTS(f.UpdatedAt),
	}
}

func toFileResponse(f sftpdb.File) *models.FileResponse {
	r := &models.FileResponse{
		ID:            f.ID.String(),
		Name:          f.Name,
		Extension:     f.Extension,
		MimeType:      f.MimeType,
		SizeBytes:     f.SizeBytes,
		FolderID:      uuidPtrStr(f.FolderID),
		IsStarred:     f.IsStarred,
		VersionNo:     f.VersionNo,
		DownloadCount: f.DownloadCount,
		CreatedAt:     fmtTS(f.CreatedAt),
		UpdatedAt:     fmtTS(f.UpdatedAt),
		DeletedAt:     fmtTS(f.DeletedAt),
	}
	if f.ChecksumSha256 != nil {
		r.Checksum = *f.ChecksumSha256
	}
	r.TransferPending = f.TransferPending
	if f.TransferDeadline.Valid {
		r.TransferDeadline = f.TransferDeadline.Time.Format(time.RFC3339)
	}
	r.LegalHold = f.LegalHold
	if f.RetainUntil.Valid {
		r.RetainUntil = f.RetainUntil.Time.Format(time.RFC3339)
	}
	return r
}
