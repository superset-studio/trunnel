package repositories

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/superset-studio/kapstan/api/internal/models"
)

type APIKeyRepository struct {
	db *sqlx.DB
}

func NewAPIKeyRepository(db *sqlx.DB) *APIKeyRepository {
	return &APIKeyRepository{db: db}
}

func (r *APIKeyRepository) Create(ctx context.Context, key *models.APIKey) error {
	query := `INSERT INTO api_keys (id, tenant_id, workspace_id, name, key_prefix, key_hash, access_level, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at`

	err := r.db.QueryRowxContext(ctx, query,
		key.ID, key.TenantID, key.WorkspaceID, key.Name,
		key.KeyPrefix, key.KeyHash, key.AccessLevel, key.CreatedBy,
	).Scan(&key.CreatedAt)
	if err != nil {
		return err
	}
	return nil
}

func (r *APIKeyRepository) ListByTenantID(ctx context.Context, tenantID uuid.UUID) ([]models.APIKey, error) {
	var keys []models.APIKey
	query := `SELECT id, tenant_id, workspace_id, name, key_prefix, access_level, created_by, created_at, last_used_at
		FROM api_keys WHERE tenant_id = $1
		ORDER BY created_at DESC`
	err := r.db.SelectContext(ctx, &keys, query, tenantID)
	if err != nil {
		return nil, err
	}
	return keys, nil
}

func (r *APIKeyRepository) GetByPrefix(ctx context.Context, prefix string) ([]models.APIKey, error) {
	var keys []models.APIKey
	query := `SELECT * FROM api_keys WHERE key_prefix = $1`
	err := r.db.SelectContext(ctx, &keys, query, prefix)
	if err != nil {
		return nil, err
	}
	return keys, nil
}

func (r *APIKeyRepository) Delete(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM api_keys WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *APIKeyRepository) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE api_keys SET last_used_at = now() WHERE id = $1`, id)
	return err
}

func (r *APIKeyRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.APIKey, error) {
	var key models.APIKey
	err := r.db.GetContext(ctx, &key, `SELECT * FROM api_keys WHERE id = $1`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &key, nil
}
