package handlers

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/JakeNeyer/ipam/server/auth"
	"github.com/JakeNeyer/ipam/server/config"
	"github.com/JakeNeyer/ipam/server/oauth"
	"github.com/JakeNeyer/ipam/server/validation"
	"github.com/JakeNeyer/ipam/store"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

const (
	oauthStateLogin  = "login"
	oauthStateInvite = "invite:"
)

// AuthConfigResponse is the response for GET /api/auth/config (no auth).
type AuthConfigResponse struct {
	OAuthProviders []string `json:"oauth_providers"`
	// GitHubOAuthEnabled is true when "github" is in OAuthProviders (backward compatibility).
	GitHubOAuthEnabled bool `json:"github_oauth_enabled"`
}

// AuthConfigHandler returns auth-related config for the frontend (enabled OAuth provider IDs).
func AuthConfigHandler(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		providers := cfg.EnabledOAuthProviders()
		githubEnabled := false
		for _, p := range providers {
			if p == "github" {
				githubEnabled = true
				break
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(AuthConfigResponse{
			OAuthProviders:     providers,
			GitHubOAuthEnabled: githubEnabled,
		})
	}
}

// OAuthStartHandler redirects to the provider's OAuth authorize URL. Path: /api/auth/oauth/:provider/start. Query: invite_token (optional).
func OAuthStartHandler(cfg *config.Config, registry *oauth.ProviderRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		provider := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/api/auth/oauth/"))
		provider = strings.TrimSuffix(provider, "/start")
		provider = strings.Trim(provider, "/")
		if provider == "" {
			auth.WriteJSONError(w, "provider required", http.StatusBadRequest)
			return
		}
		pc := cfg.OAuthProvider(provider)
		if pc == nil {
			auth.WriteJSONError(w, "OAuth provider not enabled", http.StatusNotFound)
			return
		}
		endpoint, ok := registry.Endpoint(provider)
		if !ok {
			auth.WriteJSONError(w, "OAuth provider not supported", http.StatusNotFound)
			return
		}
		inviteToken := strings.TrimSpace(r.URL.Query().Get("invite_token"))
		state := oauthStateLogin
		if inviteToken != "" {
			state = oauthStateInvite + base64.URLEncoding.EncodeToString([]byte(inviteToken))
		}
		redirectURI := redirectBase(r) + "/api/auth/oauth/" + provider + "/callback"
		conf := &oauth2.Config{
			ClientID:     pc.ClientID,
			ClientSecret: pc.ClientSecret,
			RedirectURL:  redirectURI,
			Endpoint:     endpoint,
			Scopes:       pc.Scopes,
		}
		if len(conf.Scopes) == 0 && provider == "github" {
			conf.Scopes = []string{"user:email"}
		}
		authURL := conf.AuthCodeURL(state)
		http.Redirect(w, r, authURL, http.StatusFound)
	}
}

// OAuthCallbackHandler exchanges code for token, fetches user info, then creates/links user and sets session.
// Path: /api/auth/oauth/:provider/callback.
func OAuthCallbackHandler(s store.Storer, cfg *config.Config, registry *oauth.ProviderRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		provider := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/api/auth/oauth/"))
		provider = strings.TrimSuffix(provider, "/callback")
		provider = strings.Trim(provider, "/")
		if provider == "" {
			redirectWithError(w, r, "provider required", cfg.AppOrigin)
			return
		}
		pc := cfg.OAuthProvider(provider)
		if pc == nil {
			redirectWithError(w, r, "OAuth provider not enabled", cfg.AppOrigin)
			return
		}
		endpoint, ok := registry.Endpoint(provider)
		if !ok {
			redirectWithError(w, r, "OAuth provider not supported", cfg.AppOrigin)
			return
		}
		code := strings.TrimSpace(r.URL.Query().Get("code"))
		state := strings.TrimSpace(r.URL.Query().Get("state"))
		if code == "" || state == "" {
			redirectWithError(w, r, "missing code or state", cfg.AppOrigin)
			return
		}
		redirectURI := redirectBase(r) + "/api/auth/oauth/" + provider + "/callback"
		conf := &oauth2.Config{
			ClientID:     pc.ClientID,
			ClientSecret: pc.ClientSecret,
			RedirectURL:  redirectURI,
			Endpoint:     endpoint,
			Scopes:       pc.Scopes,
		}
		if len(conf.Scopes) == 0 && provider == "github" {
			conf.Scopes = []string{"user:email"}
		}
		token, err := conf.Exchange(r.Context(), code)
		if err != nil {
			redirectWithError(w, r, "failed to exchange code", cfg.AppOrigin)
			return
		}
		providerUserID, email, err := registry.UserInfo(r.Context(), provider, token)
		if err != nil || providerUserID == "" || email == "" {
			redirectWithError(w, r, "failed to fetch user info", cfg.AppOrigin)
			return
		}
		email = strings.TrimSpace(strings.ToLower(email))
		if !validation.ValidateEmail(email) {
			redirectWithError(w, r, "invalid email from provider", cfg.AppOrigin)
			return
		}

		appOrigin := cfg.AppOrigin
		var inviteToken string
		if strings.HasPrefix(state, oauthStateInvite) {
			b, err := base64.URLEncoding.DecodeString(strings.TrimPrefix(state, oauthStateInvite))
			if err != nil {
				redirectWithError(w, r, "invalid state", appOrigin)
				return
			}
			inviteToken = string(b)
		}

		secure := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
		appRedirect := appRedirectBase(r, appOrigin) + appHashPath("dashboard")

		if inviteToken != "" {
			inv, err := s.GetSignupInviteByToken(inviteToken)
			if err != nil {
				redirectWithError(w, r, "invalid or expired invite link", appOrigin)
				return
			}
			inviter, err := s.GetUser(inv.CreatedBy)
			if err != nil {
				redirectWithError(w, r, "invalid invite", appOrigin)
				return
			}
			orgID := inv.OrganizationID
			if orgID == uuid.Nil {
				orgID = inviter.OrganizationID
			}
			role := inv.Role
			if role != store.RoleAdmin {
				role = store.RoleUser
			}
			newUser := &store.User{
				Email:                email,
				PasswordHash:         "",
				Role:                 role,
				OrganizationID:       orgID,
				OAuthProvider:        provider,
				OAuthProviderUserID:  providerUserID,
			}
			if err := s.CreateUser(newUser); err != nil {
				redirectWithError(w, r, "could not create account", appOrigin)
				return
			}
			_ = s.MarkSignupInviteUsed(inv.ID, newUser.ID)
			setSessionAndRedirect(w, r, s, newUser, secure, appRedirect)
			return
		}

		user, err := s.GetUserByOAuth(provider, providerUserID)
		if err == nil {
			setSessionAndRedirect(w, r, s, user, secure, appRedirect)
			return
		}
		user, err = s.GetUserByEmail(email)
		if err == nil {
			if user.OAuthProvider == "" || user.OAuthProviderUserID == "" {
				_ = s.SetUserOAuth(user.ID, provider, providerUserID)
			}
			setSessionAndRedirect(w, r, s, user, secure, appRedirect)
			return
		}
		users, listErr := s.ListUsers(nil)
		if listErr == nil && len(users) == 1 {
			onlyUser := users[0]
			if onlyUser.OrganizationID == uuid.Nil && onlyUser.Role == store.RoleAdmin {
				_ = s.SetUserOAuth(onlyUser.ID, provider, providerUserID)
				setSessionAndRedirect(w, r, s, onlyUser, secure, appRedirect)
				return
			}
		}
		redirectWithError(w, r, "Use a signup link or ask an admin to create your account", appOrigin)
	}
}

func redirectBase(r *http.Request) string {
	scheme := "https"
	if r.TLS == nil && r.Header.Get("X-Forwarded-Proto") != "https" {
		scheme = "http"
	}
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	host := r.Host
	if h := r.Header.Get("X-Forwarded-Host"); h != "" {
		host = h
	}
	return scheme + "://" + host
}

func appRedirectBase(r *http.Request, appOrigin string) string {
	if s := strings.TrimSpace(appOrigin); s != "" {
		return strings.TrimSuffix(s, "/")
	}
	return redirectBase(r)
}

func appHashPath(fragment string) string {
	return "/#" + fragment
}

func redirectWithError(w http.ResponseWriter, r *http.Request, msg string, appOrigin string) {
	base := appRedirectBase(r, appOrigin)
	u := base + appHashPath("login") + "?error=" + url.QueryEscape(msg)
	http.Redirect(w, r, u, http.StatusFound)
}

func setSessionAndRedirect(w http.ResponseWriter, r *http.Request, s store.Storer, user *store.User, secure bool, redirectURL string) {
	sessionID := auth.NewSessionID()
	s.CreateSession(sessionID, user.ID, time.Now().Add(auth.SessionDuration))
	auth.SetSessionCookie(w, sessionID, secure)
	http.Redirect(w, r, redirectURL, http.StatusFound)
}
