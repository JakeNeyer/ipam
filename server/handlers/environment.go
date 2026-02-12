package handlers

import (
	"context"
	"errors"

	"github.com/JakeNeyer/ipam/network"
	"github.com/JakeNeyer/ipam/server/auth"
	"github.com/JakeNeyer/ipam/store"
	"github.com/google/uuid"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
)

// CreateEnvironment handler
func NewCreateEnvironmentUseCase(s store.Storer) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input createEnvironmentInput, output *environmentOutput) error {
		if input.Name == "" {
			return status.Wrap(errors.New("name is required"), status.InvalidArgument)
		}
		user := auth.UserFromContext(ctx)
		if user == nil {
			return status.Wrap(errors.New("unauthorized"), status.Unauthenticated)
		}
		orgIDPtr := auth.ResolveOrgID(ctx, user, input.OrganizationID)
		if orgIDPtr == nil {
			return status.Wrap(errors.New("organization is required"), status.InvalidArgument)
		}
		orgID := *orgIDPtr

		env := &network.Environment{
			Id:             s.GenerateID(),
			Name:           input.Name,
			OrganizationID: orgID,
			Block:          []network.Block{},
		}

		if err := s.CreateEnvironment(env); err != nil {
			return status.Wrap(err, status.Internal)
		}

		if input.InitialBlock != nil && input.InitialBlock.Name != "" && input.InitialBlock.CIDR != "" {
			if valid := network.ValidateCIDR(input.InitialBlock.CIDR); !valid {
				return status.Wrap(errors.New("invalid initial block CIDR format"), status.InvalidArgument)
			}
			totalStored := int(network.CIDRAddressCountInt64(input.InitialBlock.CIDR))
			block := &network.Block{
				Name:          input.InitialBlock.Name,
				CIDR:          input.InitialBlock.CIDR,
				EnvironmentID: env.Id,
				Usage: network.Usage{
					TotalIPs:     totalStored,
					UsedIPs:      0,
					AvailableIPs: totalStored,
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
		user := auth.UserFromContext(ctx)
		if user == nil {
			return status.Wrap(errors.New("unauthorized"), status.Unauthenticated)
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
		envs, total, err := s.ListEnvironmentsFiltered(input.Name, orgID, limit, offset)
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
		user := auth.UserFromContext(ctx)
		if userOrg := auth.UserOrgForAccess(ctx, user); userOrg != uuid.Nil && env.OrganizationID != userOrg {
			return status.Wrap(errors.New("environment not found"), status.NotFound)
		}

		blocks, err := s.ListBlocksByEnvironment(env.Id)
		if err != nil {
			return status.Wrap(err, status.Internal)
		}

		output.Id = env.Id
		output.Name = env.Name
		envOrgID := &env.OrganizationID
		output.Blocks = make([]*blockOutput, len(blocks))
		for i, b := range blocks {
			totalStr, usedStr, availStr, _ := derivedBlockUsage(s, b.Name, b.CIDR, envOrgID)
			blockOrgID := b.OrganizationID
			if blockOrgID == uuid.Nil {
				blockOrgID = env.OrganizationID
			}
			output.Blocks[i] = &blockOutput{
				ID:             b.ID,
				Name:           b.Name,
				CIDR:           b.CIDR,
				TotalIPs:       totalStr,
				UsedIPs:        usedStr,
				Available:      availStr,
				EnvironmentID:  b.EnvironmentID,
				OrganizationID: blockOrgID,
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
		user := auth.UserFromContext(ctx)
		if userOrg := auth.UserOrgForAccess(ctx, user); userOrg != uuid.Nil && env.OrganizationID != userOrg {
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
		env, err := s.GetEnvironment(input.ID)
		if err != nil {
			return status.Wrap(errors.New("environment not found"), status.NotFound)
		}
		user := auth.UserFromContext(ctx)
		if userOrg := auth.UserOrgForAccess(ctx, user); userOrg != uuid.Nil && env.OrganizationID != userOrg {
			return status.Wrap(errors.New("environment not found"), status.NotFound)
		}
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
		env, err := s.GetEnvironment(input.ID)
		if err != nil {
			return status.Wrap(errors.New("environment not found"), status.NotFound)
		}
		user := auth.UserFromContext(ctx)
		if userOrg := auth.UserOrgForAccess(ctx, user); userOrg != uuid.Nil && env.OrganizationID != userOrg {
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
		reservedOrgID := &env.OrganizationID
		reserved, err := s.ListReservedBlocks(reservedOrgID)
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
