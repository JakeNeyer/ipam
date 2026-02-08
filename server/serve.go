package server

import (
	"github.com/JakeNeyer/ipam/server/auth"
	"github.com/JakeNeyer/ipam/server/handlers"
	"github.com/JakeNeyer/ipam/store"
	"github.com/swaggest/openapi-go/openapi31"
	"github.com/swaggest/rest/response/gzip"
	"github.com/swaggest/rest/web"
	swgui "github.com/swaggest/swgui/v5emb"
)

func NewServer(s *store.Store) *web.Service {
	svc := web.NewService(openapi31.NewReflector())

	svc.OpenAPISchema().SetTitle("IPAM Service")
	svc.OpenAPISchema().SetVersion("1.0.0")

	svc.Wrap(
		auth.Middleware(s),
		gzip.Middleware,
	)

	// Setup routes (no auth; only when no users exist).
	svc.Handle("/api/setup/status", handlers.GetSetupStatusHandler(s))
	svc.Handle("/api/setup", handlers.PostSetupHandler(s))

	// Auth routes (no auth required for login/logout).
	svc.Handle("/api/auth/login", handlers.LoginHandler(s))
	svc.Handle("/api/auth/logout", handlers.LogoutHandler(s))
	svc.Handle("/api/auth/me", handlers.MeHandler())
	svc.Handle("/api/auth/me/tour-completed", handlers.TourCompletedHandler(s))

	// Admin routes (auth + admin role required).
	svc.Handle("/api/admin/users", handlers.AdminUsersHandler(s))

	// Environment use case handlers.
	createEnvUC := handlers.NewCreateEnvironmentUseCase(s)
	svc.Post("/api/environments", createEnvUC)

	listEnvUC := handlers.NewListEnvironmentsUseCase(s)
	svc.Get("/api/environments", listEnvUC)

	getEnvUC := handlers.NewGetEnvironmentUseCase(s)
	svc.Get("/api/environments/{id}", getEnvUC)

	suggestEnvBlockCIDRUC := handlers.NewSuggestEnvironmentBlockCIDRUseCase(s)
	svc.Get("/api/environments/{id}/suggest-block-cidr", suggestEnvBlockCIDRUC)

	updateEnvUC := handlers.NewUpdateEnvironmentUseCase(s)
	svc.Put("/api/environments/{id}", updateEnvUC)

	deleteEnvUC := handlers.NewDeleteEnvironmentUseCase(s)
	svc.Delete("/api/environments/{id}", deleteEnvUC)

	// Block use case handlers.
	createBlockUC := handlers.NewCreateBlockUseCase(s)
	svc.Post("/api/blocks", createBlockUC)

	listBlocksUC := handlers.NewListBlocksUseCase(s)
	svc.Get("/api/blocks", listBlocksUC)

	getBlockUC := handlers.NewGetBlockUseCase(s)
	svc.Get("/api/blocks/{id}", getBlockUC)

	updateBlockUC := handlers.NewUpdateBlockUseCase(s)
	svc.Put("/api/blocks/{id}", updateBlockUC)

	deleteBlockUC := handlers.NewDeleteBlockUseCase(s)
	svc.Delete("/api/blocks/{id}", deleteBlockUC)

	getBlockUsageUC := handlers.NewGetBlockUsageUseCase(s)
	svc.Get("/api/blocks/{id}/usage", getBlockUsageUC)

	suggestBlockCIDRUC := handlers.NewSuggestBlockCIDRUseCase(s)
	svc.Get("/api/blocks/{id}/suggest-cidr", suggestBlockCIDRUC)

	// Allocation use case handlers.
	createAllocUC := handlers.NewCreateAllocationUseCase(s)
	svc.Post("/api/allocations", createAllocUC)

	listAllocUC := handlers.NewListAllocationsUseCase(s)
	svc.Get("/api/allocations", listAllocUC)

	getAllocUC := handlers.NewGetAllocationUseCase(s)
	svc.Get("/api/allocations/{id}", getAllocUC)

	updateAllocUC := handlers.NewUpdateAllocationUseCase(s)
	svc.Put("/api/allocations/{id}", updateAllocUC)

	deleteAllocUC := handlers.NewDeleteAllocationUseCase(s)
	svc.Delete("/api/allocations/{id}", deleteAllocUC)

	svc.Handle("/api/export/csv", handlers.ExportCSVHandler(s))

	// Swagger UI endpoint at /docs.
	svc.Docs("/docs", swgui.New)

	return svc
}
