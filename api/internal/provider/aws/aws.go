package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	"github.com/superset-studio/kapstan/api/internal/provider"
)

// Compile-time interface check.
var _ provider.Provider = (*AWSProvider)(nil)

// AWSProvider implements provider.Provider using AWS STS.
type AWSProvider struct {
	region    string
	credsProv aws.CredentialsProvider
}

// NewAWSProvider creates a new AWS provider with static credentials.
// If region is empty, defaults to us-east-1.
func NewAWSProvider(accessKeyID, secretAccessKey, region string) *AWSProvider {
	if region == "" {
		region = "us-east-1"
	}
	return &AWSProvider{
		region:    region,
		credsProv: credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, ""),
	}
}

// NewAWSProviderFromRole creates a new AWS provider that assumes an IAM role
// using Kapstan's ambient credentials (environment, instance profile, etc.).
// If region is empty, defaults to us-east-1.
func NewAWSProviderFromRole(roleARN, externalID, region string) (*AWSProvider, error) {
	if region == "" {
		region = "us-east-1"
	}

	// Load default config (ambient credentials) for the STS client.
	baseCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(region),
	)
	if err != nil {
		return nil, err
	}

	stsClient := sts.NewFromConfig(baseCfg)

	var opts []func(*stscreds.AssumeRoleOptions)
	if externalID != "" {
		opts = append(opts, func(o *stscreds.AssumeRoleOptions) {
			o.ExternalID = &externalID
		})
	}

	assumeRoleProv := stscreds.NewAssumeRoleProvider(stsClient, roleARN, opts...)

	return &AWSProvider{
		region:    region,
		credsProv: assumeRoleProv,
	}, nil
}

func (p *AWSProvider) stsClient(ctx context.Context) (*sts.Client, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(p.region),
		awsconfig.WithCredentialsProvider(p.credsProv),
	)
	if err != nil {
		return nil, err
	}
	return sts.NewFromConfig(cfg), nil
}

// ValidateCredentials calls STS GetCallerIdentity to verify the credentials are valid.
func (p *AWSProvider) ValidateCredentials(ctx context.Context) error {
	client, err := p.stsClient(ctx)
	if err != nil {
		return err
	}
	_, err = client.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	return err
}

// GetAccountInfo calls STS GetCallerIdentity and returns the account ID and ARN.
func (p *AWSProvider) GetAccountInfo(ctx context.Context) (*provider.AccountInfo, error) {
	client, err := p.stsClient(ctx)
	if err != nil {
		return nil, err
	}
	output, err := client.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, err
	}
	info := &provider.AccountInfo{}
	if output.Account != nil {
		info.AccountID = *output.Account
	}
	if output.Arn != nil {
		info.UserARN = *output.Arn
	}
	return info, nil
}
