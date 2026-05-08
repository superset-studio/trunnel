package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type ConnectionCategory string

const (
	ConnectionCategoryAWS ConnectionCategory = "aws"
	ConnectionCategoryGCP ConnectionCategory = "gcp"
)

type ConnectionStatus string

const (
	ConnectionStatusPending ConnectionStatus = "pending"
	ConnectionStatusValid   ConnectionStatus = "valid"
	ConnectionStatusInvalid ConnectionStatus = "invalid"
	ConnectionStatusExpired ConnectionStatus = "expired"
	ConnectionStatusPartial ConnectionStatus = "partial"
)

type Connection struct {
	ID            uuid.UUID          `db:"id" json:"id"`
	TenantID      uuid.UUID          `db:"tenant_id" json:"tenantId"`
	Name          string             `db:"name" json:"name"`
	Category      ConnectionCategory `db:"category" json:"category"`
	Status        ConnectionStatus   `db:"status" json:"status"`
	LastValidated *time.Time         `db:"last_validated" json:"lastValidated,omitempty"`
	Credentials   []byte             `db:"credentials" json:"-"`
	Config        json.RawMessage    `db:"config" json:"config,omitempty"`
	CreatedBy     *uuid.UUID         `db:"created_by" json:"createdBy,omitempty"`
	CreatedAt     time.Time          `db:"created_at" json:"createdAt"`
	UpdatedAt     time.Time          `db:"updated_at" json:"updatedAt"`
}

type AWSCredentials struct {
	AccessKeyID     string `json:"accessKeyId"`
	SecretAccessKey string `json:"secretAccessKey"`
	Region          string `json:"region"`
}

type AWSRoleCredentials struct {
	RoleARN    string `json:"roleArn"`
	ExternalID string `json:"externalId"`
	Region     string `json:"region"`
}

type AWSConfig struct {
	Region    string `json:"region"`
	AccountID string `json:"accountId,omitempty"`
}
