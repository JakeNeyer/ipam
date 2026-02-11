package middleware

import (
	"net/http"
	"time"

	"github.com/JakeNeyer/ipam/internal/logger"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// responseWriterWithSize wraps http.ResponseWriter to capture status code and bytes written.
type responseWriterWithSize struct {
	http.ResponseWriter
	status int
	written int64
}

func (w *responseWriterWithSize) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *responseWriterWithSize) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.written += int64(n)
	return n, err
}

// OtelRequestResponseLog wraps an http.Handler and logs every request and response to stdout,
// and records request/response attributes on the current OpenTelemetry span.
// Use inside otelhttp so the span is in context: otelhttp.NewHandler(OtelRequestResponseLog(handler), "ipam").
func OtelRequestResponseLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ctx := r.Context()

		path := r.URL.Path
		method := r.Method
		reqContentLength := r.ContentLength
		if reqContentLength < 0 {
			reqContentLength = 0
		}

		if span := trace.SpanFromContext(ctx); span.IsRecording() {
			span.SetAttributes(
				attribute.String("http.request.method", method),
				attribute.String("http.request.path", path),
				attribute.Int64("http.request.body.size", reqContentLength),
			)
		}
		logger.Info("request",
			logger.KeyMethod, method,
			logger.KeyPath, path,
			"request_body_size", reqContentLength,
		)

		wrap := &responseWriterWithSize{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(wrap, r)

		durationMs := time.Since(start).Milliseconds()

		if span := trace.SpanFromContext(ctx); span.IsRecording() {
			span.SetAttributes(
				attribute.Int("http.response.status_code", wrap.status),
				attribute.Int64("http.response.duration_ms", durationMs),
				attribute.Int64("http.response.body.size", wrap.written),
			)
		}
		logger.Info("response",
			logger.KeyMethod, method,
			logger.KeyPath, path,
			logger.KeyStatus, wrap.status,
			logger.KeyDuration, durationMs,
			"response_body_size", wrap.written,
		)
	})
}
