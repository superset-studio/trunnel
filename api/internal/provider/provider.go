package provider

import "context"

// AccountInfo contains account details returned by cloud providers.
type AccountInfo struct {
	AccountID string
	UserARN   string
}

// Provider is the interface for cloud provider operations.
type Provider interface {
	ValidateCredentials(ctx context.Context) error
	GetAccountInfo(ctx context.Context) (*AccountInfo, error)
}
