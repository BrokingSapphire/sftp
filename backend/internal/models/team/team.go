// Package team holds request/response DTOs for Team Spaces.
package team

// CreateRequest creates or updates a team.
type CreateRequest struct {
	Name         string `json:"name" validate:"required"`
	Description  string `json:"description"`
	StorageQuota int64  `json:"storage_quota"` // 0 = unlimited
}

// AddMemberRequest adds a member by email with a role.
type AddMemberRequest struct {
	Email string `json:"email" validate:"required,email"`
	Role  string `json:"role"` // admin | member | viewer
}

// TeamResponse is the public projection of a team.
type TeamResponse struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	Description  string `json:"description,omitempty"`
	StorageQuota int64  `json:"storage_quota"`
	StorageUsed  int64  `json:"storage_used"`
	MemberRole   string `json:"member_role,omitempty"`
	MemberCount  int64  `json:"member_count,omitempty"`
	CreatedAt    string `json:"created_at,omitempty"`
}

// MemberResponse is a team member.
type MemberResponse struct {
	UserID    string `json:"user_id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	HasAvatar bool   `json:"has_avatar"`
}
