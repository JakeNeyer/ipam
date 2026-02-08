package auth

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/JakeNeyer/ipam/store"
	"github.com/google/uuid"
)

const (
	SessionCookieName = "ipam_session"
	SessionDuration   = 24 * time.Hour
)

// Middleware returns a middleware that requires a valid session for /api/* except login and logout.
func Middleware(s *store.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			if path == "/api/auth/login" || path == "/api/auth/logout" ||
				path == "/api/setup/status" || path == "/api/setup" {
				next.ServeHTTP(w, r)
				return
			}
			if !strings.HasPrefix(path, "/api/") {
				next.ServeHTTP(w, r)
				return
			}
			cookie, err := r.Cookie(SessionCookieName)
			if err != nil || cookie == nil || cookie.Value == "" {
				WriteJSONError(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			sess, err := s.GetSession(cookie.Value)
			if err != nil {
				WriteJSONError(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			user, err := s.GetUser(sess.UserID)
			if err != nil {
				WriteJSONError(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			ctx := WithUser(r.Context(), user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// WriteJSONError writes a JSON error response.
func WriteJSONError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// SetSessionCookie sets the session cookie on the response.
func SetSessionCookie(w http.ResponseWriter, sessionID string) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    sessionID,
		Path:     "/",
		MaxAge:   int(SessionDuration.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearSessionCookie clears the session cookie.
func ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// NewSessionID returns a new session ID.
func NewSessionID() string {
	return uuid.New().String()
}
