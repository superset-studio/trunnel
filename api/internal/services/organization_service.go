package services

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/superset-studio/kapstan/api/internal/models"
	"github.com/superset-studio/kapstan/api/internal/repositories"
)

var (
	ErrCannotRemoveLastOwner = errors.New("cannot remove the last owner of an organization")
)

type OrganizationService struct {
	db         *sqlx.DB
	orgRepo    *repositories.OrganizationRepository
	memberRepo *repositories.OrganizationMemberRepository
	userRepo   *repositories.UserRepository
}

func NewOrganizationService(
	db *sqlx.DB,
	orgRepo *repositories.OrganizationRepository,
	memberRepo *repositories.OrganizationMemberRepository,
	userRepo *repositories.UserRepository,
) *OrganizationService {
	return &OrganizationService{
		db:         db,
		orgRepo:    orgRepo,
		memberRepo: memberRepo,
		userRepo:   userRepo,
	}
}

func (s *OrganizationService) CreateOrganization(ctx context.Context, userID uuid.UUID, displayName string) (*models.Organization, error) {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	org := &models.Organization{
		ID:          uuid.New(),
		Name:        slugify(displayName),
		DisplayName: displayName,
	}
	if err := s.orgRepo.CreateTx(ctx, tx, org); err != nil {
		return nil, err
	}

	now := time.Now()
	member := &models.OrganizationMember{
		ID:             uuid.New(),
		OrganizationID: org.ID,
		UserID:         userID,
		Role:           models.RoleOwner,
		AcceptedAt:     &now,
	}
	if err := s.memberRepo.CreateTx(ctx, tx, member); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return org, nil
}

func (s *OrganizationService) GetOrganization(ctx context.Context, orgID uuid.UUID) (*models.Organization, error) {
	return s.orgRepo.GetByID(ctx, orgID)
}

func (s *OrganizationService) ListUserOrganizations(ctx context.Context, userID uuid.UUID) ([]models.Organization, error) {
	return s.orgRepo.ListByUserID(ctx, userID)
}

func (s *OrganizationService) UpdateOrganization(ctx context.Context, org *models.Organization) error {
	return s.orgRepo.Update(ctx, org)
}

func (s *OrganizationService) ListMembers(ctx context.Context, orgID uuid.UUID) ([]models.MemberWithUser, error) {
	return s.memberRepo.ListByOrgID(ctx, orgID)
}

func (s *OrganizationService) InviteMember(ctx context.Context, orgID uuid.UUID, inviterID uuid.UUID, email string, role models.Role) (*models.OrganizationMember, error) {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Find or create user by email.
	user, err := s.userRepo.GetByEmailTx(ctx, tx, email)
	if err != nil {
		if !errors.Is(err, repositories.ErrNotFound) {
			return nil, err
		}
		// Create a stub user (no password — invited).
		user = &models.User{
			ID:    uuid.New(),
			Email: email,
			Name:  email, // Default name to email until they register.
		}
		if err := s.userRepo.CreateTx(ctx, tx, user); err != nil {
			return nil, err
		}
	}

	now := time.Now()
	member := &models.OrganizationMember{
		ID:             uuid.New(),
		OrganizationID: orgID,
		UserID:         user.ID,
		Role:           role,
		InvitedBy:      &inviterID,
		InvitedAt:      &now,
	}
	if err := s.memberRepo.CreateTx(ctx, tx, member); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return member, nil
}

func (s *OrganizationService) UpdateMemberRole(ctx context.Context, memberID uuid.UUID, role models.Role) error {
	return s.memberRepo.UpdateRole(ctx, memberID, role)
}

func (s *OrganizationService) RemoveMember(ctx context.Context, memberID uuid.UUID, orgID uuid.UUID) error {
	member, err := s.memberRepo.GetByID(ctx, memberID)
	if err != nil {
		return err
	}

	if member.Role == models.RoleOwner {
		count, err := s.memberRepo.CountOwners(ctx, orgID)
		if err != nil {
			return err
		}
		if count <= 1 {
			return ErrCannotRemoveLastOwner
		}
	}

	return s.memberRepo.Delete(ctx, memberID)
}
