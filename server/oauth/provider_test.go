package oauth

import (
	"context"
	"testing"

	"golang.org/x/oauth2"
)

func TestNewProviderRegistry(t *testing.T) {
	r := NewProviderRegistry()
	if r == nil {
		t.Fatal("NewProviderRegistry() = nil")
	}
}

func TestProviderRegistry_Endpoint(t *testing.T) {
	r := NewProviderRegistry()
	e, ok := r.Endpoint("github")
	if !ok {
		t.Fatal("Endpoint(\"github\") = false, want true")
	}
	if e.AuthURL == "" || e.TokenURL == "" {
		t.Errorf("Endpoint(\"github\") = %+v", e)
	}
	_, ok = r.Endpoint("unknown")
	if ok {
		t.Error("Endpoint(\"unknown\") = true, want false")
	}
}

func TestProviderRegistry_UserInfo_UnknownProvider(t *testing.T) {
	r := NewProviderRegistry()
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
