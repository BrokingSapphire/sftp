// Package share holds request/response DTOs for share links.
package share

// CreateRequest creates a share link for a file or a folder. Exactly one of
// FileID / FolderID must be provided (validated in the service).
type CreateRequest struct {
	FileID         string `json:"file_id"        validate:"omitempty,uuid"`
	FolderID       string `json:"folder_id"      validate:"omitempty,uuid"`
	Password       string `json:"password"`       // optional; empty = public
	DownloadLimit  *int   `json:"download_limit"` // optional; nil = unlimited
	ExpiresInDays  *int   `json:"expires_in_days"`
	Permission     string `json:"permission"`      // read (default)
	RecipientEmail string `json:"recipient_email"` // optional; emails the link
}

// CreateResponse returns the created share.
type CreateResponse struct {
	ID            string `json:"id"`
	Token         string `json:"token"`
	URL           string `json:"url"`
	Kind          string `json:"kind"` // "file" | "folder"
	HasPassword   bool   `json:"has_password"`
	DownloadLimit *int32 `json:"download_limit,omitempty"`
	ExpiresAt     string `json:"expires_at,omitempty"`
	CreatedAt     string `json:"created_at"`
	Emailed       bool   `json:"emailed"`  // an email was sent to the recipient
	External      bool   `json:"external"` // recipient is outside the organisation
}

// Response is the projection of a share for the owner's list.
type Response struct {
	ID            string `json:"id"`
	Token         string `json:"token"`
	Kind          string `json:"kind"` // "file" | "folder"
	FileID        string `json:"file_id,omitempty"`
	FolderID      string `json:"folder_id,omitempty"`
	Permission    string `json:"permission"`
	HasPassword   bool   `json:"has_password"`
	DownloadLimit *int32 `json:"download_limit,omitempty"`
	DownloadCount int32  `json:"download_count"`
	IsActive      bool   `json:"is_active"`
	ExpiresAt     string `json:"expires_at,omitempty"`
	CreatedAt     string `json:"created_at"`
}

// PublicInfo is the safe, unauthenticated view of a share. For a folder share,
// FileName holds the folder name, ItemCount is the number of files inside
// (recursive), and SizeBytes/MimeType are zero-valued.
type PublicInfo struct {
	Token       string `json:"token"`
	Kind        string `json:"kind"` // "file" | "folder"
	FileName    string `json:"file_name"`
	SizeBytes   int64  `json:"size_bytes"`
	MimeType    string `json:"mime_type"`
	ItemCount   int    `json:"item_count,omitempty"` // folder shares: number of files inside
	HasPassword bool   `json:"has_password"`
	Permission  string `json:"permission"`
}
