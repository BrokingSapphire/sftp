// Package apikey implements API-key management and authentication.
package apikey

import (
	"context"
	"net/netip"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"sapphirebroking.com/sftp_service/internal/apperrors"
	"sapphirebroking.com/sftp_service/internal/db/sftpdb"
	models "sapphirebroking.com/sftp_service/internal/models/apikey"
	keygen "sapphirebroking.com/sftp_service/pkg/apikey"
	"sapphirebroking.com/sftp_service/pkg/logger"
)

// Service manages API keys.
type Service struct {
	q   *sftpdb.Queries
	log logger.Logger
}

// New builds the API-key Service.
func New(q *sftpdb.Queries, log logger.Logger) *Service {
	return &Service{q: q, log: log.Named("service.apikey")}
}

// Principal is the identity resolved from a valid API key.
type Principal struct {
	UserID uuid.UUID
	Email  string
	Role   string
	Scopes []string
}

// Create mints a new API key for the user and returns the plaintext once.
func (s *Service) Create(ctx context.Context, userID uuid.UUID, req models.CreateRequest) (*models.CreateResponse, error) {
	gen, err := keygen.New()
	if err != nil {
		return nil, err
	}
	var expires pgtype.Timestamptz
	if req.ExpiresInDays != nil && *req.ExpiresInDays > 0 {
		expires = pgtype.Timestamptz{Time: time.Now().AddDate(0, 0, *req.ExpiresInDays), Valid: true}
	}
	scopes := req.Scopes
	if scopes == nil {
		scopes = []string{}
	}

	key, err := s.q.CreateAPIKey(ctx, sftpdb.CreateAPIKeyParams{
		UserID: userID, Name: req.Name, Prefix: gen.Prefix, KeyHash: gen.Hash,
		Scopes: scopes, ExpiresAt: expires,
	})
	if err != nil {
		return nil, err
	}
	resp := &models.CreateResponse{
		ID: key.ID.String(), Name: key.Name, Prefix: key.Prefix, Key: gen.Plaintext,
		Scopes: key.Scopes, CreatedAt: fmtTS(key.CreatedAt),
	}
	if key.ExpiresAt.Valid {
		resp.ExpiresAt = key.ExpiresAt.Time.Format(time.RFC3339)
	}
	return resp, nil
}

// List returns the user's active API keys (without secrets).
func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]models.Response, error) {
	rows, err := s.q.ListUserAPIKeys(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]models.Response, 0, len(rows))
	for _, k := range rows {
		r := models.Response{
			ID: k.ID.String(), Name: k.Name, Prefix: k.Prefix, Scopes: k.Scopes,
			LastUsedAt: fmtTS(k.LastUsedAt), CreatedAt: fmtTS(k.CreatedAt),
		}
		if k.ExpiresAt.Valid {
			r.ExpiresAt = k.ExpiresAt.Time.Format(time.RFC3339)
		}
		out = append(out, r)
	}
	return out, nil
}

// Revoke revokes one of the user's API keys.
func (s *Service) Revoke(ctx context.Context, userID, keyID uuid.UUID) error {
	return s.q.RevokeAPIKey(ctx, sftpdb.RevokeAPIKeyParams{ID: keyID, UserID: userID})
}

// Authenticate resolves a plaintext API key to a Principal, updating last-used.
func (s *Service) Authenticate(ctx context.Context, plaintext, ip string) (*Principal, error) {
	if !keygen.Valid(plaintext) {
		return nil, apperrors.ErrAPIKeyNotFound
	}
	key, err := s.q.GetAPIKeyByHash(ctx, keygen.Hash(plaintext))
	if err != nil {
		return nil, apperrors.ErrAPIKeyNotFound
	}
	user, err := s.q.GetUserByID(ctx, key.UserID)
	if err != nil || !user.IsActive {
		return nil, apperrors.ErrAPIKeyRevoked
	}
	role, err := s.q.GetRoleByID(ctx, user.RoleID)
	if err != nil {
		return nil, err
	}

	_ = s.q.TouchAPIKey(ctx, sftpdb.TouchAPIKeyParams{ID: key.ID, LastUsedIp: parseAddr(ip)})

	return &Principal{
		UserID: user.ID, Email: user.Email, Role: role.Slug, Scopes: key.Scopes,
	}, nil
}

func fmtTS(t pgtype.Timestamptz) string {
	if !t.Valid {
		return ""
	}
	return t.Time.Format(time.RFC3339)
}

func parseAddr(ip string) *netip.Addr {
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return nil
	}
	return &addr
}
