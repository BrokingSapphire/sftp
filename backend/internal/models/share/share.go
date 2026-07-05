// Package share holds request/response DTOs for share links.
package share

// CreateRequest creates a share link for a file.
type CreateRequest struct {
	FileID        string  `json:"file_id"        validate:"required,uuid"`
	Password      string  `json:"password"`       // optional; empty = public
	DownloadLimit *int    `json:"download_limit"` // optional; nil = unlimited
	ExpiresInDays *int    `json:"expires_in_days"`
	Permission    string  `json:"permission"`     // read (default)
}

// CreateResponse returns the created share.
type CreateResponse struct {
	ID            string `json:"id"`
	Token         string `json:"token"`
	URL           string `json:"url"`
	HasPassword   bool   `json:"has_password"`
	DownloadLimit *int32 `json:"download_limit,omitempty"`
	ExpiresAt     string `json:"expires_at,omitempty"`
	CreatedAt     string `json:"created_at"`
}

// Response is the projection of a share for the owner's list.
type Response struct {
	ID            string `json:"id"`
	Token         string `json:"token"`
	FileID        string `json:"file_id,omitempty"`
	Permission    string `json:"permission"`
	HasPassword   bool   `json:"has_password"`
	DownloadLimit *int32 `json:"download_limit,omitempty"`
	DownloadCount int32  `json:"download_count"`
	IsActive      bool   `json:"is_active"`
	ExpiresAt     string `json:"expires_at,omitempty"`
	CreatedAt     string `json:"created_at"`
}

// PublicInfo is the safe, unauthenticated view of a share.
type PublicInfo struct {
	Token       string `json:"token"`
	FileName    string `json:"file_name"`
	SizeBytes   int64  `json:"size_bytes"`
	MimeType    string `json:"mime_type"`
	HasPassword bool   `json:"has_password"`
	Permission  string `json:"permission"`
}
