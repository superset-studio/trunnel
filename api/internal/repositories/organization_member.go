package repositories

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/superset-studio/kapstan/api/internal/models"
)

type OrganizationMemberRepository struct {
	db *sqlx.DB
}

func NewOrganizationMemberRepository(db *sqlx.DB) *OrganizationMemberRepository {
	return &OrganizationMemberRepository{db: db}
}

func (r *OrganizationMemberRepository) CreateTx(ctx context.Context, tx *sqlx.Tx, member *models.OrganizationMember) error {
	query := `INSERT INTO organization_members (id, organization_id, user_id, role, invited_by, invited_at, accepted_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at`

	err := tx.QueryRowxContext(ctx, query,
		member.ID, member.OrganizationID, member.UserID, member.Role,
		member.InvitedBy, member.InvitedAt, member.AcceptedAt,
	).Scan(&member.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrDuplicateMembership
		}
		return err
	}
	return nil
}

func (r *OrganizationMemberRepository) Create(ctx context.Context, member *models.OrganizationMember) error {
	query := `INSERT INTO organization_members (id, organization_id, user_id, role, invited_by, invited_at, accepted_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at`

	err := r.db.QueryRowxContext(ctx, query,
		member.ID, member.OrganizationID, member.UserID, member.Role,
		member.InvitedBy, member.InvitedAt, member.AcceptedAt,
	).Scan(&member.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrDuplicateMembership
		}
		return err
	}
	return nil
}

func (r *OrganizationMemberRepository) ListByOrgID(ctx context.Context, orgID uuid.UUID) ([]models.MemberWithUser, error) {
	var members []models.MemberWithUser
	query := `SELECT om.*, u.email, u.name, u.avatar_url
		FROM organization_members om
		INNER JOIN users u ON om.user_id = u.id
		WHERE om.organization_id = $1
		ORDER BY om.created_at ASC`
	err := r.db.SelectContext(ctx, &members, query, orgID)
	if err != nil {
		return nil, err
	}
	return members, nil
}

func (r *OrganizationMemberRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]models.OrganizationMember, error) {
	var members []models.OrganizationMember
	query := `SELECT * FROM organization_members WHERE user_id = $1`
	err := r.db.SelectContext(ctx, &members, query, userID)
	if err != nil {
		return nil, err
	}
	return members, nil
}

func (r *OrganizationMemberRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.OrganizationMember, error) {
	var member models.OrganizationMember
	err := r.db.GetContext(ctx, &member, `SELECT * FROM organization_members WHERE id = $1`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &member, nil
}

func (r *OrganizationMemberRepository) UpdateRole(ctx context.Context, id uuid.UUID, role models.Role) error {
	result, err := r.db.ExecContext(ctx, `UPDATE organization_members SET role = $1 WHERE id = $2`, role, id)
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

func (r *OrganizationMemberRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM organization_members WHERE id = $1`, id)
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

func (r *OrganizationMemberRepository) CountOwners(ctx context.Context, orgID uuid.UUID) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count,
		`SELECT COUNT(*) FROM organization_members WHERE organization_id = $1 AND role = 'owner'`, orgID)
	if err != nil {
		return 0, err
	}
	return count, nil
}
