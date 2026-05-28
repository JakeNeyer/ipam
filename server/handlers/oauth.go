package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/JakeNeyer/ipam/internal/logger"
	"github.com/JakeNeyer/ipam/server/auth"
	"github.com/JakeNeyer/ipam/server/config"
	"github.com/JakeNeyer/ipam/server/oauth"
	"github.com/JakeNeyer/ipam/server/validation"
	"github.com/JakeNeyer/ipam/store"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

type OAuthProviderOption struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

type AuthConfigResponse struct {
	OAuthProviders       []string              `json:"oauth_providers"`
	OAuthProviderOptions []OAuthProviderOption `json:"oauth_provider_options"`
}

func AuthConfigHandler(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		providers := cfg.EnabledOAuthProviders()
		options := make([]OAuthProviderOption, 0, len(providers))
		for _, id := range providers {
			label := config.DefaultOAuthDisplayName(id)
			if pc := cfg.OAuthProvider(id); pc != nil && pc.DisplayName != "" {
				label = pc.DisplayName
			}
			options = append(options, OAuthProviderOption{ID: id, DisplayName: label})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(AuthConfigResponse{
			OAuthProviders:       providers,
			OAuthProviderOptions: options,
		})
	}
}

func OAuthStartHandler(cfg *config.Config, registry *oauth.ProviderRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		provider := config.NormalizeOAuthProviderID(strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/api/auth/oauth/")))
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
		secure := requestSecure(r)
		nonce, err := auth.NewOAuthStateNonce()
		if err != nil {
			auth.WriteJSONError(w, "could not start OAuth", http.StatusInternalServerError)
			return
		}
		if err := auth.SetOAuthStateCookie(w, auth.OAuthStatePayload{
			Nonce: nonce, Provider: provider, InviteToken: inviteToken,
		}, secure); err != nil {
			auth.WriteJSONError(w, "could not start OAuth", http.StatusInternalServerError)
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
		conf.Scopes = defaultOAuthScopes(provider, conf.Scopes)
		authURL := conf.AuthCodeURL(nonce)
		// #nosec G710 -- OAuth provider authorize URL is generated from trusted configured endpoint.
		http.Redirect(w, r, authURL, http.StatusFound)
	}
}

func OAuthCallbackHandler(s store.Storer, cfg *config.Config, registry *oauth.ProviderRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		provider := config.NormalizeOAuthProviderID(strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/api/auth/oauth/")))
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
		stored, ok := auth.OAuthStateFromRequest(r)
		if !ok || stored.Nonce != state || stored.Provider != provider {
			redirectWithError(w, r, "invalid or expired OAuth state", cfg.AppOrigin)
			return
		}
		inviteToken := strings.TrimSpace(stored.InviteToken)
		secure := requestSecure(r)
		auth.ClearOAuthStateCookie(w, secure)
		redirectURI := redirectBase(r) + "/api/auth/oauth/" + provider + "/callback"
		conf := &oauth2.Config{
			ClientID:     pc.ClientID,
			ClientSecret: pc.ClientSecret,
			RedirectURL:  redirectURI,
			Endpoint:     endpoint,
			Scopes:       pc.Scopes,
		}
		conf.Scopes = defaultOAuthScopes(provider, conf.Scopes)
		ctx := context.WithValue(
			r.Context(),
			oauth2.HTTPClient,
			registry.HTTPClient(),
		)
		token, err := conf.Exchange(ctx, code)
		if err != nil {
			logger.Error("failed to fetch user exchange code", logger.ErrAttr(err))
			redirectWithError(w, r, "failed to exchange code", cfg.AppOrigin)
			return
		}
		providerUserID, email, err := registry.UserInfo(ctx, provider, token)
		if err != nil || providerUserID == "" || email == "" {
			logger.Error("failed to fetch user info", logger.ErrAttr(err))
			redirectWithError(w, r, "failed to fetch user info", cfg.AppOrigin)
			return
		}
		email = strings.TrimSpace(strings.ToLower(email))
		if !validation.ValidateEmail(email) {
			redirectWithError(w, r, "invalid email from provider", cfg.AppOrigin)
			return
		}

		appOrigin := cfg.AppOrigin
		appRedirect := appRedirectBase(appOrigin) + appHashPath("dashboard")

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
				Email:               email,
				PasswordHash:        "",
				Role:                role,
				OrganizationID:      orgID,
				OAuthProvider:       provider,
				OAuthProviderUserID: providerUserID,
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
		if pc.AllowEmailMatch {
			user, err = s.GetUserByEmail(email)
			if err == nil {
				if user.OAuthProvider == "" || user.OAuthProviderUserID == "" {
					_ = s.SetUserOAuth(user.ID, provider, providerUserID)
				}
				setSessionAndRedirect(w, r, s, user, secure, appRedirect)
				return
			}
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

func defaultOAuthScopes(_ string, scopes []string) []string {
	if len(scopes) > 0 {
		return scopes
	}
	return []string{"openid", "email", "profile"}
}

func requestSecure(r *http.Request) bool {
	return r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
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

func appRedirectBase(appOrigin string) string {
	if s := strings.TrimSpace(appOrigin); s != "" {
		return strings.TrimSuffix(s, "/")
	}
	// Use relative redirects by default to avoid host-header-based open redirect risks.
	return ""
}

func appHashPath(fragment string) string {
	return "/#" + fragment
}

func redirectWithError(w http.ResponseWriter, r *http.Request, msg string, appOrigin string) {
	base := appRedirectBase(appOrigin)
	u := base + appHashPath("login") + "?error=" + url.QueryEscape(msg)
	http.Redirect(w, r, u, http.StatusFound)
}

func setSessionAndRedirect(w http.ResponseWriter, r *http.Request, s store.Storer, user *store.User, secure bool, redirectURL string) {
	sessionID := auth.NewSessionID()
	s.CreateSession(sessionID, user.ID, time.Now().Add(auth.SessionDuration))
	auth.SetSessionCookie(w, sessionID, secure)
	http.Redirect(w, r, redirectURL, http.StatusFound)
}
