package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/JakeNeyer/ipam/server/auth"
	"github.com/JakeNeyer/ipam/server/config"
	"github.com/JakeNeyer/ipam/server/oauth"
	"github.com/JakeNeyer/ipam/store"
)

func oauthTestProvider() config.OAuthProviderConfig {
	return config.OAuthProviderConfig{
		ClientID:     "cid",
		ClientSecret: "secret",
		Scopes:       []string{"openid", "email"},
		AuthURL:      "https://idp.example/oauth/authorize",
		TokenURL:     "https://idp.example/oauth/token",
		UserInfoURL:  "https://idp.example/oauth/userinfo",
	}
}

func TestAuthConfigHandler_NoConfig(t *testing.T) {
	handler := AuthConfigHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/auth/config", nil)
	rr := httptest.NewRecorder()
	handler(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
	var out AuthConfigResponse
	if err := json.NewDecoder(rr.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if len(out.OAuthProviders) != 0 {
		t.Errorf("OAuthProviders = %v, want []", out.OAuthProviders)
	}
}

func TestAuthConfigHandler_WithProvider(t *testing.T) {
	cfg := &config.Config{OAuth: config.OAuthConfig{Providers: map[string]config.OAuthProviderConfig{
		"sso": oauthTestProvider(),
	}}}
	handler := AuthConfigHandler(cfg)
	req := httptest.NewRequest(http.MethodGet, "/api/auth/config", nil)
	rr := httptest.NewRecorder()
	handler(rr, req)
	var out AuthConfigResponse
	if err := json.NewDecoder(rr.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if len(out.OAuthProviders) != 1 || out.OAuthProviders[0] != "sso" {
		t.Errorf("OAuthProviders = %v", out.OAuthProviders)
	}
}

func TestOAuthStartHandler_ProviderRequired(t *testing.T) {
	cfg := &config.Config{OAuth: config.OAuthConfig{Providers: map[string]config.OAuthProviderConfig{
		"sso": oauthTestProvider(),
	}}}
	registry, err := oauth.NewProviderRegistry(cfg)
	if err != nil {
		t.Fatalf("NewProviderRegistry() error = %v", err)
	}
	handler := OAuthStartHandler(cfg, registry)
	req := httptest.NewRequest(http.MethodGet, "/api/auth/oauth/", nil)
	rr := httptest.NewRecorder()
	handler(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rr.Code)
	}
}

func TestOAuthStartHandler_ProviderNotEnabled(t *testing.T) {
	cfg := &config.Config{OAuth: config.OAuthConfig{Providers: map[string]config.OAuthProviderConfig{}}}
	registry, err := oauth.NewProviderRegistry(cfg)
	if err != nil {
		t.Fatalf("NewProviderRegistry() error = %v", err)
	}
	handler := OAuthStartHandler(cfg, registry)
	req := httptest.NewRequest(http.MethodGet, "/api/auth/oauth/sso/start", nil)
	rr := httptest.NewRecorder()
	handler(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rr.Code)
	}
}

func TestOAuthStartHandler_SetsStateCookie(t *testing.T) {
	cfg := &config.Config{OAuth: config.OAuthConfig{Providers: map[string]config.OAuthProviderConfig{
		"sso": oauthTestProvider(),
	}}}
	registry, err := oauth.NewProviderRegistry(cfg)
	if err != nil {
		t.Fatalf("NewProviderRegistry() error = %v", err)
	}
	handler := OAuthStartHandler(cfg, registry)
	req := httptest.NewRequest(http.MethodGet, "/api/auth/oauth/sso/start?invite_token=secret-invite", nil)
	rr := httptest.NewRecorder()
	handler(rr, req)
	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", rr.Code)
	}
	var stateCookie *http.Cookie
	for _, c := range rr.Result().Cookies() {
		if c.Name == auth.OAuthStateCookieName {
			stateCookie = c
			break
		}
	}
	if stateCookie == nil {
		t.Fatal("missing oauth state cookie")
	}
	if stateCookie.HttpOnly != true || stateCookie.Path != "/api/auth/oauth" {
		t.Errorf("cookie = %+v", stateCookie)
	}
}

func TestOAuthCallbackHandler_InvalidState(t *testing.T) {
	s := store.NewStore()
	cfg := &config.Config{OAuth: config.OAuthConfig{Providers: map[string]config.OAuthProviderConfig{
		"sso": oauthTestProvider(),
	}}}
	registry, err := oauth.NewProviderRegistry(cfg)
	if err != nil {
		t.Fatalf("NewProviderRegistry() error = %v", err)
	}
	handler := OAuthCallbackHandler(s, cfg, registry)
	req := httptest.NewRequest(http.MethodGet, "/api/auth/oauth/sso/callback?code=abc&state=wrong", nil)
	rr := httptest.NewRecorder()
	handler(rr, req)
	loc := rr.Header().Get("Location")
	if !strings.Contains(loc, "error=") || !strings.Contains(loc, "invalid") {
		t.Errorf("Location = %s", loc)
	}
}

func TestOAuthStartHandler_RedirectsToProvider(t *testing.T) {
	cfg := &config.Config{OAuth: config.OAuthConfig{Providers: map[string]config.OAuthProviderConfig{
		"sso": oauthTestProvider(),
	}}}
	registry, err := oauth.NewProviderRegistry(cfg)
	if err != nil {
		t.Fatalf("NewProviderRegistry() error = %v", err)
	}
	handler := OAuthStartHandler(cfg, registry)
	req := httptest.NewRequest(http.MethodGet, "/api/auth/oauth/sso/start", nil)
	req.Host = "example.com"
	rr := httptest.NewRecorder()
	handler(rr, req)
	if rr.Code != http.StatusFound {
		t.Errorf("status = %d, want 302", rr.Code)
	}
	loc := rr.Header().Get("Location")
	if !strings.Contains(loc, "idp.example") || !strings.Contains(loc, "client_id=cid") {
		t.Errorf("Location = %s", loc)
	}
	u, err := url.Parse(loc)
	if err != nil {
		t.Fatal(err)
	}
	if got := u.Query().Get("redirect_uri"); got != "http://example.com/api/auth/oauth/sso/callback" {
		t.Errorf("redirect_uri = %q", got)
	}
}

func TestOAuthStartHandler_UsesAppOriginForRedirectURI(t *testing.T) {
	cfg := &config.Config{
		AppOrigin: "http://localhost:5173",
		OAuth: config.OAuthConfig{Providers: map[string]config.OAuthProviderConfig{
			"sso": oauthTestProvider(),
		}},
	}
	registry, err := oauth.NewProviderRegistry(cfg)
	if err != nil {
		t.Fatalf("NewProviderRegistry() error = %v", err)
	}
	handler := OAuthStartHandler(cfg, registry)
	req := httptest.NewRequest(http.MethodGet, "/api/auth/oauth/sso/start", nil)
	req.Host = "127.0.0.1:8011"
	rr := httptest.NewRecorder()
	handler(rr, req)
	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", rr.Code)
	}
	u, err := url.Parse(rr.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	if got := u.Query().Get("redirect_uri"); got != "http://localhost:5173/api/auth/oauth/sso/callback" {
		t.Errorf("redirect_uri = %q, want APP_ORIGIN callback", got)
	}
}

func TestOauthRedirectBase(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/auth/oauth/github/start", nil)
	req.Host = "127.0.0.1:8011"
	if got := oauthRedirectBase(req, "http://localhost:5173/"); got != "http://localhost:5173" {
		t.Errorf("with APP_ORIGIN = %q", got)
	}
	if got := oauthRedirectBase(req, ""); got != "http://127.0.0.1:8011" {
		t.Errorf("without APP_ORIGIN = %q", got)
	}
	req.Header.Set("X-Forwarded-Host", "localhost:5173")
	req.Header.Set("X-Forwarded-Proto", "http")
	if got := oauthRedirectBase(req, ""); got != "http://localhost:5173" {
		t.Errorf("with forwarded host = %q", got)
	}
}

func TestOAuthCallbackHandler_MissingCode(t *testing.T) {
	s := store.NewStore()
	cfg := &config.Config{OAuth: config.OAuthConfig{Providers: map[string]config.OAuthProviderConfig{
		"sso": oauthTestProvider(),
	}}}
	registry, err := oauth.NewProviderRegistry(cfg)
	if err != nil {
		t.Fatalf("NewProviderRegistry() error = %v", err)
	}
	handler := OAuthCallbackHandler(s, cfg, registry)
	req := httptest.NewRequest(http.MethodGet, "/api/auth/oauth/sso/callback", nil)
	rr := httptest.NewRecorder()
	handler(rr, req)
	if rr.Code != http.StatusFound {
		t.Errorf("status = %d, want 302", rr.Code)
	}
	loc := rr.Header().Get("Location")
	if !strings.HasPrefix(loc, "/#login?error=") {
		t.Errorf("Location = %s", loc)
	}
}
