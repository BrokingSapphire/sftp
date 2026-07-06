// Package auth implements authentication: login, token refresh, logout and
// password changes, with Argon2id hashing, account lockout and audited history.
package auth

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"sapphirebroking.com/sftp_service/internal/apperrors"
	"sapphirebroking.com/sftp_service/internal/config"
	"sapphirebroking.com/sftp_service/internal/db/sftpdb"
	models "sapphirebroking.com/sftp_service/internal/models/auth"
	"sapphirebroking.com/sftp_service/pkg/argon2"
	"sapphirebroking.com/sftp_service/pkg/headers"
	"sapphirebroking.com/sftp_service/pkg/jwt"
	"sapphirebroking.com/sftp_service/pkg/logger"
)

// RequestMeta carries client metadata for auditing.
type RequestMeta struct {
	IP        string
	UserAgent string
}

// Deps are the auth service dependencies.
type Deps struct {
	Queries  *sftpdb.Queries
	JWT      *jwt.Manager
	Security config.SecurityConfig
	Logger   logger.Logger
}

// Service provides authentication operations.
type Service struct {
	q        *sftpdb.Queries
	jwt      *jwt.Manager
	sec      config.SecurityConfig
	argon    argon2.Params
	log      logger.Logger
}

// New builds the auth Service.
func New(d Deps) *Service {
	return &Service{
		q:   d.Queries,
		jwt: d.JWT,
		sec: d.Security,
		argon: argon2.Params{
			MemoryKiB: d.Security.ArgonMemoryKiB,
			Time:      d.Security.ArgonTime,
			Threads:   d.Security.ArgonThreads,
			KeyLen:    d.Security.ArgonKeyLen,
			SaltLen:   d.Security.ArgonSaltLen,
		},
		log: d.Logger.Named("service.auth"),
	}
}

// HashPassword returns an Argon2id PHC hash using the configured parameters.
func (s *Service) HashPassword(password string) (string, error) {
	return argon2.Hash(password, s.argon)
}

// Login authenticates credentials and issues a token pair.
func (s *Service) Login(ctx context.Context, req models.LoginRequest, meta RequestMeta) (*models.TokenPair, error) {
	user, err := s.q.GetUserByEmailOrUsername(ctx, req.Identifier)
	if err != nil {
		s.recordLogin(ctx, nil, req.Identifier, false, "user_not_found", meta)
		return nil, apperrors.ErrInvalidCredentials
	}

	if !user.IsActive {
		s.recordLogin(ctx, &user.ID, user.Email, false, "disabled", meta)
		return nil, apperrors.ErrAccountDisabled
	}
	if user.IsLocked {
		if user.LockedUntil.Valid && user.LockedUntil.Time.After(time.Now()) {
			s.recordLogin(ctx, &user.ID, user.Email, false, "locked", meta)
			return nil, apperrors.ErrAccountLocked
		}
		_ = s.q.UnlockUser(ctx, user.ID) // lock window elapsed
	}

	ok, err := argon2.Verify(req.Password, user.PasswordHash)
	if err != nil || !ok {
		s.handleFailedAttempt(ctx, user)
		s.recordLogin(ctx, &user.ID, user.Email, false, "bad_password", meta)
		return nil, apperrors.ErrInvalidCredentials
	}

	if err := s.q.UpdateLastLogin(ctx, user.ID); err != nil {
		s.log.Error("update last login failed", "err", err)
	}
	s.recordLogin(ctx, &user.ID, user.Email, true, "ok", meta)

	return s.issueTokens(ctx, user, req.RememberMe, meta)
}

// VerifyPassword authenticates credentials without issuing tokens and returns
// the user id. Used by the SFTP protocol server's password auth. Applies the
// same active/locked/lockout rules as Login.
func (s *Service) VerifyPassword(ctx context.Context, identifier, password string) (uuid.UUID, error) {
	user, err := s.q.GetUserByEmailOrUsername(ctx, identifier)
	if err != nil {
		return uuid.Nil, apperrors.ErrInvalidCredentials
	}
	if !user.IsActive {
		return uuid.Nil, apperrors.ErrAccountDisabled
	}
	if user.IsLocked && user.LockedUntil.Valid && user.LockedUntil.Time.After(time.Now()) {
		return uuid.Nil, apperrors.ErrAccountLocked
	}
	ok, err := argon2.Verify(password, user.PasswordHash)
	if err != nil || !ok {
		s.handleFailedAttempt(ctx, user)
		return uuid.Nil, apperrors.ErrInvalidCredentials
	}
	_ = s.q.UpdateLastLogin(ctx, user.ID)
	return user.ID, nil
}

// Refresh rotates a refresh token and issues a new access token.
func (s *Service) Refresh(ctx context.Context, refreshToken string, meta RequestMeta) (*models.TokenPair, error) {
	session, err := s.q.GetSessionByHash(ctx, hashToken(refreshToken))
	if err != nil {
		return nil, apperrors.ErrInvalidToken
	}
	user, err := s.q.GetUserByID(ctx, session.UserID)
	if err != nil || !user.IsActive {
		return nil, apperrors.ErrInvalidToken
	}

	newRefresh, err := randomToken()
	if err != nil {
		return nil, err
	}
	ttl := s.refreshTTL(session.RememberMe)
	if err := s.q.RotateSession(ctx, sftpdb.RotateSessionParams{
		ID:               session.ID,
		RefreshTokenHash: hashToken(newRefresh),
		ExpiresAt:        ts(time.Now().Add(ttl)),
	}); err != nil {
		return nil, err
	}

	pair, err := s.accessFor(ctx, user, session.ID.String())
	if err != nil {
		return nil, err
	}
	pair.RefreshToken = newRefresh
	return pair, nil
}

// Logout revokes the session backing the given refresh token.
func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	return s.q.RevokeSessionByHash(ctx, hashToken(refreshToken))
}

// Me returns the public projection of a user by ID.
func (s *Service) Me(ctx context.Context, userID uuid.UUID) (*models.UserInfo, error) {
	user, err := s.q.GetUserByID(ctx, userID)
	if err != nil {
		return nil, apperrors.ErrUserNotFound
	}
	return s.userInfo(ctx, user), nil
}

// ChangePassword verifies the current password and sets a new one.
func (s *Service) ChangePassword(ctx context.Context, userID uuid.UUID, req models.ChangePasswordRequest) error {
	user, err := s.q.GetUserByID(ctx, userID)
	if err != nil {
		return apperrors.ErrUserNotFound
	}
	ok, err := argon2.Verify(req.CurrentPassword, user.PasswordHash)
	if err != nil || !ok {
		return apperrors.ErrInvalidCredentials
	}
	if len(req.NewPassword) < s.sec.PasswordMinLength {
		return apperrors.ErrWeakPassword
	}
	hash, err := s.HashPassword(req.NewPassword)
	if err != nil {
		return err
	}
	if err := s.q.SetUserPassword(ctx, sftpdb.SetUserPasswordParams{ID: userID, PasswordHash: hash}); err != nil {
		return err
	}
	// Revoke all sessions to force re-login on other devices.
	return s.q.RevokeAllUserSessions(ctx, userID)
}

// ── internals ──────────────────────────────────────────────

func (s *Service) handleFailedAttempt(ctx context.Context, user sftpdb.User) {
	attempts, err := s.q.IncrementFailedAttempts(ctx, user.ID)
	if err != nil {
		s.log.Error("increment failed attempts", "err", err)
		return
	}
	if int(attempts) >= s.sec.MaxLoginAttempts {
		if err := s.q.LockUser(ctx, sftpdb.LockUserParams{
			ID:          user.ID,
			LockedUntil: ts(time.Now().Add(s.sec.LockoutDuration)),
		}); err != nil {
			s.log.Error("lock user", "err", err)
		}
		s.log.Warn("account locked after failed attempts", "user_id", user.ID, "attempts", attempts)
	}
}

func (s *Service) issueTokens(ctx context.Context, user sftpdb.User, rememberMe bool, meta RequestMeta) (*models.TokenPair, error) {
	refresh, err := randomToken()
	if err != nil {
		return nil, err
	}
	ttl := s.refreshTTL(rememberMe)
	session, err := s.q.CreateSession(ctx, sftpdb.CreateSessionParams{
		UserID:           user.ID,
		RefreshTokenHash: hashToken(refresh),
		UserAgent:        meta.UserAgent,
		IpAddress:        parseAddr(meta.IP),
		RememberMe:       rememberMe,
		ExpiresAt:        ts(time.Now().Add(ttl)),
	})
	if err != nil {
		return nil, err
	}

	pair, err := s.accessFor(ctx, user, session.ID.String())
	if err != nil {
		return nil, err
	}
	pair.RefreshToken = refresh
	return pair, nil
}

func (s *Service) accessFor(ctx context.Context, user sftpdb.User, sessionID string) (*models.TokenPair, error) {
	role, err := s.q.GetRoleByID(ctx, user.RoleID)
	if err != nil {
		return nil, err
	}
	access, exp, err := s.jwt.Issue(user.ID.String(), user.Email, user.Username, role.Slug, sessionID)
	if err != nil {
		return nil, err
	}
	return &models.TokenPair{
		AccessToken: access,
		TokenType:   "Bearer",
		ExpiresIn:   int64(time.Until(exp).Seconds()),
		User:        s.userInfo(ctx, user),
	}, nil
}

func (s *Service) userInfo(ctx context.Context, user sftpdb.User) *models.UserInfo {
	roleSlug := ""
	if role, err := s.q.GetRoleByID(ctx, user.RoleID); err == nil {
		roleSlug = role.Slug
	}
	perms, err := s.q.GetPermissionsForUser(ctx, user.ID)
	if err != nil {
		perms = nil
	}
	return &models.UserInfo{
		ID:           user.ID.String(),
		Email:        user.Email,
		Username:     user.Username,
		FullName:     user.FullName,
		Role:         roleSlug,
		Permissions:  perms,
		StorageUsed:  user.StorageUsed,
		StorageQuota: user.StorageQuota,
		MustChangePw: user.MustChangePw,
		HasAvatar:    user.AvatarPath != nil && *user.AvatarPath != "",
	}
}

func (s *Service) refreshTTL(rememberMe bool) time.Duration {
	if rememberMe {
		return 30 * 24 * time.Hour
	}
	return 7 * 24 * time.Hour
}

func (s *Service) recordLogin(ctx context.Context, userID *uuid.UUID, email string, success bool, reason string, meta RequestMeta) {
	dev := headers.ParseDevice(meta.UserAgent)
	emailCopy := email
	reasonCopy := reason
	uaCopy := meta.UserAgent
	browserCopy := dev.Browser
	osCopy := dev.OS
	err := s.q.InsertLoginHistory(ctx, sftpdb.InsertLoginHistoryParams{
		UserID:    userID,
		Email:     &emailCopy,
		Success:   success,
		Reason:    &reasonCopy,
		IpAddress: parseAddr(meta.IP),
		UserAgent: &uaCopy,
		Browser:   &browserCopy,
		Os:        &osCopy,
	})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		s.log.Error("record login history failed", "err", err)
	}
}
