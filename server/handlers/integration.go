package handlers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/JakeNeyer/ipam/internal/integrations"
	"github.com/JakeNeyer/ipam/internal/integrations/aws" // register AWS provider + ParseAWSConfig
	"github.com/JakeNeyer/ipam/internal/logger"
	"github.com/JakeNeyer/ipam/server/auth"
	"github.com/JakeNeyer/ipam/store"
	"github.com/google/uuid"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
)

func normalizeSyncMode(v string) string {
	switch v {
	case "read_write":
		return "read_write"
	case "read_only":
		return "read_only"
	default:
		return "read_only"
	}
}

func normalizeConflictResolution(v string) string {
	switch v {
	case "ipam":
		return "ipam"
	case "cloud":
		return "cloud"
	default:
		return "cloud"
	}
}

func cloudConnectionToOutput(c *store.CloudConnection) *integrationOutput {
	syncMode := c.SyncMode
	if syncMode == "" {
		syncMode = "read_only"
	}
	conflictRes := c.ConflictResolution
	if conflictRes == "" {
		conflictRes = "cloud"
	}
	out := &integrationOutput{
		ID:                  c.ID,
		OrganizationID:      c.OrganizationID,
		Provider:            c.Provider,
		Name:                c.Name,
		Config:              c.Config,
		SyncIntervalMinutes: c.SyncIntervalMinutes,
		SyncMode:            syncMode,
		ConflictResolution:  conflictRes,
		CreatedAt:           c.CreatedAt.Format(time.RFC3339),
		UpdatedAt:           c.UpdatedAt.Format(time.RFC3339),
	}
	if c.LastSyncAt != nil {
		s := c.LastSyncAt.Format(time.RFC3339)
		out.LastSyncAt = &s
	}
	out.LastSyncStatus = c.LastSyncStatus
	out.LastSyncError = c.LastSyncError
	return out
}

func NewListIntegrationsUseCase(s store.Storer) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input listIntegrationsInput, output *integrationListOutput) error {
		user := auth.UserFromContext(ctx)
		if user == nil {
			return status.Wrap(errors.New("unauthorized"), status.Unauthenticated)
		}
		orgID := auth.ResolveOrgID(ctx, user, input.OrganizationID)
		if orgID == nil {
			return status.Wrap(errors.New("organization_id required"), status.InvalidArgument)
		}
		list, err := s.ListCloudConnectionsByOrganization(*orgID)
		if err != nil {
			return status.Wrap(err, status.Internal)
		}
		output.Integrations = make([]*integrationOutput, len(list))
		for i, c := range list {
			output.Integrations[i] = cloudConnectionToOutput(c)
		}
		return nil
	})
	u.SetTitle("List Integrations")
	u.SetDescription("List cloud connections for the organization")
	u.SetExpectedErrors(status.Unauthenticated, status.InvalidArgument, status.Internal)
	return u
}

func NewCreateIntegrationUseCase(s store.Storer) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input createIntegrationInput, output *integrationOutput) error {
		if input.Provider == "" || input.Name == "" {
			return status.Wrap(errors.New("provider and name are required"), status.InvalidArgument)
		}
		user := auth.UserFromContext(ctx)
		if user == nil {
			return status.Wrap(errors.New("unauthorized"), status.Unauthenticated)
		}
		orgID := auth.ResolveOrgID(ctx, user, input.OrganizationID)
		if orgID == nil {
			return status.Wrap(errors.New("organization is required: select an organization or provide organization_id"), status.InvalidArgument)
		}
		if integrations.Get(input.Provider) == nil {
			return status.Wrap(fmt.Errorf("unknown provider %q", input.Provider), status.InvalidArgument)
		}
		config := input.Config
		if config == nil {
			config = []byte("{}")
		}
		syncInterval := 5
		if input.SyncIntervalMinutes != nil {
			syncInterval = *input.SyncIntervalMinutes
			if syncInterval < 0 {
				syncInterval = 0
			}
			if syncInterval > 1440 {
				syncInterval = 1440
			}
		}
		syncMode := normalizeSyncMode(input.SyncMode)
		conflictRes := normalizeConflictResolution(input.ConflictResolution)
		c := &store.CloudConnection{
			ID:                  s.GenerateID(),
			OrganizationID:      *orgID,
			Provider:            input.Provider,
			Name:                input.Name,
			Config:              config,
			SyncIntervalMinutes: syncInterval,
			SyncMode:            syncMode,
			ConflictResolution:  conflictRes,
		}
		if err := s.CreateCloudConnection(c); err != nil {
			return status.Wrap(err, status.Internal)
		}
		*output = *cloudConnectionToOutput(c)
		return nil
	})
	u.SetTitle("Create Integration")
	u.SetDescription("Create a cloud connection (AWS, Azure, GCP)")
	u.SetExpectedErrors(status.Unauthenticated, status.InvalidArgument, status.Internal)
	return u
}

func NewGetIntegrationUseCase(s store.Storer) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input getIntegrationInput, output *integrationOutput) error {
		user := auth.UserFromContext(ctx)
		if user == nil {
			return status.Wrap(errors.New("unauthorized"), status.Unauthenticated)
		}
		c, err := s.GetCloudConnection(input.ID)
		if err != nil {
			return status.Wrap(errors.New("integration not found"), status.NotFound)
		}
		userOrg := auth.UserOrgForAccess(ctx, user)
		if userOrg != uuid.Nil && c.OrganizationID != userOrg {
			return status.Wrap(errors.New("integration not found"), status.NotFound)
		}
		*output = *cloudConnectionToOutput(c)
		return nil
	})
	u.SetTitle("Get Integration")
	u.SetDescription("Get a cloud connection by ID")
	u.SetExpectedErrors(status.Unauthenticated, status.NotFound)
	return u
}

func NewUpdateIntegrationUseCase(s store.Storer) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input updateIntegrationInput, output *integrationOutput) error {
		user := auth.UserFromContext(ctx)
		if user == nil {
			return status.Wrap(errors.New("unauthorized"), status.Unauthenticated)
		}
		c, err := s.GetCloudConnection(input.ID)
		if err != nil {
			return status.Wrap(errors.New("integration not found"), status.NotFound)
		}
		userOrg := auth.UserOrgForAccess(ctx, user)
		if userOrg != uuid.Nil && c.OrganizationID != userOrg {
			return status.Wrap(errors.New("integration not found"), status.NotFound)
		}
		c.Name = input.Name
		if input.Config != nil {
			c.Config = input.Config
		}
		if input.SyncIntervalMinutes != nil {
			v := *input.SyncIntervalMinutes
			if v < 0 {
				v = 0
			}
			if v > 1440 {
				v = 1440
			}
			c.SyncIntervalMinutes = v
		}
		if input.SyncMode != "" {
			c.SyncMode = normalizeSyncMode(input.SyncMode)
		}
		if input.ConflictResolution != "" {
			c.ConflictResolution = normalizeConflictResolution(input.ConflictResolution)
		}
		if err := s.UpdateCloudConnection(input.ID, c); err != nil {
			return status.Wrap(err, status.Internal)
		}
		updated, _ := s.GetCloudConnection(input.ID)
		*output = *cloudConnectionToOutput(updated)
		return nil
	})
	u.SetTitle("Update Integration")
	u.SetDescription("Update a cloud connection")
	u.SetExpectedErrors(status.Unauthenticated, status.NotFound, status.Internal)
	return u
}

func NewDeleteIntegrationUseCase(s store.Storer) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input getIntegrationInput, output *integrationOutput) error {
		user := auth.UserFromContext(ctx)
		if user == nil {
			return status.Wrap(errors.New("unauthorized"), status.Unauthenticated)
		}
		c, err := s.GetCloudConnection(input.ID)
		if err != nil {
			return status.Wrap(errors.New("integration not found"), status.NotFound)
		}
		userOrg := auth.UserOrgForAccess(ctx, user)
		if userOrg != uuid.Nil && c.OrganizationID != userOrg {
			return status.Wrap(errors.New("integration not found"), status.NotFound)
		}
		if err := s.DeleteCloudConnection(input.ID); err != nil {
			return status.Wrap(err, status.Internal)
		}
		return nil
	})
	u.SetTitle("Delete Integration")
	u.SetDescription("Delete a cloud connection")
	u.SetExpectedErrors(status.Unauthenticated, status.NotFound, status.Internal)
	return u
}

// RunSyncForConnection runs full sync (pools, blocks, allocations) for a connection and updates status.
// Used by both the sync use case (with auth) and the background sync runner.
func RunSyncForConnection(ctx context.Context, s store.Storer, connID uuid.UUID) error {
	c, err := s.GetCloudConnection(connID)
	if err != nil {
		logger.Error("sync full: get connection", slog.String("connection_id", connID.String()), logger.ErrAttr(err))
		return err
	}
	logger.Info("sync full started", slog.String("connection_id", connID.String()), slog.String("connection_name", c.Name))
	now := time.Now()
	statusStr := "syncing"
	c.LastSyncAt = &now
	c.LastSyncStatus = &statusStr
	c.LastSyncError = nil
	if err := s.UpdateCloudConnection(connID, c); err != nil {
		return err
	}
	// Which resources to sync (AWS config; default all true for other providers)
	syncPools, syncBlocks, syncAllocations := true, true, true
	if c.Provider == "aws" {
		if cfg, _ := aws.ParseAWSConfig(c.Config); cfg != nil {
			syncPools, syncBlocks, syncAllocations = cfg.SyncResources()
		}
	}
	if syncPools {
		if err := integrations.SyncPools(ctx, s, connID); err != nil {
			logger.Error("sync full failed: pools", slog.String("connection_id", connID.String()), slog.String("connection_name", c.Name), logger.ErrAttr(err))
			errStr := err.Error()
			c.LastSyncError = &errStr
			statusStr = "failed"
			c.LastSyncStatus = &statusStr
			_ = s.UpdateCloudConnection(connID, c)
			return err
		}
		// Push app pools (in target env with no external_id yet) to the cloud when read-write
		if c.SyncMode == "read_write" && c.Provider == "aws" {
			if cfg, _ := aws.ParseAWSConfig(c.Config); cfg != nil && cfg.EnvironmentID != uuid.Nil {
				if err := integrations.PushPoolsToCloud(ctx, s, c, cfg.EnvironmentID); err != nil {
					logger.Error("sync full failed: push pools to cloud", slog.String("connection_id", connID.String()), slog.String("connection_name", c.Name), logger.ErrAttr(err))
					errStr := err.Error()
					c.LastSyncError = &errStr
					statusStr = "failed"
					c.LastSyncStatus = &statusStr
					_ = s.UpdateCloudConnection(connID, c)
					return err
				}
			}
		}
		// Delete in cloud any pools that were soft-deleted in the app (IPAM conflict resolution)
		if c.SyncMode == "read_write" && c.ConflictResolution == "ipam" {
			if err := integrations.ApplyPoolDeletesInCloud(ctx, s, c); err != nil {
				logger.Error("sync full failed: apply pool deletes in cloud", slog.String("connection_id", connID.String()), slog.String("connection_name", c.Name), logger.ErrAttr(err))
				errStr := err.Error()
				c.LastSyncError = &errStr
				statusStr = "failed"
				c.LastSyncStatus = &statusStr
				_ = s.UpdateCloudConnection(connID, c)
				return err
			}
		}
	}
	if syncBlocks {
		if err := integrations.SyncBlocks(ctx, s, connID); err != nil {
			logger.Error("sync full failed: blocks", slog.String("connection_id", connID.String()), slog.String("connection_name", c.Name), logger.ErrAttr(err))
			errStr := err.Error()
			c.LastSyncError = &errStr
			statusStr = "failed"
			c.LastSyncStatus = &statusStr
			_ = s.UpdateCloudConnection(connID, c)
			return err
		}
		// Push app blocks (in synced pools with no external_id yet) to the cloud when read-write
		if c.SyncMode == "read_write" {
			if err := integrations.PushBlocksToCloud(ctx, s, c); err != nil {
				logger.Error("sync full failed: push blocks to cloud", slog.String("connection_id", connID.String()), slog.String("connection_name", c.Name), logger.ErrAttr(err))
				errStr := err.Error()
				c.LastSyncError = &errStr
				statusStr = "failed"
				c.LastSyncStatus = &statusStr
				_ = s.UpdateCloudConnection(connID, c)
				return err
			}
		}
	}
	syncedBlocks, _, _ := s.ListBlocksFiltered("", nil, nil, &c.OrganizationID, false, c.Provider, &connID, 10000, 0)
	if syncAllocations {
		if err := integrations.SyncAllocations(ctx, s, connID, syncedBlocks); err != nil {
			logger.Error("sync full failed: allocations", slog.String("connection_id", connID.String()), slog.String("connection_name", c.Name), logger.ErrAttr(err))
			errStr := err.Error()
			c.LastSyncError = &errStr
			statusStr = "failed"
			c.LastSyncStatus = &statusStr
			_ = s.UpdateCloudConnection(connID, c)
			return err
		}
		// Push app allocations (in synced blocks with no external_id yet) to the cloud when read-write
		if c.SyncMode == "read_write" {
			if err := integrations.PushAllocationsToCloud(ctx, s, c); err != nil {
				logger.Error("sync full failed: push allocations to cloud", slog.String("connection_id", connID.String()), slog.String("connection_name", c.Name), logger.ErrAttr(err))
				errStr := err.Error()
				c.LastSyncError = &errStr
				statusStr = "failed"
				c.LastSyncStatus = &statusStr
				_ = s.UpdateCloudConnection(connID, c)
				return err
			}
		}
	}
	// Delete in cloud any allocations/blocks that were soft-deleted in the app (IPAM conflict resolution).
	// Allocations (subnets) first, then blocks (VPCs).
	if c.SyncMode == "read_write" && c.ConflictResolution == "ipam" {
		if syncAllocations {
			if err := integrations.ApplyAllocationDeletesInCloud(ctx, s, c); err != nil {
				logger.Error("sync full failed: apply allocation deletes in cloud", slog.String("connection_id", connID.String()), slog.String("connection_name", c.Name), logger.ErrAttr(err))
				errStr := err.Error()
				c.LastSyncError = &errStr
				statusStr = "failed"
				c.LastSyncStatus = &statusStr
				_ = s.UpdateCloudConnection(connID, c)
				return err
			}
		}
		if syncBlocks {
			if err := integrations.ApplyBlockDeletesInCloud(ctx, s, c); err != nil {
				logger.Error("sync full failed: apply block deletes in cloud", slog.String("connection_id", connID.String()), slog.String("connection_name", c.Name), logger.ErrAttr(err))
				errStr := err.Error()
				c.LastSyncError = &errStr
				statusStr = "failed"
				c.LastSyncStatus = &statusStr
				_ = s.UpdateCloudConnection(connID, c)
				return err
			}
		}
	}
	logger.Info("sync full completed", slog.String("connection_id", connID.String()), slog.String("connection_name", c.Name))
	statusStr = "success"
	c.LastSyncStatus = &statusStr
	c.LastSyncError = nil
	return s.UpdateCloudConnection(connID, c)
}

func NewSyncIntegrationUseCase(s store.Storer) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input syncIntegrationInput, output *integrationOutput) error {
		user := auth.UserFromContext(ctx)
		if user == nil {
			return status.Wrap(errors.New("unauthorized"), status.Unauthenticated)
		}
		c, err := s.GetCloudConnection(input.ID)
		if err != nil {
			return status.Wrap(errors.New("integration not found"), status.NotFound)
		}
		userOrg := auth.UserOrgForAccess(ctx, user)
		if userOrg != uuid.Nil && c.OrganizationID != userOrg {
			return status.Wrap(errors.New("integration not found"), status.NotFound)
		}
		if err := RunSyncForConnection(ctx, s, input.ID); err != nil {
			return status.Wrap(err, status.Internal)
		}
		updated, _ := s.GetCloudConnection(input.ID)
		*output = *cloudConnectionToOutput(updated)
		return nil
	})
	u.SetTitle("Sync Integration")
	u.SetDescription("Trigger sync for a cloud connection (pools, blocks, and allocations e.g. VPC subnets)")
	u.SetExpectedErrors(status.Unauthenticated, status.NotFound, status.Internal)
	return u
}

// StartBackgroundSync starts a goroutine that syncs cloud connections on their configured interval (default 5 min).
func StartBackgroundSync(s store.Storer) {
	const tickInterval = time.Minute
	go func() {
		ticker := time.NewTicker(tickInterval)
		defer ticker.Stop()
		for range ticker.C {
			list, err := s.ListCloudConnections()
			if err != nil {
				logger.Error("sync background: list connections", logger.ErrAttr(err))
				continue
			}
			now := time.Now()
			for _, c := range list {
				if c.SyncIntervalMinutes <= 0 {
					continue
				}
				interval := time.Duration(c.SyncIntervalMinutes) * time.Minute
				due := c.LastSyncAt == nil || now.Sub(*c.LastSyncAt) >= interval
				if !due {
					continue
				}
				connID := c.ID
				go func() {
					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
					defer cancel()
					acquired, err := s.WithSyncLock(ctx, connID, func() error { return RunSyncForConnection(ctx, s, connID) })
					if !acquired {
						return // another instance is syncing this connection
					}
					if err != nil {
						logger.Error("sync background failed", slog.String("connection_id", connID.String()), logger.ErrAttr(err))
					}
				}()
			}
		}
	}()
}
