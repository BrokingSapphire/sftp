// Package user holds request/response DTOs for user administration.
package user

// CreateRequest creates a new user.
type CreateRequest struct {
	Email        string  `json:"email"         validate:"required,email"`
	Username     string  `json:"username"      validate:"required,min=3,max=64"`
	Password     string  `json:"password"      validate:"required,min=12"`
	FullName     string  `json:"full_name"     validate:"required"`
	RoleSlug     string  `json:"role"          validate:"required"`
	EmployeeID   *string `json:"employee_id"`
	Phone        *string `json:"phone"`
	StorageQuota int64   `json:"storage_quota"` // bytes; 0 = unlimited
}

// UpdateRequest updates mutable profile fields.
type UpdateRequest struct {
	FullName *string `json:"full_name"`
	Phone    *string `json:"phone"`
}

// SetRoleRequest changes a user's role.
type SetRoleRequest struct {
	RoleSlug string `json:"role" validate:"required"`
}

// SetQuotaRequest changes a user's storage quota.
type SetQuotaRequest struct {
	StorageQuota int64 `json:"storage_quota" validate:"min=0"`
}

// SetActiveRequest enables/disables a user.
type SetActiveRequest struct {
	IsActive bool `json:"is_active"`
}

// DeleteRequest carries the mandatory transfer target for a user's files.
type DeleteRequest struct {
	TransferTo string `json:"transfer_to" validate:"required,uuid"`
}

// ResetPasswordRequest sets a new password (admin action).
type ResetPasswordRequest struct {
	NewPassword string `json:"new_password" validate:"required,min=12"`
}

// Response is the public projection of a user for admin views.
type Response struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	Username     string `json:"username"`
	FullName     string `json:"full_name"`
	Role         string `json:"role"`
	EmployeeID   string `json:"employee_id,omitempty"`
	Phone        string `json:"phone,omitempty"`
	StorageUsed  int64  `json:"storage_used"`
	StorageQuota int64  `json:"storage_quota"`
	IsActive     bool   `json:"is_active"`
	IsLocked     bool   `json:"is_locked"`
	HasAvatar    bool   `json:"has_avatar"`
	LastLoginAt  string `json:"last_login_at,omitempty"`
	CreatedAt    string `json:"created_at"`
}

// ListMeta is pagination metadata for list responses.
type ListMeta struct {
	Total  int64 `json:"total"`
	Limit  int   `json:"limit"`
	Offset int   `json:"offset"`
}
