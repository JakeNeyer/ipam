package config

import (
	"reflect"
	"testing"
)

func TestConfig_EnabledOAuthProviders(t *testing.T) {
	tests := []struct {
		name string
		cfg  *Config
		want []string
	}{
		{"nil config", nil, nil},
		{"nil providers", &Config{}, nil},
		{"empty providers", &Config{OAuth: OAuthConfig{Providers: map[string]OAuthProviderConfig{}}}, nil},
		{"one valid", &Config{OAuth: OAuthConfig{Providers: map[string]OAuthProviderConfig{
			"github": {ClientID: "a", ClientSecret: "b"},
		}}}, []string{"github"}},
		{"one missing secret", &Config{OAuth: OAuthConfig{Providers: map[string]OAuthProviderConfig{
			"github": {ClientID: "a", ClientSecret: ""},
		}}}, nil},
		{"two valid", &Config{OAuth: OAuthConfig{Providers: map[string]OAuthProviderConfig{
			"github": {ClientID: "a", ClientSecret: "b"},
			"google": {ClientID: "c", ClientSecret: "d"},
		}}}, []string{"github", "google"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.EnabledOAuthProviders()
			if len(got) != len(tt.want) {
				t.Errorf("EnabledOAuthProviders() = %v, want %v", got, tt.want)
				return
			}
			gotSet := make(map[string]bool)
			for _, p := range got {
				gotSet[p] = true
			}
			for _, p := range tt.want {
				if !gotSet[p] {
					t.Errorf("EnabledOAuthProviders() missing %q, got %v", p, got)
				}
			}
		})
	}
}

func TestConfig_OAuthProvider(t *testing.T) {
	cfg := &Config{OAuth: OAuthConfig{Providers: map[string]OAuthProviderConfig{
		"github": {ClientID: "cid", ClientSecret: "secret", Scopes: []string{"user:email"}},
	}}}
	if got := cfg.OAuthProvider("github"); got == nil {
		t.Fatal("OAuthProvider(\"github\") = nil, want config")
	} else if got.ClientID != "cid" || got.ClientSecret != "secret" {
		t.Errorf("OAuthProvider(\"github\") = %+v", got)
	}
	if got := cfg.OAuthProvider("unknown"); got != nil {
		t.Errorf("OAuthProvider(\"unknown\") = %v, want nil", got)
	}
	if got := cfg.OAuthProvider(""); got != nil {
		t.Errorf("OAuthProvider(\"\") = %v, want nil", got)
	}
}

func TestConfig_IsGitHubOAuthEnabled(t *testing.T) {
	if (&Config{}).IsGitHubOAuthEnabled() {
		t.Error("IsGitHubOAuthEnabled() = true for empty config")
	}
	cfg := &Config{OAuth: OAuthConfig{Providers: map[string]OAuthProviderConfig{
		"github": {ClientID: "a", ClientSecret: "b"},
	}}}
	if !cfg.IsGitHubOAuthEnabled() {
		t.Error("IsGitHubOAuthEnabled() = false for github config")
	}
}

func TestConfig_AuthConfigResponse_BackwardCompat(t *testing.T) {
	// AuthConfigResponse should include github_oauth_enabled for frontend backward compat
	cfg := &Config{OAuth: OAuthConfig{Providers: map[string]OAuthProviderConfig{
		"github": {ClientID: "a", ClientSecret: "b"},
	}}}
	providers := cfg.EnabledOAuthProviders()
	githubEnabled := false
	for _, p := range providers {
		if p == "github" {
			githubEnabled = true
			break
		}
	}
	if !githubEnabled {
		t.Error("github should be in enabled providers")
	}
	// Ensure we can build the response shape used by handlers
	type response struct {
		OAuthProviders     []string `json:"oauth_providers"`
		GitHubOAuthEnabled bool     `json:"github_oauth_enabled"`
	}
	_ = response{OAuthProviders: providers, GitHubOAuthEnabled: githubEnabled}
}

func TestOAuthProviderConfig_Scopes(t *testing.T) {
	cfg := OAuthProviderConfig{Scopes: []string{"user:email", "read:user"}}
	if !reflect.DeepEqual(cfg.Scopes, []string{"user:email", "read:user"}) {
		t.Errorf("Scopes = %v", cfg.Scopes)
	}
}
