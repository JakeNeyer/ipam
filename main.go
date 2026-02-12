package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/JakeNeyer/ipam/internal/logger"
	"github.com/JakeNeyer/ipam/internal/setup"
	"github.com/JakeNeyer/ipam/internal/telemetry"
	"github.com/JakeNeyer/ipam/server"
	"github.com/JakeNeyer/ipam/server/config"
	"github.com/JakeNeyer/ipam/server/handlers"
	"github.com/JakeNeyer/ipam/server/middleware"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func otelEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("ENABLE_OTEL")))
	return v == "true" || v == "1"
}

func main() {
	ctx := context.Background()

	if otelEnabled() {
		shutdown, err := telemetry.Init(ctx)
		if err != nil {
			logger.Error("telemetry init failed", logger.ErrAttr(err))
			os.Exit(1)
		}
		defer telemetry.Shutdown(ctx, shutdown)
		logger.Info("opentelemetry enabled (stdout traces)")
	}

	st, closeStore, err := setup.NewStore(ctx)
	if err != nil {
		logger.Error("store failed", logger.ErrAttr(err))
		os.Exit(1)
	}
	defer closeStore()

	serverCfg := config.LoadFromEnv()
	oauthEnabled := len(serverCfg.EnabledOAuthProviders()) > 0
	setup.EnsureInitialAdmin(st, oauthEnabled)
	setup.EnsureDemoFixtures(st)

	s := server.NewServer(st, serverCfg)

	handler := middleware.SecurityHeaders(s)
	handler = middleware.MaxBytes(handler)
	if otelEnabled() {
		handler = middleware.OtelRequestResponseLog(handler)
		handler = otelhttp.NewHandler(handler, "ipam")
	} else {
		handler = middleware.RequestLog(handler)
	}
	handler = middleware.Recover(handler)

	appOrigin := serverCfg.AppOrigin
	if appOrigin != "" {
		handler = handlers.Unauthorized(appOrigin, handler)
		logger.Info("app origin set; non-API requests return 401", slog.String("app_origin", appOrigin))
	} else {
		staticDir := handlers.ResolveStaticDir()
		if staticDir != "" {
			handler = handlers.Static(staticDir, handler)
			logger.Info("serving static files", slog.String("dir", staticDir))
		}
	}

	addr := "0.0.0.0"
	if port := os.Getenv("PORT"); port != "" {
		addr = addr + ":" + port
	} else {
		addr = "localhost:8011"
	}
	logger.Info("server listening", slog.String("addr", "http://"+addr), slog.String("docs", "http://"+addr+"/docs"))
	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		logger.Error("server failed", logger.ErrAttr(err))
		os.Exit(1)
	}
}
