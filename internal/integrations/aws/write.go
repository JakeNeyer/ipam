package aws

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/JakeNeyer/ipam/network"
	"github.com/JakeNeyer/ipam/store"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// ec2WriteAPI abstracts EC2 write operations for testing. *ec2.Client implements it.
type ec2WriteAPI interface {
	CreateIpamPool(ctx context.Context, params *ec2.CreateIpamPoolInput, optFns ...func(*ec2.Options)) (*ec2.CreateIpamPoolOutput, error)
	DeleteIpamPool(ctx context.Context, params *ec2.DeleteIpamPoolInput, optFns ...func(*ec2.Options)) (*ec2.DeleteIpamPoolOutput, error)
	DescribeIpamPools(ctx context.Context, params *ec2.DescribeIpamPoolsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeIpamPoolsOutput, error)
	GetIpamPoolCidrs(ctx context.Context, params *ec2.GetIpamPoolCidrsInput, optFns ...func(*ec2.Options)) (*ec2.GetIpamPoolCidrsOutput, error)
	GetIpamPoolAllocations(ctx context.Context, params *ec2.GetIpamPoolAllocationsInput, optFns ...func(*ec2.Options)) (*ec2.GetIpamPoolAllocationsOutput, error)
	ProvisionIpamPoolCidr(ctx context.Context, params *ec2.ProvisionIpamPoolCidrInput, optFns ...func(*ec2.Options)) (*ec2.ProvisionIpamPoolCidrOutput, error)
	AllocateIpamPoolCidr(ctx context.Context, params *ec2.AllocateIpamPoolCidrInput, optFns ...func(*ec2.Options)) (*ec2.AllocateIpamPoolCidrOutput, error)
	ReleaseIpamPoolAllocation(ctx context.Context, params *ec2.ReleaseIpamPoolAllocationInput, optFns ...func(*ec2.Options)) (*ec2.ReleaseIpamPoolAllocationOutput, error)
	CreateVpc(ctx context.Context, params *ec2.CreateVpcInput, optFns ...func(*ec2.Options)) (*ec2.CreateVpcOutput, error)
	DescribeVpcs(ctx context.Context, params *ec2.DescribeVpcsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcsOutput, error)
	DeleteVpc(ctx context.Context, params *ec2.DeleteVpcInput, optFns ...func(*ec2.Options)) (*ec2.DeleteVpcOutput, error)
	DescribeSubnets(ctx context.Context, params *ec2.DescribeSubnetsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSubnetsOutput, error)
	CreateSubnet(ctx context.Context, params *ec2.CreateSubnetInput, optFns ...func(*ec2.Options)) (*ec2.CreateSubnetOutput, error)
	DeleteSubnet(ctx context.Context, params *ec2.DeleteSubnetInput, optFns ...func(*ec2.Options)) (*ec2.DeleteSubnetOutput, error)
}

const ipamPoolReadyPollInterval = 3 * time.Second
const ipamPoolReadyTimeout = 2 * time.Minute
const ipamPoolCidrProvisionedPollInterval = 5 * time.Second
const ipamPoolCidrProvisionedTimeout = 3 * time.Minute

// ec2WriteAPIForTest is set by tests to inject a mock; must be reset after each test.
var ec2WriteAPIForTest ec2WriteAPI

func getWriteClient(ctx context.Context, region string) (ec2WriteAPI, error) {
	if ec2WriteAPIForTest != nil {
		return ec2WriteAPIForTest, nil
	}
	return newEC2Client(ctx, region)
}

// SupportsPush returns true; AWS provider supports write (push to cloud).
func (p *Provider) SupportsPush() bool {
	return true
}

// findExistingIpamPool returns an existing IPAM pool ID in the scope (and under parent if set) with matching Name tag and optional CIDR, or "" if none.
func findExistingIpamPool(ctx context.Context, client ec2WriteAPI, scopeID, parentExternalID, name, cidr string) (string, error) {
	input := &ec2.DescribeIpamPoolsInput{
		Filters: []ec2types.Filter{
			{Name: aws.String("ipam-scope-id"), Values: []string{scopeID}},
		},
	}
	var all []ec2types.IpamPool
	for {
		out, err := client.DescribeIpamPools(ctx, input)
		if err != nil {
			return "", err
		}
		all = append(all, out.IpamPools...)
		if out.NextToken == nil {
			break
		}
		input.NextToken = out.NextToken
	}
	for _, ap := range all {
		if parentExternalID != "" && (ap.SourceIpamPoolId == nil || aws.ToString(ap.SourceIpamPoolId) != parentExternalID) {
			continue
		}
		tagName := ""
		for _, t := range ap.Tags {
			if t.Key != nil && aws.ToString(t.Key) == "Name" && t.Value != nil {
				tagName = aws.ToString(t.Value)
				break
			}
		}
		if tagName != name {
			continue
		}
		poolID := aws.ToString(ap.IpamPoolId)
		if cidr != "" {
			cidrOut, err := client.GetIpamPoolCidrs(ctx, &ec2.GetIpamPoolCidrsInput{IpamPoolId: aws.String(poolID)})
			if err != nil {
				continue
			}
			found := false
			for _, c := range cidrOut.IpamPoolCidrs {
				if aws.ToString(c.Cidr) == cidr {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		return poolID, nil
	}
	return "", nil
}

// CreatePoolInCloud creates an IPAM pool in AWS and returns its pool ID (external_id).
// If a pool already exists in the scope (and parent) with the same Name and optional CIDR, returns that pool ID to avoid duplicates.
// parentExternalID is the parent pool's AWS pool ID when creating a sub-pool; empty for top-level.
func (p *Provider) CreatePoolInCloud(ctx context.Context, conn *store.CloudConnection, pool *network.Pool, parentExternalID string) (externalID string, err error) {
	cfg, err := ParseAWSConfig(conn.Config)
	if err != nil || cfg == nil || cfg.Region == "" {
		return "", fmt.Errorf("invalid aws connection config: need region")
	}
	if cfg.IpamScopeId == "" {
		return "", fmt.Errorf("aws connection config must set ipam_scope_id to create pools")
	}
	client, err := getWriteClient(ctx, cfg.Region)
	if err != nil {
		return "", fmt.Errorf("ec2 client: %w", err)
	}

	if existing, err := findExistingIpamPool(ctx, client, cfg.IpamScopeId, parentExternalID, pool.Name, pool.CIDR); err == nil && existing != "" {
		return existing, nil
	}

	input := &ec2.CreateIpamPoolInput{
		IpamScopeId:   aws.String(cfg.IpamScopeId),
		AddressFamily: ec2types.AddressFamilyIpv4,
		Description:   aws.String(pool.Name),
		TagSpecifications: []ec2types.TagSpecification{
			{
				ResourceType: ec2types.ResourceTypeIpamPool,
				Tags: []ec2types.Tag{
					{Key: aws.String("Name"), Value: aws.String(pool.Name)},
				},
			},
		},
	}
	if parentExternalID != "" {
		input.SourceIpamPoolId = aws.String(parentExternalID)
	}

	out, err := client.CreateIpamPool(ctx, input)
	if err != nil {
		return "", fmt.Errorf("create ipam pool: %w", err)
	}
	if out.IpamPool == nil || out.IpamPool.IpamPoolId == nil {
		return "", fmt.Errorf("create ipam pool: empty response")
	}
	poolID := aws.ToString(out.IpamPool.IpamPoolId)

	// Wait for pool to leave create-in-progress before provisioning CIDR.
	if err = waitForIpamPoolReady(ctx, client, poolID); err != nil {
		return "", fmt.Errorf("wait for ipam pool: %w", err)
	}

	// Provision the pool with its CIDR so allocations (AllocateIpamPoolCidr) work later.
	if pool.CIDR != "" {
		_, err = client.ProvisionIpamPoolCidr(ctx, &ec2.ProvisionIpamPoolCidrInput{
			IpamPoolId: aws.String(poolID),
			Cidr:       aws.String(pool.CIDR),
		})
		if err != nil {
			return "", fmt.Errorf("provision ipam pool cidr: %w", err)
		}
		if err = waitForIpamPoolCidrProvisioned(ctx, client, poolID, pool.CIDR); err != nil {
			return "", fmt.Errorf("wait for ipam pool cidr provisioned: %w", err)
		}
	}
	return poolID, nil
}

// DeletePoolInCloud deletes the IPAM pool in AWS. Uses Cascade to delete the pool and its allocations/CIDRs when possible.
func (p *Provider) DeletePoolInCloud(ctx context.Context, conn *store.CloudConnection, externalID string) error {
	cfg, err := ParseAWSConfig(conn.Config)
	if err != nil || cfg == nil || cfg.Region == "" {
		return fmt.Errorf("invalid aws connection config: need region")
	}
	client, err := getWriteClient(ctx, cfg.Region)
	if err != nil {
		return fmt.Errorf("ec2 client: %w", err)
	}
	_, err = client.DeleteIpamPool(ctx, &ec2.DeleteIpamPoolInput{
		IpamPoolId: aws.String(externalID),
		Cascade:    aws.Bool(true),
	})
	if err != nil {
		return fmt.Errorf("delete ipam pool: %w", err)
	}
	return nil
}

// waitForIpamPoolReady polls until the pool state is create-complete or create-failed.
func waitForIpamPoolReady(ctx context.Context, api ec2WriteAPI, poolID string) error {
	deadline := time.Now().Add(ipamPoolReadyTimeout)
	ticker := time.NewTicker(ipamPoolReadyPollInterval)
	defer ticker.Stop()
	for {
		out, err := api.DescribeIpamPools(ctx, &ec2.DescribeIpamPoolsInput{
			IpamPoolIds: []string{poolID},
		})
		if err != nil {
			return fmt.Errorf("describe ipam pool: %w", err)
		}
		if len(out.IpamPools) == 0 {
			return fmt.Errorf("pool %s not found", poolID)
		}
		state := out.IpamPools[0].State
		switch state {
		case ec2types.IpamPoolStateCreateComplete:
			return nil
		case ec2types.IpamPoolStateCreateFailed:
			msg := "pool creation failed"
			if out.IpamPools[0].StateMessage != nil {
				msg = aws.ToString(out.IpamPools[0].StateMessage)
			}
			return fmt.Errorf("%s: %s", poolID, msg)
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("pool %s did not become ready in time (state: %s)", poolID, state)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// continue
		}
	}
}

// waitForIpamPoolCidrProvisioned polls until the given CIDR in the pool reaches "provisioned" state so AllocateIpamPoolCidr can use it.
func waitForIpamPoolCidrProvisioned(ctx context.Context, api ec2WriteAPI, poolID, cidr string) error {
	deadline := time.Now().Add(ipamPoolCidrProvisionedTimeout)
	ticker := time.NewTicker(ipamPoolCidrProvisionedPollInterval)
	defer ticker.Stop()
	for {
		out, err := api.GetIpamPoolCidrs(ctx, &ec2.GetIpamPoolCidrsInput{
			IpamPoolId: aws.String(poolID),
		})
		if err != nil {
			return fmt.Errorf("get ipam pool cidrs: %w", err)
		}
		for _, c := range out.IpamPoolCidrs {
			if aws.ToString(c.Cidr) == cidr {
				if c.State == ec2types.IpamPoolCidrStateProvisioned {
					return nil
				}
				break
			}
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("pool %s cidr %s did not reach provisioned in time", poolID, cidr)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// continue
		}
	}
}

// findExistingVpcAllocation returns a VPC ID (resource ID) if the pool already has an allocation with the given CIDR and resource type VPC,
// and the VPC still exists in AWS. If the VPC was deleted (e.g. Custom allocations remain but VPCs are gone), returns "" so caller allocates and creates.
func findExistingVpcAllocation(ctx context.Context, client ec2WriteAPI, poolExternalID, cidr string) (string, error) {
	input := &ec2.GetIpamPoolAllocationsInput{IpamPoolId: aws.String(poolExternalID)}
	for {
		out, err := client.GetIpamPoolAllocations(ctx, input)
		if err != nil {
			return "", err
		}
		for _, a := range out.IpamPoolAllocations {
			if a.ResourceType != ec2types.IpamPoolAllocationResourceTypeVpc {
				continue
			}
			if a.Cidr != nil && aws.ToString(a.Cidr) == cidr && a.ResourceId != nil {
				vpcID := aws.ToString(a.ResourceId)
				if vpcExists(ctx, client, vpcID) {
					return vpcID, nil
				}
				// VPC was deleted; allocation record may linger. Fall through to allocate+create.
			}
		}
		if out.NextToken == nil {
			break
		}
		input.NextToken = out.NextToken
	}
	return "", nil
}

// vpcExists returns true if the VPC exists in AWS (available or pending state).
func vpcExists(ctx context.Context, client ec2WriteAPI, vpcID string) bool {
	out, err := client.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{
		VpcIds: []string{vpcID},
	})
	if err != nil || out == nil || len(out.Vpcs) == 0 {
		return false
	}
	for _, v := range out.Vpcs {
		if v.VpcId != nil && aws.ToString(v.VpcId) == vpcID {
			if v.State == ec2types.VpcStateAvailable || v.State == ec2types.VpcStatePending {
				return true
			}
			return false
		}
	}
	return false
}

// releaseOrphanedVpcAllocation releases an IPAM pool allocation for the given CIDR when the VPC was deleted
// (allocation lingers in the pool). Call before AllocateIpamPoolCidr when re-creating a VPC.
func releaseOrphanedVpcAllocation(ctx context.Context, client ec2WriteAPI, poolExternalID, cidr string) error {
	input := &ec2.GetIpamPoolAllocationsInput{IpamPoolId: aws.String(poolExternalID)}
	for {
		out, err := client.GetIpamPoolAllocations(ctx, input)
		if err != nil {
			return err
		}
		for _, a := range out.IpamPoolAllocations {
			if a.ResourceType != ec2types.IpamPoolAllocationResourceTypeVpc && a.ResourceType != ec2types.IpamPoolAllocationResourceTypeCustom {
				continue
			}
			if a.Cidr == nil || aws.ToString(a.Cidr) != cidr {
				continue
			}
			vpcID := ""
			if a.ResourceId != nil {
				vpcID = aws.ToString(a.ResourceId)
			}
			if vpcID != "" && vpcExists(ctx, client, vpcID) {
				continue
			}
			allocID := ""
			if a.IpamPoolAllocationId != nil {
				allocID = aws.ToString(a.IpamPoolAllocationId)
			}
			if allocID == "" {
				continue
			}
			_, err = client.ReleaseIpamPoolAllocation(ctx, &ec2.ReleaseIpamPoolAllocationInput{
				IpamPoolId:           aws.String(poolExternalID),
				Cidr:                 aws.String(cidr),
				IpamPoolAllocationId: aws.String(allocID),
			})
			return err
		}
		if out.NextToken == nil {
			break
		}
		input.NextToken = out.NextToken
	}
	return nil
}

// AllocateBlockInCloud allocates the block's CIDR from the IPAM pool and creates a VPC with it; returns the VPC ID (block external_id).
// If the pool already has a VPC allocation with the same CIDR, returns that VPC ID to avoid duplicates.
func (p *Provider) AllocateBlockInCloud(ctx context.Context, conn *store.CloudConnection, poolExternalID string, block *network.Block) (externalID string, err error) {
	cfg, err := ParseAWSConfig(conn.Config)
	if err != nil || cfg == nil || cfg.Region == "" {
		return "", fmt.Errorf("invalid aws connection config: need region")
	}
	client, err := getWriteClient(ctx, cfg.Region)
	if err != nil {
		return "", fmt.Errorf("ec2 client: %w", err)
	}

	if block.CIDR != "" {
		if existing, err := findExistingVpcAllocation(ctx, client, poolExternalID, block.CIDR); err == nil && existing != "" {
			return existing, nil
		}
		// VPC may have been deleted manually; release orphaned allocation before re-allocate.
		if err := releaseOrphanedVpcAllocation(ctx, client, poolExternalID, block.CIDR); err != nil {
			// AWS may reject release for VPC allocations (only custom/manual can be released); continue to try allocate
			if !strings.Contains(err.Error(), "resource type") && !strings.Contains(err.Error(), "Cannot release") {
				return "", fmt.Errorf("release orphaned allocation: %w", err)
			}
		}
	}

	// Allocate the CIDR from the IPAM pool (required before creating VPC with IPAM).
	_, err = client.AllocateIpamPoolCidr(ctx, &ec2.AllocateIpamPoolCidrInput{
		IpamPoolId: aws.String(poolExternalID),
		Cidr:       aws.String(block.CIDR),
	})
	if err != nil {
		return "", fmt.Errorf("allocate ipam pool cidr: %w", err)
	}

	// Create VPC with the allocated CIDR.
	vpcOut, err := client.CreateVpc(ctx, &ec2.CreateVpcInput{
		CidrBlock: aws.String(block.CIDR),
		TagSpecifications: []ec2types.TagSpecification{
			{
				ResourceType: ec2types.ResourceTypeVpc,
				Tags: []ec2types.Tag{
					{Key: aws.String("Name"), Value: aws.String(block.Name)},
				},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("create vpc: %w", err)
	}
	if vpcOut.Vpc == nil || vpcOut.Vpc.VpcId == nil {
		return "", fmt.Errorf("create vpc: empty response")
	}
	return aws.ToString(vpcOut.Vpc.VpcId), nil
}

// findExistingSubnet returns a subnet ID in the VPC with the given CIDR, or "" if none.
func findExistingSubnet(ctx context.Context, client ec2WriteAPI, vpcID, cidr string) (string, error) {
	input := &ec2.DescribeSubnetsInput{
		Filters: []ec2types.Filter{
			{Name: aws.String("vpc-id"), Values: []string{vpcID}},
		},
	}
	for {
		out, err := client.DescribeSubnets(ctx, input)
		if err != nil {
			return "", err
		}
		for _, sn := range out.Subnets {
			if sn.CidrBlock != nil && aws.ToString(sn.CidrBlock) == cidr && sn.SubnetId != nil {
				return aws.ToString(sn.SubnetId), nil
			}
		}
		if out.NextToken == nil {
			break
		}
		input.NextToken = out.NextToken
	}
	return "", nil
}

// CreateAllocationInCloud creates a subnet in the VPC (block) and returns the subnet ID (allocation external_id).
// alloc.Block.CIDR is the allocation's CIDR (subnet CIDR).
// If the VPC already has a subnet with the same CIDR, returns that subnet ID to avoid duplicates.
func (p *Provider) CreateAllocationInCloud(ctx context.Context, conn *store.CloudConnection, blockExternalID string, alloc *network.Allocation) (externalID string, err error) {
	cfg, err := ParseAWSConfig(conn.Config)
	if err != nil || cfg == nil || cfg.Region == "" {
		return "", fmt.Errorf("invalid aws connection config: need region")
	}
	client, err := getWriteClient(ctx, cfg.Region)
	if err != nil {
		return "", fmt.Errorf("ec2 client: %w", err)
	}

	cidr := alloc.Block.CIDR
	if cidr == "" {
		return "", fmt.Errorf("allocation has no CIDR")
	}

	if existing, err := findExistingSubnet(ctx, client, blockExternalID, cidr); err == nil && existing != "" {
		return existing, nil
	}

	out, err := client.CreateSubnet(ctx, &ec2.CreateSubnetInput{
		VpcId:     aws.String(blockExternalID),
		CidrBlock: aws.String(cidr),
		TagSpecifications: []ec2types.TagSpecification{
			{
				ResourceType: ec2types.ResourceTypeSubnet,
				Tags: []ec2types.Tag{
					{Key: aws.String("Name"), Value: aws.String(alloc.Name)},
				},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("create subnet: %w", err)
	}
	if out.Subnet == nil || out.Subnet.SubnetId == nil {
		return "", fmt.Errorf("create subnet: empty response")
	}
	return aws.ToString(out.Subnet.SubnetId), nil
}

// DeleteBlockInCloud deletes the VPC in AWS (block external_id is VPC ID).
// Skips without error if externalID is not a VPC ID (e.g. legacy ipam-pool-* mistaken as block); caller will still remove the row.
func (p *Provider) DeleteBlockInCloud(ctx context.Context, conn *store.CloudConnection, externalID string) error {
	if externalID == "" || !strings.HasPrefix(externalID, "vpc-") {
		return nil
	}
	cfg, err := ParseAWSConfig(conn.Config)
	if err != nil || cfg == nil || cfg.Region == "" {
		return fmt.Errorf("invalid aws connection config: need region")
	}
	client, err := getWriteClient(ctx, cfg.Region)
	if err != nil {
		return fmt.Errorf("ec2 client: %w", err)
	}
	_, err = client.DeleteVpc(ctx, &ec2.DeleteVpcInput{
		VpcId: aws.String(externalID),
	})
	if err != nil {
		return fmt.Errorf("delete vpc: %w", err)
	}
	return nil
}

// DeleteAllocationInCloud deletes the subnet in AWS (allocation external_id is subnet ID).
func (p *Provider) DeleteAllocationInCloud(ctx context.Context, conn *store.CloudConnection, externalID string) error {
	cfg, err := ParseAWSConfig(conn.Config)
	if err != nil || cfg == nil || cfg.Region == "" {
		return fmt.Errorf("invalid aws connection config: need region")
	}
	client, err := getWriteClient(ctx, cfg.Region)
	if err != nil {
		return fmt.Errorf("ec2 client: %w", err)
	}
	_, err = client.DeleteSubnet(ctx, &ec2.DeleteSubnetInput{
		SubnetId: aws.String(externalID),
	})
	if err != nil {
		return fmt.Errorf("delete subnet: %w", err)
	}
	return nil
}
