package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID            uuid.UUID  `db:"id" json:"id"`
	Email         string     `db:"email" json:"email"`
	PasswordHash  *string    `db:"password_hash" json:"-"`
	Name          string     `db:"name" json:"name"`
	AvatarURL     *string    `db:"avatar_url" json:"avatarUrl,omitempty"`
	EmailVerified bool       `db:"email_verified" json:"emailVerified"`
	CreatedAt     time.Time  `db:"created_at" json:"createdAt"`
	UpdatedAt     time.Time  `db:"updated_at" json:"updatedAt"`
}
