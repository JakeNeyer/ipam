package handlers

import (
	"context"
	"errors"
	"fmt"

	"github.com/JakeNeyer/ipam/network"
	"github.com/JakeNeyer/ipam/server/auth"
	"github.com/JakeNeyer/ipam/store"
	"github.com/google/uuid"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
)

// SuggestPoolBlockCIDR returns a suggested CIDR for a new block in the pool at the given
// prefix length, considering existing blocks in that pool and reserved ranges overlapping the pool.
func NewSuggestPoolBlockCIDRUseCase(s store.Storer) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input suggestPoolBlockCIDRInput, output *suggestBlockCIDROutput) error {
		if input.Prefix < 9 || input.Prefix > 32 {
			return status.Wrap(errors.New("prefix must be between 9 and 32"), status.InvalidArgument)
		}
		pool, err := s.GetPool(input.ID)
		if err != nil {
			return status.Wrap(errors.New("pool not found"), status.NotFound)
		}
		user := auth.UserFromContext(ctx)
		if userOrg := auth.UserOrgForAccess(ctx, user); userOrg != uuid.Nil {
			if pool.OrganizationID != userOrg {
				return status.Wrap(errors.New("pool not found"), status.NotFound)
			}
		}
		blocks, err := s.ListBlocksByPool(input.ID)
		if err != nil {
			return status.Wrap(err, status.Internal)
		}
		var existingCIDRs []string
		for _, b := range blocks {
			if b.CIDR != "" {
				existingCIDRs = append(existingCIDRs, b.CIDR)
			}
		}
		env, _ := s.GetEnvironment(pool.EnvironmentID)
		if env != nil {
			reservedOrgID := &env.OrganizationID
			reserved, err := s.ListReservedBlocks(reservedOrgID)
			if err != nil {
				return status.Wrap(err, status.Internal)
			}
			for _, r := range reserved {
				overlap, _ := network.Overlaps(pool.CIDR, r.CIDR)
				if !overlap {
					continue
				}
				contained, _ := network.Contains(pool.CIDR, r.CIDR)
				if contained {
					existingCIDRs = append(existingCIDRs, r.CIDR)
				} else {
					existingCIDRs = append(existingCIDRs, pool.CIDR)
				}
			}
		}
		cidr, err := network.NextAvailableCIDRWithAllocations(pool.CIDR, input.Prefix, existingCIDRs)
		if err != nil {
			return status.Wrap(err, status.InvalidArgument)
		}
		output.CIDR = cidr
		return nil
	})
	u.SetTitle("Suggest Pool Block CIDR")
	u.SetDescription("Suggests a CIDR for a new block in the pool at the given prefix length, considering existing blocks in that pool")
	u.SetExpectedErrors(status.NotFound, status.InvalidArgument, status.Internal)
	return u
}

// CreatePool handler
func NewCreatePoolUseCase(s store.Storer) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input createPoolInput, output *poolOutput) error {
		if input.Name == "" || input.CIDR == "" {
			return status.Wrap(errors.New("name and CIDR are required"), status.InvalidArgument)
		}
		if valid := network.ValidateCIDR(input.CIDR); !valid {
			return status.Wrap(errors.New("invalid CIDR format"), status.InvalidArgument)
		}
		user := auth.UserFromContext(ctx)
		if user == nil {
			return status.Wrap(errors.New("unauthorized"), status.Unauthenticated)
		}
		env, err := s.GetEnvironment(input.EnvironmentID)
		if err != nil {
			return status.Wrap(errors.New("environment not found"), status.NotFound)
		}
		if userOrg := auth.UserOrgForAccess(ctx, user); userOrg != uuid.Nil && env.OrganizationID != userOrg {
			return status.Wrap(errors.New("environment not found"), status.NotFound)
		}
		existingPools, err := s.ListPoolsByOrganization(env.OrganizationID)
		if err != nil {
			return status.Wrap(err, status.Internal)
		}
		for _, other := range existingPools {
			overlap, err := network.Overlaps(input.CIDR, other.CIDR)
			if err != nil {
				return status.Wrap(err, status.Internal)
			}
			if overlap {
				return status.Wrap(
					fmt.Errorf("pool CIDR %s overlaps with existing pool %q (%s)", input.CIDR, other.Name, other.CIDR),
					status.InvalidArgument,
				)
			}
		}
		pool := &network.Pool{
			ID:             s.GenerateID(),
			OrganizationID: env.OrganizationID,
			EnvironmentID:  input.EnvironmentID,
			Name:           input.Name,
			CIDR:           input.CIDR,
		}
		if err := s.CreatePool(pool); err != nil {
			return status.Wrap(err, status.Internal)
		}
		output.ID = pool.ID
		output.OrganizationID = pool.OrganizationID
		output.EnvironmentID = pool.EnvironmentID
		output.Name = pool.Name
		output.CIDR = pool.CIDR
		return nil
	})
	u.SetTitle("Create Pool")
	u.SetDescription("Creates an environment pool (CIDR range that blocks in the environment can draw from)")
	u.SetExpectedErrors(status.InvalidArgument, status.NotFound, status.Internal)
	return u
}

// GetPool handler
func NewGetPoolUseCase(s store.Storer) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input getPoolInput, output *poolOutput) error {
		pool, err := s.GetPool(input.ID)
		if err != nil {
			return status.Wrap(errors.New("pool not found"), status.NotFound)
		}
		user := auth.UserFromContext(ctx)
		if user != nil {
			if userOrg := auth.UserOrgForAccess(ctx, user); userOrg != uuid.Nil && pool.OrganizationID != userOrg {
				return status.Wrap(errors.New("pool not found"), status.NotFound)
			}
		}
		output.ID = pool.ID
		output.OrganizationID = pool.OrganizationID
		output.EnvironmentID = pool.EnvironmentID
		output.Name = pool.Name
		output.CIDR = pool.CIDR
		return nil
	})
	u.SetTitle("Get Pool")
	u.SetDescription("Gets a pool by ID")
	u.SetExpectedErrors(status.NotFound, status.Internal)
	return u
}

// ListPools handler (by environment or by organization)
func NewListPoolsUseCase(s store.Storer) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input listPoolsInput, output *poolListOutput) error {
		user := auth.UserFromContext(ctx)
		if user == nil {
			return status.Wrap(errors.New("unauthorized"), status.Unauthenticated)
		}
		var pools []*network.Pool
		if input.OrganizationID != uuid.Nil {
			if userOrg := auth.UserOrgForAccess(ctx, user); userOrg != uuid.Nil && input.OrganizationID != userOrg {
				return status.Wrap(errors.New("organization not found"), status.NotFound)
			}
			orgPools, err := s.ListPoolsByOrganization(input.OrganizationID)
			if err != nil {
				return status.Wrap(err, status.Internal)
			}
			if input.EnvironmentID != uuid.Nil {
				for _, p := range orgPools {
					if p.EnvironmentID == input.EnvironmentID {
						pools = append(pools, p)
					}
				}
			} else {
				pools = orgPools
			}
		} else if input.EnvironmentID != uuid.Nil {
			env, err := s.GetEnvironment(input.EnvironmentID)
			if err != nil {
				return status.Wrap(errors.New("environment not found"), status.NotFound)
			}
			if userOrg := auth.UserOrgForAccess(ctx, user); userOrg != uuid.Nil && env.OrganizationID != userOrg {
				return status.Wrap(errors.New("environment not found"), status.NotFound)
			}
			pools, err = s.ListPoolsByEnvironment(input.EnvironmentID)
			if err != nil {
				return status.Wrap(err, status.Internal)
			}
		} else {
			return status.Wrap(errors.New("environment_id or organization_id is required"), status.InvalidArgument)
		}
		output.Pools = make([]*poolOutput, len(pools))
		for i, p := range pools {
			output.Pools[i] = &poolOutput{
				ID:             p.ID,
				OrganizationID: p.OrganizationID,
				EnvironmentID:  p.EnvironmentID,
				Name:           p.Name,
				CIDR:           p.CIDR,
			}
		}
		return nil
	})
	u.SetTitle("List Pools")
	u.SetDescription("Lists pools for an environment or for an organization")
	u.SetExpectedErrors(status.InvalidArgument, status.NotFound, status.Internal)
	return u
}

// UpdatePool handler
func NewUpdatePoolUseCase(s store.Storer) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input updatePoolInput, output *poolOutput) error {
		pool, err := s.GetPool(input.ID)
		if err != nil {
			return status.Wrap(errors.New("pool not found"), status.NotFound)
		}
		user := auth.UserFromContext(ctx)
		if user != nil {
			if userOrg := auth.UserOrgForAccess(ctx, user); userOrg != uuid.Nil && pool.OrganizationID != userOrg {
				return status.Wrap(errors.New("pool not found"), status.NotFound)
			}
		}
		if valid := network.ValidateCIDR(input.CIDR); !valid {
			return status.Wrap(errors.New("invalid CIDR format"), status.InvalidArgument)
		}
		existingPools, err := s.ListPoolsByOrganization(pool.OrganizationID)
		if err != nil {
			return status.Wrap(err, status.Internal)
		}
		for _, other := range existingPools {
			if other.ID == input.ID {
				continue
			}
			overlap, err := network.Overlaps(input.CIDR, other.CIDR)
			if err != nil {
				return status.Wrap(err, status.Internal)
			}
			if overlap {
				return status.Wrap(
					fmt.Errorf("pool CIDR %s overlaps with existing pool %q (%s)", input.CIDR, other.Name, other.CIDR),
					status.InvalidArgument,
				)
			}
		}
		pool.Name = input.Name
		pool.CIDR = input.CIDR
		if err := s.UpdatePool(input.ID, pool); err != nil {
			return status.Wrap(err, status.Internal)
		}
		output.ID = pool.ID
		output.OrganizationID = pool.OrganizationID
		output.EnvironmentID = pool.EnvironmentID
		output.Name = pool.Name
		output.CIDR = pool.CIDR
		return nil
	})
	u.SetTitle("Update Pool")
	u.SetDescription("Updates a pool")
	u.SetExpectedErrors(status.NotFound, status.InvalidArgument, status.Internal)
	return u
}

// DeletePool handler
func NewDeletePoolUseCase(s store.Storer) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input getPoolInput, output *struct{}) error {
		pool, err := s.GetPool(input.ID)
		if err != nil {
			return status.Wrap(errors.New("pool not found"), status.NotFound)
		}
		user := auth.UserFromContext(ctx)
		if user != nil {
			if userOrg := auth.UserOrgForAccess(ctx, user); userOrg != uuid.Nil && pool.OrganizationID != userOrg {
				return status.Wrap(errors.New("pool not found"), status.NotFound)
			}
		}
		if err := s.DeletePool(input.ID); err != nil {
			return status.Wrap(errors.New("pool not found"), status.NotFound)
		}
		return nil
	})
	u.SetTitle("Delete Pool")
	u.SetDescription("Deletes a pool (blocks referencing it will have pool_id set to null)")
	u.SetExpectedErrors(status.NotFound, status.Internal)
	return u
}
