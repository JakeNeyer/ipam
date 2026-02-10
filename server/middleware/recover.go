package middleware

import (
	"encoding/json"
	"net/http"
	"runtime/debug"

	"github.com/JakeNeyer/ipam/internal/logger"
)

// Recover wraps an http.Handler and recovers from panics: logs the panic and stack, returns 500.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				stack := debug.Stack()
				logger.Error("panic recovered",
					logger.KeyMethod, r.Method,
					logger.KeyPath, r.URL.Path,
					"panic", v,
					"stack", string(stack),
				)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "Internal Server Error"})
			}
		}()
		next.ServeHTTP(w, r)
	})
}
