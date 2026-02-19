package repositories

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/superset-studio/kapstan/api/internal/models"
)

type OrganizationRepository struct {
	db *sqlx.DB
}

func NewOrganizationRepository(db *sqlx.DB) *OrganizationRepository {
	return &OrganizationRepository{db: db}
}

func (r *OrganizationRepository) CreateTx(ctx context.Context, tx *sqlx.Tx, org *models.Organization) error {
	query := `INSERT INTO organizations (id, name, display_name, logo_url)
		VALUES ($1, $2, $3, $4)
		RETURNING created_at, updated_at`

	err := tx.QueryRowxContext(ctx, query,
		org.ID, org.Name, org.DisplayName, org.LogoURL,
	).Scan(&org.CreatedAt, &org.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrDuplicateOrgName
		}
		return err
	}
	return nil
}

func (r *OrganizationRepository) Create(ctx context.Context, org *models.Organization) error {
	query := `INSERT INTO organizations (id, name, display_name, logo_url)
		VALUES ($1, $2, $3, $4)
		RETURNING created_at, updated_at`

	err := r.db.QueryRowxContext(ctx, query,
		org.ID, org.Name, org.DisplayName, org.LogoURL,
	).Scan(&org.CreatedAt, &org.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrDuplicateOrgName
		}
		return err
	}
	return nil
}

func (r *OrganizationRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Organization, error) {
	var org models.Organization
	err := r.db.GetContext(ctx, &org, `SELECT * FROM organizations WHERE id = $1`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &org, nil
}

func (r *OrganizationRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]models.Organization, error) {
	var orgs []models.Organization
	query := `SELECT o.* FROM organizations o
		INNER JOIN organization_members om ON o.id = om.organization_id
		WHERE om.user_id = $1
		ORDER BY o.created_at DESC`
	err := r.db.SelectContext(ctx, &orgs, query, userID)
	if err != nil {
		return nil, err
	}
	return orgs, nil
}

func (r *OrganizationRepository) Update(ctx context.Context, org *models.Organization) error {
	query := `UPDATE organizations SET display_name = $1, logo_url = $2, updated_at = now()
		WHERE id = $3
		RETURNING updated_at`
	err := r.db.QueryRowxContext(ctx, query, org.DisplayName, org.LogoURL, org.ID).Scan(&org.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	return nil
}
