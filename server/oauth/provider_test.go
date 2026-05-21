package oauth

import (
	"context"
	"testing"

	"github.com/JakeNeyer/ipam/server/config"
	"golang.org/x/oauth2"
)

func TestNewProviderRegistry_Empty(t *testing.T) {
	r := NewProviderRegistry(nil)
	if r == nil {
		t.Fatal("NewProviderRegistry() = nil")
	}
	_, ok := r.Endpoint("sso")
	if ok {
		t.Error("Endpoint(sso) = true for empty registry")
	}
}

func TestNewProviderRegistry_FromConfig(t *testing.T) {
	cfg := &config.Config{OAuth: config.OAuthConfig{Providers: map[string]config.OAuthProviderConfig{
		"sso": {
			ClientID: "c", ClientSecret: "s",
			AuthURL: "https://idp/auth", TokenURL: "https://idp/token", UserInfoURL: "https://idp/userinfo",
		},
	}}}
	r := NewProviderRegistry(cfg)
	e, ok := r.Endpoint("sso")
	if !ok {
		t.Fatal("Endpoint(\"sso\") = false")
	}
	if e.AuthURL != "https://idp/auth" || e.TokenURL != "https://idp/token" {
		t.Errorf("Endpoint(\"sso\") = %+v", e)
	}
}

func TestProviderRegistry_UserInfo_UnknownProvider(t *testing.T) {
	r := NewProviderRegistry(nil)
	id, email, err := r.UserInfo(context.Background(), "unknown", &oauth2.Token{})
	if err != nil {
		t.Errorf("UserInfo(unknown) err = %v", err)
	}
	if id != "" || email != "" {
		t.Errorf("UserInfo(unknown) = %q, %q", id, email)
	}
}

func TestProviderRegistry_Register(t *testing.T) {
	r := &ProviderRegistry{
		endpoints: make(map[string]oauth2.Endpoint),
		userInfos: make(map[string]UserInfoFetcher),
	}
	e := oauth2.Endpoint{AuthURL: "https://auth.example.com", TokenURL: "https://token.example.com"}
	r.Register("custom", e, nil)
	got, ok := r.Endpoint("custom")
	if !ok {
		t.Fatal("Endpoint(\"custom\") = false")
	}
	if got.AuthURL != e.AuthURL || got.TokenURL != e.TokenURL {
		t.Errorf("Endpoint(\"custom\") = %+v", got)
	}
}
