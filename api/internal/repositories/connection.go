package repositories

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/superset-studio/kapstan/api/internal/models"
)

type ConnectionRepository struct {
	db *sqlx.DB
}

func NewConnectionRepository(db *sqlx.DB) *ConnectionRepository {
	return &ConnectionRepository{db: db}
}

func (r *ConnectionRepository) Create(ctx context.Context, conn *models.Connection) error {
	query := `INSERT INTO connections (id, tenant_id, name, category, status, credentials, config, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at, updated_at`

	err := r.db.QueryRowxContext(ctx, query,
		conn.ID, conn.TenantID, conn.Name, conn.Category, conn.Status,
		conn.Credentials, conn.Config, conn.CreatedBy,
	).Scan(&conn.CreatedAt, &conn.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrDuplicateConnectionName
		}
		return err
	}
	return nil
}

func (r *ConnectionRepository) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*models.Connection, error) {
	var conn models.Connection
	err := r.db.GetContext(ctx, &conn,
		`SELECT * FROM connections WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &conn, nil
}

func (r *ConnectionRepository) GetByIDInternal(ctx context.Context, id uuid.UUID) (*models.Connection, error) {
	var conn models.Connection
	err := r.db.GetContext(ctx, &conn, `SELECT * FROM connections WHERE id = $1`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &conn, nil
}

func (r *ConnectionRepository) ListByTenantID(ctx context.Context, tenantID uuid.UUID) ([]models.Connection, error) {
	var conns []models.Connection
	query := `SELECT id, tenant_id, name, category, status, last_validated, config, created_by, created_at, updated_at
		FROM connections WHERE tenant_id = $1 ORDER BY created_at DESC`
	err := r.db.SelectContext(ctx, &conns, query, tenantID)
	if err != nil {
		return nil, err
	}
	return conns, nil
}

func (r *ConnectionRepository) Update(ctx context.Context, conn *models.Connection) error {
	query := `UPDATE connections SET name = $1, credentials = $2, config = $3, status = $4, updated_at = now()
		WHERE id = $5 AND tenant_id = $6
		RETURNING updated_at`

	err := r.db.QueryRowxContext(ctx, query,
		conn.Name, conn.Credentials, conn.Config, conn.Status, conn.ID, conn.TenantID,
	).Scan(&conn.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if isUniqueViolation(err) {
			return ErrDuplicateConnectionName
		}
		return err
	}
	return nil
}

func (r *ConnectionRepository) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM connections WHERE id = $1 AND tenant_id = $2`, id, tenantID)
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

func (r *ConnectionRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status models.ConnectionStatus, lastValidated time.Time) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE connections SET status = $1, last_validated = $2, updated_at = now() WHERE id = $3`,
		status, lastValidated, id)
	return err
}

func (r *ConnectionRepository) UpdateStatusAndConfig(ctx context.Context, id uuid.UUID, status models.ConnectionStatus, lastValidated time.Time, config []byte) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE connections SET status = $1, last_validated = $2, config = $3, updated_at = now() WHERE id = $4`,
		status, lastValidated, config, id)
	return err
}

func (r *ConnectionRepository) ListAll(ctx context.Context) ([]models.Connection, error) {
	var conns []models.Connection
	err := r.db.SelectContext(ctx, &conns, `SELECT * FROM connections ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	return conns, nil
}
