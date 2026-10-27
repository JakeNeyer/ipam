package config

import "testing"

func TestLoadFromEnv_JITAndClaimMaps(t *testing.T) {
	t.Setenv("OAUTH_PROVIDERS", "sso")
	t.Setenv("OAUTH_SSO_CLIENT_ID", "client")
	t.Setenv("OAUTH_SSO_CLIENT_SECRET", "secret")
	t.Setenv("OAUTH_SSO_AUTH_URL", "https://idp.example/realms/test/protocol/openid-connect/auth")
	t.Setenv("OAUTH_SSO_TOKEN_URL", "https://idp.example/realms/test/protocol/openid-connect/token")
	t.Setenv("OAUTH_SSO_USERINFO_URL", "https://idp.example/realms/test/protocol/openid-connect/userinfo")
	t.Setenv("OAUTH_SSO_JIT_ENABLED", "true")
	t.Setenv("OAUTH_SSO_JIT_DEFAULT_ROLE", "user")
	t.Setenv("OAUTH_SSO_JIT_DEFAULT_ORGANIZATION", "Acme")
	t.Setenv("OAUTH_SSO_ROLE_CLAIM", "groups")
	t.Setenv("OAUTH_SSO_ROLE_MAP", "ipam-admins:admin,ipam-users:user")
	t.Setenv("OAUTH_SSO_ORG_CLAIM", "groups")
	t.Setenv("OAUTH_SSO_ORG_MAP", "team-network:Network,team-platform:Platform")
	t.Setenv("OAUTH_SSO_JIT_ALLOWED_EMAIL_DOMAINS", "Acme.com, @platform.io")
	t.Setenv("OAUTH_SSO_JIT_DOMAIN_ORG_MAP", "acme.com:Acme,platform.io:Platform")

	p := LoadFromEnv().OAuthProvider("sso")
	if p == nil {
		t.Fatal("sso provider not loaded")
	}
	if !p.JITEnabled {
		t.Fatal("JITEnabled = false")
	}
	if p.JITDefaultRole != "user" || p.JITDefaultOrganization != "Acme" {
		t.Errorf("JIT defaults = %q %q", p.JITDefaultRole, p.JITDefaultOrganization)
	}
	if p.RoleClaim != "groups" || p.OrgClaim != "groups" {
		t.Errorf("claims = %q %q", p.RoleClaim, p.OrgClaim)
	}
	wantRole := []ClaimMapEntry{{Value: "ipam-admins", Mapped: "admin"}, {Value: "ipam-users", Mapped: "user"}}
	if !claimMapEqual(p.RoleMap, wantRole) {
		t.Errorf("RoleMap = %+v", p.RoleMap)
	}
	wantOrg := []ClaimMapEntry{{Value: "team-network", Mapped: "Network"}, {Value: "team-platform", Mapped: "Platform"}}
	if !claimMapEqual(p.OrgMap, wantOrg) {
		t.Errorf("OrgMap = %+v", p.OrgMap)
	}
	if !stringSliceEqual(p.JITAllowedEmailDomains, []string{"acme.com", "platform.io"}) {
		t.Errorf("JITAllowedEmailDomains = %v", p.JITAllowedEmailDomains)
	}
	wantDomain := []ClaimMapEntry{{Value: "acme.com", Mapped: "Acme"}, {Value: "platform.io", Mapped: "Platform"}}
	if !claimMapEqual(p.JITDomainOrgMap, wantDomain) {
		t.Errorf("JITDomainOrgMap = %+v", p.JITDomainOrgMap)
	}
}

func TestParseClaimMap(t *testing.T) {
	got := parseClaimMap(" ipam-admins:admin , ipam-users:user, :skip, nosep ")
	want := []ClaimMapEntry{{Value: "ipam-admins", Mapped: "admin"}, {Value: "ipam-users", Mapped: "user"}}
	if !claimMapEqual(got, want) {
		t.Errorf("parseClaimMap = %+v, want %+v", got, want)
	}
}

func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func claimMapEqual(a, b []ClaimMapEntry) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

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

func TestLoadFromEnv_Metrics(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		t.Setenv("METRICS_ENABLED", "")
		t.Setenv("METRICS_TOKEN", "")
		cfg := LoadFromEnv()
		if !cfg.Metrics.Enabled {
			t.Error("Metrics.Enabled should default to true")
		}
		if cfg.Metrics.Token != "" {
			t.Errorf("Metrics.Token = %q, want empty", cfg.Metrics.Token)
		}
	})
	t.Run("disabled and token trimmed", func(t *testing.T) {
		t.Setenv("METRICS_ENABLED", "false")
		t.Setenv("METRICS_TOKEN", "  s3cret \n")
		cfg := LoadFromEnv()
		if cfg.Metrics.Enabled {
			t.Error("Metrics.Enabled should be false")
		}
		if cfg.Metrics.Token != "s3cret" {
			t.Errorf("Metrics.Token = %q, want %q", cfg.Metrics.Token, "s3cret")
		}
	})
	t.Run("garbage keeps default", func(t *testing.T) {
		t.Setenv("METRICS_ENABLED", "maybe")
		if !LoadFromEnv().Metrics.Enabled {
			t.Error("unparseable METRICS_ENABLED should keep default true")
		}
	})
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
