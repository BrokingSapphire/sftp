// Package auth holds request/response DTOs for the authentication API.
package auth

// LoginRequest is the credentials payload.
type LoginRequest struct {
	Identifier string `json:"identifier" validate:"required"` // email or username
	Password   string `json:"password"   validate:"required"`
	RememberMe bool   `json:"remember_me"`
}

// TokenPair is issued on successful login/refresh.
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int64     `json:"expires_in"` // access token TTL, seconds
	User         *UserInfo `json:"user,omitempty"`
}

// RefreshRequest carries the opaque refresh token.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// LogoutRequest carries the refresh token to revoke.
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// ChangePasswordRequest changes the current user's password.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password"     validate:"required,min=12"`
}

// UserInfo is the public projection of a user.
type UserInfo struct {
	ID          string   `json:"id"`
	Email       string   `json:"email"`
	Username    string   `json:"username"`
	FullName    string   `json:"full_name"`
	Role        string   `json:"role"`
	Permissions []string `json:"permissions,omitempty"`
	StorageUsed int64    `json:"storage_used"`
	StorageQuota int64   `json:"storage_quota"`
	MustChangePw bool    `json:"must_change_pw"`
	HasAvatar   bool     `json:"has_avatar"`
}
