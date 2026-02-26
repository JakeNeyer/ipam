package handlers

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/JakeNeyer/ipam/internal/integrations"
	"github.com/JakeNeyer/ipam/network"
	"github.com/JakeNeyer/ipam/server/auth"
	"github.com/JakeNeyer/ipam/store"
	"github.com/google/uuid"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
)

func allocationBlockNamesMatch(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

// CreateAllocation handler
func NewCreateAllocationUseCase(s store.Storer) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input createAllocationInput, output *allocationOutput) error {
		if input.Name == "" || input.BlockName == "" || input.CIDR == "" {
			return status.Wrap(errors.New("name, block_name, and CIDR are required"), status.InvalidArgument)
		}

		if valid := network.ValidateCIDR(input.CIDR); !valid {
			return status.Wrap(errors.New("invalid CIDR format"), status.InvalidArgument)
		}

		user := auth.UserFromContext(ctx)
		if user == nil {
			return status.Wrap(errors.New("unauthorized"), status.Unauthenticated)
		}
		orgID := auth.ResolveOrgID(ctx, user, uuid.Nil)
		blocks, _, err := s.ListBlocksFiltered(input.BlockName, nil, nil, orgID, false, "", nil, 0, 0)
		if err != nil {
			return status.Wrap(err, status.Internal)
		}
		var parentBlock *network.Block
		for _, b := range blocks {
			if allocationBlockNamesMatch(b.Name, input.BlockName) {
				parentBlock = b
				break
			}
		}
		if parentBlock == nil {
			return status.Wrap(errors.New("block not found"), status.NotFound)
		}
		var reservedOrgID *uuid.UUID
		if parentBlock.EnvironmentID != uuid.Nil {
			if env, err := s.GetEnvironment(parentBlock.EnvironmentID); err == nil {
				reservedOrgID = &env.OrganizationID
			}
		}

		contained, err := network.Contains(parentBlock.CIDR, input.CIDR)
		if err != nil {
			return status.Wrap(err, status.InvalidArgument)
		}
		if !contained {
			return status.Wrap(errors.New("allocation CIDR must fall within the parent block's CIDR range"), status.InvalidArgument)
		}

		allAllocs, _, err := s.ListAllocationsFiltered("", input.BlockName, uuid.Nil, orgID, "", nil, 0, 0)
		if err != nil {
			return status.Wrap(err, status.Internal)
		}
		for _, existing := range allAllocs {
			if !allocationBlockNamesMatch(existing.Block.Name, input.BlockName) {
				continue
			}
			overlap, err := network.Overlaps(input.CIDR, existing.Block.CIDR)
			if err != nil {
				return status.Wrap(err, status.Internal)
			}
			if overlap {
				return status.Wrap(
					fmt.Errorf("CIDR %s overlaps with existing allocation %q in block %q", input.CIDR, existing.Name, input.BlockName),
					status.InvalidArgument,
				)
			}
		}
		if reserved, err := s.OverlapsReservedBlock(input.CIDR, reservedOrgID); err != nil {
			return status.Wrap(err, status.Internal)
		} else if reserved != nil {
			return status.Wrap(
				fmt.Errorf("CIDR %s overlaps reserved block %s", input.CIDR, reserved.CIDR),
				status.InvalidArgument,
			)
		}

		id := s.GenerateID()
		allocation := &network.Allocation{
			Id:   id,
			Name: input.Name,
			Block: network.Block{
				Name: input.BlockName,
				CIDR: input.CIDR,
			},
		}

		if parentBlock.ConnectionID != nil && parentBlock.ExternalID != "" {
			conn, err := s.GetCloudConnection(*parentBlock.ConnectionID)
			if err == nil && conn.SyncMode == "read_write" {
				prov := integrations.Get(conn.Provider)
				if prov != nil {
					if pushProv, ok := prov.(integrations.PushProvider); ok && pushProv.SupportsPush() {
						extID, err := pushProv.CreateAllocationInCloud(ctx, conn, parentBlock.ExternalID, allocation)
						if err != nil {
							return status.Wrap(fmt.Errorf("push allocation to cloud: %w", err), status.Internal)
						}
						allocation.Provider = conn.Provider
						allocation.ExternalID = extID
						allocation.ConnectionID = parentBlock.ConnectionID
					}
				}
			}
		}

		if err := s.CreateAllocation(id, allocation); err != nil {
			return status.Wrap(err, status.Internal)
		}

		output.Id = id
		output.Name = allocation.Name
		output.BlockName = allocation.Block.Name
		output.CIDR = allocation.Block.CIDR
		output.Provider = allocation.Provider
		output.ExternalID = allocation.ExternalID
		output.ConnectionID = allocation.ConnectionID
		return nil
	})

	u.SetTitle("Create Allocation")
	u.SetDescription("Creates a new IP allocation")
	u.SetExpectedErrors(status.InvalidArgument, status.FailedPrecondition, status.NotFound, status.Internal)
	return u
}

// AutoAllocate handler: find the next available CIDR in a block via bin-packing and create the allocation.
func NewAutoAllocateUseCase(s store.Storer) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input autoAllocateInput, output *allocationOutput) error {
		if input.Name == "" || input.BlockName == "" {
			return status.Wrap(errors.New("name and block_name are required"), status.InvalidArgument)
		}
		if input.PrefixLength < 1 || input.PrefixLength > 32 {
			return status.Wrap(errors.New("prefix_length must be between 1 and 32"), status.InvalidArgument)
		}

		user := auth.UserFromContext(ctx)
		if user == nil {
			return status.Wrap(errors.New("unauthorized"), status.Unauthenticated)
		}
		orgID := auth.ResolveOrgID(ctx, user, uuid.Nil)
		blocks, _, err := s.ListBlocksFiltered(input.BlockName, nil, nil, orgID, false, "", nil, 0, 0)
		if err != nil {
			return status.Wrap(err, status.Internal)
		}
		var parentBlock *network.Block
		for _, b := range blocks {
			if allocationBlockNamesMatch(b.Name, input.BlockName) {
				parentBlock = b
				break
			}
		}
		if parentBlock == nil {
			return status.Wrap(errors.New("block not found"), status.NotFound)
		}

		allAllocs, _, err := s.ListAllocationsFiltered("", input.BlockName, uuid.Nil, orgID, "", nil, 0, 0)
		if err != nil {
			return status.Wrap(err, status.Internal)
		}
		var allocatedCIDRs []string
		for _, a := range allAllocs {
			if allocationBlockNamesMatch(a.Block.Name, input.BlockName) {
				allocatedCIDRs = append(allocatedCIDRs, a.Block.CIDR)
			}
		}

		reserved, err := s.ListReservedBlocks(orgID)
		if err != nil {
			return status.Wrap(err, status.Internal)
		}
		for _, r := range reserved {
			overlap, _ := network.Overlaps(parentBlock.CIDR, r.CIDR)
			if !overlap {
				continue
			}
			contained, _ := network.Contains(parentBlock.CIDR, r.CIDR)
			if contained {
				allocatedCIDRs = append(allocatedCIDRs, r.CIDR)
			} else {
				allocatedCIDRs = append(allocatedCIDRs, parentBlock.CIDR)
			}
		}

		cidr, err := network.NextAvailableCIDRWithAllocations(parentBlock.CIDR, input.PrefixLength, allocatedCIDRs)
		if err != nil {
			return status.Wrap(fmt.Errorf("no available CIDR with prefix /%d in block %q: %w", input.PrefixLength, input.BlockName, err), status.FailedPrecondition)
		}

		id := s.GenerateID()
		allocation := &network.Allocation{
			Id:   id,
			Name: input.Name,
			Block: network.Block{
				Name: input.BlockName,
				CIDR: cidr,
			},
		}

		if parentBlock.ConnectionID != nil && parentBlock.ExternalID != "" {
			conn, err := s.GetCloudConnection(*parentBlock.ConnectionID)
			if err == nil && conn.SyncMode == "read_write" {
				prov := integrations.Get(conn.Provider)
				if prov != nil {
					if pushProv, ok := prov.(integrations.PushProvider); ok && pushProv.SupportsPush() {
						extID, err := pushProv.CreateAllocationInCloud(ctx, conn, parentBlock.ExternalID, allocation)
						if err != nil {
							return status.Wrap(fmt.Errorf("push allocation to cloud: %w", err), status.Internal)
						}
						allocation.Provider = conn.Provider
						allocation.ExternalID = extID
						allocation.ConnectionID = parentBlock.ConnectionID
					}
				}
			}
		}

		if err := s.CreateAllocation(id, allocation); err != nil {
			return status.Wrap(err, status.Internal)
		}

		output.Id = id
		output.Name = allocation.Name
		output.BlockName = allocation.Block.Name
		output.CIDR = allocation.Block.CIDR
		output.Provider = allocation.Provider
		output.ExternalID = allocation.ExternalID
		output.ConnectionID = allocation.ConnectionID
		return nil
	})

	u.SetTitle("Auto-allocate")
	u.SetDescription("Finds the next available CIDR in a block using bin-packing and creates an allocation")
	u.SetExpectedErrors(status.InvalidArgument, status.FailedPrecondition, status.NotFound, status.Internal)
	return u
}

// ListAllocations handler
func NewListAllocationsUseCase(s store.Storer) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input listAllocationsInput, output *allocationListOutput) error {
		user := auth.UserFromContext(ctx)
		if user == nil {
			return status.Wrap(errors.New("unauthorized"), status.Unauthenticated)
		}
		if input.EnvironmentID != uuid.Nil {
			env, err := s.GetEnvironment(input.EnvironmentID)
			if err != nil {
				return status.Wrap(errors.New("environment not found"), status.NotFound)
			}
			if userOrg := auth.UserOrgForAccess(ctx, user); userOrg != uuid.Nil && env.OrganizationID != userOrg {
				return status.Wrap(errors.New("environment not found"), status.NotFound)
			}
		}
		orgID := auth.ResolveOrgID(ctx, user, input.OrganizationID)
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
		var connectionID *uuid.UUID
		if input.ConnectionID != uuid.Nil {
			connectionID = &input.ConnectionID
		}
		allocations, total, err := s.ListAllocationsFiltered(input.Name, input.BlockName, input.EnvironmentID, orgID, input.Provider, connectionID, limit, offset)
		if err != nil {
			return status.Wrap(err, status.Internal)
		}
		output.Total = total
		output.Allocations = make([]*allocationOutput, len(allocations))
		for i, alloc := range allocations {
			output.Allocations[i] = &allocationOutput{
				Id:           alloc.Id,
				Name:         alloc.Name,
				BlockName:    alloc.Block.Name,
				CIDR:         alloc.Block.CIDR,
				Provider:     alloc.Provider,
				ExternalID:   alloc.ExternalID,
				ConnectionID: alloc.ConnectionID,
			}
		}
		return nil
	})

	u.SetTitle("List Allocations")
	u.SetDescription("Lists IP allocations with optional name/block_name filter and pagination (limit, offset)")
	u.SetExpectedErrors(status.Internal)
	return u
}

// allocationInOrg returns true if the allocation's block belongs to an environment in the given org.
func allocationInOrg(s store.Storer, orgID uuid.UUID, alloc *network.Allocation) bool {
	if orgID == uuid.Nil {
		return true
	}
	blocks, _, err := s.ListBlocksFiltered(strings.TrimSpace(alloc.Block.Name), nil, nil, &orgID, false, "", nil, 0, 0)
	if err != nil || len(blocks) == 0 {
		return false
	}
	for _, b := range blocks {
		contained, cErr := network.Contains(b.CIDR, alloc.Block.CIDR)
		if cErr == nil && contained {
			return true
		}
	}
	return false
}

// allocationInEffectiveOrg returns true if the allocation is accessible: effective org from context (e.g. token) or user's org, or full access when global admin unscoped.
func allocationInEffectiveOrg(ctx context.Context, s store.Storer, user *store.User, alloc *network.Allocation) bool {
	if user == nil {
		return false
	}
	userOrg := auth.UserOrgForAccess(ctx, user)
	return allocationInOrg(s, userOrg, alloc)
}

// GetAllocation handler
func NewGetAllocationUseCase(s store.Storer) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input getAllocationInput, output *allocationOutput) error {
		alloc, err := s.GetAllocation(input.ID)
		if err != nil {
			return status.Wrap(errors.New("allocation not found"), status.NotFound)
		}
		user := auth.UserFromContext(ctx)
		if user != nil && !allocationInEffectiveOrg(ctx, s, user, alloc) {
			return status.Wrap(errors.New("allocation not found"), status.NotFound)
		}

		output.Id = alloc.Id
		output.Name = alloc.Name
		output.BlockName = alloc.Block.Name
		output.CIDR = alloc.Block.CIDR
		output.Provider = alloc.Provider
		output.ExternalID = alloc.ExternalID
		output.ConnectionID = alloc.ConnectionID
		return nil
	})

	u.SetTitle("Get Allocation")
	u.SetDescription("Gets a specific allocation by ID")
	u.SetExpectedErrors(status.NotFound, status.Internal)
	return u
}

// UpdateAllocation handler
func NewUpdateAllocationUseCase(s store.Storer) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input updateAllocationInput, output *allocationOutput) error {
		alloc, err := s.GetAllocation(input.ID)
		if err != nil {
			return status.Wrap(errors.New("allocation not found"), status.NotFound)
		}
		user := auth.UserFromContext(ctx)
		if user != nil && !allocationInEffectiveOrg(ctx, s, user, alloc) {
			return status.Wrap(errors.New("allocation not found"), status.NotFound)
		}

		alloc.Name = input.Name

		orgID := auth.ResolveOrgID(ctx, user, uuid.Nil)
		allAllocs, _, err := s.ListAllocationsFiltered("", alloc.Block.Name, uuid.Nil, orgID, "", nil, 0, 0)
		if err != nil {
			return status.Wrap(err, status.Internal)
		}
		for _, existing := range allAllocs {
			if existing.Id == input.ID {
				continue
			}
			if !allocationBlockNamesMatch(existing.Block.Name, alloc.Block.Name) {
				continue
			}
			overlap, err := network.Overlaps(alloc.Block.CIDR, existing.Block.CIDR)
			if err != nil {
				return status.Wrap(err, status.Internal)
			}
			if overlap {
				return status.Wrap(
					fmt.Errorf("CIDR %s overlaps with existing allocation %q in block %q", alloc.Block.CIDR, existing.Name, alloc.Block.Name),
					status.InvalidArgument,
				)
			}
		}

		if err := s.UpdateAllocation(input.ID, alloc); err != nil {
			return status.Wrap(err, status.Internal)
		}

		output.Id = alloc.Id
		output.Name = alloc.Name
		output.BlockName = alloc.Block.Name
		output.CIDR = alloc.Block.CIDR
		output.Provider = alloc.Provider
		output.ExternalID = alloc.ExternalID
		output.ConnectionID = alloc.ConnectionID
		return nil
	})

	u.SetTitle("Update Allocation")
	u.SetDescription("Updates an existing allocation")
	u.SetExpectedErrors(status.NotFound, status.InvalidArgument, status.Internal)
	return u
}

// DeleteAllocation handler. When the allocation is linked to a read-write integration with IPAM conflict resolution,
// the allocation is soft-deleted so the next sync will delete it in the cloud then remove the row.
func NewDeleteAllocationUseCase(s store.Storer) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input struct {
		Id uuid.UUID `path:"id"`
	}, output *struct{}) error {
		alloc, err := s.GetAllocation(input.Id)
		if err != nil {
			return status.Wrap(errors.New("allocation not found"), status.NotFound)
		}
		user := auth.UserFromContext(ctx)
		if user != nil && !allocationInEffectiveOrg(ctx, s, user, alloc) {
			return status.Wrap(errors.New("allocation not found"), status.NotFound)
		}
		// Soft-delete when allocation has an external ID and is linked to a read-write connection with IPAM conflict resolution.
		if alloc.ConnectionID != nil && *alloc.ConnectionID != uuid.Nil && alloc.ExternalID != "" {
			conn, err := s.GetCloudConnection(*alloc.ConnectionID)
			if err == nil && conn.SyncMode == "read_write" && conn.ConflictResolution == "ipam" {
				if err := s.SoftDeleteAllocation(input.Id); err != nil {
					return status.Wrap(err, status.Internal)
				}
				return nil
			}
		}
		if err := s.DeleteAllocation(input.Id); err != nil {
			return status.Wrap(errors.New("allocation not found"), status.NotFound)
		}
		return nil
	})

	u.SetTitle("Delete Allocation")
	u.SetDescription("Deletes an allocation")
	u.SetExpectedErrors(status.NotFound, status.Internal)
	return u
}
