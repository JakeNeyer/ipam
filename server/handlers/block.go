package handlers

import (
	"context"
	"errors"
	"net"

	"github.com/JakeNeyer/ipam/network"
	"github.com/JakeNeyer/ipam/store"
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
			Name: input.Name,
			CIDR: input.CIDR,
			Usage: network.Usage{
				TotalIPs:     totalIPs,
				UsedIPs:      0,
				AvailableIPs: totalIPs,
			},
			Children: []network.Block{},
		}

		if err := s.CreateBlock(block); err != nil {
			return status.Wrap(err, status.Internal)
		}

		output.Name = block.Name
		output.CIDR = block.CIDR
		output.TotalIPs = block.Usage.TotalIPs
		output.UsedIPs = block.Usage.UsedIPs
		output.Available = block.Usage.AvailableIPs
		return nil
	})

	u.SetTitle("Create Block")
	u.SetDescription("Creates a new network block")
	u.SetExpectedErrors(status.InvalidArgument, status.Internal)
	return u
}

// ListBlocks handler
func NewListBlocksUseCase(s *store.Store) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input struct{}, output *blockListOutput) error {
		blocks, err := s.ListBlocks()
		if err != nil {
			return status.Wrap(err, status.Internal)
		}

		output.Blocks = make([]*blockOutput, len(blocks))
		for i, block := range blocks {
			output.Blocks[i] = &blockOutput{
				Name:      block.Name,
				CIDR:      block.CIDR,
				TotalIPs:  block.Usage.TotalIPs,
				UsedIPs:   block.Usage.UsedIPs,
				Available: block.Usage.AvailableIPs,
			}
		}
		return nil
	})

	u.SetTitle("List Blocks")
	u.SetDescription("Lists all network blocks")
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

		output.Name = block.Name
		output.CIDR = block.CIDR
		output.TotalIPs = block.Usage.TotalIPs
		output.UsedIPs = block.Usage.UsedIPs
		output.Available = block.Usage.AvailableIPs
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
		if err := s.UpdateBlock(input.ID, block); err != nil {
			return status.Wrap(err, status.Internal)
		}

		output.Name = block.Name
		output.CIDR = block.CIDR
		output.TotalIPs = block.Usage.TotalIPs
		output.UsedIPs = block.Usage.UsedIPs
		output.Available = block.Usage.AvailableIPs
		return nil
	})

	u.SetTitle("Update Block")
	u.SetDescription("Updates an existing block")
	u.SetExpectedErrors(status.NotFound, status.Internal)
	return u
}

// DeleteBlock handler
func NewDeleteBlockUseCase(s *store.Store) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input getBlockInput, output *struct{}) error {
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

		utilPercent := 0.0
		if block.Usage.TotalIPs > 0 {
			utilPercent = (float64(block.Usage.UsedIPs) / float64(block.Usage.TotalIPs)) * 100.0
		}

		output.Name = block.Name
		output.CIDR = block.CIDR
		output.TotalIPs = block.Usage.TotalIPs
		output.UsedIPs = block.Usage.UsedIPs
		output.Available = block.Usage.AvailableIPs
		output.Utilized = utilPercent
		return nil
	})

	u.SetTitle("Get Block Usage")
	u.SetDescription("Gets usage statistics for a block")
	u.SetExpectedErrors(status.NotFound, status.Internal)
	return u
}
