package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JakeNeyer/ipam/server/config"
	"github.com/JakeNeyer/ipam/server/oauth"
	"github.com/JakeNeyer/ipam/store"
)

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
	if out.GitHubOAuthEnabled {
		t.Error("GitHubOAuthEnabled = true, want false")
	}
}

func TestAuthConfigHandler_WithGitHub(t *testing.T) {
	cfg := &config.Config{OAuth: config.OAuthConfig{Providers: map[string]config.OAuthProviderConfig{
		"github": {ClientID: "cid", ClientSecret: "secret"},
	}}}
	handler := AuthConfigHandler(cfg)
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
	if len(out.OAuthProviders) != 1 || out.OAuthProviders[0] != "github" {
		t.Errorf("OAuthProviders = %v, want [github]", out.OAuthProviders)
	}
	if !out.GitHubOAuthEnabled {
		t.Error("GitHubOAuthEnabled = false, want true")
	}
}

func TestOAuthStartHandler_ProviderRequired(t *testing.T) {
	cfg := &config.Config{OAuth: config.OAuthConfig{Providers: map[string]config.OAuthProviderConfig{
		"github": {ClientID: "cid", ClientSecret: "secret"},
	}}}
	registry := oauth.NewProviderRegistry()
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
	registry := oauth.NewProviderRegistry()
	handler := OAuthStartHandler(cfg, registry)
	req := httptest.NewRequest(http.MethodGet, "/api/auth/oauth/github/start", nil)
	req.Host = "example.com"
	rr := httptest.NewRecorder()
	handler(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rr.Code)
	}
}

func TestOAuthStartHandler_RedirectsToProvider(t *testing.T) {
	cfg := &config.Config{OAuth: config.OAuthConfig{Providers: map[string]config.OAuthProviderConfig{
		"github": {ClientID: "cid", ClientSecret: "secret", Scopes: []string{"user:email"}},
	}}}
	registry := oauth.NewProviderRegistry()
	handler := OAuthStartHandler(cfg, registry)
	req := httptest.NewRequest(http.MethodGet, "/api/auth/oauth/github/start", nil)
	req.Host = "example.com"
	rr := httptest.NewRecorder()
	handler(rr, req)
	if rr.Code != http.StatusFound {
		t.Errorf("status = %d, want 302", rr.Code)
	}
	loc := rr.Header().Get("Location")
	if loc == "" {
		t.Fatal("missing Location header")
	}
	if loc[:30] != "https://github.com/login/oauth" {
		t.Errorf("Location = %s", loc)
	}
}

func TestOAuthCallbackHandler_MissingCode(t *testing.T) {
	s := store.NewStore()
	cfg := &config.Config{OAuth: config.OAuthConfig{Providers: map[string]config.OAuthProviderConfig{
		"github": {ClientID: "cid", ClientSecret: "secret"},
	}}}
	registry := oauth.NewProviderRegistry()
	handler := OAuthCallbackHandler(s, cfg, registry)
	req := httptest.NewRequest(http.MethodGet, "/api/auth/oauth/github/callback", nil)
	req.Host = "example.com"
	rr := httptest.NewRecorder()
	handler(rr, req)
	if rr.Code != http.StatusFound {
		t.Errorf("status = %d, want 302", rr.Code)
	}
	loc := rr.Header().Get("Location")
	if loc == "" {
		t.Fatal("missing Location header")
	}
	if loc[:8] != "https://" && loc[:7] != "http://" {
		t.Errorf("Location = %s", loc)
	}
}

func TestRedirectBase(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.com/foo", nil)
	base := redirectBase(req)
	if base != "http://example.com" {
		t.Errorf("redirectBase() = %s", base)
	}
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("X-Forwarded-Host", "app.example.com")
	base = redirectBase(req)
	if base != "https://app.example.com" {
		t.Errorf("redirectBase() with headers = %s", base)
	}
}
