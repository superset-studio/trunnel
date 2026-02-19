package repositories

import (
	"context"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/superset-studio/kapstan/api/internal/models"
)

type RefreshTokenRepository struct {
	db *sqlx.DB
}

func NewRefreshTokenRepository(db *sqlx.DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

func (r *RefreshTokenRepository) Create(ctx context.Context, token *models.RefreshToken) error {
	query := `INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at)
		VALUES ($1, $2, $3, $4)
		RETURNING created_at`

	err := r.db.QueryRowxContext(ctx, query,
		token.ID, token.UserID, token.TokenHash, token.ExpiresAt,
	).Scan(&token.CreatedAt)
	if err != nil {
		return err
	}
	return nil
}

func (r *RefreshTokenRepository) GetByHash(ctx context.Context, tokenHash string) (*models.RefreshToken, error) {
	var token models.RefreshToken
	query := `SELECT * FROM refresh_tokens WHERE token_hash = $1 AND expires_at > now()`
	err := r.db.GetContext(ctx, &token, query, tokenHash)
	if err != nil {
		return nil, err
	}
	return &token, nil
}

func (r *RefreshTokenRepository) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM refresh_tokens WHERE user_id = $1`, userID)
	return err
}

func (r *RefreshTokenRepository) DeleteExpired(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM refresh_tokens WHERE expires_at <= now()`)
	return err
}

func (r *RefreshTokenRepository) DeleteByHash(ctx context.Context, tokenHash string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM refresh_tokens WHERE token_hash = $1`, tokenHash)
	return err
}
