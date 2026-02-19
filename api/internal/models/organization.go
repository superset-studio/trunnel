package models

import (
	"time"

	"github.com/google/uuid"
)

type Organization struct {
	ID          uuid.UUID  `db:"id" json:"id"`
	Name        string     `db:"name" json:"name"`
	DisplayName string     `db:"display_name" json:"displayName"`
	LogoURL     *string    `db:"logo_url" json:"logoUrl,omitempty"`
	CreatedAt   time.Time  `db:"created_at" json:"createdAt"`
	UpdatedAt   time.Time  `db:"updated_at" json:"updatedAt"`
}
