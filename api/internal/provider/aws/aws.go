package aws

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	"github.com/superset-studio/kapstan/api/internal/provider"
)

// GetAmbientIdentity calls STS GetCallerIdentity using the server's ambient
// AWS credentials (instance profile, task role, env vars, etc.) and returns
// the account ID and ARN. This reveals Kapstan's own identity, not a
// connection's credentials.
func GetAmbientIdentity(ctx context.Context, region string) (*provider.AccountInfo, error) {
	if region == "" {
		region = "us-east-1"
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return nil, err
	}
	client := sts.NewFromConfig(cfg)
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

func (p *AWSProvider) awsConfig(ctx context.Context) (aws.Config, error) {
	return awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(p.region),
		awsconfig.WithCredentialsProvider(p.credsProv),
	)
}

func (p *AWSProvider) stsClient(ctx context.Context) (*sts.Client, error) {
	cfg, err := p.awsConfig(ctx)
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

// TestPermissions runs 6 lightweight read-only API calls concurrently to verify
// that the credentials have the required service permissions.
func (p *AWSProvider) TestPermissions(ctx context.Context) []provider.PermissionCheckResult {
	services := []string{"ec2", "eks", "vpc", "iam", "s3", "rds"}
	results := make([]provider.PermissionCheckResult, len(services))
	for i, svc := range services {
		results[i].Service = svc
	}

	cfg, err := p.awsConfig(ctx)
	if err != nil {
		errMsg := err.Error()
		for i := range results {
			results[i].Error = errMsg
		}
		return results
	}

	ec2Client := ec2.NewFromConfig(cfg)
	eksClient := eks.NewFromConfig(cfg)
	iamClient := iam.NewFromConfig(cfg)
	s3Client := s3.NewFromConfig(cfg)
	rdsClient := rds.NewFromConfig(cfg)

	var wg sync.WaitGroup
	wg.Add(len(services))

	// ec2: DescribeRegions
	go func() {
		defer wg.Done()
		_, err := ec2Client.DescribeRegions(ctx, &ec2.DescribeRegionsInput{})
		if err != nil {
			results[0].Error = err.Error()
		} else {
			results[0].Passed = true
		}
	}()

	// eks: ListClusters
	go func() {
		defer wg.Done()
		_, err := eksClient.ListClusters(ctx, &eks.ListClustersInput{})
		if err != nil {
			results[1].Error = err.Error()
		} else {
			results[1].Passed = true
		}
	}()

	// vpc: DescribeVpcs
	go func() {
		defer wg.Done()
		_, err := ec2Client.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{})
		if err != nil {
			results[2].Error = err.Error()
		} else {
			results[2].Passed = true
		}
	}()

	// iam: ListRoles (MaxItems=1)
	go func() {
		defer wg.Done()
		maxItems := int32(1)
		_, err := iamClient.ListRoles(ctx, &iam.ListRolesInput{MaxItems: &maxItems})
		if err != nil {
			results[3].Error = err.Error()
		} else {
			results[3].Passed = true
		}
	}()

	// s3: ListBuckets
	go func() {
		defer wg.Done()
		_, err := s3Client.ListBuckets(ctx, &s3.ListBucketsInput{})
		if err != nil {
			results[4].Error = err.Error()
		} else {
			results[4].Passed = true
		}
	}()

	// rds: DescribeDBInstances
	go func() {
		defer wg.Done()
		_, err := rdsClient.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{})
		if err != nil {
			results[5].Error = err.Error()
		} else {
			results[5].Passed = true
		}
	}()

	wg.Wait()
	return results
}
