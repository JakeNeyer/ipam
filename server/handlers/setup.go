package handlers

import (
	"context"
	"errors"
	"strings"

	"github.com/JakeNeyer/ipam/internal/logger"
	"github.com/JakeNeyer/ipam/server/validation"
	"github.com/JakeNeyer/ipam/store"
	"github.com/google/uuid"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"golang.org/x/crypto/bcrypt"
)

// getSetupStatusOutput is the response for GET /api/setup/status.
type getSetupStatusOutput struct {
	SetupRequired bool `json:"setup_required"`
}

// NewGetSetupStatusUseCase returns a use case for GET /api/setup/status.
func NewGetSetupStatusUseCase(s store.Storer) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input struct{}, output *getSetupStatusOutput) error {
		users, err := s.ListUsers(nil)
		if err != nil {
			logger.Error(logger.MsgSetupStatusFailed, logger.KeyOperation, "get_setup_status", logger.ErrAttr(err))
			return status.Wrap(errors.New("setup check failed"), status.InvalidArgument)
		}
		output.SetupRequired = len(users) == 0
		logger.Info("setup status", logger.KeyOperation, "get_setup_status", logger.KeySetupRequired, output.SetupRequired)
		return nil
	})
	u.SetTitle("Get setup status")
	u.SetDescription("Returns whether initial setup is required (no users exist)")
	u.SetExpectedErrors(status.InvalidArgument)
	return u
}

// postSetupInput is the body for POST /api/setup.
type postSetupInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// postSetupOutput is the response for POST /api/setup.
type postSetupOutput struct {
	User UserResponse `json:"user"`
}

// NewPostSetupUseCase returns a use case for POST /api/setup. Creates the first admin only when no users exist.
func NewPostSetupUseCase(s store.Storer) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input postSetupInput, output *postSetupOutput) error {
		logger.Info("setup request", logger.KeyOperation, "post_setup", logger.KeyEmail, input.Email)
		users, err := s.ListUsers(nil)
		if err != nil {
			logger.Error(logger.MsgSetupStatusFailed, logger.KeyOperation, "post_setup", logger.ErrAttr(err))
			return status.Wrap(errors.New("setup check failed, please try again"), status.InvalidArgument)
		}
		if len(users) > 0 {
			logger.Info(logger.MsgSetupAlreadyDone, logger.KeyOperation, "post_setup")
			return status.Wrap(errors.New("setup already completed"), status.PermissionDenied)
		}
		if !validation.ValidateEmail(input.Email) {
			logger.Info(logger.MsgSetupMissingCreds, logger.KeyOperation, "post_setup")
			return status.Wrap(errors.New("valid email required"), status.InvalidArgument)
		}
		if !validation.ValidatePassword(input.Password) {
			logger.Info(logger.MsgSetupMissingCreds, logger.KeyOperation, "post_setup")
			return status.Wrap(errors.New("password must be at least 8 characters"), status.InvalidArgument)
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
		if err != nil {
			logger.Error(logger.MsgSetupPasswordFailed, logger.KeyOperation, "post_setup", logger.ErrAttr(err))
			return status.Wrap(errors.New("password setup failed, please try again"), status.InvalidArgument)
		}
		admin := &store.User{
			Email:          strings.TrimSpace(strings.ToLower(input.Email)),
			PasswordHash:   string(hash),
			Role:           store.RoleAdmin,
			OrganizationID: uuid.Nil,
		}
		if err := s.CreateUser(admin); err != nil {
			logger.Error(logger.MsgSetupCreateUserFailed, logger.KeyOperation, "post_setup", logger.ErrAttr(err))
			msg := err.Error()
			if msg == "" {
				msg = logger.MsgSetupCreateUserFailed
			}
			return status.Wrap(errors.New(msg), status.InvalidArgument)
		}
		logger.Info("setup success", logger.KeyOperation, "post_setup", logger.KeyUserID, admin.ID.String(), logger.KeyEmail, admin.Email)
		output.User = userToResponse(admin)
		return nil
	})
	u.SetTitle("Post setup")
	u.SetDescription("Creates the first admin user. Only succeeds when no users exist.")
	u.SetExpectedErrors(status.PermissionDenied, status.InvalidArgument, status.Internal)
	return u
}
