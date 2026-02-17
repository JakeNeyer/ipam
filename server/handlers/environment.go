package handlers

import (
	"context"
	"errors"
	"fmt"
	"strings"

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

		if len(input.Pools) > 0 {
			for i, p := range input.Pools {
				if p.Name == "" || p.CIDR == "" {
					return status.Wrap(fmt.Errorf("pool at index %d: name and cidr are required", i), status.InvalidArgument)
				}
				if valid := network.ValidateCIDR(p.CIDR); !valid {
					return status.Wrap(fmt.Errorf("pool %q: invalid CIDR format", p.Name), status.InvalidArgument)
				}
			}
			for i := 0; i < len(input.Pools); i++ {
				for j := i + 1; j < len(input.Pools); j++ {
					overlap, err := network.Overlaps(input.Pools[i].CIDR, input.Pools[j].CIDR)
					if err != nil {
						return status.Wrap(err, status.InvalidArgument)
					}
					if overlap {
						return status.Wrap(
							fmt.Errorf("pools %q and %q overlap", input.Pools[i].Name, input.Pools[j].Name),
							status.InvalidArgument,
						)
					}
				}
			}
			existingPools, err := s.ListPoolsByOrganization(orgID)
			if err != nil {
				return status.Wrap(err, status.Internal)
			}
			for _, newPool := range input.Pools {
				for _, other := range existingPools {
					overlap, err := network.Overlaps(newPool.CIDR, other.CIDR)
					if err != nil {
						return status.Wrap(err, status.Internal)
					}
					if overlap {
						return status.Wrap(
							fmt.Errorf("pool CIDR %s overlaps with existing pool %q (%s) in this organization", newPool.CIDR, other.Name, other.CIDR),
							status.InvalidArgument,
						)
					}
				}
			}
		}

		env := &network.Environment{
			Id:             s.GenerateID(),
			Name:           input.Name,
			OrganizationID: orgID,
			Block:          []network.Block{},
		}

		if err := s.CreateEnvironment(env); err != nil {
			return status.Wrap(err, status.Internal)
		}

		output.PoolIDs = make([]uuid.UUID, 0, len(input.Pools))
		for _, p := range input.Pools {
			pool := &network.Pool{
				ID:             s.GenerateID(),
				OrganizationID: orgID,
				EnvironmentID:  env.Id,
				Name:           p.Name,
				CIDR:           p.CIDR,
			}
			if err := s.CreatePool(pool); err != nil {
				return status.Wrap(err, status.Internal)
			}
			output.PoolIDs = append(output.PoolIDs, pool.ID)
		}
		if len(output.PoolIDs) > 0 {
			output.InitialPoolID = &output.PoolIDs[0]
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
				PoolID:         b.PoolID,
				Provider:       b.Provider,
				ExternalID:     b.ExternalID,
				ConnectionID:   b.ConnectionID,
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
		name := strings.TrimSpace(input.Name)
		if name == "" {
			return status.Wrap(errors.New("name is required"), status.InvalidArgument)
		}
		if len(name) > 255 {
			return status.Wrap(errors.New("name must be at most 255 characters"), status.InvalidArgument)
		}

		env, err := s.GetEnvironment(input.ID)
		if err != nil {
			return status.Wrap(errors.New("environment not found"), status.NotFound)
		}
		user := auth.UserFromContext(ctx)
		if userOrg := auth.UserOrgForAccess(ctx, user); userOrg != uuid.Nil && env.OrganizationID != userOrg {
			return status.Wrap(errors.New("environment not found"), status.NotFound)
		}

		env.Name = name
		if err := s.UpdateEnvironment(input.ID, env); err != nil {
			return status.Wrap(err, status.Internal)
		}

		output.Id = env.Id
		output.Name = env.Name
		return nil
	})

	u.SetTitle("Update Environment")
	u.SetDescription("Updates an existing environment")
	u.SetExpectedErrors(status.InvalidArgument, status.NotFound, status.Internal)
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
