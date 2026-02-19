package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/superset-studio/kapstan/api/internal/models"
	"github.com/superset-studio/kapstan/api/internal/platform/auth"
	"github.com/superset-studio/kapstan/api/internal/repositories"
)

var (
	ErrInvalidAPIKey = errors.New("invalid API key")
)

type APIKeyService struct {
	keyRepo *repositories.APIKeyRepository
}

func NewAPIKeyService(keyRepo *repositories.APIKeyRepository) *APIKeyService {
	return &APIKeyService{keyRepo: keyRepo}
}

func (s *APIKeyService) CreateAPIKey(ctx context.Context, tenantID uuid.UUID, name, accessLevel string, createdBy uuid.UUID) (*models.APIKey, string, error) {
	// Generate raw key: kap_ + 32 random hex bytes.
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return nil, "", err
	}
	rawKey := "kap_" + hex.EncodeToString(randomBytes)

	// Prefix is first 12 chars for lookup.
	prefix := rawKey[:12]

	// Bcrypt hash the full key for storage.
	keyHash, err := auth.HashPassword(rawKey)
	if err != nil {
		return nil, "", err
	}

	key := &models.APIKey{
		ID:          uuid.New(),
		TenantID:    tenantID,
		Name:        name,
		KeyPrefix:   prefix,
		KeyHash:     keyHash,
		AccessLevel: accessLevel,
		CreatedBy:   &createdBy,
	}

	if err := s.keyRepo.Create(ctx, key); err != nil {
		return nil, "", err
	}

	return key, rawKey, nil
}

func (s *APIKeyService) ListAPIKeys(ctx context.Context, tenantID uuid.UUID) ([]models.APIKey, error) {
	return s.keyRepo.ListByTenantID(ctx, tenantID)
}

func (s *APIKeyService) RevokeAPIKey(ctx context.Context, keyID uuid.UUID, tenantID uuid.UUID) error {
	return s.keyRepo.Delete(ctx, keyID, tenantID)
}

func (s *APIKeyService) ValidateAPIKey(ctx context.Context, rawKey string) (*models.APIKey, error) {
	if len(rawKey) < 12 {
		return nil, fmt.Errorf("%w: key too short", ErrInvalidAPIKey)
	}

	prefix := rawKey[:12]
	candidates, err := s.keyRepo.GetByPrefix(ctx, prefix)
	if err != nil {
		return nil, err
	}

	for i := range candidates {
		if err := auth.CheckPassword(candidates[i].KeyHash, rawKey); err == nil {
			// Update last used.
			_ = s.keyRepo.UpdateLastUsed(ctx, candidates[i].ID)
			return &candidates[i], nil
		}
	}

	return nil, ErrInvalidAPIKey
}
