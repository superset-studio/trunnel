package models

import (
	"time"

	"github.com/google/uuid"
)

type APIKey struct {
	ID          uuid.UUID  `db:"id" json:"id"`
	TenantID    uuid.UUID  `db:"tenant_id" json:"tenantId"`
	WorkspaceID *uuid.UUID `db:"workspace_id" json:"workspaceId,omitempty"`
	Name        string     `db:"name" json:"name"`
	KeyPrefix   string     `db:"key_prefix" json:"keyPrefix"`
	KeyHash     string     `db:"key_hash" json:"-"`
	AccessLevel string     `db:"access_level" json:"accessLevel"`
	CreatedBy   *uuid.UUID `db:"created_by" json:"createdBy,omitempty"`
	CreatedAt   time.Time  `db:"created_at" json:"createdAt"`
	LastUsedAt  *time.Time `db:"last_used_at" json:"lastUsedAt,omitempty"`
}
