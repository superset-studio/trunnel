package provider

import "context"

// AccountInfo contains account details returned by cloud providers.
type AccountInfo struct {
	AccountID string
	UserARN   string
}

// PermissionCheckResult reports pass/fail for a single AWS service permission check.
type PermissionCheckResult struct {
	Service string `json:"service"` // "ec2", "eks", "vpc", "iam", "s3", "rds"
	Passed  bool   `json:"passed"`
	Error   string `json:"error,omitempty"` // empty if passed
}

// Provider is the interface for cloud provider operations.
type Provider interface {
	ValidateCredentials(ctx context.Context) error
	GetAccountInfo(ctx context.Context) (*AccountInfo, error)
	TestPermissions(ctx context.Context) []PermissionCheckResult
}
