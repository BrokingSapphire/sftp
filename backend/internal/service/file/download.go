package file

import (
	"context"
	"io"
	"net/netip"
	"time"

	"github.com/google/uuid"

	"sapphirebroking.com/sftp_service/internal/db/sftpdb"
)

// DownloadHandle carries an open file plus the metadata a handler needs to
// serve it (with range support via http.ServeContent).
type DownloadHandle struct {
	File     io.ReadSeekCloser
	Name     string
	MimeType string
	Size     int64
	ModTime  time.Time
}

// DownloadMeta carries client info for the download audit record.
type DownloadMeta struct {
	IP        string
	UserAgent string
}

// OpenForDownload opens a file for streaming and records the download.
// The caller must Close the returned handle's File.
func (s *Service) OpenForDownload(ctx context.Context, owner, id uuid.UUID, meta DownloadMeta) (*DownloadHandle, error) {
	f, err := s.ownedFile(ctx, owner, id)
	if err != nil {
		return nil, err
	}
	fh, err := s.store.Open(f.StorageKey)
	if err != nil {
		return nil, err
	}
	modTime := time.Now()
	if f.UpdatedAt.Valid {
		modTime = f.UpdatedAt.Time
	}

	// Best-effort audit; failures must not block the download.
	_ = s.q.IncrementDownloadCount(ctx, id)
	ua := meta.UserAgent
	ownerCopy := owner
	fileCopy := id
	_ = s.q.InsertDownload(ctx, sftpdb.InsertDownloadParams{
		FileID:    &fileCopy,
		UserID:    &ownerCopy,
		BytesSent: f.SizeBytes,
		IpAddress: parseAddr(meta.IP),
		UserAgent: &ua,
	})

	return &DownloadHandle{
		File: fh, Name: f.Name, MimeType: f.MimeType, Size: f.SizeBytes, ModTime: modTime,
	}, nil
}

func parseAddr(ip string) *netip.Addr {
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return nil
	}
	return &addr
}
