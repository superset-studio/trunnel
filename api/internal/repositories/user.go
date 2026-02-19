package repositories

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/superset-studio/kapstan/api/internal/models"
)

type UserRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) CreateTx(ctx context.Context, tx *sqlx.Tx, user *models.User) error {
	query := `INSERT INTO users (id, email, password_hash, name, avatar_url, email_verified)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at, updated_at`

	err := tx.QueryRowxContext(ctx, query,
		user.ID, user.Email, user.PasswordHash, user.Name, user.AvatarURL, user.EmailVerified,
	).Scan(&user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) && strings.Contains(err.Error(), "email") {
			return ErrDuplicateEmail
		}
		return err
	}
	return nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var user models.User
	err := r.db.GetContext(ctx, &user, `SELECT * FROM users WHERE id = $1`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := r.db.GetContext(ctx, &user, `SELECT * FROM users WHERE email = $1`, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetByEmailTx(ctx context.Context, tx *sqlx.Tx, email string) (*models.User, error) {
	var user models.User
	err := tx.GetContext(ctx, &user, `SELECT * FROM users WHERE email = $1`, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}
