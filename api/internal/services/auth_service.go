package services

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/superset-studio/kapstan/api/internal/models"
	"github.com/superset-studio/kapstan/api/internal/platform/auth"
	"github.com/superset-studio/kapstan/api/internal/repositories"
)

var (
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrInvalidRefreshToken = errors.New("invalid or expired refresh token")
)

type AuthService struct {
	db         *sqlx.DB
	jwtManager *auth.JWTManager
	userRepo   *repositories.UserRepository
	orgRepo    *repositories.OrganizationRepository
	memberRepo *repositories.OrganizationMemberRepository
	tokenRepo  *repositories.RefreshTokenRepository
}

func NewAuthService(
	db *sqlx.DB,
	jwtManager *auth.JWTManager,
	userRepo *repositories.UserRepository,
	orgRepo *repositories.OrganizationRepository,
	memberRepo *repositories.OrganizationMemberRepository,
	tokenRepo *repositories.RefreshTokenRepository,
) *AuthService {
	return &AuthService{
		db:         db,
		jwtManager: jwtManager,
		userRepo:   userRepo,
		orgRepo:    orgRepo,
		memberRepo: memberRepo,
		tokenRepo:  tokenRepo,
	}
}

func (s *AuthService) Register(ctx context.Context, req *models.RegisterRequest) (*models.AuthResponse, error) {
	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	user := &models.User{
		ID:            uuid.New(),
		Email:         req.Email,
		PasswordHash:  &hashedPassword,
		Name:          req.Name,
		EmailVerified: false,
	}
	if err := s.userRepo.CreateTx(ctx, tx, user); err != nil {
		return nil, err
	}

	org := &models.Organization{
		ID:          uuid.New(),
		Name:        slugify(req.OrgName),
		DisplayName: req.OrgName,
	}
	if err := s.orgRepo.CreateTx(ctx, tx, org); err != nil {
		return nil, err
	}

	now := time.Now()
	member := &models.OrganizationMember{
		ID:             uuid.New(),
		OrganizationID: org.ID,
		UserID:         user.ID,
		Role:           models.RoleOwner,
		AcceptedAt:     &now,
	}
	if err := s.memberRepo.CreateTx(ctx, tx, member); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	memberships := []models.OrgMembership{
		{OrgID: org.ID, Role: models.RoleOwner},
	}
	tokenPair, err := s.generateTokenPair(ctx, user, memberships)
	if err != nil {
		return nil, err
	}

	return &models.AuthResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		User:         user,
		Organization: org,
	}, nil
}

func (s *AuthService) Login(ctx context.Context, req *models.LoginRequest) (*models.AuthResponse, error) {
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if user.PasswordHash == nil {
		return nil, ErrInvalidCredentials
	}

	if err := auth.CheckPassword(*user.PasswordHash, req.Password); err != nil {
		return nil, ErrInvalidCredentials
	}

	members, err := s.memberRepo.ListByUserID(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	memberships := make([]models.OrgMembership, len(members))
	for i, m := range members {
		memberships[i] = models.OrgMembership{OrgID: m.OrganizationID, Role: m.Role}
	}

	tokenPair, err := s.generateTokenPair(ctx, user, memberships)
	if err != nil {
		return nil, err
	}

	// Return the first org for convenience.
	var org *models.Organization
	if len(members) > 0 {
		org, _ = s.orgRepo.GetByID(ctx, members[0].OrganizationID)
	}

	return &models.AuthResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		User:         user,
		Organization: org,
	}, nil
}

func (s *AuthService) Refresh(ctx context.Context, rawToken string) (*models.TokenPair, error) {
	tokenHash := hashRefreshToken(rawToken)

	stored, err := s.tokenRepo.GetByHash(ctx, tokenHash)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}

	// Delete the old token (rotation).
	if err := s.tokenRepo.DeleteByHash(ctx, tokenHash); err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetByID(ctx, stored.UserID)
	if err != nil {
		return nil, err
	}

	members, err := s.memberRepo.ListByUserID(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	memberships := make([]models.OrgMembership, len(members))
	for i, m := range members {
		memberships[i] = models.OrgMembership{OrgID: m.OrganizationID, Role: m.Role}
	}

	return s.generateTokenPair(ctx, user, memberships)
}

func (s *AuthService) generateTokenPair(ctx context.Context, user *models.User, memberships []models.OrgMembership) (*models.TokenPair, error) {
	accessToken, err := s.jwtManager.GenerateAccessToken(user, memberships)
	if err != nil {
		return nil, err
	}

	rawRefresh := s.jwtManager.GenerateRefreshToken()
	refreshToken := &models.RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: hashRefreshToken(rawRefresh),
		ExpiresAt: time.Now().Add(auth.RefreshTokenExpiry),
	}

	if err := s.tokenRepo.Create(ctx, refreshToken); err != nil {
		return nil, err
	}

	return &models.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
	}, nil
}
