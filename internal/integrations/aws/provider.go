package aws

import (
	"context"

	"github.com/JakeNeyer/ipam/internal/integrations"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

const providerID = "aws"

type Provider struct{}

// Ensure Provider implements integrations.CloudProvider and integrations.PushProvider.
var _ integrations.CloudProvider = (*Provider)(nil)
var _ integrations.PushProvider = (*Provider)(nil)

func (p *Provider) ProviderID() string        { return providerID }
func (p *Provider) SupportsPools() bool       { return true }
func (p *Provider) SupportsBlocks() bool      { return true }
func (p *Provider) SupportsAllocations() bool { return true }

func init() {
	integrations.Register(&Provider{})
}

// newEC2Client creates an EC2 client for the given region. Shared by read (via getEC2API) and write.
func newEC2Client(ctx context.Context, region string) (*ec2.Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, err
	}
	return ec2.NewFromConfig(cfg), nil
}
