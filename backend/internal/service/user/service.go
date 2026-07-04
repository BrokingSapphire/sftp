// Package user implements user administration (CRUD, roles, quotas).
package user

import (
	"context"
	"time"

	"github.com/google/uuid"

	"sapphirebroking.com/sftp_service/internal/apperrors"
	"sapphirebroking.com/sftp_service/internal/config"
	"sapphirebroking.com/sftp_service/internal/db/sftpdb"
	models "sapphirebroking.com/sftp_service/internal/models/user"
	"sapphirebroking.com/sftp_service/pkg/argon2"
	"sapphirebroking.com/sftp_service/pkg/logger"
)

// Deps are the user service dependencies.
type Deps struct {
	Queries  *sftpdb.Queries
	Security config.SecurityConfig
	Logger   logger.Logger
}

// Service provides user administration operations.
type Service struct {
	q     *sftpdb.Queries
	argon argon2.Params
	minPw int
	log   logger.Logger
}

// New builds the user Service.
func New(d Deps) *Service {
	return &Service{
		q: d.Queries,
		argon: argon2.Params{
			MemoryKiB: d.Security.ArgonMemoryKiB, Time: d.Security.ArgonTime,
			Threads: d.Security.ArgonThreads, KeyLen: d.Security.ArgonKeyLen, SaltLen: d.Security.ArgonSaltLen,
		},
		minPw: d.Security.PasswordMinLength,
		log:   d.Logger.Named("service.user"),
	}
}

// EnsureSuperAdmin creates the first super-admin when the database has no users.
func (s *Service) EnsureSuperAdmin(ctx context.Context, bc config.BootstrapConfig) error {
	count, err := s.q.CountUsers(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	if bc.AdminPassword == "" {
		s.log.Warn("no users exist and BOOTSTRAP_ADMIN_PASSWORD is unset; skipping admin creation")
		return nil
	}
	role, err := s.q.GetRoleBySlug(ctx, "super_admin")
	if err != nil {
		return err
	}
	hash, err := argon2.Hash(bc.AdminPassword, s.argon)
	if err != nil {
		return err
	}
	if _, err := s.q.CreateUser(ctx, sftpdb.CreateUserParams{
		Email: bc.AdminEmail, Username: bc.AdminUsername, PasswordHash: hash,
		FullName: "Super Admin", RoleID: role.ID, MustChangePw: true,
	}); err != nil {
		return err
	}
	s.log.Info("bootstrap super-admin created", "email", bc.AdminEmail, "username", bc.AdminUsername)
	return nil
}

// Create provisions a new user with the given role.
func (s *Service) Create(ctx context.Context, req models.CreateRequest, createdBy uuid.UUID) (*models.Response, error) {
	role, err := s.q.GetRoleBySlug(ctx, req.RoleSlug)
	if err != nil {
		return nil, apperrors.ErrRoleNotFound
	}
	if len(req.Password) < s.minPw {
		return nil, apperrors.ErrWeakPassword
	}
	hash, err := argon2.Hash(req.Password, s.argon)
	if err != nil {
		return nil, err
	}
	created := createdBy
	user, err := s.q.CreateUser(ctx, sftpdb.CreateUserParams{
		Email: req.Email, Username: req.Username, PasswordHash: hash,
		FullName: req.FullName, EmployeeID: req.EmployeeID, RoleID: role.ID,
		Phone: req.Phone, StorageQuota: req.StorageQuota, MustChangePw: true,
		CreatedBy: &created,
	})
	if err != nil {
		return nil, mapCreateErr(err)
	}
	return s.toResponse(ctx, user), nil
}

// List returns a page of users plus the total count.
func (s *Service) List(ctx context.Context, limit, offset int) ([]models.Response, int64, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.q.ListUsers(ctx, sftpdb.ListUsersParams{Limit: int32(limit), Offset: int32(offset)})
	if err != nil {
		return nil, 0, err
	}
	total, err := s.q.CountUsers(ctx)
	if err != nil {
		return nil, 0, err
	}
	out := make([]models.Response, 0, len(rows))
	cache := map[uuid.UUID]string{}
	for _, u := range rows {
		out = append(out, *s.toResponseCached(ctx, u, cache))
	}
	return out, total, nil
}

// Get returns a single user.
func (s *Service) Get(ctx context.Context, id uuid.UUID) (*models.Response, error) {
	user, err := s.q.GetUserByID(ctx, id)
	if err != nil {
		return nil, apperrors.ErrUserNotFound
	}
	return s.toResponse(ctx, user), nil
}

// Update changes mutable profile fields.
func (s *Service) Update(ctx context.Context, id uuid.UUID, req models.UpdateRequest) (*models.Response, error) {
	user, err := s.q.GetUserByID(ctx, id)
	if err != nil {
		return nil, apperrors.ErrUserNotFound
	}
	full := user.FullName
	if req.FullName != nil {
		full = *req.FullName
	}
	phone := user.Phone
	if req.Phone != nil {
		phone = req.Phone
	}
	updated, err := s.q.UpdateUserProfile(ctx, sftpdb.UpdateUserProfileParams{
		ID: id, FullName: full, Phone: phone, DepartmentID: user.DepartmentID, AvatarPath: user.AvatarPath,
	})
	if err != nil {
		return nil, err
	}
	return s.toResponse(ctx, updated), nil
}

// SetRole reassigns a user's role.
func (s *Service) SetRole(ctx context.Context, id uuid.UUID, roleSlug string) error {
	role, err := s.q.GetRoleBySlug(ctx, roleSlug)
	if err != nil {
		return apperrors.ErrRoleNotFound
	}
	return s.q.UpdateUserRole(ctx, sftpdb.UpdateUserRoleParams{ID: id, RoleID: role.ID})
}

// SetQuota changes a user's storage quota.
func (s *Service) SetQuota(ctx context.Context, id uuid.UUID, quota int64) error {
	return s.q.UpdateUserQuota(ctx, sftpdb.UpdateUserQuotaParams{ID: id, StorageQuota: quota})
}

// SetActive enables or disables a user.
func (s *Service) SetActive(ctx context.Context, id uuid.UUID, active bool) error {
	if err := s.q.SetUserActive(ctx, sftpdb.SetUserActiveParams{ID: id, IsActive: active}); err != nil {
		return err
	}
	if !active {
		return s.q.RevokeAllUserSessions(ctx, id)
	}
	return nil
}

// ResetPassword sets a new password (admin) and revokes sessions.
func (s *Service) ResetPassword(ctx context.Context, id uuid.UUID, newPassword string) error {
	if len(newPassword) < s.minPw {
		return apperrors.ErrWeakPassword
	}
	hash, err := argon2.Hash(newPassword, s.argon)
	if err != nil {
		return err
	}
	if err := s.q.SetUserPassword(ctx, sftpdb.SetUserPasswordParams{ID: id, PasswordHash: hash}); err != nil {
		return err
	}
	return s.q.RevokeAllUserSessions(ctx, id)
}

// Delete soft-deletes a user and revokes their sessions.
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.q.SoftDeleteUser(ctx, id); err != nil {
		return err
	}
	return s.q.RevokeAllUserSessions(ctx, id)
}

// RoleInfo is a role with its permission slugs.
type RoleInfo struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Slug        string   `json:"slug"`
	Description string   `json:"description"`
	IsSystem    bool     `json:"is_system"`
	Priority    int32    `json:"priority"`
	Permissions []string `json:"permissions"`
}

// ListRoles returns all roles with their permissions.
func (s *Service) ListRoles(ctx context.Context) ([]RoleInfo, error) {
	roles, err := s.q.ListRoles(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]RoleInfo, 0, len(roles))
	for _, r := range roles {
		perms, err := s.q.GetPermissionsForRole(ctx, r.ID)
		if err != nil {
			perms = nil
		}
		out = append(out, RoleInfo{
			ID: r.ID.String(), Name: r.Name, Slug: r.Slug, Description: r.Description,
			IsSystem: r.IsSystem, Priority: r.Priority, Permissions: perms,
		})
	}
	return out, nil
}

// ListPermissions returns the full permission catalogue.
func (s *Service) ListPermissions(ctx context.Context) ([]sftpdb.Permission, error) {
	return s.q.ListPermissions(ctx)
}

// ── mapping helpers ────────────────────────────────────────

func (s *Service) toResponse(ctx context.Context, u sftpdb.User) *models.Response {
	return s.toResponseCached(ctx, u, map[uuid.UUID]string{})
}

func (s *Service) toResponseCached(ctx context.Context, u sftpdb.User, cache map[uuid.UUID]string) *models.Response {
	slug, ok := cache[u.RoleID]
	if !ok {
		if role, err := s.q.GetRoleByID(ctx, u.RoleID); err == nil {
			slug = role.Slug
			cache[u.RoleID] = slug
		}
	}
	r := &models.Response{
		ID: u.ID.String(), Email: u.Email, Username: u.Username, FullName: u.FullName,
		Role: slug, StorageUsed: u.StorageUsed, StorageQuota: u.StorageQuota,
		IsActive: u.IsActive, IsLocked: u.IsLocked,
	}
	if u.EmployeeID != nil {
		r.EmployeeID = *u.EmployeeID
	}
	if u.Phone != nil {
		r.Phone = *u.Phone
	}
	if u.LastLoginAt.Valid {
		r.LastLoginAt = u.LastLoginAt.Time.Format(time.RFC3339)
	}
	if u.CreatedAt.Valid {
		r.CreatedAt = u.CreatedAt.Time.Format(time.RFC3339)
	}
	return r
}
