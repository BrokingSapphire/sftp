// Package team implements Team Spaces — group-owned shared drives with
// membership roles (owner > admin > member > viewer).
package team

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"sapphirebroking.com/sftp_service/internal/apperrors"
	"sapphirebroking.com/sftp_service/internal/db/sftpdb"
	models "sapphirebroking.com/sftp_service/internal/models/team"
	"sapphirebroking.com/sftp_service/pkg/logger"
)

// Service manages teams and their membership.
type Service struct {
	q   *sftpdb.Queries
	log logger.Logger
}

// New builds the team Service.
func New(q *sftpdb.Queries, log logger.Logger) *Service {
	return &Service{q: q, log: log.Named("service.team")}
}

var roleRank = map[string]int{"viewer": 0, "member": 1, "admin": 2, "owner": 3}

func validRole(r string) bool { _, ok := roleRank[r]; return ok }

// Role returns the caller's role in a team, or an error if they're not a member.
func (s *Service) Role(ctx context.Context, teamID, user uuid.UUID) (string, error) {
	r, err := s.q.GetTeamMembership(ctx, sftpdb.GetTeamMembershipParams{TeamID: teamID, UserID: user})
	if err != nil {
		return "", apperrors.ErrForbidden
	}
	return r, nil
}

// requireRole checks the caller has at least the given role in the team.
func (s *Service) requireRole(ctx context.Context, teamID, user uuid.UUID, min string) error {
	r, err := s.Role(ctx, teamID, user)
	if err != nil {
		return err
	}
	if roleRank[r] < roleRank[min] {
		return apperrors.ErrForbidden
	}
	return nil
}

// Create makes a team and adds the creator as its owner.
func (s *Service) Create(ctx context.Context, creator uuid.UUID, req models.CreateRequest) (*models.TeamResponse, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, apperrors.ErrInvalidRequest
	}
	t, err := s.q.CreateTeam(ctx, sftpdb.CreateTeamParams{
		Name: name, Slug: slugify(name), Description: strings.TrimSpace(req.Description),
		StorageQuota: req.StorageQuota, Color: req.Color, CreatedBy: &creator,
	})
	if err != nil {
		return nil, err
	}
	if err := s.q.AddTeamMember(ctx, sftpdb.AddTeamMemberParams{TeamID: t.ID, UserID: creator, Role: "owner"}); err != nil {
		return nil, err
	}
	resp := toTeam(t)
	resp.MemberRole = "owner"
	resp.MemberCount = 1
	return &resp, nil
}

// ListForUser returns the teams the caller belongs to.
func (s *Service) ListForUser(ctx context.Context, user uuid.UUID) ([]models.TeamResponse, error) {
	rows, err := s.q.ListTeamsForUser(ctx, user)
	if err != nil {
		return nil, err
	}
	out := make([]models.TeamResponse, 0, len(rows))
	for _, r := range rows {
		t := toTeam(sftpdb.Team{
			ID: r.ID, Name: r.Name, Slug: r.Slug, Description: r.Description,
			StorageQuota: r.StorageQuota, StorageUsed: r.StorageUsed, Color: r.Color, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
		})
		t.MemberRole = r.MemberRole
		t.MemberCount = r.MemberCount
		out = append(out, t)
	}
	return out, nil
}

// Get returns a team (caller must be a member).
func (s *Service) Get(ctx context.Context, actor, teamID uuid.UUID) (*models.TeamResponse, error) {
	role, err := s.Role(ctx, teamID, actor)
	if err != nil {
		return nil, err
	}
	t, err := s.q.GetTeam(ctx, teamID)
	if err != nil {
		return nil, apperrors.ErrNotFound
	}
	resp := toTeam(t)
	resp.MemberRole = role
	return &resp, nil
}

// Update edits team settings (admin+).
func (s *Service) Update(ctx context.Context, actor, teamID uuid.UUID, req models.CreateRequest) error {
	if err := s.requireRole(ctx, teamID, actor, "admin"); err != nil {
		return err
	}
	return s.q.UpdateTeam(ctx, sftpdb.UpdateTeamParams{
		ID: teamID, Name: strings.TrimSpace(req.Name), Description: strings.TrimSpace(req.Description), StorageQuota: req.StorageQuota, Color: req.Color,
	})
}

// Delete removes a team and its drive (owner only).
func (s *Service) Delete(ctx context.Context, actor, teamID uuid.UUID) error {
	if err := s.requireRole(ctx, teamID, actor, "owner"); err != nil {
		return err
	}
	return s.q.DeleteTeam(ctx, teamID)
}

// ListMembers lists a team's members (any member).
func (s *Service) ListMembers(ctx context.Context, actor, teamID uuid.UUID) ([]models.MemberResponse, error) {
	if _, err := s.Role(ctx, teamID, actor); err != nil {
		return nil, err
	}
	rows, err := s.q.ListTeamMembers(ctx, teamID)
	if err != nil {
		return nil, err
	}
	out := make([]models.MemberResponse, 0, len(rows))
	for _, r := range rows {
		name := r.FullName
		if name == "" {
			name = r.Username
		}
		out = append(out, models.MemberResponse{
			UserID: r.UserID.String(), Name: name, Email: r.Email, Role: r.Role,
			HasAvatar: r.HasAvatar != nil && *r.HasAvatar,
		})
	}
	return out, nil
}

// AddMember adds/updates a member by email (admin+). Cannot grant owner.
func (s *Service) AddMember(ctx context.Context, actor, teamID uuid.UUID, email, role string) (*models.MemberResponse, error) {
	if err := s.requireRole(ctx, teamID, actor, "admin"); err != nil {
		return nil, err
	}
	if role == "" {
		role = "member"
	}
	if !validRole(role) || role == "owner" {
		return nil, apperrors.ErrInvalidRequest
	}
	u, err := s.q.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, apperrors.ErrUserNotFound
	}
	if err := s.q.AddTeamMember(ctx, sftpdb.AddTeamMemberParams{TeamID: teamID, UserID: u.ID, Role: role}); err != nil {
		return nil, err
	}
	name := u.FullName
	if name == "" {
		name = u.Username
	}
	return &models.MemberResponse{
		UserID: u.ID.String(), Name: name, Email: u.Email, Role: role,
		HasAvatar: u.AvatarPath != nil && *u.AvatarPath != "",
	}, nil
}

// RemoveMember removes a member (admin+). Owners cannot be removed here.
func (s *Service) RemoveMember(ctx context.Context, actor, teamID, target uuid.UUID) error {
	if err := s.requireRole(ctx, teamID, actor, "admin"); err != nil {
		return err
	}
	if r, err := s.Role(ctx, teamID, target); err == nil && r == "owner" {
		return apperrors.ErrForbidden
	}
	return s.q.RemoveTeamMember(ctx, sftpdb.RemoveTeamMemberParams{TeamID: teamID, UserID: target})
}

func toTeam(t sftpdb.Team) models.TeamResponse {
	r := models.TeamResponse{
		ID: t.ID.String(), Name: t.Name, Slug: t.Slug, Description: t.Description,
		StorageQuota: t.StorageQuota, StorageUsed: t.StorageUsed, Color: t.Color,
	}
	if t.CreatedAt.Valid {
		r.CreatedAt = t.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00")
	}
	return r
}

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(name string) string {
	s := slugRe.ReplaceAllString(strings.ToLower(name), "-")
	s = strings.Trim(s, "-")
	if len(s) > 48 {
		s = s[:48]
	}
	b := make([]byte, 3)
	_, _ = rand.Read(b)
	return s + "-" + hex.EncodeToString(b)
}
