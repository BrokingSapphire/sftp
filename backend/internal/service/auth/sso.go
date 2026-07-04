package auth

import (
	"context"
	"strings"

	"sapphirebroking.com/sftp_service/internal/apperrors"
	"sapphirebroking.com/sftp_service/internal/db/sftpdb"
	models "sapphirebroking.com/sftp_service/internal/models/auth"
)

// SSOProfile is the normalised identity returned by an OIDC provider.
type SSOProfile struct {
	Email    string
	FullName string
	Provider string // e.g. "microsoft"
}

// LoginSSO signs in (provisioning on first login) a user authenticated by an
// external identity provider, then issues platform tokens. Provisioned users
// get a random unusable password; they authenticate only via SSO until they
// set a password through a reset flow.
func (s *Service) LoginSSO(ctx context.Context, p SSOProfile, defaultRoleSlug string, allowedDomains []string, meta RequestMeta) (*models.TokenPair, error) {
	email := strings.ToLower(strings.TrimSpace(p.Email))
	if email == "" {
		return nil, apperrors.ErrInvalidCredentials
	}
	if !domainAllowed(email, allowedDomains) {
		return nil, apperrors.ErrSSODomainNotAllowed
	}

	user, err := s.q.GetUserByEmail(ctx, email)
	if err != nil {
		user, err = s.provisionSSOUser(ctx, email, p.FullName, defaultRoleSlug)
		if err != nil {
			return nil, err
		}
	}
	if !user.IsActive {
		s.recordLogin(ctx, &user.ID, user.Email, false, "disabled", meta)
		return nil, apperrors.ErrAccountDisabled
	}

	if err := s.q.UpdateLastLogin(ctx, user.ID); err != nil {
		s.log.Error("update last login failed", "err", err)
	}
	s.recordLogin(ctx, &user.ID, user.Email, true, "sso:"+p.Provider, meta)
	return s.issueTokens(ctx, user, false, meta)
}

func (s *Service) provisionSSOUser(ctx context.Context, email, fullName, roleSlug string) (sftpdb.User, error) {
	role, err := s.q.GetRoleBySlug(ctx, roleSlug)
	if err != nil {
		return sftpdb.User{}, apperrors.ErrRoleNotFound
	}
	// Random, unusable password so credential login is impossible until reset.
	randomPw, err := randomToken()
	if err != nil {
		return sftpdb.User{}, err
	}
	hash, err := s.HashPassword(randomPw)
	if err != nil {
		return sftpdb.User{}, err
	}
	user, err := s.q.CreateUser(ctx, sftpdb.CreateUserParams{
		Email:        email,
		Username:     email, // email is unique; guarantees a unique username
		PasswordHash: hash,
		FullName:     fullName,
		RoleID:       role.ID,
		StorageQuota: 0,
		MustChangePw: false,
	})
	if err != nil {
		return sftpdb.User{}, err
	}
	s.log.Info("provisioned SSO user", "email", email, "role", roleSlug)
	return user, nil
}

func domainAllowed(email string, allowed []string) bool {
	if len(allowed) == 0 {
		return true
	}
	at := strings.LastIndex(email, "@")
	if at < 0 {
		return false
	}
	domain := strings.ToLower(email[at+1:])
	for _, d := range allowed {
		if strings.EqualFold(strings.TrimSpace(d), domain) {
			return true
		}
	}
	return false
}
