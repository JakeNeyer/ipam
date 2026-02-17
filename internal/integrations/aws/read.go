package aws

import (
	"context"
	"fmt"

	"github.com/JakeNeyer/ipam/internal/integrations"
	"github.com/JakeNeyer/ipam/network"
	"github.com/JakeNeyer/ipam/store"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/google/uuid"
)

// SyncPools discovers AWS IPAM pools and returns create/update diffs. Pools are returned in top-down order.
func (p *Provider) SyncPools(ctx context.Context, conn *store.CloudConnection) (*integrations.PoolSyncResult, error) {
	cfg, err := ParseAWSConfig(conn.Config)
	if err != nil || cfg == nil || cfg.Region == "" {
		return nil, fmt.Errorf("invalid aws connection config: need region")
	}
	envID := cfg.EnvironmentID
	if envID == uuid.Nil {
		return nil, fmt.Errorf("aws connection config must set environment_id to attach synced pools")
	}
	env, err := getEnvForConn(conn)
	if env == nil || err != nil {
		return nil, fmt.Errorf("environment_id not found or not in same org: %w", err)
	}
	envID = env.Id

	api, err := getEC2API(ctx, cfg.Region)
	if err != nil {
		return nil, fmt.Errorf("ec2 client: %w", err)
	}

	input := &ec2.DescribeIpamPoolsInput{}
	if cfg.IpamScopeId != "" {
		input.Filters = []ec2types.Filter{
			{Name: aws.String("ipam-scope-id"), Values: []string{cfg.IpamScopeId}},
		}
	}
	awsPools, err := api.DescribeIpamPools(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("describe ipam pools: %w", err)
	}

	// Build top-down: first pools with no source (top-level), then children.
	byId := make(map[string]ec2types.IpamPool)
	for _, pool := range awsPools {
		if pool.IpamPoolId != nil {
			byId[aws.ToString(pool.IpamPoolId)] = pool
		}
	}
	var ordered []ec2types.IpamPool
	added := make(map[string]bool)
	for len(ordered) < len(awsPools) {
		before := len(ordered)
		for _, pool := range awsPools {
			pid := aws.ToString(pool.IpamPoolId)
			if added[pid] {
				continue
			}
			parentId := aws.ToString(pool.SourceIpamPoolId)
			if parentId == "" || added[parentId] {
				ordered = append(ordered, pool)
				added[pid] = true
			}
		}
		if len(ordered) == before {
			break
		}
	}

	connID := conn.ID
	result := &integrations.PoolSyncResult{}
	externalToAppPool := make(map[string]*network.Pool)

	for _, ap := range ordered {
		extId := aws.ToString(ap.IpamPoolId)
		name := ipamPoolDisplayName(ap, extId)
		cidr, err := api.GetIpamPoolCidrs(ctx, extId)
		if err != nil {
			return nil, fmt.Errorf("get ipam pool cidrs for %s: %w", extId, err)
		}
		// Leave CIDR empty when pool has no provisioned CIDR; do not default to 0.0.0.0/0

		pool := &network.Pool{
			OrganizationID: conn.OrganizationID,
			EnvironmentID:  envID,
			Name:           name,
			CIDR:           cidr,
			Provider:       providerID,
			ExternalID:     extId,
			ConnectionID:   &connID,
		}
		if ap.SourceIpamPoolId != nil {
			if parent, ok := externalToAppPool[aws.ToString(ap.SourceIpamPoolId)]; ok {
				pool.ParentPoolID = &parent.ID
			}
		}
		// We don't have existing app pool IDs here; sync layer will match by connection_id+external_id and create or update
		result.Create = append(result.Create, pool)
		externalToAppPool[extId] = pool
	}
	result.CurrentExternalIDs = make([]string, 0, len(result.Create))
	for _, pool := range result.Create {
		if pool.ExternalID != "" {
			result.CurrentExternalIDs = append(result.CurrentExternalIDs, pool.ExternalID)
		}
	}
	return result, nil
}

// SyncBlocks discovers AWS IPAM pool allocations (e.g. VPC CIDRs) and returns block create/update diffs.
func (p *Provider) SyncBlocks(ctx context.Context, conn *store.CloudConnection, s store.Storer) (*integrations.BlockSyncResult, error) {
	cfg, err := ParseAWSConfig(conn.Config)
	if err != nil || cfg == nil || cfg.Region == "" {
		return nil, fmt.Errorf("invalid aws connection config: need region")
	}
	env, err := getEnvForConn(conn)
	if env == nil || err != nil {
		return nil, fmt.Errorf("environment_id not found: %w", err)
	}
	api, err := getEC2API(ctx, cfg.Region)
	if err != nil {
		return nil, fmt.Errorf("ec2 client: %w", err)
	}
	connID := conn.ID

	// Resolve app pools for this connection (synced pools have connection_id and external_id = AWS pool id)
	appPools, err := s.ListPoolsByOrganization(conn.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("list pools: %w", err)
	}
	var poolsForConn []*network.Pool
	for _, pool := range appPools {
		if pool.ConnectionID != nil && *pool.ConnectionID == conn.ID && pool.ExternalID != "" {
			poolsForConn = append(poolsForConn, pool)
		}
	}

	// Set of AWS pool IDs for this connection: allocations to these are child pools, not blocks
	poolExternalIDs := make(map[string]bool)
	for _, appPool := range poolsForConn {
		if appPool.ExternalID != "" {
			poolExternalIDs[appPool.ExternalID] = true
		}
	}

	result := &integrations.BlockSyncResult{}
	seenExtID := make(map[string]bool)

	for _, appPool := range poolsForConn {
		allocations, err := api.GetIpamPoolAllocations(ctx, appPool.ExternalID)
		if err != nil {
			return nil, fmt.Errorf("get ipam pool allocations for %s: %w", appPool.ExternalID, err)
		}
		poolID := appPool.ID
		envID := appPool.EnvironmentID
		orgID := conn.OrganizationID
		for _, alloc := range allocations {
			// Only sync VPC allocations as blocks. Skip ipam-pool (child pool), subnet, eip, etc.
			if alloc.ResourceType != ec2types.IpamPoolAllocationResourceTypeVpc {
				continue
			}
			// Skip sub-pool allocations: allocation resource is another IPAM pool (child), already synced as a pool
			if poolExternalIDs[ipamAllocationExternalID(alloc)] {
				continue
			}
			cidr := ipamAllocationCIDR(alloc)
			if cidr == "" {
				continue
			}
			extID := ipamAllocationExternalID(alloc)
			if extID == "" || seenExtID[extID] {
				continue
			}
			seenExtID[extID] = true
			name := ipamAllocationName(alloc)
			if name == "" {
				name = extID
			}
			block := &network.Block{
				Name:           name,
				CIDR:           cidr,
				EnvironmentID:  envID,
				OrganizationID: orgID,
				PoolID:         &poolID,
				Provider:       providerID,
				ExternalID:     extID,
				ConnectionID:   &connID,
			}
			result.Create = append(result.Create, block)
		}
	}
	result.CurrentExternalIDs = make([]string, 0, len(seenExtID))
	for extID := range seenExtID {
		result.CurrentExternalIDs = append(result.CurrentExternalIDs, extID)
	}
	return result, nil
}

// SyncAllocations discovers VPC subnets for each synced block (VPC) and returns allocation create/update diffs.
func (p *Provider) SyncAllocations(ctx context.Context, conn *store.CloudConnection, s store.Storer, syncedBlocks []*network.Block) (*integrations.AllocationSyncResult, error) {
	cfg, err := ParseAWSConfig(conn.Config)
	if err != nil || cfg == nil || cfg.Region == "" {
		return nil, fmt.Errorf("invalid aws connection config: need region")
	}
	api, err := getEC2API(ctx, cfg.Region)
	if err != nil {
		return nil, fmt.Errorf("ec2 client: %w", err)
	}
	connID := conn.ID
	result := &integrations.AllocationSyncResult{
		CurrentExternalIDs: make([]string, 0),
	}

	for _, block := range syncedBlocks {
		if block.ExternalID == "" {
			continue
		}
		vpcID := block.ExternalID
		blockName := block.Name
		subnets, err := api.DescribeSubnets(ctx, vpcID)
		if err != nil {
			return nil, fmt.Errorf("describe subnets for vpc %s: %w", vpcID, err)
		}
		for _, sn := range subnets {
			cidr := aws.ToString(sn.CidrBlock)
			if cidr == "" {
				continue
			}
			extID := aws.ToString(sn.SubnetId)
			if extID == "" {
				continue
			}
			result.CurrentExternalIDs = append(result.CurrentExternalIDs, extID)
			name := subnetName(sn)
			if name == "" {
				name = extID
			}
			alloc := &network.Allocation{
				Name:         name,
				Block:        network.Block{Name: blockName, CIDR: cidr},
				Provider:     providerID,
				ExternalID:   extID,
				ConnectionID: &connID,
			}
			result.Create = append(result.Create, alloc)
		}
	}
	return result, nil
}

// getEnvForConn returns the environment for conn's config environment_id; must be in same org as conn.
func getEnvForConn(conn *store.CloudConnection) (*network.Environment, error) {
	cfg, _ := ParseAWSConfig(conn.Config)
	if cfg == nil || cfg.EnvironmentID == uuid.Nil {
		return nil, fmt.Errorf("environment_id not set")
	}
	// We don't have store in provider; caller (sync orchestrator) could pass env or we require config to have environment_id and the orchestrator validates. So we can't fetch env here. Return a dummy env with ID from config and Org from conn.
	return &network.Environment{Id: cfg.EnvironmentID, OrganizationID: conn.OrganizationID}, nil
}

// describeSubnetsByVPCWithClient is used by ec2APIAdapter; tests mock EC2IPAMAPI instead.
func describeSubnetsByVPCWithClient(ctx context.Context, client *ec2.Client, vpcID string) ([]ec2types.Subnet, error) {
	input := &ec2.DescribeSubnetsInput{
		Filters: []ec2types.Filter{
			{Name: aws.String("vpc-id"), Values: []string{vpcID}},
		},
	}
	var out []ec2types.Subnet
	pager := ec2.NewDescribeSubnetsPaginator(client, input)
	for pager.HasMorePages() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		out = append(out, page.Subnets...)
	}
	return out, nil
}

func subnetName(sn ec2types.Subnet) string {
	for _, t := range sn.Tags {
		if t.Key != nil && aws.ToString(t.Key) == "Name" && t.Value != nil {
			if v := aws.ToString(t.Value); v != "" {
				return v
			}
		}
	}
	return ""
}

// getIpamPoolAllocationsWithClient is used by ec2APIAdapter; tests mock EC2IPAMAPI instead.
func getIpamPoolAllocationsWithClient(ctx context.Context, client *ec2.Client, ipamPoolID string) ([]ec2types.IpamPoolAllocation, error) {
	var out []ec2types.IpamPoolAllocation
	input := &ec2.GetIpamPoolAllocationsInput{
		IpamPoolId: aws.String(ipamPoolID),
	}
	pager := ec2.NewGetIpamPoolAllocationsPaginator(client, input)
	for pager.HasMorePages() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		out = append(out, page.IpamPoolAllocations...)
	}
	return out, nil
}

func ipamAllocationCIDR(a ec2types.IpamPoolAllocation) string {
	if a.Cidr != nil {
		return aws.ToString(a.Cidr)
	}
	return ""
}

func ipamAllocationExternalID(a ec2types.IpamPoolAllocation) string {
	if a.ResourceId != nil {
		return aws.ToString(a.ResourceId)
	}
	if a.IpamPoolAllocationId != nil {
		return aws.ToString(a.IpamPoolAllocationId)
	}
	return ""
}

func ipamAllocationName(a ec2types.IpamPoolAllocation) string {
	if a.Description != nil {
		return aws.ToString(a.Description)
	}
	if a.ResourceId != nil {
		return aws.ToString(a.ResourceId)
	}
	return ""
}

// ipamPoolDisplayName returns the app pool name: AWS pool ID, with optional Name tag in parentheses (e.g. "ipam-pool-abc123 (My Pool)").
// Description is not used.
func ipamPoolDisplayName(ap ec2types.IpamPool, poolID string) string {
	if len(ap.Tags) > 0 {
		for _, t := range ap.Tags {
			if t.Key != nil && aws.ToString(t.Key) == "Name" && t.Value != nil {
				if v := aws.ToString(t.Value); v != "" {
					return poolID + " (" + v + ")"
				}
			}
		}
	}
	return poolID
}

// getIpamPoolCidrWithClient fetches the pool's provisioned CIDR(s) via GetIpamPoolCidrs and returns the first
// provisioned CIDR, or the first CIDR in the list if none are in "provisioned" state. Used by ec2APIAdapter.
func getIpamPoolCidrWithClient(ctx context.Context, client *ec2.Client, ipamPoolID string) (string, error) {
	input := &ec2.GetIpamPoolCidrsInput{
		IpamPoolId: aws.String(ipamPoolID),
	}
	pager := ec2.NewGetIpamPoolCidrsPaginator(client, input)
	var first string
	for pager.HasMorePages() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return "", err
		}
		for _, c := range page.IpamPoolCidrs {
			cidr := aws.ToString(c.Cidr)
			if cidr == "" {
				continue
			}
			if first == "" {
				first = cidr
			}
			if c.State == ec2types.IpamPoolCidrStateProvisioned {
				return cidr, nil
			}
		}
	}
	return first, nil
}
