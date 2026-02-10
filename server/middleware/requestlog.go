package middleware

import (
	"net/http"
	"time"

	"github.com/JakeNeyer/ipam/internal/logger"
)

// responseWriter wraps http.ResponseWriter to capture status code and bytes written.
type responseWriter struct {
	http.ResponseWriter
	status int
	written int64
}

func (w *responseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *responseWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.written += int64(n)
	return n, err
}

// RequestLog wraps an http.Handler and logs each request (method, path, status, duration) to stdout.
// Use with OpenTelemetry: otelhttp.NewHandler(RequestLog(handler), "service name").
func RequestLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrap := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(wrap, r)
		durationMs := time.Since(start).Milliseconds()
		logger.Request(r.Method, r.URL.Path, wrap.status, durationMs)
	})
}
