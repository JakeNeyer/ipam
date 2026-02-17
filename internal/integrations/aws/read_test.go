package aws

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/JakeNeyer/ipam/network"
	"github.com/JakeNeyer/ipam/store"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/google/uuid"
)

func TestSyncPools(t *testing.T) {
	ctx := context.Background()
	envID := uuid.New()
	conn := connWithConfig(t, "us-east-1", envID, "")

	poolID := "ipam-pool-abc123"
	poolName := "My Pool"
	mock := &mockEC2IPAMAPI{
		describeIpamPoolsFunc: func(_ context.Context, input *ec2.DescribeIpamPoolsInput) ([]ec2types.IpamPool, error) {
			return []ec2types.IpamPool{
				{
					IpamPoolId: aws.String(poolID),
					Tags: []ec2types.Tag{
						{Key: aws.String("Name"), Value: aws.String(poolName)},
					},
				},
			}, nil
		},
		getIpamPoolCidrsFunc: func(_ context.Context, id string) (string, error) {
			if id != poolID {
				t.Errorf("GetIpamPoolCidrs got pool id %q, want %q", id, poolID)
			}
			return "10.0.0.0/8", nil
		},
	}

	ec2APIForTest = mock
	defer func() { ec2APIForTest = nil }()

	provider := &Provider{}
	result, err := provider.SyncPools(ctx, conn)
	if err != nil {
		t.Fatalf("SyncPools: %v", err)
	}
	if len(result.Create) != 1 {
		t.Fatalf("SyncPools: got %d pools, want 1", len(result.Create))
	}
	p := result.Create[0]
	if p.ExternalID != poolID {
		t.Errorf("ExternalID = %q, want %q", p.ExternalID, poolID)
	}
	if p.Name != poolID+" ("+poolName+")" {
		t.Errorf("Name = %q, want %q", p.Name, poolID+" ("+poolName+")")
	}
	if p.CIDR != "10.0.0.0/8" {
		t.Errorf("CIDR = %q, want 10.0.0.0/8", p.CIDR)
	}
	if p.Provider != "aws" {
		t.Errorf("Provider = %q, want aws", p.Provider)
	}
	if len(result.CurrentExternalIDs) != 1 || result.CurrentExternalIDs[0] != poolID {
		t.Errorf("CurrentExternalIDs = %v, want [%q]", result.CurrentExternalIDs, poolID)
	}
}

func TestSyncPools_InvalidConfig(t *testing.T) {
	ctx := context.Background()
	provider := &Provider{}

	t.Run("missing region", func(t *testing.T) {
		conn := connWithConfig(t, "", uuid.New(), "")
		_, err := provider.SyncPools(ctx, conn)
		if err == nil {
			t.Error("SyncPools: expected error for missing region")
		}
	})

	t.Run("missing environment_id", func(t *testing.T) {
		conn := connWithConfig(t, "us-east-1", uuid.Nil, "")
		ec2APIForTest = &mockEC2IPAMAPI{}
		defer func() { ec2APIForTest = nil }()
		_, err := provider.SyncPools(ctx, conn)
		if err == nil {
			t.Error("SyncPools: expected error for missing environment_id")
		}
	})
}

func TestSyncPools_APIError(t *testing.T) {
	ctx := context.Background()
	conn := connWithConfig(t, "us-east-1", uuid.New(), "")
	wantErr := errors.New("api denied")
	mock := &mockEC2IPAMAPI{
		describeIpamPoolsFunc: func(_ context.Context, _ *ec2.DescribeIpamPoolsInput) ([]ec2types.IpamPool, error) {
			return nil, wantErr
		},
	}
	ec2APIForTest = mock
	defer func() { ec2APIForTest = nil }()

	provider := &Provider{}
	_, err := provider.SyncPools(ctx, conn)
	if err == nil {
		t.Fatal("SyncPools: expected error")
	}
	if !errors.Is(err, wantErr) && err.Error() == "" {
		t.Errorf("SyncPools: want error containing %q, got %v", wantErr, err)
	}
}

func TestSyncPools_TopDownOrder(t *testing.T) {
	ctx := context.Background()
	envID := uuid.New()
	conn := connWithConfig(t, "us-east-1", envID, "")

	parentID := "ipam-pool-parent"
	childID := "ipam-pool-child"
	mock := &mockEC2IPAMAPI{
		describeIpamPoolsFunc: func(_ context.Context, _ *ec2.DescribeIpamPoolsInput) ([]ec2types.IpamPool, error) {
			return []ec2types.IpamPool{
				{IpamPoolId: aws.String(childID), SourceIpamPoolId: aws.String(parentID)},
				{IpamPoolId: aws.String(parentID)},
			}, nil
		},
		getIpamPoolCidrsFunc: func(_ context.Context, id string) (string, error) {
			return "10.0.0.0/8", nil
		},
	}
	ec2APIForTest = mock
	defer func() { ec2APIForTest = nil }()

	provider := &Provider{}
	result, err := provider.SyncPools(ctx, conn)
	if err != nil {
		t.Fatalf("SyncPools: %v", err)
	}
	if len(result.Create) != 2 {
		t.Fatalf("SyncPools: got %d pools, want 2", len(result.Create))
	}
	// First pool should be parent (no SourceIpamPoolId)
	if result.Create[0].ExternalID != parentID {
		t.Errorf("first pool ExternalID = %q, want parent %q", result.Create[0].ExternalID, parentID)
	}
	if result.Create[1].ExternalID != childID {
		t.Errorf("second pool ExternalID = %q, want child %q", result.Create[1].ExternalID, childID)
	}
	if result.Create[1].ParentPoolID == nil || *result.Create[1].ParentPoolID != result.Create[0].ID {
		t.Errorf("child ParentPoolID = %v, want parent ID %v", result.Create[1].ParentPoolID, result.Create[0].ID)
	}
}

func TestSyncBlocks(t *testing.T) {
	ctx := context.Background()
	conn := connWithConfig(t, "us-east-1", uuid.New(), "")
	connID := conn.ID
	orgID := conn.OrganizationID

	s := store.NewStore()
	envID := uuid.New()
	poolID := uuid.New()
	pool := &network.Pool{
		ID:             poolID,
		OrganizationID: orgID,
		EnvironmentID:  envID,
		Name:           "pool1",
		CIDR:           "10.0.0.0/8",
		Provider:       "aws",
		ExternalID:     "ipam-pool-abc",
		ConnectionID:   &connID,
	}
	if err := s.CreatePool(pool); err != nil {
		t.Fatal(err)
	}

	vpcID := "vpc-123"
	cidr := "10.1.0.0/16"
	mock := &mockEC2IPAMAPI{
		getIpamPoolAllocationsFunc: func(_ context.Context, id string) ([]ec2types.IpamPoolAllocation, error) {
			if id != "ipam-pool-abc" {
				return nil, nil
			}
			return []ec2types.IpamPoolAllocation{
				{
					ResourceId:   aws.String(vpcID),
					ResourceType: ec2types.IpamPoolAllocationResourceTypeVpc,
					Cidr:         aws.String(cidr),
					Description:  aws.String("my-vpc"),
				},
			}, nil
		},
	}
	ec2APIForTest = mock
	defer func() { ec2APIForTest = nil }()

	provider := &Provider{}
	result, err := provider.SyncBlocks(ctx, conn, s)
	if err != nil {
		t.Fatalf("SyncBlocks: %v", err)
	}
	if len(result.Create) != 1 {
		t.Fatalf("SyncBlocks: got %d blocks, want 1", len(result.Create))
	}
	b := result.Create[0]
	if b.ExternalID != vpcID {
		t.Errorf("ExternalID = %q, want %q", b.ExternalID, vpcID)
	}
	if b.CIDR != cidr {
		t.Errorf("CIDR = %q, want %q", b.CIDR, cidr)
	}
	if b.Name != "my-vpc" {
		t.Errorf("Name = %q, want my-vpc", b.Name)
	}
	if b.PoolID == nil || *b.PoolID != poolID {
		t.Errorf("PoolID = %v, want %q", b.PoolID, poolID)
	}
}

func TestSyncAllocations(t *testing.T) {
	ctx := context.Background()
	conn := connWithConfig(t, "us-east-1", uuid.New(), "")
	connID := conn.ID

	vpcID := "vpc-123"
	blockName := "my-vpc"
	syncedBlocks := []*network.Block{
		{Name: blockName, ExternalID: vpcID, CIDR: "10.1.0.0/16"},
	}

	subnetID := "subnet-abc"
	subnetCIDR := "10.1.1.0/24"
	subnetName := "private-1a"
	mock := &mockEC2IPAMAPI{
		describeSubnetsFunc: func(_ context.Context, vpc string) ([]ec2types.Subnet, error) {
			if vpc != vpcID {
				return nil, nil
			}
			return []ec2types.Subnet{
				{
					SubnetId:  aws.String(subnetID),
					CidrBlock: aws.String(subnetCIDR),
					Tags: []ec2types.Tag{
						{Key: aws.String("Name"), Value: aws.String(subnetName)},
					},
				},
			}, nil
		},
	}
	ec2APIForTest = mock
	defer func() { ec2APIForTest = nil }()

	provider := &Provider{}
	result, err := provider.SyncAllocations(ctx, conn, nil, syncedBlocks)
	if err != nil {
		t.Fatalf("SyncAllocations: %v", err)
	}
	if len(result.Create) != 1 {
		t.Fatalf("SyncAllocations: got %d allocations, want 1", len(result.Create))
	}
	a := result.Create[0]
	if a.ExternalID != subnetID {
		t.Errorf("ExternalID = %q, want %q", a.ExternalID, subnetID)
	}
	if a.Block.CIDR != subnetCIDR {
		t.Errorf("Block.CIDR = %q, want %q", a.Block.CIDR, subnetCIDR)
	}
	if a.Name != subnetName {
		t.Errorf("Name = %q, want %q", a.Name, subnetName)
	}
	if a.Block.Name != blockName {
		t.Errorf("Block.Name = %q, want %q", a.Block.Name, blockName)
	}
	if a.ConnectionID == nil || *a.ConnectionID != connID {
		t.Errorf("ConnectionID = %v, want %q", a.ConnectionID, connID)
	}
}

func TestSyncAllocations_SkipsBlockWithNoExternalID(t *testing.T) {
	ctx := context.Background()
	conn := connWithConfig(t, "us-east-1", uuid.New(), "")
	syncedBlocks := []*network.Block{
		{Name: "vpc1", ExternalID: "", CIDR: "10.1.0.0/16"},
	}
	mock := &mockEC2IPAMAPI{}
	ec2APIForTest = mock
	defer func() { ec2APIForTest = nil }()

	provider := &Provider{}
	result, err := provider.SyncAllocations(ctx, conn, nil, syncedBlocks)
	if err != nil {
		t.Fatalf("SyncAllocations: %v", err)
	}
	if len(result.Create) != 0 {
		t.Errorf("SyncAllocations: got %d allocations, want 0 (block has no external_id)", len(result.Create))
	}
}

func Test_ipamPoolDisplayName(t *testing.T) {
	poolID := "ipam-pool-xyz"
	t.Run("with Name tag", func(t *testing.T) {
		ap := ec2types.IpamPool{
			IpamPoolId: aws.String(poolID),
			Tags: []ec2types.Tag{
				{Key: aws.String("Name"), Value: aws.String("Prod Pool")},
			},
		}
		got := ipamPoolDisplayName(ap, poolID)
		want := poolID + " (Prod Pool)"
		if got != want {
			t.Errorf("ipamPoolDisplayName = %q, want %q", got, want)
		}
	})
	t.Run("no Name tag", func(t *testing.T) {
		ap := ec2types.IpamPool{IpamPoolId: aws.String(poolID)}
		got := ipamPoolDisplayName(ap, poolID)
		if got != poolID {
			t.Errorf("ipamPoolDisplayName = %q, want %q", got, poolID)
		}
	})
}

func Test_getEnvForConn(t *testing.T) {
	orgID := uuid.New()
	envID := uuid.New()
	cfg := AWSConnectionConfig{Region: "us-east-1", EnvironmentID: envID}
	raw, _ := json.Marshal(cfg)
	conn := &store.CloudConnection{OrganizationID: orgID, Config: raw}

	env, err := getEnvForConn(conn)
	if err != nil {
		t.Fatalf("getEnvForConn: %v", err)
	}
	if env.Id != envID {
		t.Errorf("env.Id = %v, want %v", env.Id, envID)
	}
	if env.OrganizationID != orgID {
		t.Errorf("env.OrganizationID = %v, want %v", env.OrganizationID, orgID)
	}

	connNilEnv := &store.CloudConnection{Config: mustMarshal(t, AWSConnectionConfig{EnvironmentID: uuid.Nil})}
	_, err = getEnvForConn(connNilEnv)
	if err == nil {
		t.Error("getEnvForConn: expected error for nil environment_id")
	}
}

func mustMarshal(t *testing.T, v interface{}) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
