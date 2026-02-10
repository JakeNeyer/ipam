package server

import (
	"github.com/JakeNeyer/ipam/server/auth"
	"github.com/JakeNeyer/ipam/server/handlers"
	"github.com/JakeNeyer/ipam/store"
	"github.com/swaggest/openapi-go/openapi31"
	"github.com/swaggest/rest/nethttp"
	"github.com/swaggest/rest/response/gzip"
	"github.com/swaggest/rest/web"
	swguicfg "github.com/swaggest/swgui"
	swgui "github.com/swaggest/swgui/v5emb"
)

func NewServer(s store.Storer) *web.Service {
	svc := web.NewService(openapi31.NewReflector())

	svc.OpenAPISchema().SetTitle("IPAM Service")
	svc.OpenAPISchema().SetVersion("1.0.0")

	svc.Wrap(
		auth.Middleware(s),
		gzip.Middleware,
	)

	// Setup routes (no auth; only when no users exist).
	getSetupStatusUC := handlers.NewGetSetupStatusUseCase(s)
	svc.Get("/api/setup/status", getSetupStatusUC)

	postSetupUC := handlers.NewPostSetupUseCase(s)
	svc.Post("/api/setup", postSetupUC)
	svc.Post("/api/setup/status", postSetupUC) // alias so POST to status (e.g. form action) also creates admin

	// Auth routes (no auth required for login/logout).
	loginUC := handlers.NewLoginUseCase(s)
	svc.Post("/api/auth/login", loginUC)
	logoutUC := handlers.NewLogoutUseCase(s)
	svc.Post("/api/auth/logout", logoutUC, nethttp.SuccessStatus(204))

	meUC := handlers.NewMeUseCase()
	svc.Get("/api/auth/me", meUC)

	tourCompletedUC := handlers.NewTourCompletedUseCase(s)
	svc.Post("/api/auth/me/tour-completed", tourCompletedUC)

	listTokensUC := handlers.NewListTokensUseCase(s)
	svc.Get("/api/auth/me/tokens", listTokensUC)
	createTokenUC := handlers.NewCreateTokenUseCase(s)
	svc.Post("/api/auth/me/tokens", createTokenUC)
	deleteTokenUC := handlers.NewDeleteTokenUseCase(s)
	svc.Delete("/api/auth/me/tokens/{id}", deleteTokenUC)

	// Admin routes (auth + admin role required).
	svc.Handle("/api/admin/users", handlers.AdminUsersHandler(s))

	listReservedUC := handlers.NewListReservedBlocksUseCase(s)
	svc.Get("/api/admin/reserved-blocks", listReservedUC)
	createReservedUC := handlers.NewCreateReservedBlockUseCase(s)
	svc.Post("/api/admin/reserved-blocks", createReservedUC)
	deleteReservedUC := handlers.NewDeleteReservedBlockUseCase(s)
	svc.Delete("/api/admin/reserved-blocks/{id}", deleteReservedUC)

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

	svc.Method("GET", "/api/export/csv", handlers.ExportCSVHandler(s))

	// Swagger UI endpoint at /docs with Wintry-style dark theme.
	svc.Docs("/docs", swgui.NewWithConfig(swguicfg.Config{
		AppendHead: swaggerThemeCSS(),
	}))

	return svc
}
