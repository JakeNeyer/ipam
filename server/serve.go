package server

import (
	"github.com/JakeNeyer/ipam/server/auth"
	"github.com/JakeNeyer/ipam/server/config"
	"github.com/JakeNeyer/ipam/server/handlers"
	"github.com/JakeNeyer/ipam/server/oauth"
	"github.com/JakeNeyer/ipam/store"
	"github.com/swaggest/openapi-go/openapi31"
	"github.com/swaggest/rest/nethttp"
	"github.com/swaggest/rest/response/gzip"
	"github.com/swaggest/rest/web"
	swguicfg "github.com/swaggest/swgui"
	swgui "github.com/swaggest/swgui/v5emb"
)

func NewServer(s store.Storer, cfg *config.Config) *web.Service {
	svc := web.NewService(openapi31.NewReflector())

	svc.OpenAPISchema().SetTitle("IPAM Service")
	svc.OpenAPISchema().SetVersion("1.0.0")

	svc.Wrap(
		auth.Middleware(s),
		gzip.Middleware,
	)

	getSetupStatusUC := handlers.NewGetSetupStatusUseCase(s)
	svc.Get("/api/setup/status", getSetupStatusUC)

	postSetupUC := handlers.NewPostSetupUseCase(s, cfg)
	svc.Post("/api/setup", postSetupUC)
	svc.Post("/api/setup/status", postSetupUC)

	svc.Handle("/api/signup/validate", handlers.ValidateSignupInviteHandler(s))
	svc.Handle("/api/signup/register", handlers.RegisterWithInviteHandler(s))

	svc.Handle("/api/auth/config", handlers.AuthConfigHandler(cfg))
	loginLimiter := auth.NewLoginAttemptLimiter(auth.DefaultLoginMaxAttempts, auth.DefaultLoginWindow)
	loginUC := handlers.NewLoginUseCase(s, loginLimiter, cfg)
	svc.Post("/api/auth/login", loginUC)
	logoutUC := handlers.NewLogoutUseCase(s)
	svc.Post("/api/auth/logout", logoutUC, nethttp.SuccessStatus(204))
	if cfg != nil && len(cfg.EnabledOAuthProviders()) > 0 {
		registry := oauth.NewProviderRegistry()
		for _, provider := range cfg.EnabledOAuthProviders() {
			svc.Handle("/api/auth/oauth/"+provider+"/start", handlers.OAuthStartHandler(cfg, registry))
			svc.Handle("/api/auth/oauth/"+provider+"/callback", handlers.OAuthCallbackHandler(s, cfg, registry))
		}
	}

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

	svc.Handle("/api/admin/users", handlers.AdminUsersHandler(s))
	svc.Handle("/api/admin/users/{id}/role", handlers.UpdateUserRoleHandler(s))
	svc.Handle("/api/admin/users/{id}/organization", handlers.UpdateUserOrganizationHandler(s))
	svc.Handle("/api/admin/users/{id}", handlers.DeleteUserHandler(s))
	svc.Handle("/api/admin/organizations", handlers.AdminOrganizationsHandler(s))
	svc.Handle("/api/admin/organizations/{id}", handlers.AdminOrganizationByIDHandler(s))
	svc.Handle("/api/admin/signup-invites", handlers.AdminSignupInvitesHandler(s, cfg))
	svc.Handle("/api/admin/signup-invites/{id}", handlers.RevokeSignupInviteHandler(s))

	listReservedUC := handlers.NewListReservedBlocksUseCase(s)
	svc.Get("/api/reserved-blocks", listReservedUC)
	createReservedUC := handlers.NewCreateReservedBlockUseCase(s)
	svc.Post("/api/reserved-blocks", createReservedUC)
	updateReservedUC := handlers.NewUpdateReservedBlockUseCase(s)
	svc.Put("/api/reserved-blocks/{id}", updateReservedUC)
	deleteReservedUC := handlers.NewDeleteReservedBlockUseCase(s)
	svc.Delete("/api/reserved-blocks/{id}", deleteReservedUC)

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

	createAllocUC := handlers.NewCreateAllocationUseCase(s)
	svc.Post("/api/allocations", createAllocUC)

	autoAllocUC := handlers.NewAutoAllocateUseCase(s)
	svc.Post("/api/allocations/auto", autoAllocUC)

	listAllocUC := handlers.NewListAllocationsUseCase(s)
	svc.Get("/api/allocations", listAllocUC)

	getAllocUC := handlers.NewGetAllocationUseCase(s)
	svc.Get("/api/allocations/{id}", getAllocUC)

	updateAllocUC := handlers.NewUpdateAllocationUseCase(s)
	svc.Put("/api/allocations/{id}", updateAllocUC)

	deleteAllocUC := handlers.NewDeleteAllocationUseCase(s)
	svc.Delete("/api/allocations/{id}", deleteAllocUC)

	svc.Method("GET", "/api/export/csv", handlers.ExportCSVHandler(s))

	svc.Docs("/docs", swgui.NewWithConfig(swguicfg.Config{
		AppendHead: swaggerThemeCSS(),
	}))

	return svc
}
