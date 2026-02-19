package models

import (
	"time"

	"github.com/google/uuid"
)

type RefreshToken struct {
	ID        uuid.UUID `db:"id" json:"-"`
	UserID    uuid.UUID `db:"user_id" json:"-"`
	TokenHash string    `db:"token_hash" json:"-"`
	ExpiresAt time.Time `db:"expires_at" json:"-"`
	CreatedAt time.Time `db:"created_at" json:"-"`
}
