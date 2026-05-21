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
			"sso": {
				ClientID: "a", ClientSecret: "b",
				AuthURL: "https://idp/auth", TokenURL: "https://idp/token", UserInfoURL: "https://idp/userinfo",
			},
		}}}, []string{"sso"}},
		{"one missing secret", &Config{OAuth: OAuthConfig{Providers: map[string]OAuthProviderConfig{
			"sso": {
				ClientID: "a", ClientSecret: "",
				AuthURL: "https://idp/auth", TokenURL: "https://idp/token", UserInfoURL: "https://idp/userinfo",
			},
		}}}, nil},
		{"two valid", &Config{OAuth: OAuthConfig{Providers: map[string]OAuthProviderConfig{
			"sso": {
				ClientID: "a", ClientSecret: "b",
				AuthURL: "https://idp/auth", TokenURL: "https://idp/token", UserInfoURL: "https://idp/userinfo",
			},
			"acme": {
				ClientID: "c", ClientSecret: "d",
				AuthURL: "https://acme/auth", TokenURL: "https://acme/token", UserInfoURL: "https://acme/userinfo",
			},
		}}}, []string{"sso", "acme"}},
		{"missing endpoints", &Config{OAuth: OAuthConfig{Providers: map[string]OAuthProviderConfig{
			"sso": {ClientID: "a", ClientSecret: "b"},
		}}}, nil},
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
		"sso": {
			ClientID: "cid", ClientSecret: "secret", Scopes: []string{"openid", "email"},
			AuthURL: "https://idp/auth", TokenURL: "https://idp/token", UserInfoURL: "https://idp/userinfo",
		},
	}}}
	if got := cfg.OAuthProvider("sso"); got == nil {
		t.Fatal("OAuthProvider(sso) = nil, want config")
	} else if got.ClientID != "cid" || got.ClientSecret != "secret" {
		t.Errorf("OAuthProvider(sso) = %+v", got)
	}
	if got := cfg.OAuthProvider("unknown"); got != nil {
		t.Errorf("OAuthProvider(\"unknown\") = %v, want nil", got)
	}
}

func TestOAuthProviderConfig_Scopes(t *testing.T) {
	cfg := OAuthProviderConfig{Scopes: []string{"openid", "email"}}
	if !reflect.DeepEqual(cfg.Scopes, []string{"openid", "email"}) {
		t.Errorf("Scopes = %v", cfg.Scopes)
	}
}
