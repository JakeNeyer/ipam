package aws

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/JakeNeyer/ipam/store"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/google/uuid"
)

// mockEC2IPAMAPI implements EC2IPAMAPI for unit tests (see AWS SDK unit testing guide).
type mockEC2IPAMAPI struct {
	describeIpamPoolsFunc      func(context.Context, *ec2.DescribeIpamPoolsInput) ([]ec2types.IpamPool, error)
	getIpamPoolCidrsFunc       func(context.Context, string) (string, error)
	getIpamPoolAllocationsFunc func(context.Context, string) ([]ec2types.IpamPoolAllocation, error)
	describeSubnetsFunc        func(context.Context, string) ([]ec2types.Subnet, error)
}

func (m *mockEC2IPAMAPI) DescribeIpamPools(ctx context.Context, input *ec2.DescribeIpamPoolsInput) ([]ec2types.IpamPool, error) {
	if m.describeIpamPoolsFunc != nil {
		return m.describeIpamPoolsFunc(ctx, input)
	}
	return nil, nil
}

func (m *mockEC2IPAMAPI) GetIpamPoolCidrs(ctx context.Context, ipamPoolID string) (string, error) {
	if m.getIpamPoolCidrsFunc != nil {
		return m.getIpamPoolCidrsFunc(ctx, ipamPoolID)
	}
	return "10.0.0.0/8", nil
}

func (m *mockEC2IPAMAPI) GetIpamPoolAllocations(ctx context.Context, ipamPoolID string) ([]ec2types.IpamPoolAllocation, error) {
	if m.getIpamPoolAllocationsFunc != nil {
		return m.getIpamPoolAllocationsFunc(ctx, ipamPoolID)
	}
	return nil, nil
}

func (m *mockEC2IPAMAPI) DescribeSubnets(ctx context.Context, vpcID string) ([]ec2types.Subnet, error) {
	if m.describeSubnetsFunc != nil {
		return m.describeSubnetsFunc(ctx, vpcID)
	}
	return nil, nil
}

func connWithConfig(t *testing.T, region string, envID uuid.UUID, ipamScopeID string) *store.CloudConnection {
	t.Helper()
	cfg := AWSConnectionConfig{Region: region, EnvironmentID: envID, IpamScopeId: ipamScopeID}
	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return &store.CloudConnection{
		ID:             uuid.New(),
		OrganizationID: uuid.New(),
		Provider:       "aws",
		Name:           "test-conn",
		Config:         raw,
	}
}
