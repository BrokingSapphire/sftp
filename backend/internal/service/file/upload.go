package file

import (
	"context"
	"io"
	"mime"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"sapphirebroking.com/sftp_service/internal/apperrors"
	"sapphirebroking.com/sftp_service/internal/db/sftpdb"
	models "sapphirebroking.com/sftp_service/internal/models/file"
	"sapphirebroking.com/sftp_service/internal/utils"
)

const uploadTTL = 24 * time.Hour

// InitUpload starts a resumable upload session, enforcing size and quota.
func (s *Service) InitUpload(ctx context.Context, owner uuid.UUID, req models.InitUploadRequest) (*models.InitUploadResponse, error) {
	name, err := utils.SanitizeName(req.Filename)
	if err != nil {
		return nil, err
	}
	if s.maxUploadSize > 0 && req.TotalSize > s.maxUploadSize {
		return nil, apperrors.ErrPayloadTooLarge
	}
	if err := s.checkQuota(ctx, owner, req.TotalSize); err != nil {
		return nil, err
	}

	folderID, err := s.resolveFolder(ctx, owner, req.FolderID)
	if err != nil {
		return nil, err
	}

	chunkSize := req.ChunkSize
	if chunkSize <= 0 {
		chunkSize = s.chunkSize
	}
	totalChunks := int((req.TotalSize + chunkSize - 1) / chunkSize)
	if totalChunks < 1 {
		totalChunks = 1
	}

	up, err := s.q.CreateUpload(ctx, sftpdb.CreateUploadParams{
		UserID:      owner,
		FolderID:    folderID,
		Filename:    name,
		TotalSize:   req.TotalSize,
		ChunkSize:   chunkSize,
		TotalChunks: int32(totalChunks),
		TempKey:     "chunked",
		Checksum:    req.Checksum,
		ExpiresAt:   pgtype.Timestamptz{Time: time.Now().Add(uploadTTL), Valid: true},
	})
	if err != nil {
		return nil, err
	}
	return &models.InitUploadResponse{
		UploadID:       up.ID.String(),
		TotalChunks:    totalChunks,
		ChunkSize:      chunkSize,
		ReceivedChunks: []int{},
	}, nil
}

// PutChunk stores a single chunk and updates progress. Safe to retry.
func (s *Service) PutChunk(ctx context.Context, owner, uploadID uuid.UUID, index int, r io.Reader) (*models.UploadStatusResponse, error) {
	up, err := s.activeUpload(ctx, owner, uploadID)
	if err != nil {
		return nil, err
	}
	if index < 0 || index >= int(up.TotalChunks) {
		return nil, apperrors.ErrInvalidRequest
	}

	size, err := s.store.WriteChunk(uploadID.String(), index, r)
	if err != nil {
		return nil, err
	}
	if err := s.q.RecordChunk(ctx, sftpdb.RecordChunkParams{
		UploadID: uploadID, ChunkIndex: int32(index), SizeBytes: size,
	}); err != nil {
		return nil, err
	}

	received, err := s.q.ListReceivedChunks(ctx, uploadID)
	if err != nil {
		return nil, err
	}
	receivedBytes := int64(len(received)) * up.ChunkSize
	if receivedBytes > up.TotalSize {
		receivedBytes = up.TotalSize
	}
	if err := s.q.UpdateUploadProgress(ctx, sftpdb.UpdateUploadProgressParams{
		ID: uploadID, UploadedChunks: int32(len(received)), ReceivedBytes: receivedBytes,
	}); err != nil {
		return nil, err
	}

	return &models.UploadStatusResponse{
		UploadID: uploadID.String(), Status: "in_progress",
		TotalChunks: int(up.TotalChunks), UploadedChunks: len(received),
		ReceivedBytes: receivedBytes, ReceivedChunks: toIntSlice(received),
	}, nil
}

// UploadStatus reports progress for resume.
func (s *Service) UploadStatus(ctx context.Context, owner, uploadID uuid.UUID) (*models.UploadStatusResponse, error) {
	up, err := s.q.GetUploadForUser(ctx, sftpdb.GetUploadForUserParams{ID: uploadID, UserID: owner})
	if err != nil {
		return nil, apperrors.ErrUploadNotFound
	}
	received, err := s.q.ListReceivedChunks(ctx, uploadID)
	if err != nil {
		return nil, err
	}
	return &models.UploadStatusResponse{
		UploadID: up.ID.String(), Status: up.Status,
		TotalChunks: int(up.TotalChunks), UploadedChunks: len(received),
		ReceivedBytes: up.ReceivedBytes, ReceivedChunks: toIntSlice(received),
	}, nil
}

// CompleteUpload assembles the chunks into a stored file.
func (s *Service) CompleteUpload(ctx context.Context, owner, uploadID uuid.UUID) (*models.FileResponse, error) {
	up, err := s.activeUpload(ctx, owner, uploadID)
	if err != nil {
		return nil, err
	}
	received, err := s.q.ListReceivedChunks(ctx, uploadID)
	if err != nil {
		return nil, err
	}
	if len(received) != int(up.TotalChunks) {
		return nil, apperrors.ErrUploadIncomplete
	}

	res, err := s.store.AssembleAndSave(uploadID.String(), int(up.TotalChunks))
	if err != nil {
		return nil, err
	}
	if up.ChecksumSha256 != nil && *up.ChecksumSha256 != "" && *up.ChecksumSha256 != res.Checksum {
		_ = s.store.Delete(res.Key)
		_ = s.q.SetUploadStatus(ctx, sftpdb.SetUploadStatusParams{ID: uploadID, Status: "failed"})
		return nil, apperrors.ErrChecksumMismatch
	}

	checksum := res.Checksum
	file, err := s.q.CreateFile(ctx, sftpdb.CreateFileParams{
		OwnerID: owner, FolderID: up.FolderID, Name: up.Filename,
		Extension: utils.FileExtension(up.Filename), MimeType: mimeByName(up.Filename),
		SizeBytes: res.Size, ChecksumSha256: &checksum, StorageKey: res.Key,
	})
	if err != nil {
		_ = s.store.Delete(res.Key)
		return nil, mapConflict(err)
	}
	if err := s.q.AddStorageUsed(ctx, sftpdb.AddStorageUsedParams{ID: owner, StorageUsed: res.Size}); err != nil {
		s.log.Error("increment storage used failed", "err", err)
	}
	if err := s.q.CompleteUpload(ctx, sftpdb.CompleteUploadParams{ID: uploadID, FileID: &file.ID}); err != nil {
		s.log.Error("mark upload complete failed", "err", err)
	}
	return toFileResponse(file), nil
}

// AbortUpload cancels a session and removes its chunks.
func (s *Service) AbortUpload(ctx context.Context, owner, uploadID uuid.UUID) error {
	if _, err := s.q.GetUploadForUser(ctx, sftpdb.GetUploadForUserParams{ID: uploadID, UserID: owner}); err != nil {
		return apperrors.ErrUploadNotFound
	}
	s.store.CleanupUpload(uploadID.String())
	return s.q.SetUploadStatus(ctx, sftpdb.SetUploadStatusParams{ID: uploadID, Status: "aborted"})
}

// SimpleUpload stores a small file in a single request.
func (s *Service) SimpleUpload(ctx context.Context, owner uuid.UUID, folderID *string, filename string, r io.Reader) (*models.FileResponse, error) {
	name, err := utils.SanitizeName(filename)
	if err != nil {
		return nil, err
	}
	folder, err := s.resolveFolder(ctx, owner, folderID)
	if err != nil {
		return nil, err
	}
	res, err := s.store.Save(r)
	if err != nil {
		return nil, err
	}
	if s.maxUploadSize > 0 && res.Size > s.maxUploadSize {
		_ = s.store.Delete(res.Key)
		return nil, apperrors.ErrPayloadTooLarge
	}
	if err := s.checkQuota(ctx, owner, res.Size); err != nil {
		_ = s.store.Delete(res.Key)
		return nil, err
	}

	checksum := res.Checksum
	file, err := s.q.CreateFile(ctx, sftpdb.CreateFileParams{
		OwnerID: owner, FolderID: folder, Name: name,
		Extension: utils.FileExtension(name), MimeType: mimeByName(name),
		SizeBytes: res.Size, ChecksumSha256: &checksum, StorageKey: res.Key,
	})
	if err != nil {
		_ = s.store.Delete(res.Key)
		return nil, mapConflict(err)
	}
	if err := s.q.AddStorageUsed(ctx, sftpdb.AddStorageUsedParams{ID: owner, StorageUsed: res.Size}); err != nil {
		s.log.Error("increment storage used failed", "err", err)
	}
	return toFileResponse(file), nil
}

// ── helpers ────────────────────────────────────────────────

func (s *Service) activeUpload(ctx context.Context, owner, uploadID uuid.UUID) (sftpdb.Upload, error) {
	up, err := s.q.GetUploadForUser(ctx, sftpdb.GetUploadForUserParams{ID: uploadID, UserID: owner})
	if err != nil {
		return sftpdb.Upload{}, apperrors.ErrUploadNotFound
	}
	if up.Status != "in_progress" {
		return sftpdb.Upload{}, apperrors.ErrUploadNotFound
	}
	if up.ExpiresAt.Valid && up.ExpiresAt.Time.Before(time.Now()) {
		return sftpdb.Upload{}, apperrors.ErrUploadExpired
	}
	return up, nil
}

func (s *Service) checkQuota(ctx context.Context, owner uuid.UUID, addBytes int64) error {
	user, err := s.q.GetUserByID(ctx, owner)
	if err != nil {
		return apperrors.ErrUserNotFound
	}
	if user.StorageQuota > 0 && user.StorageUsed+addBytes > user.StorageQuota {
		return apperrors.ErrQuotaExceeded
	}
	return nil
}

func (s *Service) resolveFolder(ctx context.Context, owner uuid.UUID, folderID *string) (*uuid.UUID, error) {
	if folderID == nil || *folderID == "" {
		return nil, nil
	}
	id, err := uuid.Parse(*folderID)
	if err != nil {
		return nil, apperrors.ErrInvalidRequest
	}
	if _, err := s.ownedFolder(ctx, owner, id); err != nil {
		return nil, err
	}
	return &id, nil
}

func mimeByName(name string) string {
	if t := mime.TypeByExtension("." + utils.FileExtension(name)); t != "" {
		return t
	}
	return "application/octet-stream"
}

func toIntSlice(vals []int32) []int {
	out := make([]int, len(vals))
	for i, v := range vals {
		out[i] = int(v)
	}
	return out
}
