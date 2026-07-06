package file

import (
	"context"

	"github.com/google/uuid"

	"sapphirebroking.com/sftp_service/internal/db/sftpdb"
	"sapphirebroking.com/sftp_service/pkg/dlp"
	"sapphirebroking.com/sftp_service/pkg/textextract"
)

// IndexText extracts searchable plain text from a file's content and stores it
// for full-text search. Unsupported types get an empty row so they are not
// re-scanned. Reads decrypt transparently via the storage engine.
func (s *Service) IndexText(ctx context.Context, fileID uuid.UUID) error {
	f, err := s.q.GetFileByID(ctx, fileID)
	if err != nil {
		return err
	}

	var text string
	if textextract.Supported(f.Extension, f.MimeType) {
		rc, err := s.store.Open(f.StorageKey)
		if err != nil {
			return err
		}
		text, err = textextract.Extract(f.Extension, f.MimeType, rc)
		rc.Close()
		if err != nil {
			return err
		}
	}

	if err := s.q.UpsertFileText(ctx, sftpdb.UpsertFileTextParams{
		FileID:  fileID,
		Content: text,
		Ocr:     false,
		Bytes:   int64(len(text)),
	}); err != nil {
		return err
	}

	// Classify content for DLP (PII types + sensitivity level).
	res := dlp.Scan(text)
	return s.q.SetFileClassification(ctx, sftpdb.SetFileClassificationParams{
		ID: fileID, Sensitivity: res.Sensitivity, PiiTypes: res.PIITypes,
	})
}

// indexAsync indexes a file in the background so uploads stay fast. Failures are
// logged; the backfill sweeper retries anything left unindexed.
func (s *Service) indexAsync(fileID uuid.UUID) {
	go func() {
		if err := s.IndexText(context.Background(), fileID); err != nil {
			s.log.Warn("text index failed", "file", fileID, "err", err)
		}
	}()
}

// BackfillTextIndex indexes up to `limit` files that have no extracted text yet
// (called periodically by the cleanup worker). Returns the number indexed.
func (s *Service) BackfillTextIndex(ctx context.Context, limit int) (int, error) {
	rows, err := s.q.ListFilesMissingText(ctx, int32(limit))
	if err != nil {
		return 0, err
	}
	n := 0
	for _, r := range rows {
		if err := s.IndexText(ctx, r.ID); err != nil {
			s.log.Warn("backfill index failed", "file", r.ID, "err", err)
			continue
		}
		n++
	}
	return n, nil
}
