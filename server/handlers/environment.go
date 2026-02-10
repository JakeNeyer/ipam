package handlers

import (
	"context"
	"errors"

	"github.com/JakeNeyer/ipam/network"
	"github.com/JakeNeyer/ipam/store"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
)

// func calculateTotalIPs(cidr string) int {
// 	_, ipnet, err := net.ParseCIDR(cidr)
// 	if err != nil {
// 		return 0
// 	}
// 	ones, bits := ipnet.Mask.Size()
// 	return 1 << uint(bits-ones)
// }

// CreateEnvironment handler
func NewCreateEnvironmentUseCase(s store.Storer) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input createEnvironmentInput, output *environmentOutput) error {
		if input.Name == "" {
			return status.Wrap(errors.New("name is required"), status.InvalidArgument)
		}

		env := &network.Environment{
			Id:    s.GenerateID(),
			Name:  input.Name,
			Block: []network.Block{},
		}

		if err := s.CreateEnvironment(env); err != nil {
			return status.Wrap(err, status.Internal)
		}

		if input.InitialBlock != nil && input.InitialBlock.Name != "" && input.InitialBlock.CIDR != "" {
			if valid := network.ValidateCIDR(input.InitialBlock.CIDR); !valid {
				return status.Wrap(errors.New("invalid initial block CIDR format"), status.InvalidArgument)
			}
			totalIPs := calculateTotalIPs(input.InitialBlock.CIDR)
			block := &network.Block{
				Name:          input.InitialBlock.Name,
				CIDR:          input.InitialBlock.CIDR,
				EnvironmentID: env.Id,
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
		}

		output.Id = env.Id
		output.Name = env.Name
		return nil
	})

	u.SetTitle("Create Environment")
	u.SetDescription("Creates a new network environment")
	u.SetExpectedErrors(status.InvalidArgument, status.Internal)
	return u
}

// ListEnvironments handler
func NewListEnvironmentsUseCase(s store.Storer) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input listEnvironmentsInput, output *environmentListOutput) error {
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
		envs, total, err := s.ListEnvironmentsFiltered(input.Name, limit, offset)
		if err != nil {
			return status.Wrap(err, status.Internal)
		}
		output.Total = total
		output.Environments = make([]*environmentOutput, len(envs))
		for i, env := range envs {
			output.Environments[i] = &environmentOutput{
				Id:   env.Id,
				Name: env.Name,
			}
		}
		return nil
	})

	u.SetTitle("List Environments")
	u.SetDescription("Lists network environments with optional name filter and pagination (limit, offset)")
	u.SetExpectedErrors(status.Internal)
	return u
}

// GetEnvironment handler returns the environment with its blocks.
func NewGetEnvironmentUseCase(s store.Storer) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input getEnvironmentInput, output *environmentDetailOutput) error {
		env, err := s.GetEnvironment(input.ID)
		if err != nil {
			return status.Wrap(errors.New("environment not found"), status.NotFound)
		}

		blocks, err := s.ListBlocksByEnvironment(env.Id)
		if err != nil {
			return status.Wrap(err, status.Internal)
		}

		output.Id = env.Id
		output.Name = env.Name
		output.Blocks = make([]*blockOutput, len(blocks))
		for i, b := range blocks {
			used := computeUsedIPsForBlock(s, b.Name)
			avail := b.Usage.TotalIPs - used
			if avail < 0 {
				avail = 0
			}
			output.Blocks[i] = &blockOutput{
				ID:            b.ID,
				Name:          b.Name,
				CIDR:          b.CIDR,
				TotalIPs:      b.Usage.TotalIPs,
				UsedIPs:       used,
				Available:     avail,
				EnvironmentID: b.EnvironmentID,
			}
		}
		return nil
	})

	u.SetTitle("Get Environment")
	u.SetDescription("Gets a specific environment by ID")
	u.SetExpectedErrors(status.NotFound, status.Internal)
	return u
}

// UpdateEnvironment handler
func NewUpdateEnvironmentUseCase(s store.Storer) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input updateEnvironmentInput, output *environmentOutput) error {
		env, err := s.GetEnvironment(input.ID)
		if err != nil {
			return status.Wrap(errors.New("environment not found"), status.NotFound)
		}

		env.Name = input.Name
		if err := s.UpdateEnvironment(input.ID, env); err != nil {
			return status.Wrap(err, status.Internal)
		}

		output.Id = env.Id
		output.Name = env.Name
		return nil
	})

	u.SetTitle("Update Environment")
	u.SetDescription("Updates an existing environment")
	u.SetExpectedErrors(status.NotFound, status.Internal)
	return u
}

// DeleteEnvironment handler
func NewDeleteEnvironmentUseCase(s store.Storer) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input getEnvironmentInput, output *struct{}) error {
		if err := s.DeleteEnvironment(input.ID); err != nil {
			return status.Wrap(errors.New("environment not found"), status.NotFound)
		}
		return nil
	})

	u.SetTitle("Delete Environment")
	u.SetDescription("Deletes an environment")
	u.SetExpectedErrors(status.NotFound, status.Internal)
	return u
}

// SuggestEnvironmentBlockCIDR returns a suggested CIDR for a new block in the environment
// at the given prefix length, considering existing blocks in that environment (no overlap).
const defaultBlockSupernet = "10.0.0.0/8"

// SuggestEnvironmentBlockCIDR handler
func NewSuggestEnvironmentBlockCIDRUseCase(s store.Storer) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input suggestEnvironmentBlockCIDRInput, output *suggestBlockCIDROutput) error {
		if input.Prefix < 9 || input.Prefix > 32 {
			return status.Wrap(errors.New("prefix must be between 9 and 32"), status.InvalidArgument)
		}
		if _, err := s.GetEnvironment(input.ID); err != nil {
			return status.Wrap(errors.New("environment not found"), status.NotFound)
		}
		blocks, err := s.ListBlocksByEnvironment(input.ID)
		if err != nil {
			return status.Wrap(err, status.Internal)
		}
		var existingCIDRs []string
		for _, b := range blocks {
			if b.CIDR != "" {
				existingCIDRs = append(existingCIDRs, b.CIDR)
			}
		}
		// Exclude reserved ranges that overlap the supernet (contained in supernet, or reserve contains/overlaps supernet)
		reserved, err := s.ListReservedBlocks()
		if err != nil {
			return status.Wrap(err, status.Internal)
		}
		for _, r := range reserved {
			overlap, _ := network.Overlaps(defaultBlockSupernet, r.CIDR)
			if !overlap {
				continue
			}
			contained, _ := network.Contains(defaultBlockSupernet, r.CIDR)
			if contained {
				existingCIDRs = append(existingCIDRs, r.CIDR)
			} else {
				// Reserved contains or partially overlaps supernet; no suggestion possible in that range
				existingCIDRs = append(existingCIDRs, defaultBlockSupernet)
			}
		}
		cidr, err := network.NextAvailableCIDRWithAllocations(defaultBlockSupernet, input.Prefix, existingCIDRs)
		if err != nil {
			return status.Wrap(err, status.InvalidArgument)
		}
		output.CIDR = cidr
		return nil
	})
	u.SetTitle("Suggest Environment Block CIDR")
	u.SetDescription("Suggests a CIDR for a new block in the environment at the given prefix length, considering existing blocks in that environment")
	u.SetExpectedErrors(status.NotFound, status.InvalidArgument, status.Internal)
	return u
}
