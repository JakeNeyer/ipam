package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// EC2IPAMAPI abstracts EC2/IPAM API operations used by the provider for sync.
// Use this interface for dependency injection so tests can mock API calls.
// See: https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/unit-testing.html
type EC2IPAMAPI interface {
	DescribeIpamPools(ctx context.Context, input *ec2.DescribeIpamPoolsInput) ([]ec2types.IpamPool, error)
	GetIpamPoolCidrs(ctx context.Context, ipamPoolID string) (string, error)
	GetIpamPoolAllocations(ctx context.Context, ipamPoolID string) ([]ec2types.IpamPoolAllocation, error)
	DescribeSubnets(ctx context.Context, vpcID string) ([]ec2types.Subnet, error)
}

// ec2APIAdapter wraps *ec2.Client and implements EC2IPAMAPI (handles pagination).
type ec2APIAdapter struct {
	client *ec2.Client
}

func (a *ec2APIAdapter) DescribeIpamPools(ctx context.Context, input *ec2.DescribeIpamPoolsInput) ([]ec2types.IpamPool, error) {
	var out []ec2types.IpamPool
	pager := ec2.NewDescribeIpamPoolsPaginator(a.client, input)
	for pager.HasMorePages() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		out = append(out, page.IpamPools...)
	}
	return out, nil
}

func (a *ec2APIAdapter) GetIpamPoolCidrs(ctx context.Context, ipamPoolID string) (string, error) {
	return getIpamPoolCidrWithClient(ctx, a.client, ipamPoolID)
}

func (a *ec2APIAdapter) GetIpamPoolAllocations(ctx context.Context, ipamPoolID string) ([]ec2types.IpamPoolAllocation, error) {
	return getIpamPoolAllocationsWithClient(ctx, a.client, ipamPoolID)
}

func (a *ec2APIAdapter) DescribeSubnets(ctx context.Context, vpcID string) ([]ec2types.Subnet, error) {
	return describeSubnetsByVPCWithClient(ctx, a.client, vpcID)
}

// getEC2API returns the EC2 IPAM API for the given region. Tests can set ec2APIForTest to inject a mock.
var getEC2API = func(ctx context.Context, region string) (EC2IPAMAPI, error) {
	if ec2APIForTest != nil {
		return ec2APIForTest, nil
	}
	client, err := newEC2Client(ctx, region)
	if err != nil {
		return nil, err
	}
	return &ec2APIAdapter{client: client}, nil
}

// ec2APIForTest is set by tests to inject a mock; must be reset after each test.
var ec2APIForTest EC2IPAMAPI
