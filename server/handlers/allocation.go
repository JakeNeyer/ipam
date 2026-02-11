package handlers

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/JakeNeyer/ipam/network"
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

		blocks, err := s.ListBlocks()
		if err != nil {
			return status.Wrap(err, status.Internal)
		}
		if len(blocks) == 0 {
			return status.Wrap(errors.New("no network blocks exist; create a block first"), status.FailedPrecondition)
		}
		var parentBlock *network.Block
		for _, b := range blocks {
			if b.Name == input.BlockName {
				parentBlock = b
				break
			}
		}
		if parentBlock == nil {
			return status.Wrap(errors.New("block not found"), status.NotFound)
		}

		contained, err := network.Contains(parentBlock.CIDR, input.CIDR)
		if err != nil {
			return status.Wrap(err, status.InvalidArgument)
		}
		if !contained {
			return status.Wrap(errors.New("allocation CIDR must fall within the parent block's CIDR range"), status.InvalidArgument)
		}

		allAllocs, err := s.ListAllocations()
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
		if reserved, err := s.OverlapsReservedBlock(input.CIDR); err != nil {
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

		if err := s.CreateAllocation(id, allocation); err != nil {
			return status.Wrap(err, status.Internal)
		}

		output.Id = id
		output.Name = allocation.Name
		output.BlockName = allocation.Block.Name
		output.CIDR = allocation.Block.CIDR
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

		blocks, err := s.ListBlocks()
		if err != nil {
			return status.Wrap(err, status.Internal)
		}
		if len(blocks) == 0 {
			return status.Wrap(errors.New("no network blocks exist; create a block first"), status.FailedPrecondition)
		}
		var parentBlock *network.Block
		for _, b := range blocks {
			if b.Name == input.BlockName {
				parentBlock = b
				break
			}
		}
		if parentBlock == nil {
			return status.Wrap(errors.New("block not found"), status.NotFound)
		}

		allAllocs, err := s.ListAllocations()
		if err != nil {
			return status.Wrap(err, status.Internal)
		}
		var allocatedCIDRs []string
		for _, a := range allAllocs {
			if allocationBlockNamesMatch(a.Block.Name, input.BlockName) {
				allocatedCIDRs = append(allocatedCIDRs, a.Block.CIDR)
			}
		}

		reserved, err := s.ListReservedBlocks()
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

		if err := s.CreateAllocation(id, allocation); err != nil {
			return status.Wrap(err, status.Internal)
		}

		output.Id = id
		output.Name = allocation.Name
		output.BlockName = allocation.Block.Name
		output.CIDR = allocation.Block.CIDR
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
		allocations, total, err := s.ListAllocationsFiltered(input.Name, input.BlockName, input.EnvironmentID, limit, offset)
		if err != nil {
			return status.Wrap(err, status.Internal)
		}
		output.Total = total
		output.Allocations = make([]*allocationOutput, len(allocations))
		for i, alloc := range allocations {
			output.Allocations[i] = &allocationOutput{
				Id:        alloc.Id,
				Name:      alloc.Name,
				BlockName: alloc.Block.Name,
				CIDR:      alloc.Block.CIDR,
			}
		}
		return nil
	})

	u.SetTitle("List Allocations")
	u.SetDescription("Lists IP allocations with optional name/block_name filter and pagination (limit, offset)")
	u.SetExpectedErrors(status.Internal)
	return u
}

// GetAllocation handler
func NewGetAllocationUseCase(s store.Storer) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input getAllocationInput, output *allocationOutput) error {
		alloc, err := s.GetAllocation(input.ID)
		if err != nil {
			return status.Wrap(errors.New("allocation not found"), status.NotFound)
		}

		output.Id = alloc.Id
		output.Name = alloc.Name
		output.BlockName = alloc.Block.Name
		output.CIDR = alloc.Block.CIDR
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

		alloc.Name = input.Name

		allAllocs, err := s.ListAllocations()
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
		return nil
	})

	u.SetTitle("Update Allocation")
	u.SetDescription("Updates an existing allocation")
	u.SetExpectedErrors(status.NotFound, status.InvalidArgument, status.Internal)
	return u
}

// DeleteAllocation handler
func NewDeleteAllocationUseCase(s store.Storer) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input struct {
		Id uuid.UUID `path:"id"`
	}, output *struct{}) error {
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
