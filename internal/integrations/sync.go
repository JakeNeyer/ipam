package integrations

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/JakeNeyer/ipam/internal/logger"
	"github.com/JakeNeyer/ipam/network"
	"github.com/JakeNeyer/ipam/store"
	"github.com/google/uuid"
)

// SyncPools runs pool sync for a connection using its registered provider and applies diffs to the store.
// Returns an error if the provider is not registered or sync fails.
func SyncPools(ctx context.Context, s store.Storer, connID uuid.UUID) error {
	conn, err := s.GetCloudConnection(connID)
	if err != nil {
		logger.Error("sync pools: get connection", slog.String("connection_id", connID.String()), logger.ErrAttr(err))
		return fmt.Errorf("get connection: %w", err)
	}
	logger.Info("sync pools started", slog.String("connection_id", connID.String()), slog.String("connection_name", conn.Name))
	p := Get(conn.Provider)
	if p == nil {
		logger.Error("sync pools: provider not registered", slog.String("connection_id", connID.String()), slog.String("connection_name", conn.Name), slog.String("provider", conn.Provider))
		return fmt.Errorf("provider %q not registered", conn.Provider)
	}
	if !p.SupportsPools() {
		logger.Error("sync pools: provider does not support pool sync", slog.String("connection_id", connID.String()), slog.String("connection_name", conn.Name))
		return fmt.Errorf("provider %q does not support pool sync", conn.Provider)
	}
	result, err := p.SyncPools(ctx, conn)
	if err != nil {
		logger.Error("sync pools failed", slog.String("connection_id", connID.String()), slog.String("connection_name", conn.Name), logger.ErrAttr(err))
		return fmt.Errorf("sync pools: %w", err)
	}
	if err := applyPoolDiffs(s, conn, result); err != nil {
		logger.Error("sync pools: apply diffs failed", slog.String("connection_id", connID.String()), slog.String("connection_name", conn.Name), logger.ErrAttr(err))
		return err
	}
	logger.Info("sync pools completed", slog.String("connection_id", connID.String()), slog.String("connection_name", conn.Name))
	return nil
}

// SyncBlocks runs block sync for a connection using its registered provider and applies diffs to the store.
func SyncBlocks(ctx context.Context, s store.Storer, connID uuid.UUID) error {
	conn, err := s.GetCloudConnection(connID)
	if err != nil {
		logger.Error("sync blocks: get connection", slog.String("connection_id", connID.String()), logger.ErrAttr(err))
		return fmt.Errorf("get connection: %w", err)
	}
	logger.Info("sync blocks started", slog.String("connection_id", connID.String()), slog.String("connection_name", conn.Name))
	p := Get(conn.Provider)
	if p == nil {
		logger.Error("sync blocks: provider not registered", slog.String("connection_id", connID.String()), slog.String("connection_name", conn.Name), slog.String("provider", conn.Provider))
		return fmt.Errorf("provider %q not registered", conn.Provider)
	}
	if !p.SupportsBlocks() {
		logger.Error("sync blocks: provider does not support block sync", slog.String("connection_id", connID.String()), slog.String("connection_name", conn.Name))
		return fmt.Errorf("provider %q does not support block sync", conn.Provider)
	}
	result, err := p.SyncBlocks(ctx, conn, s)
	if err != nil {
		logger.Error("sync blocks failed", slog.String("connection_id", connID.String()), slog.String("connection_name", conn.Name), logger.ErrAttr(err))
		return fmt.Errorf("sync blocks: %w", err)
	}
	if err := applyBlockDiffs(s, conn, result); err != nil {
		logger.Error("sync blocks: apply diffs failed", slog.String("connection_id", connID.String()), slog.String("connection_name", conn.Name), logger.ErrAttr(err))
		return err
	}
	logger.Info("sync blocks completed", slog.String("connection_id", connID.String()), slog.String("connection_name", conn.Name))
	return nil
}

// poolNamesMatch returns true when cloud name and app name refer to the same logical pool (e.g. "ipam-pool-xxx (Staging pool)" and "Staging pool").
func poolNamesMatch(cloudName, appName string) bool {
	if cloudName == appName {
		return true
	}
	// Cloud sync uses "id (Name tag)"; app pool may be just "Staging pool".
	if appName != "" && len(cloudName) > len(appName)+3 && cloudName[len(cloudName)-len(appName)-2:] == " ("+appName+")" {
		return true
	}
	return false
}

func applyPoolDiffs(s store.Storer, conn *store.CloudConnection, result *PoolSyncResult) error {
	if result == nil || conn == nil {
		return nil
	}
	connID := conn.ID
	conflictIPAM := conn.ConflictResolution == "ipam"
	// Build map of existing pools by external_id (org-wide, including soft-deleted) so we don't duplicate and so we match soft-deleted for IPAM conflict
	existingByExtID := make(map[string]*network.Pool)
	// Unlinked pools (no external_id): adopt instead of creating duplicate when cloud pool matches by name (any connection or none)
	var unlinkedPools []*network.Pool
	if len(result.Create) > 0 && result.Create[0].OrganizationID != uuid.Nil {
		existing, _ := s.ListPoolsByOrganizationIncludingDeleted(result.Create[0].OrganizationID)
		for _, p := range existing {
			if p.ExternalID != "" {
				existingByExtID[p.ExternalID] = p
			} else if p.DeletedAt == nil {
				unlinkedPools = append(unlinkedPools, p)
			}
		}
	}
	for _, pool := range result.Create {
		if pool.ConnectionID == nil {
			pool.ConnectionID = &connID
		}
		if existing, ok := existingByExtID[pool.ExternalID]; ok {
			if conflictIPAM {
				continue // app wins: do not overwrite existing
			}
			pool.ID = existing.ID
			pool.ConnectionID = &connID // associate with this connection (last synced from)
			if err := s.UpdatePool(pool.ID, pool); err != nil {
				return err
			}
			continue
		}
		// Adopt an unlinked app pool so we don't create a duplicate row. Match by name+env, or by CIDR+env when names don't match (e.g. AWS has no Name tag).
		var adopted *network.Pool
		for i, u := range unlinkedPools {
			if u.EnvironmentID != pool.EnvironmentID {
				continue
			}
			nameMatch := poolNamesMatch(pool.Name, u.Name)
			cidrMatch := pool.CIDR != "" && u.CIDR != "" && pool.CIDR == u.CIDR
			if !nameMatch && !cidrMatch {
				continue
			}
			adopted = u
			unlinkedPools = append(unlinkedPools[:i], unlinkedPools[i+1:]...)
			break
		}
		if adopted != nil {
			pool.ID = adopted.ID
			// Prefer app pool name when adopting (user's label vs cloud "ipam-pool-xxx" or "ipam-pool-xxx (Name)")
			if adopted.Name != "" {
				pool.Name = adopted.Name
			}
			// Keep app pool's CIDR when cloud has no provisioned CIDR yet (don't overwrite e.g. 0.0.0.0/0 with empty)
			if pool.CIDR == "" && adopted.CIDR != "" {
				pool.CIDR = adopted.CIDR
			}
			if err := s.UpdatePool(pool.ID, pool); err != nil {
				return err
			}
			existingByExtID[pool.ExternalID] = pool
			continue
		}
		if pool.ID == uuid.Nil {
			pool.ID = s.GenerateID()
		}
		if err := s.CreatePool(pool); err != nil {
			return err
		}
		existingByExtID[pool.ExternalID] = pool
	}
	for _, pool := range result.Update {
		if pool.ID == uuid.Nil {
			continue
		}
		if err := s.UpdatePool(pool.ID, pool); err != nil {
			return err
		}
	}
	// Prune pools that no longer exist in the cloud (only when provider reported current set)
	if result.CurrentExternalIDs != nil {
		connForPrune, err := s.GetCloudConnection(connID)
		if err != nil {
			return fmt.Errorf("get connection for prune: %w", err)
		}
		existing, _ := s.ListPoolsByOrganization(connForPrune.OrganizationID)
		currentSet := make(map[string]bool)
		for _, extID := range result.CurrentExternalIDs {
			currentSet[extID] = true
		}
		for _, p := range existing {
			if p.ConnectionID == nil || *p.ConnectionID != connID || p.ExternalID == "" {
				continue
			}
			if !currentSet[p.ExternalID] {
				// When read-write + IPAM: pool was deleted in cloud; clear external_id so PushPoolsToCloud re-creates it this sync.
				if conn.SyncMode == "read_write" && conn.ConflictResolution == "ipam" {
					p.ExternalID = ""
					if err := s.UpdatePool(p.ID, p); err != nil {
						return fmt.Errorf("clear pool %s external_id for re-push: %w", p.Name, err)
					}
					logger.Info("sync cleared pool for re-push (pool deleted in cloud, IPAM source of truth)", slog.String("connection_id", connID.String()), slog.String("pool_name", p.Name))
				} else {
					if err := s.DeletePool(p.ID); err != nil {
						return fmt.Errorf("delete removed pool %s: %w", p.ExternalID, err)
					}
				}
			}
		}
	}
	return nil
}

// blockIdentMatch returns true when cloud and app block refer to the same logical block (same pool+CIDR, or CIDR+env when app has no pool).
func blockIdentMatch(cloud, app *network.Block) bool {
	if cloud.CIDR != "" && app.CIDR != "" && cloud.CIDR != app.CIDR {
		return false
	}
	if cloud.PoolID != nil && app.PoolID != nil {
		return *cloud.PoolID == *app.PoolID
	}
	// App block has no pool (created before pool assignment): match by CIDR + environment
	if app.PoolID == nil && cloud.EnvironmentID != uuid.Nil && app.EnvironmentID == cloud.EnvironmentID {
		return true
	}
	return false
}

func applyBlockDiffs(s store.Storer, conn *store.CloudConnection, result *BlockSyncResult) error {
	if result == nil || conn == nil {
		return nil
	}
	connID := conn.ID
	conflictIPAM := conn.ConflictResolution == "ipam"
	// Include soft-deleted blocks so we match cloud resources to existing rows (IPAM conflict: do not duplicate).
	existingByExtID := make(map[string]*network.Block)
	var unlinkedBlocks []*network.Block
	if len(result.Create) > 0 && result.Create[0].OrganizationID != uuid.Nil {
		existing, _, _ := s.ListBlocksFilteredIncludingDeleted("", nil, nil, &result.Create[0].OrganizationID, false, "", nil, 0, 0)
		for _, b := range existing {
			if b.ExternalID != "" && b.ConnectionID != nil && *b.ConnectionID == connID {
				existingByExtID[b.ExternalID] = b
			} else if b.ExternalID == "" && b.DeletedAt == nil {
				unlinkedBlocks = append(unlinkedBlocks, b)
			}
		}
	}
	for _, block := range result.Create {
		if block.ConnectionID == nil {
			block.ConnectionID = &connID
		}
		if existing, ok := existingByExtID[block.ExternalID]; ok {
			if conflictIPAM {
				continue
			}
			block.ID = existing.ID
			if err := s.UpdateBlock(block.ID, block); err != nil {
				return err
			}
			continue
		}
		// Adopt an unlinked app block with same pool and CIDR/name so we don't create a duplicate
		var adopted *network.Block
		for i, u := range unlinkedBlocks {
			if !blockIdentMatch(block, u) {
				continue
			}
			adopted = u
			unlinkedBlocks = append(unlinkedBlocks[:i], unlinkedBlocks[i+1:]...)
			break
		}
		if adopted != nil {
			block.ID = adopted.ID
			// Prefer app block name when adopting so allocations (referenced by block_name) continue to match.
			if adopted.Name != "" {
				block.Name = adopted.Name
			}
			if err := s.UpdateBlock(block.ID, block); err != nil {
				return err
			}
			existingByExtID[block.ExternalID] = block
			continue
		}
		if block.ID == uuid.Nil {
			block.ID = s.GenerateID()
		}
		if err := s.CreateBlock(block); err != nil {
			return err
		}
		existingByExtID[block.ExternalID] = block
	}
	for _, block := range result.Update {
		if block.ID == uuid.Nil {
			continue
		}
		if err := s.UpdateBlock(block.ID, block); err != nil {
			return err
		}
	}
	if result.CurrentExternalIDs != nil {
		connForPrune, err := s.GetCloudConnection(connID)
		if err != nil {
			return fmt.Errorf("get connection for prune: %w", err)
		}
		existing, _, _ := s.ListBlocksFiltered("", nil, nil, &connForPrune.OrganizationID, false, "", &connID, 0, 0)
		currentSet := make(map[string]bool)
		for _, extID := range result.CurrentExternalIDs {
			currentSet[extID] = true
		}
		for _, b := range existing {
			if b.ExternalID == "" {
				continue
			}
			if !currentSet[b.ExternalID] {
				allocs, _, _ := s.ListAllocationsFiltered("", b.Name, uuid.Nil, &connForPrune.OrganizationID, "", &connID, 10000, 0)
				// When read-write + IPAM conflict: VPC was deleted in AWS; clear external_id so PushBlocksToCloud re-creates it this sync.
				if conn.SyncMode == "read_write" && conn.ConflictResolution == "ipam" {
					for _, a := range allocs {
						if a.ConnectionID != nil && *a.ConnectionID == connID && a.ExternalID != "" {
							a.ExternalID = ""
							_ = s.UpdateAllocation(a.Id, a)
						}
					}
					b.ExternalID = ""
					// Keep ConnectionID and PoolID so PushBlocksToCloud finds this block in the pool's block list
					if err := s.UpdateBlock(b.ID, b); err != nil {
						return fmt.Errorf("clear block %s external_id for re-push: %w", b.Name, err)
					}
					logger.Info("sync cleared block for re-push (VPC deleted in cloud, IPAM source of truth)", slog.String("connection_id", connID.String()), slog.String("block_name", b.Name))
				} else {
					// Read-only or cloud source of truth: remove block and its allocations from app
					for _, a := range allocs {
						if a.ConnectionID != nil && *a.ConnectionID == connID {
							_ = s.DeleteAllocation(a.Id)
						}
					}
					if err := s.DeleteBlock(b.ID); err != nil {
						return fmt.Errorf("delete removed block %s: %w", b.ExternalID, err)
					}
				}
			}
		}
	}
	return nil
}

// SyncAllocations runs allocation sync for a connection when the provider supports it.
// syncedBlocks should be the list of blocks for this connection (e.g. VPCs) from the store.
func SyncAllocations(ctx context.Context, s store.Storer, connID uuid.UUID, syncedBlocks []*network.Block) error {
	if len(syncedBlocks) == 0 {
		return nil
	}
	conn, err := s.GetCloudConnection(connID)
	if err != nil {
		logger.Error("sync allocations: get connection", slog.String("connection_id", connID.String()), logger.ErrAttr(err))
		return fmt.Errorf("get connection: %w", err)
	}
	logger.Info("sync allocations started", slog.String("connection_id", connID.String()), slog.String("connection_name", conn.Name))
	p := Get(conn.Provider)
	if p == nil {
		logger.Error("sync allocations: provider not registered", slog.String("connection_id", connID.String()), slog.String("connection_name", conn.Name), slog.String("provider", conn.Provider))
		return fmt.Errorf("provider %q not registered", conn.Provider)
	}
	if !p.SupportsAllocations() {
		logger.Info("sync allocations skipped: provider does not support allocation sync", slog.String("connection_id", connID.String()), slog.String("connection_name", conn.Name))
		return nil
	}
	result, err := p.SyncAllocations(ctx, conn, s, syncedBlocks)
	if err != nil {
		logger.Error("sync allocations failed", slog.String("connection_id", connID.String()), slog.String("connection_name", conn.Name), logger.ErrAttr(err))
		return fmt.Errorf("sync allocations: %w", err)
	}
	if err := applyAllocationDiffs(s, conn, result); err != nil {
		logger.Error("sync allocations: apply diffs failed", slog.String("connection_id", connID.String()), slog.String("connection_name", conn.Name), logger.ErrAttr(err))
		return err
	}
	logger.Info("sync allocations completed", slog.String("connection_id", connID.String()), slog.String("connection_name", conn.Name))
	return nil
}

// PushPoolsToCloud pushes app pools (in targetEnvID with no external_id yet) to the cloud for read-write connections.
// targetEnvID is the connection's target environment (e.g. from provider config); pass uuid.Nil to skip.
func PushPoolsToCloud(ctx context.Context, s store.Storer, conn *store.CloudConnection, targetEnvID uuid.UUID) error {
	if conn.SyncMode != "read_write" || targetEnvID == uuid.Nil {
		return nil
	}
	p := Get(conn.Provider)
	if p == nil {
		return nil
	}
	pushProv, ok := p.(PushProvider)
	if !ok || !pushProv.SupportsPush() {
		return nil
	}
	pools, err := s.ListPoolsByOrganization(conn.OrganizationID)
	if err != nil {
		return err
	}
	connID := conn.ID
	for _, pool := range pools {
		if pool.EnvironmentID != targetEnvID || pool.ExternalID != "" {
			continue
		}
		if pool.ConnectionID != nil && *pool.ConnectionID != connID {
			continue
		}
		extID, err := pushProv.CreatePoolInCloud(ctx, conn, pool, "")
		if err != nil {
			logger.Error("sync push pool to cloud failed", slog.String("connection_id", connID.String()), slog.String("pool_name", pool.Name), logger.ErrAttr(err))
			return fmt.Errorf("push pool %q: %w", pool.Name, err)
		}
		pool.Provider = conn.Provider
		pool.ExternalID = extID
		pool.ConnectionID = &connID
		if err := s.UpdatePool(pool.ID, pool); err != nil {
			return err
		}
		logger.Info("sync pushed pool to cloud", slog.String("connection_id", connID.String()), slog.String("pool_name", pool.Name), slog.String("external_id", extID))
	}
	return nil
}

// ApplyPoolDeletesInCloud deletes in the cloud any pools that were soft-deleted in the app (IPAM conflict resolution).
// After each successful cloud delete, the pool row is permanently removed from the store.
func ApplyPoolDeletesInCloud(ctx context.Context, s store.Storer, conn *store.CloudConnection) error {
	if conn.SyncMode != "read_write" || conn.ConflictResolution != "ipam" {
		return nil
	}
	p := Get(conn.Provider)
	if p == nil {
		return nil
	}
	pushProv, ok := p.(PushProvider)
	if !ok || !pushProv.SupportsPush() {
		return nil
	}
	pending, err := s.ListPoolsPendingCloudDelete(conn.ID)
	if err != nil {
		return fmt.Errorf("list pools pending cloud delete: %w", err)
	}
	for _, pool := range pending {
		if err := pushProv.DeletePoolInCloud(ctx, conn, pool.ExternalID); err != nil {
			logger.Error("sync delete pool in cloud failed", slog.String("connection_id", conn.ID.String()), slog.String("pool_id", pool.ID.String()), slog.String("external_id", pool.ExternalID), logger.ErrAttr(err))
			return fmt.Errorf("delete pool %s in cloud: %w", pool.ExternalID, err)
		}
		if err := s.DeletePool(pool.ID); err != nil {
			return fmt.Errorf("delete pool %s after cloud delete: %w", pool.ID, err)
		}
		logger.Info("sync deleted pool in cloud", slog.String("connection_id", conn.ID.String()), slog.String("pool_name", pool.Name), slog.String("external_id", pool.ExternalID))
	}
	return nil
}

// ApplyAllocationDeletesInCloud deletes in the cloud any allocations that were soft-deleted in the app (IPAM conflict resolution).
// Must run before ApplyBlockDeletesInCloud so subnets are deleted before their VPC.
func ApplyAllocationDeletesInCloud(ctx context.Context, s store.Storer, conn *store.CloudConnection) error {
	if conn.SyncMode != "read_write" || conn.ConflictResolution != "ipam" {
		return nil
	}
	p := Get(conn.Provider)
	if p == nil {
		return nil
	}
	pushProv, ok := p.(PushProvider)
	if !ok || !pushProv.SupportsPush() {
		return nil
	}
	pending, err := s.ListAllocationsPendingCloudDelete(conn.ID)
	if err != nil {
		return fmt.Errorf("list allocations pending cloud delete: %w", err)
	}
	for _, alloc := range pending {
		if err := pushProv.DeleteAllocationInCloud(ctx, conn, alloc.ExternalID); err != nil {
			logger.Error("sync delete allocation in cloud failed", slog.String("connection_id", conn.ID.String()), slog.String("allocation_id", alloc.Id.String()), slog.String("external_id", alloc.ExternalID), logger.ErrAttr(err))
			return fmt.Errorf("delete allocation %s in cloud: %w", alloc.ExternalID, err)
		}
		if err := s.DeleteAllocation(alloc.Id); err != nil {
			return fmt.Errorf("delete allocation %s after cloud delete: %w", alloc.Id, err)
		}
		logger.Info("sync deleted allocation in cloud", slog.String("connection_id", conn.ID.String()), slog.String("allocation_name", alloc.Name), slog.String("external_id", alloc.ExternalID))
	}
	return nil
}

// ApplyBlockDeletesInCloud deletes in the cloud any blocks that were soft-deleted in the app (IPAM conflict resolution).
// Run after ApplyAllocationDeletesInCloud so subnets are deleted before their VPC.
func ApplyBlockDeletesInCloud(ctx context.Context, s store.Storer, conn *store.CloudConnection) error {
	if conn.SyncMode != "read_write" || conn.ConflictResolution != "ipam" {
		return nil
	}
	p := Get(conn.Provider)
	if p == nil {
		return nil
	}
	pushProv, ok := p.(PushProvider)
	if !ok || !pushProv.SupportsPush() {
		return nil
	}
	pending, err := s.ListBlocksPendingCloudDelete(conn.ID)
	if err != nil {
		return fmt.Errorf("list blocks pending cloud delete: %w", err)
	}
	for _, block := range pending {
		// Only call provider when external_id looks like a VPC ID. Skip ipam-pool-* etc. (legacy blocks created from pool allocations); still remove the row.
		extID := block.ExternalID
		if extID != "" && len(extID) >= 4 && (extID[:4] == "vpc-") {
			if err := pushProv.DeleteBlockInCloud(ctx, conn, extID); err != nil {
				logger.Error("sync delete block in cloud failed", slog.String("connection_id", conn.ID.String()), slog.String("block_id", block.ID.String()), slog.String("external_id", extID), logger.ErrAttr(err))
				return fmt.Errorf("delete block %s in cloud: %w", extID, err)
			}
			logger.Info("sync deleted block in cloud", slog.String("connection_id", conn.ID.String()), slog.String("block_name", block.Name), slog.String("external_id", extID))
		} else {
			logger.Info("sync skipping block cloud delete (not a VPC id)", slog.String("connection_id", conn.ID.String()), slog.String("block_id", block.ID.String()), slog.String("block_name", block.Name), slog.String("external_id", extID))
		}
		if err := s.DeleteBlock(block.ID); err != nil {
			return fmt.Errorf("delete block %s after cloud delete: %w", block.ID, err)
		}
	}
	return nil
}

// PushBlocksToCloud pushes app blocks (in synced pools with no external_id yet) to the cloud for read-write connections.
func PushBlocksToCloud(ctx context.Context, s store.Storer, conn *store.CloudConnection) error {
	if conn.SyncMode != "read_write" {
		return nil
	}
	p := Get(conn.Provider)
	if p == nil {
		return nil
	}
	pushProv, ok := p.(PushProvider)
	if !ok || !pushProv.SupportsPush() {
		return nil
	}
	pools, err := s.ListPoolsByOrganization(conn.OrganizationID)
	if err != nil {
		return err
	}
	connID := conn.ID
	for _, pool := range pools {
		if pool.ConnectionID == nil || *pool.ConnectionID != connID || pool.ExternalID == "" {
			continue
		}
		blocks, err := s.ListBlocksByPool(pool.ID)
		if err != nil {
			return err
		}
		// When IPAM conflict: also include env blocks with no PoolID whose CIDR is contained in pool (blocks created before pool adoption)
		if conn.ConflictResolution == "ipam" && pool.CIDR != "" {
			envBlocks, _, _ := s.ListBlocksFiltered("", &pool.EnvironmentID, nil, nil, false, "", nil, 10000, 0)
			for _, b := range envBlocks {
				if b.ExternalID != "" || b.PoolID != nil {
					continue
				}
				if b.CIDR == "" {
					continue
				}
				contained, _ := network.Contains(pool.CIDR, b.CIDR)
				if !contained {
					continue
				}
				// Avoid dupes (block might already be in blocks via ListBlocksByPool)
				seen := false
				for _, exist := range blocks {
					if exist.ID == b.ID {
						seen = true
						break
					}
				}
				if !seen {
					blocks = append(blocks, b)
				}
			}
		}
		for _, block := range blocks {
			if block.ConnectionID != nil && *block.ConnectionID == connID && block.ExternalID != "" {
				continue
			}
			if block.ExternalID != "" {
				continue
			}
			// Pre-check: block CIDR must be contained in pool's CIDR (avoids AllocateIpamPoolCidr errors after adoption when app pool CIDR differs from AWS)
			if pool.CIDR != "" {
				contained, _ := network.Contains(pool.CIDR, block.CIDR)
				if !contained {
					logger.Info("sync push block skipped (block CIDR not contained in pool CIDR)",
						slog.String("connection_id", connID.String()), slog.String("block_name", block.Name),
						slog.String("block_cidr", block.CIDR), slog.String("pool_cidr", pool.CIDR))
					continue
				}
			}
			extID, err := pushProv.AllocateBlockInCloud(ctx, conn, pool.ExternalID, block)
			if err != nil {
				errStr := err.Error()
				// Pool CIDR may still be provisioning, or orphaned allocation (VPC deleted, IPAM not released) - skip and retry on next sync.
				if strings.Contains(errStr, "InvalidParameterValue") && (strings.Contains(errStr, "overlaps with an existing allocation") || strings.Contains(errStr, "not a subnet of any pool cidrs")) {
					logger.Info("sync push block skipped (pool CIDR may still be provisioning or orphaned allocation)", slog.String("connection_id", connID.String()), slog.String("block_name", block.Name), logger.ErrAttr(err))
					continue
				}
				if strings.Contains(errStr, "release orphaned allocation") {
					logger.Info("sync push block skipped (release orphaned allocation failed)", slog.String("connection_id", connID.String()), slog.String("block_name", block.Name), logger.ErrAttr(err))
					continue
				}
				logger.Error("sync push block to cloud failed", slog.String("connection_id", connID.String()), slog.String("block_name", block.Name), logger.ErrAttr(err))
				return fmt.Errorf("push block %q: %w", block.Name, err)
			}
			block.Provider = conn.Provider
			block.ExternalID = extID
			block.ConnectionID = &connID
			if block.PoolID == nil {
				block.PoolID = &pool.ID
			}
			if err := s.UpdateBlock(block.ID, block); err != nil {
				return err
			}
			logger.Info("sync pushed block to cloud", slog.String("connection_id", connID.String()), slog.String("block_name", block.Name), slog.String("external_id", extID))
		}
	}
	return nil
}

// PushAllocationsToCloud pushes app allocations (in synced blocks with no external_id yet) to the cloud for read-write connections.
func PushAllocationsToCloud(ctx context.Context, s store.Storer, conn *store.CloudConnection) error {
	if conn.SyncMode != "read_write" {
		return nil
	}
	p := Get(conn.Provider)
	if p == nil {
		return nil
	}
	pushProv, ok := p.(PushProvider)
	if !ok || !pushProv.SupportsPush() {
		return nil
	}
	blocks, _, err := s.ListBlocksFiltered("", nil, nil, &conn.OrganizationID, false, "", &conn.ID, 10000, 0)
	if err != nil {
		return err
	}
	connID := conn.ID
	for _, block := range blocks {
		if block.ExternalID == "" {
			continue
		}
		allocs, _, err := s.ListAllocationsFiltered("", block.Name, uuid.Nil, &conn.OrganizationID, "", nil, 10000, 0)
		if err != nil {
			return err
		}
		for _, a := range allocs {
			if a.ExternalID != "" {
				continue
			}
			if a.Block.Name != block.Name {
				continue
			}
			extID, err := pushProv.CreateAllocationInCloud(ctx, conn, block.ExternalID, a)
			if err != nil {
				logger.Error("sync push allocation to cloud failed", slog.String("connection_id", connID.String()), slog.String("allocation_name", a.Name), logger.ErrAttr(err))
				return fmt.Errorf("push allocation %q: %w", a.Name, err)
			}
			a.Provider = conn.Provider
			a.ExternalID = extID
			a.ConnectionID = &connID
			if err := s.UpdateAllocation(a.Id, a); err != nil {
				return err
			}
			logger.Info("sync pushed allocation to cloud", slog.String("connection_id", connID.String()), slog.String("allocation_name", a.Name), slog.String("external_id", extID))
		}
	}
	return nil
}

// allocationIdentMatch returns true when cloud and app allocation refer to the same logical allocation (same subnet CIDR; block match by name or CIDR).
func allocationIdentMatch(cloud, app *network.Allocation) bool {
	if cloud.Block.CIDR != "" && app.Block.CIDR != "" && cloud.Block.CIDR != app.Block.CIDR {
		return false
	}
	return poolNamesMatch(cloud.Block.Name, app.Block.Name) || cloud.Block.Name == app.Block.Name
}

func applyAllocationDiffs(s store.Storer, conn *store.CloudConnection, result *AllocationSyncResult) error {
	if result == nil || conn == nil {
		return nil
	}
	connID := conn.ID
	conflictIPAM := conn.ConflictResolution == "ipam"
	// Include soft-deleted allocations so we match cloud resources to existing rows (IPAM conflict: do not duplicate).
	existingByExtID := make(map[string]*network.Allocation)
	var unlinkedAllocs []*network.Allocation
	if len(result.Create) > 0 && conn.OrganizationID != uuid.Nil {
		// Use connectionID=nil to include user-created allocations (no connection yet) for adoption
		existing, _, _ := s.ListAllocationsFilteredIncludingDeleted("", "", uuid.Nil, &conn.OrganizationID, "", nil, 0, 0)
		for _, a := range existing {
			if a.ExternalID != "" && a.ConnectionID != nil && *a.ConnectionID == connID {
				existingByExtID[a.ExternalID] = a
			} else if a.ExternalID == "" && a.DeletedAt == nil {
				unlinkedAllocs = append(unlinkedAllocs, a)
			}
		}
	}
	for _, alloc := range result.Create {
		if alloc.ConnectionID == nil {
			alloc.ConnectionID = &connID
		}
		if alloc.Provider == "" {
			alloc.Provider = "aws"
		}
		if existing, ok := existingByExtID[alloc.ExternalID]; ok {
			if conflictIPAM {
				continue
			}
			alloc.Id = existing.Id
			if err := s.UpdateAllocation(alloc.Id, alloc); err != nil {
				return err
			}
			continue
		}
		// Adopt an unlinked app allocation with same block + CIDR so we don't create a duplicate
		var adopted *network.Allocation
		for i, u := range unlinkedAllocs {
			if !allocationIdentMatch(alloc, u) {
				continue
			}
			adopted = u
			unlinkedAllocs = append(unlinkedAllocs[:i], unlinkedAllocs[i+1:]...)
			break
		}
		if adopted != nil {
			alloc.Id = adopted.Id
			// Prefer app allocation name when adopting to preserve user labels
			if adopted.Name != "" {
				alloc.Name = adopted.Name
			}
			if err := s.UpdateAllocation(alloc.Id, alloc); err != nil {
				return err
			}
			existingByExtID[alloc.ExternalID] = alloc
			continue
		}
		if alloc.Id == uuid.Nil {
			alloc.Id = s.GenerateID()
		}
		if err := s.CreateAllocation(alloc.Id, alloc); err != nil {
			return err
		}
		existingByExtID[alloc.ExternalID] = alloc
	}
	for _, alloc := range result.Update {
		if alloc.Id == uuid.Nil {
			continue
		}
		if err := s.UpdateAllocation(alloc.Id, alloc); err != nil {
			return err
		}
	}
	// Prune allocations that no longer exist in the cloud (when provider reported current set)
	if result.CurrentExternalIDs != nil {
		connForPrune, err := s.GetCloudConnection(connID)
		if err != nil {
			return fmt.Errorf("get connection for prune: %w", err)
		}
		currentSet := make(map[string]bool)
		for _, extID := range result.CurrentExternalIDs {
			currentSet[extID] = true
		}
		existing, _, _ := s.ListAllocationsFiltered("", "", uuid.Nil, &connForPrune.OrganizationID, "", &connID, 10000, 0)
		for _, a := range existing {
			if a.ExternalID == "" {
				continue
			}
			if !currentSet[a.ExternalID] {
				// When read-write + IPAM: subnet was deleted in cloud; clear external_id so PushAllocationsToCloud re-creates it this sync.
				if conn.SyncMode == "read_write" && conn.ConflictResolution == "ipam" {
					a.ExternalID = ""
					if err := s.UpdateAllocation(a.Id, a); err != nil {
						return fmt.Errorf("clear allocation %s external_id for re-push: %w", a.Name, err)
					}
					logger.Info("sync cleared allocation for re-push (subnet deleted in cloud, IPAM source of truth)", slog.String("connection_id", connID.String()), slog.String("allocation_name", a.Name))
				} else {
					if err := s.DeleteAllocation(a.Id); err != nil {
						return fmt.Errorf("delete removed allocation %s: %w", a.ExternalID, err)
					}
				}
			}
		}
	}
	return nil
}
