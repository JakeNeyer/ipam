package auth

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"
)

const (
	OAuthStateCookieName = "ipam_oauth_state"
	oAuthStateDuration   = 10 * time.Minute
)

// OAuthStatePayload is stored in the OAuth state cookie during the authorize redirect.
type OAuthStatePayload struct {
	Nonce       string `json:"nonce"`
	Provider    string `json:"provider"`
	InviteToken string `json:"invite_token,omitempty"`
}

// NewOAuthStateNonce returns a URL-safe random nonce for OAuth state.
func NewOAuthStateNonce() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// SetOAuthStateCookie stores the OAuth CSRF state (must match the state query param on callback).
func SetOAuthStateCookie(w http.ResponseWriter, payload OAuthStatePayload, secure bool) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	// #nosec G124 -- Secure matches session cookie behavior for local HTTP vs HTTPS.
	http.SetCookie(w, &http.Cookie{
		Name:     OAuthStateCookieName,
		Value:    base64.RawURLEncoding.EncodeToString(data),
		Path:     "/api/auth/oauth",
		MaxAge:   int(oAuthStateDuration.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
	})
	return nil
}

// OAuthStateFromRequest reads and validates the OAuth state cookie.
func OAuthStateFromRequest(r *http.Request) (OAuthStatePayload, bool) {
	c, err := r.Cookie(OAuthStateCookieName)
	if err != nil || c.Value == "" {
		return OAuthStatePayload{}, false
	}
	raw, err := base64.RawURLEncoding.DecodeString(c.Value)
	if err != nil {
		return OAuthStatePayload{}, false
	}
	var p OAuthStatePayload
	if err := json.Unmarshal(raw, &p); err != nil || p.Nonce == "" || p.Provider == "" {
		return OAuthStatePayload{}, false
	}
	return p, true
}

// ClearOAuthStateCookie removes the OAuth state cookie after callback.
func ClearOAuthStateCookie(w http.ResponseWriter, secure bool) {
	// #nosec G124 -- Secure must match the cookie originally set.
	http.SetCookie(w, &http.Cookie{
		Name:     OAuthStateCookieName,
		Value:    "",
		Path:     "/api/auth/oauth",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
	})
}
