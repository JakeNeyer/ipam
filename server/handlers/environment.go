package handlers

import (
	"context"
	"errors"

	"github.com/JakeNeyer/ipam/network"
	"github.com/JakeNeyer/ipam/store"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
)

// CreateEnvironment handler
func NewCreateEnvironmentUseCase(s *store.Store) usecase.Interactor {
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
func NewListEnvironmentsUseCase(s *store.Store) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input struct{}, output *environmentListOutput) error {
		envs, err := s.ListEnvironments()
		if err != nil {
			return status.Wrap(err, status.Internal)
		}

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
	u.SetDescription("Lists all network environments")
	u.SetExpectedErrors(status.Internal)
	return u
}

// GetEnvironment handler
func NewGetEnvironmentUseCase(s *store.Store) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input getEnvironmentInput, output *environmentOutput) error {
		env, err := s.GetEnvironment(input.ID)
		if err != nil {
			return status.Wrap(errors.New("environment not found"), status.NotFound)
		}

		output.Id = env.Id
		output.Name = env.Name
		return nil
	})

	u.SetTitle("Get Environment")
	u.SetDescription("Gets a specific environment by ID")
	u.SetExpectedErrors(status.NotFound, status.Internal)
	return u
}

// UpdateEnvironment handler
func NewUpdateEnvironmentUseCase(s *store.Store) usecase.Interactor {
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
func NewDeleteEnvironmentUseCase(s *store.Store) usecase.Interactor {
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
