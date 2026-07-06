// Package file holds request/response DTOs for file and folder operations.
package file

// ── Folders ───────────────────────────────────────────────

// CreateFolderRequest creates a folder (optionally nested).
type CreateFolderRequest struct {
	Name     string  `json:"name"      validate:"required"`
	ParentID *string `json:"parent_id"` // nil = root
}

// RenameRequest renames a file or folder.
type RenameRequest struct {
	Name string `json:"name" validate:"required"`
}

// MoveRequest moves a file or folder to a new parent/folder.
type MoveRequest struct {
	TargetID *string `json:"target_id"` // nil = root
}

// StarRequest toggles the starred flag.
type StarRequest struct {
	Starred bool `json:"starred"`
}

// ColorRequest sets a folder's colour ("" clears it).
type ColorRequest struct {
	Color string `json:"color"`
}

// FolderResponse is the public projection of a folder.
type FolderResponse struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	ParentID  *string `json:"parent_id,omitempty"`
	Path      string  `json:"path"`
	Depth     int32   `json:"depth"`
	SizeBytes int64   `json:"size_bytes"`
	Color     string  `json:"color,omitempty"`
	IsStarred bool    `json:"is_starred"`
	IsPinned  bool    `json:"is_pinned"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

// ── Files ─────────────────────────────────────────────────

// FileResponse is the public projection of a file.
type FileResponse struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Extension     string  `json:"extension"`
	MimeType      string  `json:"mime_type"`
	SizeBytes     int64   `json:"size_bytes"`
	Checksum      string  `json:"checksum_sha256,omitempty"`
	FolderID      *string `json:"folder_id,omitempty"`
	IsStarred     bool    `json:"is_starred"`
	VersionNo     int32   `json:"version_no"`
	DownloadCount int64   `json:"download_count"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
	DeletedAt     string  `json:"deleted_at,omitempty"`
	// Set when this file was inherited from a deleted user and awaits action.
	TransferPending  bool   `json:"transfer_pending,omitempty"`
	TransferDeadline string `json:"transfer_deadline,omitempty"`
}

// CommonFileResponse is a file in the organisation-wide Common area.
type CommonFileResponse struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Extension    string `json:"extension"`
	MimeType     string `json:"mime_type"`
	SizeBytes    int64  `json:"size_bytes"`
	IsStarred    bool   `json:"is_starred"`
	UploaderID   string `json:"uploader_id"`
	UploaderName string `json:"uploader_name"`
	CanDelete    bool   `json:"can_delete"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
	// Mirror of FileResponse fields the viewer needs.
	FolderID      *string `json:"folder_id,omitempty"`
	VersionNo     int32   `json:"version_no"`
	DownloadCount int64   `json:"download_count"`
	Checksum      string  `json:"checksum_sha256,omitempty"`
}

// ListingResponse is a combined folder + file listing.
type ListingResponse struct {
	Folders []FolderResponse `json:"folders"`
	Files   []FileResponse   `json:"files"`
}

// ListMeta is pagination metadata.
type ListMeta struct {
	Total  int64 `json:"total"`
	Limit  int   `json:"limit"`
	Offset int   `json:"offset"`
}

// ── Uploads (chunked/resumable) ───────────────────────────

// InitUploadRequest starts a resumable upload session.
type InitUploadRequest struct {
	Filename  string  `json:"filename"   validate:"required"`
	TotalSize int64   `json:"total_size" validate:"required,min=1"`
	ChunkSize int64   `json:"chunk_size" validate:"required,min=1"`
	FolderID  *string `json:"folder_id"`
	Checksum  *string `json:"checksum_sha256"`
}

// InitUploadResponse returns the session id and any already-received chunks.
type InitUploadResponse struct {
	UploadID       string `json:"upload_id"`
	TotalChunks    int    `json:"total_chunks"`
	ChunkSize      int64  `json:"chunk_size"`
	ReceivedChunks []int  `json:"received_chunks"`
}

// UploadStatusResponse reports progress for resume.
type UploadStatusResponse struct {
	UploadID       string `json:"upload_id"`
	Status         string `json:"status"`
	TotalChunks    int    `json:"total_chunks"`
	UploadedChunks int    `json:"uploaded_chunks"`
	ReceivedBytes  int64  `json:"received_bytes"`
	ReceivedChunks []int  `json:"received_chunks"`
}
