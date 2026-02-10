package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/JakeNeyer/ipam/store"
	"github.com/google/uuid"
)

const (
	SessionCookieName = "ipam_session"
	SessionDuration  = 24 * time.Hour
)

// Middleware returns a middleware that requires a valid session or API key for /api/* except login and logout.
func Middleware(s store.Storer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			if path == "/api/auth/login" || path == "/api/auth/logout" ||
				path == "/api/setup/status" || path == "/api/setup" {
				ctx := WithRequest(r.Context(), r)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			if !strings.HasPrefix(path, "/api/") {
				next.ServeHTTP(w, r)
				return
			}

			var user *store.User

			// Try session cookie first
			if cookie, err := r.Cookie(SessionCookieName); err == nil && cookie != nil && cookie.Value != "" {
				if sess, err := s.GetSession(cookie.Value); err == nil {
					if u, err := s.GetUser(sess.UserID); err == nil {
						user = u
					}
				}
			}

			// If no session, try Bearer token (API key)
			if user == nil {
				if bearer := r.Header.Get("Authorization"); strings.HasPrefix(bearer, "Bearer ") {
					rawToken := strings.TrimSpace(strings.TrimPrefix(bearer, "Bearer "))
					if rawToken != "" {
						keyHash := hashToken(rawToken)
						if u, err := s.GetUserByTokenHash(keyHash); err == nil {
							user = u
						}
					}
				}
			}

			if user == nil {
				WriteJSONError(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			ctx := WithRequest(r.Context(), r)
			ctx = WithUser(ctx, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func hashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

// WriteJSONError writes a JSON error response.
func WriteJSONError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// SetSessionCookie sets the session cookie on the response. secure should be true when using HTTPS.
func SetSessionCookie(w http.ResponseWriter, sessionID string, secure bool) {
	c := &http.Cookie{
		Name:     SessionCookieName,
		Value:    sessionID,
		Path:     "/",
		MaxAge:   int(SessionDuration.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
	}
	http.SetCookie(w, c)
}

// ClearSessionCookie clears the session cookie. secure should match the cookie that was set (e.g. request was HTTPS).
func ClearSessionCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
	})
}

// NewSessionID returns a new session ID.
func NewSessionID() string {
	return uuid.New().String()
}
