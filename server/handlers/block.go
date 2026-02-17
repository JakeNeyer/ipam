package handlers

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/JakeNeyer/ipam/internal/integrations"
	"github.com/JakeNeyer/ipam/network"
	"github.com/JakeNeyer/ipam/server/auth"
	"github.com/JakeNeyer/ipam/store"
	"github.com/google/uuid"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
)

// blockNamesMatch compares block names for usage attribution (case-insensitive, trimmed).
func blockNamesMatch(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

// derivedBlockUsage returns total, used, available (as strings) and utilization percent for a block.
// All values are derived from CIDR; used is the sum of allocation sizes for this block.
func derivedBlockUsage(s store.Storer, blockName string, blockCIDR string, orgID *uuid.UUID) (totalStr, usedStr, availableStr string, utilPercent float64) {
	totalStr = network.CIDRAddressCountString(blockCIDR)
	total, err := network.CIDRAddressCount(blockCIDR)
	if err != nil {
		return totalStr, "0", totalStr, 0
	}
	var allocs []*network.Allocation
	var listErr error
	if orgID != nil {
		allocs, _, listErr = s.ListAllocationsFiltered("", blockName, uuid.Nil, orgID, "", nil, 0, 0)
	} else {
		allocs, listErr = s.ListAllocations()
	}
	if listErr != nil {
		return totalStr, "0", totalStr, 0
	}
	used := new(big.Int)
	for _, a := range allocs {
		if !blockNamesMatch(a.Block.Name, blockName) {
			continue
		}
		c, err := network.CIDRAddressCount(a.Block.CIDR)
		if err != nil {
			continue
		}
		used.Add(used, c)
	}
	usedStr = used.String()
	available := new(big.Int).Sub(total, used)
	if available.Sign() < 0 {
		available.SetInt64(0)
	}
	availableStr = available.String()
	if total.Sign() > 0 {
		// utilization: used/total * 100 (approximate for huge numbers via float)
		tf, _ := new(big.Float).SetInt(total).Float64()
		uf, _ := new(big.Float).SetInt(used).Float64()
		if tf > 0 {
			utilPercent = (uf / tf) * 100.0
		}
	}
	return totalStr, usedStr, availableStr, utilPercent
}

// CreateBlock handler
func NewCreateBlockUseCase(s store.Storer) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input createBlockInput, output *blockOutput) error {
		if input.Name == "" || input.CIDR == "" {
			return status.Wrap(errors.New("name and CIDR are required"), status.InvalidArgument)
		}

		if valid := network.ValidateCIDR(input.CIDR); !valid {
			return status.Wrap(errors.New("invalid CIDR format"), status.InvalidArgument)
		}

		user := auth.UserFromContext(ctx)
		orgID := auth.ResolveOrgID(ctx, user, uuid.Nil)
		if user != nil && input.EnvironmentID != uuid.Nil {
			env, err := s.GetEnvironment(input.EnvironmentID)
			if err != nil {
				return status.Wrap(errors.New("environment not found"), status.NotFound)
			}
			if userOrg := auth.UserOrgForAccess(ctx, user); userOrg != uuid.Nil && env.OrganizationID != userOrg {
				return status.Wrap(errors.New("environment not found"), status.NotFound)
			}
		}

		// Orphan blocks (no environment) must be scoped to an organization
		blockOrgID := input.OrganizationID
		if input.EnvironmentID == uuid.Nil {
			if blockOrgID == uuid.Nil && orgID != nil {
				blockOrgID = *orgID
			}
			if blockOrgID == uuid.Nil {
				return status.Wrap(errors.New("organization is required for orphan blocks (blocks without an environment)"), status.InvalidArgument)
			}
		}

		// Pool is optional; if set, pool must belong to the block's environment and block CIDR must be contained in pool CIDR
		if input.PoolID != nil && *input.PoolID != uuid.Nil {
			if input.EnvironmentID == uuid.Nil {
				return status.Wrap(errors.New("pool_id can only be set for blocks in an environment"), status.InvalidArgument)
			}
			pool, err := s.GetPool(*input.PoolID)
			if err != nil {
				return status.Wrap(errors.New("pool not found"), status.NotFound)
			}
			if pool.EnvironmentID != input.EnvironmentID {
				return status.Wrap(errors.New("pool does not belong to the block's environment"), status.InvalidArgument)
			}
			contained, err := network.Contains(pool.CIDR, input.CIDR)
			if err != nil {
				return status.Wrap(err, status.InvalidArgument)
			}
			if !contained {
				return status.Wrap(
					fmt.Errorf("block CIDR %s must be contained within pool %q CIDR %s", input.CIDR, pool.Name, pool.CIDR),
					status.InvalidArgument,
				)
			}
		}

		// Derive-only: total_ips stored only when it fits in BIGINT; API always derives from CIDR
		totalStored := int(network.CIDRAddressCountInt64(input.CIDR))
		block := &network.Block{
			Name:           input.Name,
			CIDR:           input.CIDR,
			EnvironmentID:  input.EnvironmentID,
			OrganizationID: blockOrgID,
			PoolID:         input.PoolID,
			Usage: network.Usage{
				TotalIPs:     totalStored,
				UsedIPs:      0,
				AvailableIPs: totalStored,
			},
			Children: []network.Block{},
		}

		var existing []*network.Block
		var err error
		if block.EnvironmentID != uuid.Nil {
			existing, err = s.ListBlocksByEnvironment(block.EnvironmentID)
		} else {
			existing, _, err = s.ListBlocksFiltered("", nil, nil, &block.OrganizationID, true, "", nil, 0, 0)
		}
		if err != nil {
			return status.Wrap(err, status.Internal)
		}
		for _, other := range existing {
			overlap, err := network.Overlaps(block.CIDR, other.CIDR)
			if err != nil {
				return status.Wrap(err, status.Internal)
			}
			if overlap {
				envLabel := "the target environment"
				if block.EnvironmentID == uuid.Nil {
					envLabel = "orphaned blocks"
				}
				return status.Wrap(
					fmt.Errorf("CIDR %s overlaps with existing block %q in %s", block.CIDR, other.Name, envLabel),
					status.InvalidArgument,
				)
			}
		}
		reservedOrgID := &block.OrganizationID
		if block.EnvironmentID != uuid.Nil {
			if env, err := s.GetEnvironment(block.EnvironmentID); err == nil {
				reservedOrgID = &env.OrganizationID
			}
		}
		if reserved, err := s.OverlapsReservedBlock(block.CIDR, reservedOrgID); err != nil {
			return status.Wrap(err, status.Internal)
		} else if reserved != nil {
			return status.Wrap(
				fmt.Errorf("CIDR %s overlaps reserved block %s", block.CIDR, reserved.CIDR),
				status.InvalidArgument,
			)
		}

		if block.PoolID != nil {
			pool, err := s.GetPool(*block.PoolID)
			if err == nil && pool.ConnectionID != nil && pool.ExternalID != "" {
				conn, err := s.GetCloudConnection(*pool.ConnectionID)
				if err == nil && conn.SyncMode == "read_write" {
					prov := integrations.Get(conn.Provider)
					if prov != nil {
						if pushProv, ok := prov.(integrations.PushProvider); ok && pushProv.SupportsPush() {
							extID, err := pushProv.AllocateBlockInCloud(ctx, conn, pool.ExternalID, block)
							if err != nil {
								return status.Wrap(fmt.Errorf("push block to cloud: %w", err), status.Internal)
							}
							block.Provider = conn.Provider
							block.ExternalID = extID
							block.ConnectionID = pool.ConnectionID
						}
					}
				}
			}
		}

		if err := s.CreateBlock(block); err != nil {
			return status.Wrap(err, status.Internal)
		}

		totalStr, usedStr, availStr, _ := derivedBlockUsage(s, block.Name, block.CIDR, orgID)
		output.ID = block.ID
		output.Name = block.Name
		output.CIDR = block.CIDR
		output.TotalIPs = totalStr
		output.UsedIPs = usedStr
		output.Available = availStr
		output.EnvironmentID = block.EnvironmentID
		output.OrganizationID = block.OrganizationID
		output.PoolID = block.PoolID
		output.Provider = block.Provider
		output.ExternalID = block.ExternalID
		output.ConnectionID = block.ConnectionID
		return nil
	})

	u.SetTitle("Create Block")
	u.SetDescription("Creates a new network block")
	u.SetExpectedErrors(status.InvalidArgument, status.Internal)
	return u
}

// ListBlocks handler
func NewListBlocksUseCase(s store.Storer) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input listBlocksInput, output *blockListOutput) error {
		user := auth.UserFromContext(ctx)
		if user == nil {
			return status.Wrap(errors.New("unauthorized"), status.Unauthenticated)
		}
		var envID *uuid.UUID
		if input.EnvironmentID != uuid.Nil {
			envID = &input.EnvironmentID
			env, err := s.GetEnvironment(input.EnvironmentID)
			if err != nil {
				return status.Wrap(errors.New("environment not found"), status.NotFound)
			}
			if userOrg := auth.UserOrgForAccess(ctx, user); userOrg != uuid.Nil && env.OrganizationID != userOrg {
				return status.Wrap(errors.New("environment not found"), status.NotFound)
			}
		}
		orgID := auth.ResolveOrgID(ctx, user, input.OrganizationID)
		var poolID *uuid.UUID
		if input.PoolID != uuid.Nil {
			poolID = &input.PoolID
			pool, err := s.GetPool(input.PoolID)
			if err != nil {
				return status.Wrap(errors.New("pool not found"), status.NotFound)
			}
			if userOrg := auth.UserOrgForAccess(ctx, user); userOrg != uuid.Nil {
				env, err := s.GetEnvironment(pool.EnvironmentID)
				if err != nil || env.OrganizationID != userOrg {
					return status.Wrap(errors.New("pool not found"), status.NotFound)
				}
			}
		}
		limit, offset := input.Limit, input.Offset
		if limit <= 0 {
			limit = defaultListLimit
		}
		if limit > maxListLimit {
			limit = maxListLimit
		}
		if offset < 0 {
			offset = 0
		}
		var blockConnID *uuid.UUID
		if input.ConnectionID != uuid.Nil {
			blockConnID = &input.ConnectionID
		}
		blocks, total, err := s.ListBlocksFiltered(input.Name, envID, poolID, orgID, input.OrphanedOnly, input.Provider, blockConnID, limit, offset)
		if err != nil {
			return status.Wrap(err, status.Internal)
		}
		output.Total = total
		output.Blocks = make([]*blockOutput, len(blocks))
		for i, block := range blocks {
			totalStr, usedStr, availStr, _ := derivedBlockUsage(s, block.Name, block.CIDR, orgID)
			output.Blocks[i] = &blockOutput{
				ID:             block.ID,
				Name:           block.Name,
				CIDR:           block.CIDR,
				TotalIPs:       totalStr,
				UsedIPs:        usedStr,
				Available:      availStr,
				EnvironmentID:  block.EnvironmentID,
				OrganizationID: block.OrganizationID,
				PoolID:         block.PoolID,
				Provider:       block.Provider,
				ExternalID:     block.ExternalID,
				ConnectionID:   block.ConnectionID,
			}
		}
		return nil
	})

	u.SetTitle("List Blocks")
	u.SetDescription("Lists network blocks with optional name/environment/provider/connection_id filter and pagination (limit, offset)")
	u.SetExpectedErrors(status.Internal)
	return u
}

// GetBlock handler
func NewGetBlockUseCase(s store.Storer) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input getBlockInput, output *blockOutput) error {
		block, err := s.GetBlock(input.ID)
		if err != nil {
			return status.Wrap(errors.New("block not found"), status.NotFound)
		}
		user := auth.UserFromContext(ctx)
		if user != nil && block.EnvironmentID != uuid.Nil {
			env, err := s.GetEnvironment(block.EnvironmentID)
			if err != nil {
				return status.Wrap(errors.New("block not found"), status.NotFound)
			}
			if userOrg := auth.UserOrgForAccess(ctx, user); userOrg != uuid.Nil && env.OrganizationID != userOrg {
				return status.Wrap(errors.New("block not found"), status.NotFound)
			}
		}

		orgID := auth.ResolveOrgID(ctx, user, uuid.Nil)
		totalStr, usedStr, availStr, _ := derivedBlockUsage(s, block.Name, block.CIDR, orgID)
		output.ID = block.ID
		output.Name = block.Name
		output.CIDR = block.CIDR
		output.TotalIPs = totalStr
		output.UsedIPs = usedStr
		output.Available = availStr
		output.EnvironmentID = block.EnvironmentID
		output.OrganizationID = block.OrganizationID
		output.PoolID = block.PoolID
		output.Provider = block.Provider
		output.ExternalID = block.ExternalID
		output.ConnectionID = block.ConnectionID
		return nil
	})

	u.SetTitle("Get Block")
	u.SetDescription("Gets a specific block by ID")
	u.SetExpectedErrors(status.NotFound, status.Internal)
	return u
}

// UpdateBlock handler
func NewUpdateBlockUseCase(s store.Storer) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input updateBlockInput, output *blockOutput) error {
		block, err := s.GetBlock(input.ID)
		if err != nil {
			return status.Wrap(errors.New("block not found"), status.NotFound)
		}
		user := auth.UserFromContext(ctx)
		if user != nil && block.EnvironmentID != uuid.Nil {
			env, err := s.GetEnvironment(block.EnvironmentID)
			if err != nil {
				return status.Wrap(errors.New("block not found"), status.NotFound)
			}
			if userOrg := auth.UserOrgForAccess(ctx, user); userOrg != uuid.Nil && env.OrganizationID != userOrg {
				return status.Wrap(errors.New("block not found"), status.NotFound)
			}
		}

		block.Name = input.Name
		if input.EnvironmentID != nil {
			block.EnvironmentID = *input.EnvironmentID
		}
		if input.OrganizationID != nil {
			block.OrganizationID = *input.OrganizationID
		}
		// Allow setting or clearing pool (send pool_id: null to clear)
		if input.PoolID != nil {
			block.PoolID = input.PoolID
		}
		orgID := auth.ResolveOrgID(ctx, user, uuid.Nil)
		if block.EnvironmentID == uuid.Nil && block.OrganizationID == uuid.Nil && orgID != nil {
			block.OrganizationID = *orgID
		}
		// If pool is set, validate it belongs to block's environment and block CIDR is contained in pool
		if block.PoolID != nil && *block.PoolID != uuid.Nil {
			if block.EnvironmentID == uuid.Nil {
				return status.Wrap(errors.New("pool_id can only be set for blocks in an environment"), status.InvalidArgument)
			}
			pool, err := s.GetPool(*block.PoolID)
			if err != nil {
				return status.Wrap(errors.New("pool not found"), status.NotFound)
			}
			if pool.EnvironmentID != block.EnvironmentID {
				return status.Wrap(errors.New("pool does not belong to the block's environment"), status.InvalidArgument)
			}
			contained, err := network.Contains(pool.CIDR, block.CIDR)
			if err != nil {
				return status.Wrap(err, status.InvalidArgument)
			}
			if !contained {
				return status.Wrap(
					fmt.Errorf("block CIDR %s must be contained within pool %q CIDR %s", block.CIDR, pool.Name, pool.CIDR),
					status.InvalidArgument,
				)
			}
		}
		var existing []*network.Block
		if block.EnvironmentID != uuid.Nil {
			existing, err = s.ListBlocksByEnvironment(block.EnvironmentID)
		} else {
			existing, _, err = s.ListBlocksFiltered("", nil, nil, &block.OrganizationID, true, "", nil, 0, 0)
		}
		if err != nil {
			return status.Wrap(err, status.Internal)
		}
		for _, other := range existing {
			if other.ID == input.ID {
				continue
			}
			overlap, err := network.Overlaps(block.CIDR, other.CIDR)
			if err != nil {
				return status.Wrap(err, status.Internal)
			}
			if overlap {
				envLabel := "the environment"
				if block.EnvironmentID == uuid.Nil {
					envLabel = "orphaned blocks"
				}
				return status.Wrap(
					fmt.Errorf("CIDR %s overlaps with existing block %q in %s", block.CIDR, other.Name, envLabel),
					status.InvalidArgument,
				)
			}
		}
		reservedOrgID := &block.OrganizationID
		if block.EnvironmentID != uuid.Nil {
			if env, err := s.GetEnvironment(block.EnvironmentID); err == nil {
				reservedOrgID = &env.OrganizationID
			}
		}
		if reserved, err := s.OverlapsReservedBlock(block.CIDR, reservedOrgID); err != nil {
			return status.Wrap(err, status.Internal)
		} else if reserved != nil {
			return status.Wrap(
				fmt.Errorf("CIDR %s overlaps reserved block %s", block.CIDR, reserved.CIDR),
				status.InvalidArgument,
			)
		}
		if err := s.UpdateBlock(input.ID, block); err != nil {
			return status.Wrap(err, status.Internal)
		}

		outOrgID := &block.OrganizationID
		if block.EnvironmentID != uuid.Nil {
			if env, err := s.GetEnvironment(block.EnvironmentID); err == nil {
				outOrgID = &env.OrganizationID
			}
		}
		totalStr, usedStr, availStr, _ := derivedBlockUsage(s, block.Name, block.CIDR, outOrgID)
		output.ID = block.ID
		output.Name = block.Name
		output.CIDR = block.CIDR
		output.TotalIPs = totalStr
		output.UsedIPs = usedStr
		output.Available = availStr
		output.EnvironmentID = block.EnvironmentID
		output.OrganizationID = block.OrganizationID
		output.PoolID = block.PoolID
		output.Provider = block.Provider
		output.ExternalID = block.ExternalID
		output.ConnectionID = block.ConnectionID
		return nil
	})

	u.SetTitle("Update Block")
	u.SetDescription("Updates an existing block")
	u.SetExpectedErrors(status.NotFound, status.InvalidArgument, status.Internal)
	return u
}

// DeleteBlock handler. When the block is linked to a read-write integration with IPAM conflict resolution,
// the block (and its allocations) are soft-deleted so the next sync will delete them in the cloud then remove the rows.
// Otherwise cascades: deletes all allocations in this block, then the block.
func NewDeleteBlockUseCase(s store.Storer) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input getBlockInput, output *struct{}) error {
		block, err := s.GetBlock(input.ID)
		if err != nil {
			return status.Wrap(errors.New("block not found"), status.NotFound)
		}
		user := auth.UserFromContext(ctx)
		if user != nil && block.EnvironmentID != uuid.Nil {
			env, err := s.GetEnvironment(block.EnvironmentID)
			if err != nil {
				return status.Wrap(errors.New("block not found"), status.NotFound)
			}
			if userOrg := auth.UserOrgForAccess(ctx, user); userOrg != uuid.Nil && env.OrganizationID != userOrg {
				return status.Wrap(errors.New("block not found"), status.NotFound)
			}
		}
		// Soft-delete when block has an external ID and is linked to a read-write connection with IPAM conflict resolution.
		if block.ConnectionID != nil && *block.ConnectionID != uuid.Nil && block.ExternalID != "" {
			conn, err := s.GetCloudConnection(*block.ConnectionID)
			if err == nil && conn.SyncMode == "read_write" && conn.ConflictResolution == "ipam" {
				allocs, err := s.ListAllocations()
				if err != nil {
					return status.Wrap(err, status.Internal)
				}
				for _, a := range allocs {
					if blockNamesMatch(a.Block.Name, block.Name) {
						_ = s.SoftDeleteAllocation(a.Id)
					}
				}
				if err := s.SoftDeleteBlock(input.ID); err != nil {
					return status.Wrap(err, status.Internal)
				}
				return nil
			}
		}
		allocs, err := s.ListAllocations()
		if err != nil {
			return status.Wrap(err, status.Internal)
		}
		for _, a := range allocs {
			if blockNamesMatch(a.Block.Name, block.Name) {
				_ = s.DeleteAllocation(a.Id)
			}
		}
		if err := s.DeleteBlock(input.ID); err != nil {
			return status.Wrap(errors.New("block not found"), status.NotFound)
		}
		return nil
	})

	u.SetTitle("Delete Block")
	u.SetDescription("Deletes a block")
	u.SetExpectedErrors(status.NotFound, status.Internal)
	return u
}

// GetBlockUsage handler
func NewGetBlockUsageUseCase(s store.Storer) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input getBlockInput, output *blockUsageOutput) error {
		block, err := s.GetBlock(input.ID)
		if err != nil {
			return status.Wrap(errors.New("block not found"), status.NotFound)
		}
		user := auth.UserFromContext(ctx)
		if user != nil && block.EnvironmentID != uuid.Nil {
			env, err := s.GetEnvironment(block.EnvironmentID)
			if err != nil {
				return status.Wrap(errors.New("block not found"), status.NotFound)
			}
			if userOrg := auth.UserOrgForAccess(ctx, user); userOrg != uuid.Nil && env.OrganizationID != userOrg {
				return status.Wrap(errors.New("block not found"), status.NotFound)
			}
		}

		orgID := auth.ResolveOrgID(ctx, user, uuid.Nil)
		totalStr, usedStr, availStr, utilPercent := derivedBlockUsage(s, block.Name, block.CIDR, orgID)
		output.Name = block.Name
		output.CIDR = block.CIDR
		output.TotalIPs = totalStr
		output.UsedIPs = usedStr
		output.Available = availStr
		output.Utilized = utilPercent
		return nil
	})

	u.SetTitle("Get Block Usage")
	u.SetDescription("Gets usage statistics for a block")
	u.SetExpectedErrors(status.NotFound, status.Internal)
	return u
}

// SuggestBlockCIDR handler returns a suggested CIDR for the block at the given prefix length,
// considering existing allocations and bin-packing to fill gaps first.
func NewSuggestBlockCIDRUseCase(s store.Storer) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input suggestBlockCIDRInput, output *suggestBlockCIDROutput) error {
		if input.Prefix < 1 || input.Prefix > 32 {
			return status.Wrap(errors.New("prefix must be between 1 and 32"), status.InvalidArgument)
		}
		block, err := s.GetBlock(input.ID)
		if err != nil {
			return status.Wrap(errors.New("block not found"), status.NotFound)
		}
		user := auth.UserFromContext(ctx)
		if user != nil && block.EnvironmentID != uuid.Nil {
			env, err := s.GetEnvironment(block.EnvironmentID)
			if err != nil {
				return status.Wrap(errors.New("block not found"), status.NotFound)
			}
			if userOrg := auth.UserOrgForAccess(ctx, user); userOrg != uuid.Nil && env.OrganizationID != userOrg {
				return status.Wrap(errors.New("block not found"), status.NotFound)
			}
		}

		orgID := auth.ResolveOrgID(ctx, user, uuid.Nil)
		allocs, _, err := s.ListAllocationsFiltered("", block.Name, uuid.Nil, orgID, "", nil, 0, 0)
		if err != nil {
			return status.Wrap(err, status.Internal)
		}
		var allocatedCIDRs []string
		for _, a := range allocs {
			if blockNamesMatch(a.Block.Name, block.Name) {
				allocatedCIDRs = append(allocatedCIDRs, a.Block.CIDR)
			}
		}
		reserved, err := s.ListReservedBlocks(orgID)
		if err != nil {
			return status.Wrap(err, status.Internal)
		}
		for _, r := range reserved {
			overlap, _ := network.Overlaps(block.CIDR, r.CIDR)
			if !overlap {
				continue
			}
			contained, _ := network.Contains(block.CIDR, r.CIDR)
			if contained {
				allocatedCIDRs = append(allocatedCIDRs, r.CIDR)
			} else {
				allocatedCIDRs = append(allocatedCIDRs, block.CIDR)
			}
		}

		cidr, err := network.NextAvailableCIDRWithAllocations(block.CIDR, input.Prefix, allocatedCIDRs)
		if err != nil {
			return status.Wrap(err, status.InvalidArgument)
		}

		output.CIDR = cidr
		return nil
	})

	u.SetTitle("Suggest Block CIDR")
	u.SetDescription("Suggests the next available CIDR in the block at the given prefix length, bin-packing to fill gaps")
	u.SetExpectedErrors(status.NotFound, status.InvalidArgument, status.Internal)
	return u
}
