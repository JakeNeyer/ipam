package handlers

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/JakeNeyer/ipam/network"
	"github.com/JakeNeyer/ipam/store"
	"github.com/google/uuid"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
)

// Helper function to calculate total IPs in a CIDR
func calculateTotalIPs(cidr string) int {
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return 0
	}
	ones, bits := ipnet.Mask.Size()
	return 1 << uint(bits-ones)
}

// blockNamesMatch compares block names for usage attribution (case-insensitive, trimmed).
func blockNamesMatch(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

// computeUsedIPsForBlock returns the sum of IPs allocated from this block (allocations with matching block name).
func computeUsedIPsForBlock(s *store.Store, blockName string) int {
	allocs, err := s.ListAllocations()
	if err != nil {
		return 0
	}
	var sum int
	for _, a := range allocs {
		if blockNamesMatch(a.Block.Name, blockName) {
			sum += calculateTotalIPs(a.Block.CIDR)
		}
	}
	return sum
}

// CreateBlock handler
func NewCreateBlockUseCase(s *store.Store) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input createBlockInput, output *blockOutput) error {
		if input.Name == "" || input.CIDR == "" {
			return status.Wrap(errors.New("name and CIDR are required"), status.InvalidArgument)
		}

		if valid := network.ValidateCIDR(input.CIDR); !valid {
			return status.Wrap(errors.New("invalid CIDR format"), status.InvalidArgument)
		}

		totalIPs := calculateTotalIPs(input.CIDR)
		block := &network.Block{
			Name:          input.Name,
			CIDR:          input.CIDR,
			EnvironmentID: input.EnvironmentID,
			Usage: network.Usage{
				TotalIPs:     totalIPs,
				UsedIPs:      0,
				AvailableIPs: totalIPs,
			},
			Children: []network.Block{},
		}

		// Ensure the new block's CIDR does not overlap any existing block in the same environment.
		existing, err := s.ListBlocksByEnvironment(block.EnvironmentID)
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

		if err := s.CreateBlock(block); err != nil {
			return status.Wrap(err, status.Internal)
		}

		output.ID = block.ID
		output.Name = block.Name
		output.CIDR = block.CIDR
		output.TotalIPs = block.Usage.TotalIPs
		output.UsedIPs = block.Usage.UsedIPs
		output.Available = block.Usage.AvailableIPs
		output.EnvironmentID = block.EnvironmentID
		return nil
	})

	u.SetTitle("Create Block")
	u.SetDescription("Creates a new network block")
	u.SetExpectedErrors(status.InvalidArgument, status.Internal)
	return u
}

// ListBlocks handler
func NewListBlocksUseCase(s *store.Store) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input listBlocksInput, output *blockListOutput) error {
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
		var envID *uuid.UUID
		if input.EnvironmentID != uuid.Nil {
			envID = &input.EnvironmentID
		}
		blocks, total, err := s.ListBlocksFiltered(input.Name, envID, input.OrphanedOnly, limit, offset)
		if err != nil {
			return status.Wrap(err, status.Internal)
		}
		output.Total = total
		output.Blocks = make([]*blockOutput, len(blocks))
		for i, block := range blocks {
			used := computeUsedIPsForBlock(s, block.Name)
			avail := block.Usage.TotalIPs - used
			if avail < 0 {
				avail = 0
			}
			output.Blocks[i] = &blockOutput{
				ID:            block.ID,
				Name:          block.Name,
				CIDR:          block.CIDR,
				TotalIPs:      block.Usage.TotalIPs,
				UsedIPs:       used,
				Available:     avail,
				EnvironmentID: block.EnvironmentID,
			}
		}
		return nil
	})

	u.SetTitle("List Blocks")
	u.SetDescription("Lists network blocks with optional name/environment filter and pagination (limit, offset)")
	u.SetExpectedErrors(status.Internal)
	return u
}

// GetBlock handler
func NewGetBlockUseCase(s *store.Store) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input getBlockInput, output *blockOutput) error {
		block, err := s.GetBlock(input.ID)
		if err != nil {
			return status.Wrap(errors.New("block not found"), status.NotFound)
		}

		used := computeUsedIPsForBlock(s, block.Name)
		avail := block.Usage.TotalIPs - used
		if avail < 0 {
			avail = 0
		}

		output.ID = block.ID
		output.Name = block.Name
		output.CIDR = block.CIDR
		output.TotalIPs = block.Usage.TotalIPs
		output.UsedIPs = used
		output.Available = avail
		output.EnvironmentID = block.EnvironmentID
		return nil
	})

	u.SetTitle("Get Block")
	u.SetDescription("Gets a specific block by ID")
	u.SetExpectedErrors(status.NotFound, status.Internal)
	return u
}

// UpdateBlock handler
func NewUpdateBlockUseCase(s *store.Store) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input updateBlockInput, output *blockOutput) error {
		block, err := s.GetBlock(input.ID)
		if err != nil {
			return status.Wrap(errors.New("block not found"), status.NotFound)
		}

		block.Name = input.Name
		if input.EnvironmentID != nil {
			block.EnvironmentID = *input.EnvironmentID
		}
		// Ensure this block's CIDR does not overlap any other block in the same environment (create and update).
		existing, err := s.ListBlocksByEnvironment(block.EnvironmentID)
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
		if err := s.UpdateBlock(input.ID, block); err != nil {
			return status.Wrap(err, status.Internal)
		}

		used := computeUsedIPsForBlock(s, block.Name)
		avail := block.Usage.TotalIPs - used
		if avail < 0 {
			avail = 0
		}

		output.ID = block.ID
		output.Name = block.Name
		output.CIDR = block.CIDR
		output.TotalIPs = block.Usage.TotalIPs
		output.UsedIPs = used
		output.Available = avail
		output.EnvironmentID = block.EnvironmentID
		return nil
	})

	u.SetTitle("Update Block")
	u.SetDescription("Updates an existing block")
	u.SetExpectedErrors(status.NotFound, status.InvalidArgument, status.Internal)
	return u
}

// DeleteBlock handler. Cascades: deletes all allocations in this block, then the block.
func NewDeleteBlockUseCase(s *store.Store) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input getBlockInput, output *struct{}) error {
		block, err := s.GetBlock(input.ID)
		if err != nil {
			return status.Wrap(errors.New("block not found"), status.NotFound)
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
func NewGetBlockUsageUseCase(s *store.Store) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input getBlockInput, output *blockUsageOutput) error {
		block, err := s.GetBlock(input.ID)
		if err != nil {
			return status.Wrap(errors.New("block not found"), status.NotFound)
		}

		used := computeUsedIPsForBlock(s, block.Name)
		avail := block.Usage.TotalIPs - used
		if avail < 0 {
			avail = 0
		}

		utilPercent := 0.0
		if block.Usage.TotalIPs > 0 {
			utilPercent = (float64(used) / float64(block.Usage.TotalIPs)) * 100.0
		}

		output.Name = block.Name
		output.CIDR = block.CIDR
		output.TotalIPs = block.Usage.TotalIPs
		output.UsedIPs = used
		output.Available = avail
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
func NewSuggestBlockCIDRUseCase(s *store.Store) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input suggestBlockCIDRInput, output *suggestBlockCIDROutput) error {
		if input.Prefix < 1 || input.Prefix > 32 {
			return status.Wrap(errors.New("prefix must be between 1 and 32"), status.InvalidArgument)
		}
		block, err := s.GetBlock(input.ID)
		if err != nil {
			return status.Wrap(errors.New("block not found"), status.NotFound)
		}

		allocs, err := s.ListAllocations()
		if err != nil {
			return status.Wrap(err, status.Internal)
		}
		var allocatedCIDRs []string
		for _, a := range allocs {
			if blockNamesMatch(a.Block.Name, block.Name) {
				allocatedCIDRs = append(allocatedCIDRs, a.Block.CIDR)
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
