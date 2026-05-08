package services

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/superset-studio/kapstan/api/internal/models"
	"github.com/superset-studio/kapstan/api/internal/platform/crypto"
	"github.com/superset-studio/kapstan/api/internal/provider"
	awsprovider "github.com/superset-studio/kapstan/api/internal/provider/aws"
	"github.com/superset-studio/kapstan/api/internal/repositories"
)

var (
	ErrUnsupportedCategory      = errors.New("unsupported connection category")
	ErrInvalidConnectionCredentials = errors.New("invalid connection credentials")
)

type ConnectionService struct {
	connRepo      *repositories.ConnectionRepository
	encryptionKey []byte
}

func NewConnectionService(connRepo *repositories.ConnectionRepository, encryptionKey []byte) *ConnectionService {
	return &ConnectionService{
		connRepo:      connRepo,
		encryptionKey: encryptionKey,
	}
}

func (s *ConnectionService) CreateConnection(ctx context.Context, tenantID uuid.UUID, name string, category models.ConnectionCategory, rawCreds json.RawMessage, config json.RawMessage, createdBy uuid.UUID) (*models.Connection, error) {
	encrypted, err := crypto.Encrypt(s.encryptionKey, []byte(rawCreds))
	if err != nil {
		return nil, err
	}

	conn := &models.Connection{
		ID:          uuid.New(),
		TenantID:    tenantID,
		Name:        name,
		Category:    category,
		Status:      models.ConnectionStatusPending,
		Credentials: encrypted,
		Config:      config,
		CreatedBy:   &createdBy,
	}

	if err := s.connRepo.Create(ctx, conn); err != nil {
		return nil, err
	}

	return conn, nil
}

func (s *ConnectionService) GetConnection(ctx context.Context, connID, tenantID uuid.UUID) (*models.Connection, error) {
	return s.connRepo.GetByID(ctx, connID, tenantID)
}

func (s *ConnectionService) ListConnections(ctx context.Context, tenantID uuid.UUID) ([]models.Connection, error) {
	return s.connRepo.ListByTenantID(ctx, tenantID)
}

func (s *ConnectionService) UpdateConnection(ctx context.Context, connID, tenantID uuid.UUID, name *string, rawCreds json.RawMessage, config json.RawMessage) (*models.Connection, error) {
	conn, err := s.connRepo.GetByID(ctx, connID, tenantID)
	if err != nil {
		return nil, err
	}

	if name != nil {
		conn.Name = *name
	}

	if rawCreds != nil {
		encrypted, err := crypto.Encrypt(s.encryptionKey, []byte(rawCreds))
		if err != nil {
			return nil, err
		}
		conn.Credentials = encrypted
		conn.Status = models.ConnectionStatusPending
	}

	if config != nil {
		conn.Config = config
	}

	if err := s.connRepo.Update(ctx, conn); err != nil {
		return nil, err
	}

	return conn, nil
}

func (s *ConnectionService) ListAllConnections(ctx context.Context) ([]models.Connection, error) {
	return s.connRepo.ListAll(ctx)
}

func (s *ConnectionService) DeleteConnection(ctx context.Context, connID, tenantID uuid.UUID) error {
	return s.connRepo.Delete(ctx, connID, tenantID)
}

func (s *ConnectionService) ValidateConnection(ctx context.Context, connID, tenantID uuid.UUID) (*models.Connection, error) {
	conn, err := s.connRepo.GetByID(ctx, connID, tenantID)
	if err != nil {
		return nil, err
	}
	return s.validateAndUpdate(ctx, conn)
}

func (s *ConnectionService) ValidateConnectionByID(ctx context.Context, connID uuid.UUID) (*models.Connection, error) {
	conn, err := s.connRepo.GetByIDInternal(ctx, connID)
	if err != nil {
		return nil, err
	}
	return s.validateAndUpdate(ctx, conn)
}

func (s *ConnectionService) validateAndUpdate(ctx context.Context, conn *models.Connection) (*models.Connection, error) {
	decrypted, err := crypto.Decrypt(s.encryptionKey, conn.Credentials)
	if err != nil {
		return nil, err
	}

	prov, err := s.buildProvider(conn.Category, decrypted)
	if err != nil {
		return nil, err
	}

	now := time.Now()

	if err := prov.ValidateCredentials(ctx); err != nil {
		conn.Status = models.ConnectionStatusInvalid
		conn.LastValidated = &now
		_ = s.connRepo.UpdateStatus(ctx, conn.ID, conn.Status, now)
		return conn, nil
	}

	conn.Status = models.ConnectionStatusValid
	conn.LastValidated = &now

	info, err := prov.GetAccountInfo(ctx)
	if err == nil {
		s.mergeAccountInfo(conn, info)
	}

	// Run permission checks and determine final status.
	permResults := prov.TestPermissions(ctx)
	s.mergePermissionResults(conn, permResults)

	allPassed := true
	for _, r := range permResults {
		if !r.Passed {
			allPassed = false
			break
		}
	}
	if !allPassed {
		conn.Status = models.ConnectionStatusPartial
	}

	_ = s.connRepo.UpdateStatusAndConfig(ctx, conn.ID, conn.Status, now, conn.Config)
	return conn, nil
}

func (s *ConnectionService) buildProvider(category models.ConnectionCategory, decryptedCreds []byte) (provider.Provider, error) {
	switch category {
	case models.ConnectionCategoryAWS:
		// Try static keys first.
		var staticCreds models.AWSCredentials
		if err := json.Unmarshal(decryptedCreds, &staticCreds); err == nil && staticCreds.AccessKeyID != "" {
			return awsprovider.NewAWSProvider(staticCreds.AccessKeyID, staticCreds.SecretAccessKey, staticCreds.Region), nil
		}
		// Try IAM role.
		var roleCreds models.AWSRoleCredentials
		if err := json.Unmarshal(decryptedCreds, &roleCreds); err == nil && roleCreds.RoleARN != "" {
			return awsprovider.NewAWSProviderFromRole(roleCreds.RoleARN, roleCreds.ExternalID, roleCreds.Region)
		}
		return nil, ErrInvalidConnectionCredentials
	default:
		return nil, ErrUnsupportedCategory
	}
}

func (s *ConnectionService) mergeAccountInfo(conn *models.Connection, info *provider.AccountInfo) {
	existing := make(map[string]interface{})
	if conn.Config != nil {
		_ = json.Unmarshal(conn.Config, &existing)
	}
	if info.AccountID != "" {
		existing["accountId"] = info.AccountID
	}
	if info.UserARN != "" {
		existing["userArn"] = info.UserARN
	}
	merged, err := json.Marshal(existing)
	if err == nil {
		conn.Config = merged
	}
}

func (s *ConnectionService) mergePermissionResults(conn *models.Connection, results []provider.PermissionCheckResult) {
	existing := make(map[string]interface{})
	if conn.Config != nil {
		_ = json.Unmarshal(conn.Config, &existing)
	}
	existing["permissions"] = results
	merged, err := json.Marshal(existing)
	if err == nil {
		conn.Config = merged
	}
}
