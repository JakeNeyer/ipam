package aws

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/JakeNeyer/ipam/network"
	"github.com/JakeNeyer/ipam/store"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/google/uuid"
)

// mockEC2WriteAPI implements ec2WriteAPI for unit tests.
type mockEC2WriteAPI struct {
	createIpamPoolFunc            func(context.Context, *ec2.CreateIpamPoolInput, ...func(*ec2.Options)) (*ec2.CreateIpamPoolOutput, error)
	deleteIpamPoolFunc            func(context.Context, *ec2.DeleteIpamPoolInput, ...func(*ec2.Options)) (*ec2.DeleteIpamPoolOutput, error)
	describeIpamPoolsFunc         func(context.Context, *ec2.DescribeIpamPoolsInput, ...func(*ec2.Options)) (*ec2.DescribeIpamPoolsOutput, error)
	getIpamPoolCidrsFunc          func(context.Context, *ec2.GetIpamPoolCidrsInput, ...func(*ec2.Options)) (*ec2.GetIpamPoolCidrsOutput, error)
	getIpamPoolAllocationsFunc    func(context.Context, *ec2.GetIpamPoolAllocationsInput, ...func(*ec2.Options)) (*ec2.GetIpamPoolAllocationsOutput, error)
	provisionIpamPoolCidrFunc     func(context.Context, *ec2.ProvisionIpamPoolCidrInput, ...func(*ec2.Options)) (*ec2.ProvisionIpamPoolCidrOutput, error)
	allocateIpamPoolCidrFunc      func(context.Context, *ec2.AllocateIpamPoolCidrInput, ...func(*ec2.Options)) (*ec2.AllocateIpamPoolCidrOutput, error)
	releaseIpamPoolAllocationFunc func(context.Context, *ec2.ReleaseIpamPoolAllocationInput, ...func(*ec2.Options)) (*ec2.ReleaseIpamPoolAllocationOutput, error)
	describeSubnetsFunc           func(context.Context, *ec2.DescribeSubnetsInput, ...func(*ec2.Options)) (*ec2.DescribeSubnetsOutput, error)
	createVpcFunc                 func(context.Context, *ec2.CreateVpcInput, ...func(*ec2.Options)) (*ec2.CreateVpcOutput, error)
	describeVpcsFunc              func(context.Context, *ec2.DescribeVpcsInput, ...func(*ec2.Options)) (*ec2.DescribeVpcsOutput, error)
	deleteVpcFunc                 func(context.Context, *ec2.DeleteVpcInput, ...func(*ec2.Options)) (*ec2.DeleteVpcOutput, error)
	createSubnetFunc              func(context.Context, *ec2.CreateSubnetInput, ...func(*ec2.Options)) (*ec2.CreateSubnetOutput, error)
	deleteSubnetFunc              func(context.Context, *ec2.DeleteSubnetInput, ...func(*ec2.Options)) (*ec2.DeleteSubnetOutput, error)
}

func (m *mockEC2WriteAPI) CreateIpamPool(ctx context.Context, params *ec2.CreateIpamPoolInput, optFns ...func(*ec2.Options)) (*ec2.CreateIpamPoolOutput, error) {
	if m.createIpamPoolFunc != nil {
		return m.createIpamPoolFunc(ctx, params, optFns...)
	}
	return &ec2.CreateIpamPoolOutput{IpamPool: &ec2types.IpamPool{IpamPoolId: aws.String("ipam-pool-mock")}}, nil
}

func (m *mockEC2WriteAPI) DeleteIpamPool(ctx context.Context, params *ec2.DeleteIpamPoolInput, optFns ...func(*ec2.Options)) (*ec2.DeleteIpamPoolOutput, error) {
	if m.deleteIpamPoolFunc != nil {
		return m.deleteIpamPoolFunc(ctx, params, optFns...)
	}
	return &ec2.DeleteIpamPoolOutput{}, nil
}

func (m *mockEC2WriteAPI) DescribeIpamPools(ctx context.Context, params *ec2.DescribeIpamPoolsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeIpamPoolsOutput, error) {
	if m.describeIpamPoolsFunc != nil {
		return m.describeIpamPoolsFunc(ctx, params, optFns...)
	}
	// Default: pool is create-complete so wait returns immediately.
	if params != nil && len(params.IpamPoolIds) > 0 {
		return &ec2.DescribeIpamPoolsOutput{
			IpamPools: []ec2types.IpamPool{{
				IpamPoolId: aws.String(params.IpamPoolIds[0]),
				State:      ec2types.IpamPoolStateCreateComplete,
			}},
		}, nil
	}
	return &ec2.DescribeIpamPoolsOutput{IpamPools: nil}, nil
}

func (m *mockEC2WriteAPI) GetIpamPoolCidrs(ctx context.Context, params *ec2.GetIpamPoolCidrsInput, optFns ...func(*ec2.Options)) (*ec2.GetIpamPoolCidrsOutput, error) {
	if m.getIpamPoolCidrsFunc != nil {
		return m.getIpamPoolCidrsFunc(ctx, params, optFns...)
	}
	// Default: return common provisioned CIDRs so waitForIpamPoolCidrProvisioned returns for typical test pools.
	if params != nil && params.IpamPoolId != nil {
		return &ec2.GetIpamPoolCidrsOutput{
			IpamPoolCidrs: []ec2types.IpamPoolCidr{
				{Cidr: aws.String("10.0.0.0/8"), State: ec2types.IpamPoolCidrStateProvisioned},
				{Cidr: aws.String("10.0.0.0/16"), State: ec2types.IpamPoolCidrStateProvisioned},
			},
		}, nil
	}
	return &ec2.GetIpamPoolCidrsOutput{IpamPoolCidrs: nil}, nil
}

func (m *mockEC2WriteAPI) GetIpamPoolAllocations(ctx context.Context, params *ec2.GetIpamPoolAllocationsInput, optFns ...func(*ec2.Options)) (*ec2.GetIpamPoolAllocationsOutput, error) {
	if m.getIpamPoolAllocationsFunc != nil {
		return m.getIpamPoolAllocationsFunc(ctx, params, optFns...)
	}
	return &ec2.GetIpamPoolAllocationsOutput{IpamPoolAllocations: nil}, nil
}

func (m *mockEC2WriteAPI) DescribeSubnets(ctx context.Context, params *ec2.DescribeSubnetsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSubnetsOutput, error) {
	if m.describeSubnetsFunc != nil {
		return m.describeSubnetsFunc(ctx, params, optFns...)
	}
	return &ec2.DescribeSubnetsOutput{Subnets: nil}, nil
}

func (m *mockEC2WriteAPI) ProvisionIpamPoolCidr(ctx context.Context, params *ec2.ProvisionIpamPoolCidrInput, optFns ...func(*ec2.Options)) (*ec2.ProvisionIpamPoolCidrOutput, error) {
	if m.provisionIpamPoolCidrFunc != nil {
		return m.provisionIpamPoolCidrFunc(ctx, params, optFns...)
	}
	return &ec2.ProvisionIpamPoolCidrOutput{}, nil
}

func (m *mockEC2WriteAPI) AllocateIpamPoolCidr(ctx context.Context, params *ec2.AllocateIpamPoolCidrInput, optFns ...func(*ec2.Options)) (*ec2.AllocateIpamPoolCidrOutput, error) {
	if m.allocateIpamPoolCidrFunc != nil {
		return m.allocateIpamPoolCidrFunc(ctx, params, optFns...)
	}
	return &ec2.AllocateIpamPoolCidrOutput{}, nil
}

func (m *mockEC2WriteAPI) ReleaseIpamPoolAllocation(ctx context.Context, params *ec2.ReleaseIpamPoolAllocationInput, optFns ...func(*ec2.Options)) (*ec2.ReleaseIpamPoolAllocationOutput, error) {
	if m.releaseIpamPoolAllocationFunc != nil {
		return m.releaseIpamPoolAllocationFunc(ctx, params, optFns...)
	}
	return &ec2.ReleaseIpamPoolAllocationOutput{Success: aws.Bool(true)}, nil
}

func (m *mockEC2WriteAPI) CreateVpc(ctx context.Context, params *ec2.CreateVpcInput, optFns ...func(*ec2.Options)) (*ec2.CreateVpcOutput, error) {
	if m.createVpcFunc != nil {
		return m.createVpcFunc(ctx, params, optFns...)
	}
	return &ec2.CreateVpcOutput{Vpc: &ec2types.Vpc{VpcId: aws.String("vpc-mock")}}, nil
}

func (m *mockEC2WriteAPI) DescribeVpcs(ctx context.Context, params *ec2.DescribeVpcsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcsOutput, error) {
	if m.describeVpcsFunc != nil {
		return m.describeVpcsFunc(ctx, params, optFns...)
	}
	return &ec2.DescribeVpcsOutput{Vpcs: nil}, nil
}

func (m *mockEC2WriteAPI) CreateSubnet(ctx context.Context, params *ec2.CreateSubnetInput, optFns ...func(*ec2.Options)) (*ec2.CreateSubnetOutput, error) {
	if m.createSubnetFunc != nil {
		return m.createSubnetFunc(ctx, params, optFns...)
	}
	return &ec2.CreateSubnetOutput{Subnet: &ec2types.Subnet{SubnetId: aws.String("subnet-mock")}}, nil
}

func (m *mockEC2WriteAPI) DeleteVpc(ctx context.Context, params *ec2.DeleteVpcInput, optFns ...func(*ec2.Options)) (*ec2.DeleteVpcOutput, error) {
	if m.deleteVpcFunc != nil {
		return m.deleteVpcFunc(ctx, params, optFns...)
	}
	return &ec2.DeleteVpcOutput{}, nil
}

func (m *mockEC2WriteAPI) DeleteSubnet(ctx context.Context, params *ec2.DeleteSubnetInput, optFns ...func(*ec2.Options)) (*ec2.DeleteSubnetOutput, error) {
	if m.deleteSubnetFunc != nil {
		return m.deleteSubnetFunc(ctx, params, optFns...)
	}
	return &ec2.DeleteSubnetOutput{}, nil
}

func writeConnWithConfig(t *testing.T, region string, ipamScopeID string) *store.CloudConnection {
	t.Helper()
	cfg := AWSConnectionConfig{Region: region, IpamScopeId: ipamScopeID}
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

func TestSupportsPush(t *testing.T) {
	provider := &Provider{}
	if !provider.SupportsPush() {
		t.Error("SupportsPush() = false, want true")
	}
}

func TestCreatePoolInCloud(t *testing.T) {
	ctx := context.Background()
	conn := writeConnWithConfig(t, "us-east-1", "ipam-scope-123")
	pool := &network.Pool{Name: "my-pool", CIDR: "10.0.0.0/8"}

	wantID := "ipam-pool-created"
	mock := &mockEC2WriteAPI{
		createIpamPoolFunc: func(_ context.Context, input *ec2.CreateIpamPoolInput, _ ...func(*ec2.Options)) (*ec2.CreateIpamPoolOutput, error) {
			if aws.ToString(input.IpamScopeId) != "ipam-scope-123" {
				t.Errorf("IpamScopeId = %q, want ipam-scope-123", aws.ToString(input.IpamScopeId))
			}
			if aws.ToString(input.Description) != "my-pool" {
				t.Errorf("Description = %q, want my-pool", aws.ToString(input.Description))
			}
			return &ec2.CreateIpamPoolOutput{
				IpamPool: &ec2types.IpamPool{IpamPoolId: aws.String(wantID)},
			}, nil
		},
	}
	ec2WriteAPIForTest = mock
	defer func() { ec2WriteAPIForTest = nil }()

	provider := &Provider{}
	got, err := provider.CreatePoolInCloud(ctx, conn, pool, "")
	if err != nil {
		t.Fatalf("CreatePoolInCloud: %v", err)
	}
	if got != wantID {
		t.Errorf("CreatePoolInCloud = %q, want %q", got, wantID)
	}
}

func TestCreatePoolInCloud_WithParent(t *testing.T) {
	ctx := context.Background()
	conn := writeConnWithConfig(t, "us-east-1", "ipam-scope-123")
	pool := &network.Pool{Name: "child-pool", CIDR: "10.0.0.0/16"}
	parentExtID := "ipam-pool-parent"

	mock := &mockEC2WriteAPI{
		createIpamPoolFunc: func(_ context.Context, input *ec2.CreateIpamPoolInput, _ ...func(*ec2.Options)) (*ec2.CreateIpamPoolOutput, error) {
			if aws.ToString(input.SourceIpamPoolId) != parentExtID {
				t.Errorf("SourceIpamPoolId = %q, want %q", aws.ToString(input.SourceIpamPoolId), parentExtID)
			}
			return &ec2.CreateIpamPoolOutput{
				IpamPool: &ec2types.IpamPool{IpamPoolId: aws.String("ipam-pool-child")},
			}, nil
		},
	}
	ec2WriteAPIForTest = mock
	defer func() { ec2WriteAPIForTest = nil }()

	provider := &Provider{}
	got, err := provider.CreatePoolInCloud(ctx, conn, pool, parentExtID)
	if err != nil {
		t.Fatalf("CreatePoolInCloud: %v", err)
	}
	if got != "ipam-pool-child" {
		t.Errorf("CreatePoolInCloud = %q, want ipam-pool-child", got)
	}
}

func TestCreatePoolInCloud_InvalidConfig(t *testing.T) {
	ctx := context.Background()
	provider := &Provider{}
	pool := &network.Pool{Name: "p", CIDR: "10.0.0.0/8"}

	t.Run("missing region", func(t *testing.T) {
		conn := writeConnWithConfig(t, "", "ipam-scope-123")
		_, err := provider.CreatePoolInCloud(ctx, conn, pool, "")
		if err == nil {
			t.Error("CreatePoolInCloud: expected error for missing region")
		}
	})

	t.Run("missing ipam_scope_id", func(t *testing.T) {
		conn := writeConnWithConfig(t, "us-east-1", "")
		_, err := provider.CreatePoolInCloud(ctx, conn, pool, "")
		if err == nil {
			t.Error("CreatePoolInCloud: expected error for missing ipam_scope_id")
		}
	})
}

func TestCreatePoolInCloud_APIError(t *testing.T) {
	ctx := context.Background()
	conn := writeConnWithConfig(t, "us-east-1", "ipam-scope-123")
	pool := &network.Pool{Name: "p", CIDR: "10.0.0.0/8"}
	wantErr := errors.New("api denied")
	mock := &mockEC2WriteAPI{
		createIpamPoolFunc: func(_ context.Context, _ *ec2.CreateIpamPoolInput, _ ...func(*ec2.Options)) (*ec2.CreateIpamPoolOutput, error) {
			return nil, wantErr
		},
	}
	ec2WriteAPIForTest = mock
	defer func() { ec2WriteAPIForTest = nil }()

	provider := &Provider{}
	_, err := provider.CreatePoolInCloud(ctx, conn, pool, "")
	if err == nil {
		t.Fatal("CreatePoolInCloud: expected error")
	}
	if !errors.Is(err, wantErr) && err.Error() == "" {
		t.Errorf("CreatePoolInCloud: want error containing %q, got %v", wantErr, err)
	}
}

func TestCreatePoolInCloud_PoolCreateFailed(t *testing.T) {
	ctx := context.Background()
	conn := writeConnWithConfig(t, "us-east-1", "ipam-scope-123")
	pool := &network.Pool{Name: "p", CIDR: "10.0.0.0/8"}
	poolID := "ipam-pool-failed"
	mock := &mockEC2WriteAPI{
		createIpamPoolFunc: func(_ context.Context, _ *ec2.CreateIpamPoolInput, _ ...func(*ec2.Options)) (*ec2.CreateIpamPoolOutput, error) {
			return &ec2.CreateIpamPoolOutput{
				IpamPool: &ec2types.IpamPool{IpamPoolId: aws.String(poolID)},
			}, nil
		},
		describeIpamPoolsFunc: func(_ context.Context, input *ec2.DescribeIpamPoolsInput, _ ...func(*ec2.Options)) (*ec2.DescribeIpamPoolsOutput, error) {
			return &ec2.DescribeIpamPoolsOutput{
				IpamPools: []ec2types.IpamPool{{
					IpamPoolId:   aws.String(poolID),
					State:        ec2types.IpamPoolStateCreateFailed,
					StateMessage: aws.String("quota exceeded"),
				}},
			}, nil
		},
	}
	ec2WriteAPIForTest = mock
	defer func() { ec2WriteAPIForTest = nil }()

	provider := &Provider{}
	_, err := provider.CreatePoolInCloud(ctx, conn, pool, "")
	if err == nil {
		t.Fatal("CreatePoolInCloud: expected error when pool state is create-failed")
	}
	if !strings.Contains(err.Error(), "quota exceeded") && !strings.Contains(err.Error(), poolID) {
		t.Errorf("CreatePoolInCloud: want error mentioning pool or message, got %v", err)
	}
}

func TestAllocateBlockInCloud(t *testing.T) {
	ctx := context.Background()
	conn := writeConnWithConfig(t, "us-east-1", "")
	poolExternalID := "ipam-pool-abc"
	block := &network.Block{Name: "my-vpc", CIDR: "10.1.0.0/16"}

	wantVpcID := "vpc-created"
	mock := &mockEC2WriteAPI{
		allocateIpamPoolCidrFunc: func(_ context.Context, input *ec2.AllocateIpamPoolCidrInput, _ ...func(*ec2.Options)) (*ec2.AllocateIpamPoolCidrOutput, error) {
			if aws.ToString(input.IpamPoolId) != poolExternalID {
				t.Errorf("IpamPoolId = %q, want %q", aws.ToString(input.IpamPoolId), poolExternalID)
			}
			if aws.ToString(input.Cidr) != block.CIDR {
				t.Errorf("Cidr = %q, want %q", aws.ToString(input.Cidr), block.CIDR)
			}
			return &ec2.AllocateIpamPoolCidrOutput{}, nil
		},
		createVpcFunc: func(_ context.Context, input *ec2.CreateVpcInput, _ ...func(*ec2.Options)) (*ec2.CreateVpcOutput, error) {
			if aws.ToString(input.CidrBlock) != block.CIDR {
				t.Errorf("CreateVpc CidrBlock = %q, want %q", aws.ToString(input.CidrBlock), block.CIDR)
			}
			return &ec2.CreateVpcOutput{
				Vpc: &ec2types.Vpc{VpcId: aws.String(wantVpcID)},
			}, nil
		},
	}
	ec2WriteAPIForTest = mock
	defer func() { ec2WriteAPIForTest = nil }()

	provider := &Provider{}
	got, err := provider.AllocateBlockInCloud(ctx, conn, poolExternalID, block)
	if err != nil {
		t.Fatalf("AllocateBlockInCloud: %v", err)
	}
	if got != wantVpcID {
		t.Errorf("AllocateBlockInCloud = %q, want %q", got, wantVpcID)
	}
}

func TestAllocateBlockInCloud_InvalidConfig(t *testing.T) {
	ctx := context.Background()
	provider := &Provider{}
	block := &network.Block{Name: "v", CIDR: "10.0.0.0/16"}
	conn := writeConnWithConfig(t, "", "")
	ec2WriteAPIForTest = &mockEC2WriteAPI{}
	defer func() { ec2WriteAPIForTest = nil }()

	_, err := provider.AllocateBlockInCloud(ctx, conn, "ipam-pool-x", block)
	if err == nil {
		t.Error("AllocateBlockInCloud: expected error for missing region")
	}
}

// TestAllocateBlockInCloud_ReuseExistingVpc verifies that when IPAM has an allocation and the VPC exists, we reuse it.
func TestAllocateBlockInCloud_ReuseExistingVpc(t *testing.T) {
	ctx := context.Background()
	conn := writeConnWithConfig(t, "us-east-1", "")
	poolExternalID := "ipam-pool-abc"
	block := &network.Block{Name: "my-vpc", CIDR: "10.1.0.0/16"}
	existingVpcID := "vpc-existing"

	mock := &mockEC2WriteAPI{
		getIpamPoolAllocationsFunc: func(_ context.Context, input *ec2.GetIpamPoolAllocationsInput, _ ...func(*ec2.Options)) (*ec2.GetIpamPoolAllocationsOutput, error) {
			return &ec2.GetIpamPoolAllocationsOutput{
				IpamPoolAllocations: []ec2types.IpamPoolAllocation{{
					Cidr:         aws.String(block.CIDR),
					ResourceId:   aws.String(existingVpcID),
					ResourceType: ec2types.IpamPoolAllocationResourceTypeVpc,
				}},
			}, nil
		},
		describeVpcsFunc: func(_ context.Context, input *ec2.DescribeVpcsInput, _ ...func(*ec2.Options)) (*ec2.DescribeVpcsOutput, error) {
			return &ec2.DescribeVpcsOutput{
				Vpcs: []ec2types.Vpc{{
					VpcId: aws.String(existingVpcID),
					State: ec2types.VpcStateAvailable,
				}},
			}, nil
		},
	}
	ec2WriteAPIForTest = mock
	defer func() { ec2WriteAPIForTest = nil }()

	provider := &Provider{}
	got, err := provider.AllocateBlockInCloud(ctx, conn, poolExternalID, block)
	if err != nil {
		t.Fatalf("AllocateBlockInCloud: %v", err)
	}
	if got != existingVpcID {
		t.Errorf("AllocateBlockInCloud = %q, want %q (should reuse existing VPC)", got, existingVpcID)
	}
}

// TestAllocateBlockInCloud_StaleVpcAllocation verifies that when IPAM has an allocation for a deleted VPC,
// we release the orphaned allocation and re-allocate + create the VPC.
func TestAllocateBlockInCloud_StaleVpcAllocation(t *testing.T) {
	ctx := context.Background()
	conn := writeConnWithConfig(t, "us-east-1", "")
	poolExternalID := "ipam-pool-abc"
	block := &network.Block{Name: "my-vpc", CIDR: "10.1.0.0/16"}
	deletedVpcID := "vpc-deleted"
	allocID := "ipam-pool-alloc-orphaned"

	wantVpcID := "vpc-recreated"
	mock := &mockEC2WriteAPI{
		getIpamPoolAllocationsFunc: func(_ context.Context, input *ec2.GetIpamPoolAllocationsInput, _ ...func(*ec2.Options)) (*ec2.GetIpamPoolAllocationsOutput, error) {
			return &ec2.GetIpamPoolAllocationsOutput{
				IpamPoolAllocations: []ec2types.IpamPoolAllocation{{
					Cidr:                 aws.String(block.CIDR),
					ResourceId:           aws.String(deletedVpcID),
					ResourceType:         ec2types.IpamPoolAllocationResourceTypeVpc,
					IpamPoolAllocationId: aws.String(allocID),
				}},
			}, nil
		},
		releaseIpamPoolAllocationFunc: func(_ context.Context, input *ec2.ReleaseIpamPoolAllocationInput, _ ...func(*ec2.Options)) (*ec2.ReleaseIpamPoolAllocationOutput, error) {
			if aws.ToString(input.IpamPoolAllocationId) != allocID || aws.ToString(input.Cidr) != block.CIDR {
				t.Errorf("ReleaseIpamPoolAllocation: want allocation %q and CIDR %q", allocID, block.CIDR)
			}
			return &ec2.ReleaseIpamPoolAllocationOutput{Success: aws.Bool(true)}, nil
		},
		describeVpcsFunc: func(_ context.Context, input *ec2.DescribeVpcsInput, _ ...func(*ec2.Options)) (*ec2.DescribeVpcsOutput, error) {
			// VPC was deleted; DescribeVpcs returns empty
			return &ec2.DescribeVpcsOutput{Vpcs: nil}, nil
		},
		allocateIpamPoolCidrFunc: func(_ context.Context, input *ec2.AllocateIpamPoolCidrInput, _ ...func(*ec2.Options)) (*ec2.AllocateIpamPoolCidrOutput, error) {
			return &ec2.AllocateIpamPoolCidrOutput{}, nil
		},
		createVpcFunc: func(_ context.Context, input *ec2.CreateVpcInput, _ ...func(*ec2.Options)) (*ec2.CreateVpcOutput, error) {
			return &ec2.CreateVpcOutput{
				Vpc: &ec2types.Vpc{VpcId: aws.String(wantVpcID)},
			}, nil
		},
	}
	ec2WriteAPIForTest = mock
	defer func() { ec2WriteAPIForTest = nil }()

	provider := &Provider{}
	got, err := provider.AllocateBlockInCloud(ctx, conn, poolExternalID, block)
	if err != nil {
		t.Fatalf("AllocateBlockInCloud: %v", err)
	}
	if got != wantVpcID {
		t.Errorf("AllocateBlockInCloud = %q, want %q (should re-create VPC, not return stale %q)", got, wantVpcID, deletedVpcID)
	}
}

func TestCreateAllocationInCloud(t *testing.T) {
	ctx := context.Background()
	conn := writeConnWithConfig(t, "us-east-1", "")
	blockExternalID := "vpc-123"
	alloc := &network.Allocation{
		Name:  "my-subnet",
		Block: network.Block{Name: "my-vpc", CIDR: "10.1.1.0/24"},
	}

	wantSubnetID := "subnet-created"
	mock := &mockEC2WriteAPI{
		createSubnetFunc: func(_ context.Context, input *ec2.CreateSubnetInput, _ ...func(*ec2.Options)) (*ec2.CreateSubnetOutput, error) {
			if aws.ToString(input.VpcId) != blockExternalID {
				t.Errorf("VpcId = %q, want %q", aws.ToString(input.VpcId), blockExternalID)
			}
			if aws.ToString(input.CidrBlock) != alloc.Block.CIDR {
				t.Errorf("CidrBlock = %q, want %q", aws.ToString(input.CidrBlock), alloc.Block.CIDR)
			}
			return &ec2.CreateSubnetOutput{
				Subnet: &ec2types.Subnet{SubnetId: aws.String(wantSubnetID)},
			}, nil
		},
	}
	ec2WriteAPIForTest = mock
	defer func() { ec2WriteAPIForTest = nil }()

	provider := &Provider{}
	got, err := provider.CreateAllocationInCloud(ctx, conn, blockExternalID, alloc)
	if err != nil {
		t.Fatalf("CreateAllocationInCloud: %v", err)
	}
	if got != wantSubnetID {
		t.Errorf("CreateAllocationInCloud = %q, want %q", got, wantSubnetID)
	}
}

func TestCreateAllocationInCloud_NoCIDR(t *testing.T) {
	ctx := context.Background()
	conn := writeConnWithConfig(t, "us-east-1", "")
	alloc := &network.Allocation{Name: "s", Block: network.Block{CIDR: ""}}
	ec2WriteAPIForTest = &mockEC2WriteAPI{}
	defer func() { ec2WriteAPIForTest = nil }()

	provider := &Provider{}
	_, err := provider.CreateAllocationInCloud(ctx, conn, "vpc-123", alloc)
	if err == nil {
		t.Error("CreateAllocationInCloud: expected error for allocation with no CIDR")
	}
}

func TestCreateAllocationInCloud_InvalidConfig(t *testing.T) {
	ctx := context.Background()
	provider := &Provider{}
	alloc := &network.Allocation{Name: "s", Block: network.Block{CIDR: "10.1.1.0/24"}}
	conn := writeConnWithConfig(t, "", "")
	ec2WriteAPIForTest = &mockEC2WriteAPI{}
	defer func() { ec2WriteAPIForTest = nil }()

	_, err := provider.CreateAllocationInCloud(ctx, conn, "vpc-123", alloc)
	if err == nil {
		t.Error("CreateAllocationInCloud: expected error for missing region")
	}
}
