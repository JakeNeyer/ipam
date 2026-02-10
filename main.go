package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/JakeNeyer/ipam/internal/logger"
	"github.com/JakeNeyer/ipam/internal/telemetry"
	"github.com/JakeNeyer/ipam/server"
	"github.com/JakeNeyer/ipam/server/middleware"
	"github.com/JakeNeyer/ipam/store"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func main() {
	ctx := context.Background()

	// OpenTelemetry: stdout traces for request spans
	shutdown, err := telemetry.Init(ctx)
	if err != nil {
		logger.Error("telemetry init failed", logger.ErrAttr(err))
		os.Exit(1)
	}
	defer telemetry.Shutdown(ctx, shutdown)

	var st store.Storer
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		var err error
		st, err = store.NewPostgresStore(ctx, dsn)
		if err != nil {
			logger.Error("postgres store failed", logger.ErrAttr(err))
			os.Exit(1)
		}
		if c, ok := st.(*store.PostgresStore); ok {
			defer c.Close()
		}
		logger.Info("store", slog.String("type", "postgres"))
	} else {
		st = store.NewStore()
		logger.Info("store", slog.String("type", "in_memory"))
	}
	s := server.NewServer(st)

	// Panic recovery (outermost), OpenTelemetry (spans + request/response logging on span and stdout)
	handler := middleware.OtelRequestResponseLog(s)
	handler = otelhttp.NewHandler(handler, "ipam")
	handler = middleware.Recover(handler)

	logger.Info("server listening", slog.String("addr", "http://localhost:8011"), slog.String("docs", "http://localhost:8011/docs"))
	if err := http.ListenAndServe("localhost:8011", handler); err != nil {
		logger.Error("server failed", logger.ErrAttr(err))
		os.Exit(1)
	}
}
