package integrations

import (
	"context"

	"github.com/JakeNeyer/ipam/network"
	"github.com/JakeNeyer/ipam/store"
	"github.com/google/uuid"
)

// CloudProvider is the interface that cloud integrations (AWS, Azure, GCP) implement.
// Sync methods return diffs that the orchestrator applies to the store.
type CloudProvider interface {
	// ProviderID returns the provider identifier, e.g. "aws", "azure", "gcp".
	ProviderID() string
	// SupportsPools returns true if this provider can sync pools (e.g. AWS IPAM pools).
	SupportsPools() bool
	// SupportsBlocks returns true if this provider can sync blocks (e.g. AWS allocations → blocks).
	SupportsBlocks() bool
	// SupportsAllocations returns true if this provider can sync allocations (e.g. AWS VPC subnets → app allocations).
	SupportsAllocations() bool
	// SyncPools discovers pools from the cloud and returns create/update diffs for the app store.
	SyncPools(ctx context.Context, conn *store.CloudConnection) (*PoolSyncResult, error)
	// SyncBlocks discovers allocations/resources from the cloud and returns create/update diffs for blocks.
	// Store is passed so providers can resolve cloud resource IDs to app pool/block IDs (e.g. AWS pool external_id -> app pool id).
	SyncBlocks(ctx context.Context, conn *store.CloudConnection, s store.Storer) (*BlockSyncResult, error)
	// SyncAllocations discovers subnets/resources from the cloud and returns create/update diffs for allocations.
	// syncedBlocks are the app blocks for this connection (e.g. VPCs) so the provider knows block names and external IDs.
	SyncAllocations(ctx context.Context, conn *store.CloudConnection, s store.Storer, syncedBlocks []*network.Block) (*AllocationSyncResult, error)
}

// PoolSyncResult holds the result of a pool sync: pools to create or update, and the set of external IDs still present in the cloud (for pruning removed pools).
type PoolSyncResult struct {
	Create             []*network.Pool
	Update             []*network.Pool // must have ID set (existing app pool)
	CurrentExternalIDs []string        // external IDs that exist in the cloud after this sync; pools for this connection not in this set are deleted
}

// BlockSyncResult holds the result of a block sync: blocks to create or update, and the set of block external IDs still present in the cloud (for pruning removed blocks).
type BlockSyncResult struct {
	Create             []*network.Block
	Update             []*network.Block // must have ID set (existing app block)
	CurrentExternalIDs []string         // external IDs that exist in the cloud after this sync; blocks for this connection not in this set are deleted
}

// AllocationSyncResult holds the result of an allocation sync: allocations to create or update, and the set of allocation external IDs still in the cloud (for pruning).
type AllocationSyncResult struct {
	Create             []*network.Allocation
	Update             []*network.Allocation // must have Id set (existing app allocation)
	CurrentExternalIDs []string             // external IDs that exist in the cloud after this sync; allocations for this connection not in this set are cleared (IPAM) or deleted
}

// ConnectionWithEnvMapping is optional: when a provider needs to map cloud scope/region to app environment.
type ConnectionWithEnvMapping interface {
	// EnvironmentIDForScope returns the app environment ID to use for the given cloud scope/region.
	// Returns uuid.Nil if not configured.
	EnvironmentIDForScope(conn *store.CloudConnection, scopeOrRegion string) uuid.UUID
}

// PushProvider is optional: when a provider supports bi-directional sync (read_write mode).
// Used when connection.SyncMode == "read_write" to push app changes to the cloud.
type PushProvider interface {
	CloudProvider
	// SupportsPush returns true if this provider can push create/update to the cloud.
	SupportsPush() bool
	// CreatePoolInCloud creates a pool in the cloud and returns its external ID.
	// parentExternalID is the parent pool's external ID when creating a sub-pool; empty for top-level.
	CreatePoolInCloud(ctx context.Context, conn *store.CloudConnection, pool *network.Pool, parentExternalID string) (externalID string, err error)
	// DeletePoolInCloud deletes the pool in the cloud (e.g. when IPAM conflict resolution and user deleted the pool in the app).
	DeletePoolInCloud(ctx context.Context, conn *store.CloudConnection, externalID string) error
	// AllocateBlockInCloud allocates CIDR from the pool to a resource (e.g. VPC) and returns the block's external ID.
	AllocateBlockInCloud(ctx context.Context, conn *store.CloudConnection, poolExternalID string, block *network.Block) (externalID string, err error)
	// CreateAllocationInCloud creates a subnet/allocation in the cloud and returns its external ID.
	CreateAllocationInCloud(ctx context.Context, conn *store.CloudConnection, blockExternalID string, alloc *network.Allocation) (externalID string, err error)
	// DeleteBlockInCloud deletes the block in the cloud (e.g. VPC) when IPAM conflict resolution and user deleted the block in the app.
	DeleteBlockInCloud(ctx context.Context, conn *store.CloudConnection, externalID string) error
	// DeleteAllocationInCloud deletes the allocation in the cloud (e.g. subnet) when IPAM conflict resolution and user deleted the allocation in the app.
	DeleteAllocationInCloud(ctx context.Context, conn *store.CloudConnection, externalID string) error
}
