package handlers

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/JakeNeyer/ipam/network"
	"github.com/JakeNeyer/ipam/server/auth"
	"github.com/JakeNeyer/ipam/store"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
)

// NewListReservedBlocksUseCase returns a use case for GET /api/admin/reserved-blocks. Admin only.
func NewListReservedBlocksUseCase(s store.Storer) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input struct{}, output *reservedBlockListOutput) error {
		user := auth.UserFromContext(ctx)
		if user == nil || user.Role != store.RoleAdmin {
			return status.Wrap(errors.New("forbidden"), status.PermissionDenied)
		}
		list, err := s.ListReservedBlocks()
		if err != nil {
			return status.Wrap(err, status.Internal)
		}
		out := make([]*reservedBlockOutput, 0, len(list))
		for _, r := range list {
			out = append(out, &reservedBlockOutput{
				ID:        r.ID.String(),
				Name:      r.Name,
				CIDR:      r.CIDR,
				Reason:    r.Reason,
				CreatedAt: r.CreatedAt.Format(time.RFC3339),
			})
		}
		output.ReservedBlocks = out
		return nil
	})
	u.SetTitle("List reserved blocks")
	u.SetDescription("List CIDR ranges that are reserved (blacklisted) and cannot be used as blocks or allocations. Admin only.")
	u.SetExpectedErrors(status.PermissionDenied, status.Internal)
	return u
}

// NewCreateReservedBlockUseCase returns a use case for POST /api/admin/reserved-blocks. Admin only.
func NewCreateReservedBlockUseCase(s store.Storer) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input createReservedBlockInput, output *reservedBlockOutput) error {
		user := auth.UserFromContext(ctx)
		if user == nil || user.Role != store.RoleAdmin {
			return status.Wrap(errors.New("forbidden"), status.PermissionDenied)
		}
		cidr := strings.TrimSpace(input.CIDR)
		if cidr == "" {
			return status.Wrap(errors.New("CIDR is required"), status.InvalidArgument)
		}
		if !network.ValidateCIDR(cidr) {
			return status.Wrap(errors.New("invalid CIDR format"), status.InvalidArgument)
		}
		overlap, err := s.OverlapsReservedBlock(cidr)
		if err != nil {
			return status.Wrap(err, status.Internal)
		}
		if overlap != nil {
			return status.Wrap(
				errors.New("CIDR overlaps with existing reserved block "+overlap.CIDR),
				status.InvalidArgument,
			)
		}
		r := &store.ReservedBlock{
			Name:   strings.TrimSpace(input.Name),
			CIDR:   cidr,
			Reason: strings.TrimSpace(input.Reason),
		}
		if err := s.CreateReservedBlock(r); err != nil {
			return status.Wrap(err, status.Internal)
		}
		output.ID = r.ID.String()
		output.Name = r.Name
		output.CIDR = r.CIDR
		output.Reason = r.Reason
		output.CreatedAt = r.CreatedAt.Format(time.RFC3339)
		return nil
	})
	u.SetTitle("Create reserved block")
	u.SetDescription("Reserve a CIDR range so it cannot be used as a block or allocation. Admin only.")
	u.SetExpectedErrors(status.PermissionDenied, status.InvalidArgument, status.Internal)
	return u
}

// NewDeleteReservedBlockUseCase returns a use case for DELETE /api/admin/reserved-blocks/:id. Admin only.
func NewDeleteReservedBlockUseCase(s store.Storer) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input getReservedBlockInput, output *struct{}) error {
		user := auth.UserFromContext(ctx)
		if user == nil || user.Role != store.RoleAdmin {
			return status.Wrap(errors.New("forbidden"), status.PermissionDenied)
		}
		if err := s.DeleteReservedBlock(input.ID); err != nil {
			if err.Error() == "reserved block not found" {
				return status.Wrap(err, status.NotFound)
			}
			return status.Wrap(err, status.Internal)
		}
		return nil
	})
	u.SetTitle("Delete reserved block")
	u.SetDescription("Remove a reserved CIDR range. Admin only.")
	u.SetExpectedErrors(status.PermissionDenied, status.NotFound, status.Internal)
	return u
}

// NewUpdateReservedBlockUseCase returns a use case for PUT /api/admin/reserved-blocks/:id. Admin only.
func NewUpdateReservedBlockUseCase(s store.Storer) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input updateReservedBlockInput, output *reservedBlockOutput) error {
		user := auth.UserFromContext(ctx)
		if user == nil || user.Role != store.RoleAdmin {
			return status.Wrap(errors.New("forbidden"), status.PermissionDenied)
		}
		r, err := s.GetReservedBlock(input.ID)
		if err != nil {
			return status.Wrap(errors.New("reserved block not found"), status.NotFound)
		}
		r.Name = strings.TrimSpace(input.Name)
		if err := s.UpdateReservedBlock(input.ID, r); err != nil {
			if err.Error() == "reserved block not found" {
				return status.Wrap(err, status.NotFound)
			}
			return status.Wrap(err, status.Internal)
		}
		output.ID = r.ID.String()
		output.Name = r.Name
		output.CIDR = r.CIDR
		output.Reason = r.Reason
		output.CreatedAt = r.CreatedAt.Format(time.RFC3339)
		return nil
	})
	u.SetTitle("Update reserved block")
	u.SetDescription("Update reserved block metadata (name). Admin only.")
	u.SetExpectedErrors(status.PermissionDenied, status.NotFound, status.Internal)
	return u
}
