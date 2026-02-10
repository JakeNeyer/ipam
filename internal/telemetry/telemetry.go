// Package telemetry initializes OpenTelemetry for the IPAM backend (traces to stdout).
// See https://opentelemetry.io/docs/languages/go/
package telemetry

import (
	"context"
	"io"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Init initializes the global TracerProvider with a stdout trace exporter.
// Call Shutdown before process exit.
func Init(ctx context.Context) (shutdown func(context.Context) error, err error) {
	var exp io.Writer = os.Stdout
	traceExporter, err := stdouttrace.New(stdouttrace.WithWriter(exp))
	if err != nil {
		return nil, err
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	otel.SetTracerProvider(tp)
	return tp.Shutdown, nil
}

// Shutdown flushes and shuts down the global TracerProvider. Call in main defer.
func Shutdown(ctx context.Context, shutdown func(context.Context) error) {
	if shutdown != nil {
		_ = shutdown(ctx)
	}
}
