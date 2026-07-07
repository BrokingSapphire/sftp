// Package user implements user administration (CRUD, roles, quotas).
package user

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"sapphirebroking.com/sftp_service/internal/apperrors"
	"sapphirebroking.com/sftp_service/internal/config"
	"sapphirebroking.com/sftp_service/internal/db/sftpdb"
	models "sapphirebroking.com/sftp_service/internal/models/user"
	"sapphirebroking.com/sftp_service/internal/storage"
	"sapphirebroking.com/sftp_service/pkg/argon2"
	"sapphirebroking.com/sftp_service/pkg/logger"
)

// Deps are the user service dependencies.
type Deps struct {
	Queries  *sftpdb.Queries
	Storage  *storage.Engine
	Security config.SecurityConfig
	Logger   logger.Logger
}

// Service provides user administration operations.
type Service struct {
	q     *sftpdb.Queries
	store *storage.Engine
	argon argon2.Params
	minPw int
	log   logger.Logger
}

// New builds the user Service.
func New(d Deps) *Service {
	return &Service{
		q:     d.Queries,
		store: d.Storage,
		argon: argon2.Params{
			MemoryKiB: d.Security.ArgonMemoryKiB, Time: d.Security.ArgonTime,
			Threads: d.Security.ArgonThreads, KeyLen: d.Security.ArgonKeyLen, SaltLen: d.Security.ArgonSaltLen,
		},
		minPw: d.Security.PasswordMinLength,
		log:   d.Logger.Named("service.user"),
	}
}

// SetAvatar stores a user's profile photo and records its storage key.
func (s *Service) SetAvatar(ctx context.Context, userID uuid.UUID, r io.Reader) error {
	// Cap avatars at 5 MiB via a limited reader.
	res, err := s.store.Save(io.LimitReader(r, 5<<20))
	if err != nil {
		return err
	}
	key := res.Key
	return s.q.SetUserAvatar(ctx, sftpdb.SetUserAvatarParams{ID: userID, AvatarPath: &key})
}

// OpenAvatar opens a user's avatar for streaming.
func (s *Service) OpenAvatar(ctx context.Context, userID uuid.UUID) (io.ReadSeekCloser, error) {
	path, err := s.q.GetUserAvatar(ctx, userID)
	if err != nil || path == nil || *path == "" {
		return nil, apperrors.ErrNotFound
	}
	return s.store.Open(*path)
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
		// If the email/username belongs to a previously soft-deleted user,
		// reactivate that account instead of failing with "already exists".
		if errors.Is(mapCreateErr(err), apperrors.ErrAlreadyExists) {
			if del, derr := s.q.GetDeletedUserByEmailOrUsername(ctx, sftpdb.GetDeletedUserByEmailOrUsernameParams{
				Email: req.Email, Username: req.Username,
			}); derr == nil {
				if u, rerr := s.q.ReactivateUser(ctx, sftpdb.ReactivateUserParams{
					ID: del.ID, Email: req.Email, Username: req.Username, PasswordHash: hash,
					FullName: req.FullName, RoleID: role.ID, StorageQuota: req.StorageQuota,
					EmployeeID: req.EmployeeID, Phone: req.Phone,
				}); rerr == nil {
					return s.toResponse(ctx, u), nil
				}
			}
		}
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
	// Force the user to set their own password on next login.
	if err := s.q.ResetUserPassword(ctx, sftpdb.ResetUserPasswordParams{ID: id, PasswordHash: hash}); err != nil {
		return err
	}
	return s.q.RevokeAllUserSessions(ctx, id)
}

// transferWindow is how long the heir has to act on inherited files.
const transferWindow = 30 * 24 * time.Hour

// DeleteWithTransfer soft-deletes a user after mandatorily transferring all of
// their files and folders to another (active) user, who must then keep or
// delete them within the transfer window. Nothing is ever auto-deleted.
func (s *Service) DeleteWithTransfer(ctx context.Context, id, transferTo uuid.UUID) error {
	if id == transferTo {
		return apperrors.ErrInvalidRequest
	}
	if _, err := s.q.GetUserByID(ctx, id); err != nil {
		return apperrors.ErrUserNotFound
	}
	heir, err := s.q.GetUserByID(ctx, transferTo)
	if err != nil {
		return apperrors.ErrUserNotFound
	}
	if !heir.IsActive {
		return apperrors.ErrForbidden
	}

	sum, err := s.q.SumFileSizesByOwner(ctx, id)
	if err != nil {
		return err
	}
	deadline := pgtype.Timestamptz{Time: time.Now().Add(transferWindow), Valid: true}

	if err := s.q.ReassignUserFiles(ctx, sftpdb.ReassignUserFilesParams{ToUser: transferTo, Deadline: deadline, FromUser: &id}); err != nil {
		return err
	}
	if err := s.q.ReassignUserFolders(ctx, sftpdb.ReassignUserFoldersParams{ToUser: transferTo, FromUser: id}); err != nil {
		return err
	}
	// Move the storage accounting from the deleted user to the heir.
	_ = s.q.AddStorageUsed(ctx, sftpdb.AddStorageUsedParams{ID: transferTo, StorageUsed: sum})
	_ = s.q.AddStorageUsed(ctx, sftpdb.AddStorageUsedParams{ID: id, StorageUsed: -sum})

	// Notify the heir.
	_ = s.q.CreateNotification(ctx, sftpdb.CreateNotificationParams{
		UserID:   transferTo,
		Type:     "inherited_files",
		Title:    "Files transferred to you",
		Body:     "Files from a removed account are now assigned to you. Please review them (keep or delete) within 30 days, or your account may be disabled.",
		Link:     strPtr("/inherited"),
		Metadata: []byte(`{"kind":"inherited_files"}`),
	})

	if err := s.q.SoftDeleteUser(ctx, id); err != nil {
		return err
	}
	return s.q.RevokeAllUserSessions(ctx, id)
}

// Enable reactivates a disabled account (super-admin action; enforced at handler).
func (s *Service) Enable(ctx context.Context, id uuid.UUID) error {
	return s.q.SetUserActive(ctx, sftpdb.SetUserActiveParams{ID: id, IsActive: true})
}

func strPtr(s string) *string { return &s }

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

// UserStorage is a per-user storage usage row.
type UserStorage struct {
	ID           string `json:"id"`
	Username     string `json:"username"`
	FullName     string `json:"full_name"`
	Email        string `json:"email"`
	Role         string `json:"role"`
	StorageUsed  int64  `json:"storage_used"`
	StorageQuota int64  `json:"storage_quota"`
	Unlimited    bool   `json:"unlimited"`
	FileCount    int64  `json:"file_count"`
	PercentUsed  int    `json:"percent_used"`
}

// MediaSlice is a media-category size bucket.
type MediaSlice struct {
	Category string `json:"category"`
	Total    int64  `json:"total"`
	Files    int64  `json:"files"`
}

// StorageOverview is the admin storage-monitoring payload.
type StorageOverview struct {
	Users      []UserStorage `json:"users"`
	Media      []MediaSlice  `json:"media"`
	SystemUsed int64         `json:"system_used"`
}

// StorageOverview aggregates per-user usage and a system-wide media breakdown.
func (s *Service) StorageOverview(ctx context.Context) (*StorageOverview, error) {
	rows, err := s.q.StorageByUser(ctx)
	if err != nil {
		return nil, err
	}
	media, err := s.q.MediaBreakdown(ctx)
	if err != nil {
		return nil, err
	}

	out := &StorageOverview{Users: make([]UserStorage, 0, len(rows)), Media: make([]MediaSlice, 0, len(media))}
	for _, r := range rows {
		pct := 0
		if r.StorageQuota > 0 {
			pct = int(float64(r.StorageUsed) / float64(r.StorageQuota) * 100)
			if pct > 100 {
				pct = 100
			}
		}
		out.Users = append(out.Users, UserStorage{
			ID: r.ID.String(), Username: r.Username, FullName: r.FullName, Email: r.Email, Role: r.Role,
			StorageUsed: r.StorageUsed, StorageQuota: r.StorageQuota, Unlimited: r.StorageQuota == 0,
			FileCount: r.FileCount, PercentUsed: pct,
		})
	}
	for _, m := range media {
		out.Media = append(out.Media, MediaSlice{Category: m.Category, Total: m.Total, Files: m.Files})
		out.SystemUsed += m.Total
	}
	return out, nil
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
		IsActive: u.IsActive, IsLocked: u.IsLocked, HasAvatar: u.AvatarPath != nil && *u.AvatarPath != "",
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
