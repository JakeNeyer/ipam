package config

import "testing"

func TestLoadFromEnv_OIDCProvider(t *testing.T) {
	t.Setenv("OAUTH_PROVIDERS", "sso")
	t.Setenv("OAUTH_SSO_CLIENT_ID", "client")
	t.Setenv("OAUTH_SSO_CLIENT_SECRET", "secret")
	t.Setenv("OAUTH_SSO_AUTH_URL", "https://idp.example/realms/test/protocol/openid-connect/auth")
	t.Setenv("OAUTH_SSO_TOKEN_URL", "https://idp.example/realms/test/protocol/openid-connect/token")
	t.Setenv("OAUTH_SSO_USERINFO_URL", "https://idp.example/realms/test/protocol/openid-connect/userinfo")
	t.Setenv("OAUTH_SSO_SCOPES", "openid,email,profile")
	t.Setenv("OAUTH_SSO_DISPLAY_NAME", "Sign in with SSO")

	cfg := LoadFromEnv()
	p := cfg.OAuthProvider("sso")
	if p == nil {
		t.Fatal("sso provider not loaded")
	}
	if p.ClientID != "client" || p.AuthURL == "" || len(p.Scopes) != 3 {
		t.Errorf("provider = %+v", p)
	}
	if p.DisplayName != "Sign in with SSO" {
		t.Errorf("DisplayName = %q", p.DisplayName)
	}
}

func TestOAuthProviderConfig_Enabled(t *testing.T) {
	p := OAuthProviderConfig{
		ClientID: "x", ClientSecret: "y",
		AuthURL: "a", TokenURL: "b", UserInfoURL: "c",
	}
	if !p.Enabled() {
		t.Error("Enabled() = false")
	}
	if (OAuthProviderConfig{ClientID: "x", ClientSecret: "y"}).Enabled() {
		t.Error("Enabled() = true without endpoints")
	}
}

func TestNormalizeOAuthProviderID(t *testing.T) {
	if got := NormalizeOAuthProviderID(" SSO "); got != "sso" {
		t.Errorf("NormalizeOAuthProviderID = %q", got)
	}
}
