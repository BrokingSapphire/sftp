// Package apikey holds request/response DTOs for API-key management.
package apikey

// CreateRequest creates a new API key.
type CreateRequest struct {
	Name          string   `json:"name"           validate:"required,min=1,max=128"`
	Scopes        []string `json:"scopes"`
	ExpiresInDays *int     `json:"expires_in_days"`
}

// CreateResponse returns the plaintext key exactly once.
type CreateResponse struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Prefix    string   `json:"prefix"`
	Key       string   `json:"key"` // shown only on creation
	Scopes    []string `json:"scopes"`
	ExpiresAt string   `json:"expires_at,omitempty"`
	CreatedAt string   `json:"created_at"`
}

// Response is the non-secret projection of an API key.
type Response struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Prefix     string   `json:"prefix"`
	Scopes     []string `json:"scopes"`
	LastUsedAt string   `json:"last_used_at,omitempty"`
	ExpiresAt  string   `json:"expires_at,omitempty"`
	CreatedAt  string   `json:"created_at"`
}
