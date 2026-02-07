package handlers

import (
	"context"
	"errors"

	"github.com/JakeNeyer/ipam/network"
	"github.com/JakeNeyer/ipam/store"
	"github.com/google/uuid"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
)

// CreateAllocation handler
func NewCreateAllocationUseCase(s *store.Store) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input createAllocationInput, output *allocationOutput) error {
		if input.Name == "" || input.BlockName == "" || input.CIDR == "" {
			return status.Wrap(errors.New("name, block_name, and CIDR are required"), status.InvalidArgument)
		}

		if valid := network.ValidateCIDR(input.CIDR); !valid {
			return status.Wrap(errors.New("invalid CIDR format"), status.InvalidArgument)
		}

		allocation := &network.Allocation{
			Id:   uuid.New(),
			Name: input.Name,
			Block: network.Block{
				Name: input.BlockName,
				CIDR: input.CIDR,
			},
		}

		id := s.GenerateID()
		if err := s.CreateAllocation(id, allocation); err != nil {
			return status.Wrap(err, status.Internal)
		}

		output.Id = allocation.Id
		output.Name = allocation.Name
		output.BlockName = allocation.Block.Name
		output.CIDR = allocation.Block.CIDR
		return nil
	})

	u.SetTitle("Create Allocation")
	u.SetDescription("Creates a new IP allocation")
	u.SetExpectedErrors(status.InvalidArgument, status.Internal)
	return u
}

// ListAllocations handler
func NewListAllocationsUseCase(s *store.Store) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input struct{}, output *allocationListOutput) error {
		allocations, err := s.ListAllocations()
		if err != nil {
			return status.Wrap(err, status.Internal)
		}

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
	u.SetDescription("Lists all IP allocations")
	u.SetExpectedErrors(status.Internal)
	return u
}

// GetAllocation handler
func NewGetAllocationUseCase(s *store.Store) usecase.Interactor {
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
func NewUpdateAllocationUseCase(s *store.Store) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input updateAllocationInput, output *allocationOutput) error {
		alloc, err := s.GetAllocation(input.ID)
		if err != nil {
			return status.Wrap(errors.New("allocation not found"), status.NotFound)
		}

		alloc.Name = input.Name
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
	u.SetExpectedErrors(status.NotFound, status.Internal)
	return u
}

// DeleteAllocation handler
func NewDeleteAllocationUseCase(s *store.Store) usecase.Interactor {
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
