package handlers

import (
	"encoding/base64"
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
	"github.com/google/uuid"
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

func TestOAuthCallbackHandler_JITCreatesUser(t *testing.T) {
	org := &store.Organization{Name: "Acme"}
	s := store.NewStore()
	if err := s.CreateOrganization(org); err != nil {
		t.Fatal(err)
	}
	idp := newOAuthTestIdP(t, map[string]any{
		"sub":            "user-1",
		"email":          "jit@example.com",
		"email_verified": true,
		"groups":         []any{"ipam-admins", "team-network"},
	}, "")
	defer idp.Close()

	pc := oauthTestProvider()
	pc.TokenURL = idp.URL + "/token"
	pc.UserInfoURL = idp.URL + "/userinfo"
	pc.JITEnabled = true
	pc.JITDefaultOrganization = "Acme"
	pc.RoleClaim = "groups"
	pc.RoleMap = []config.ClaimMapEntry{{Value: "ipam-admins", Mapped: "admin"}}
	pc.OrgClaim = "groups"
	pc.OrgMap = []config.ClaimMapEntry{{Value: "team-network", Mapped: "Acme"}}

	cfg := &config.Config{OAuth: config.OAuthConfig{Providers: map[string]config.OAuthProviderConfig{"sso": pc}}}
	registry, err := oauth.NewProviderRegistry(cfg)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	OAuthCallbackHandler(s, cfg, registry).ServeHTTP(rr, oauthCallbackRequest(t, "sso"))
	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, location = %s", rr.Code, rr.Header().Get("Location"))
	}
	if strings.Contains(rr.Header().Get("Location"), "error=") {
		t.Fatalf("Location = %s", rr.Header().Get("Location"))
	}
	user, err := s.GetUserByEmail("jit@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if user.Role != store.RoleAdmin {
		t.Errorf("role = %q, want admin", user.Role)
	}
	if user.OrganizationID != org.ID {
		t.Errorf("org = %v, want %v", user.OrganizationID, org.ID)
	}
	if user.OAuthProvider != "sso" || user.OAuthProviderUserID != "user-1" {
		t.Errorf("oauth = %q %q", user.OAuthProvider, user.OAuthProviderUserID)
	}
}

func TestOAuthCallbackHandler_JITDefaultOrgWhenClaimMissing(t *testing.T) {
	org := &store.Organization{Name: "Acme"}
	s := store.NewStore()
	if err := s.CreateOrganization(org); err != nil {
		t.Fatal(err)
	}
	idp := newOAuthTestIdP(t, map[string]any{
		"sub":            "user-2",
		"email":          "default@example.com",
		"email_verified": true,
	}, "")
	defer idp.Close()

	pc := oauthTestProvider()
	pc.TokenURL = idp.URL + "/token"
	pc.UserInfoURL = idp.URL + "/userinfo"
	pc.JITEnabled = true
	pc.JITDefaultOrganization = "Acme"
	pc.OrgClaim = "groups"
	pc.OrgMap = []config.ClaimMapEntry{{Value: "team-network", Mapped: "Acme"}}

	cfg := &config.Config{OAuth: config.OAuthConfig{Providers: map[string]config.OAuthProviderConfig{"sso": pc}}}
	registry, err := oauth.NewProviderRegistry(cfg)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	OAuthCallbackHandler(s, cfg, registry).ServeHTTP(rr, oauthCallbackRequest(t, "sso"))
	if strings.Contains(rr.Header().Get("Location"), "error=") {
		t.Fatalf("Location = %s", rr.Header().Get("Location"))
	}
	user, err := s.GetUserByEmail("default@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if user.Role != store.RoleUser {
		t.Errorf("role = %q, want user", user.Role)
	}
	if user.OrganizationID != org.ID {
		t.Errorf("org = %v, want %v", user.OrganizationID, org.ID)
	}
}

func TestOAuthCallbackHandler_JITDeniesUnmappedOrgClaim(t *testing.T) {
	s := store.NewStore()
	if err := s.CreateOrganization(&store.Organization{Name: "Acme"}); err != nil {
		t.Fatal(err)
	}
	idp := newOAuthTestIdP(t, map[string]any{
		"sub":            "user-3",
		"email":          "unknown@example.com",
		"email_verified": true,
		"groups":         []any{"other-team"},
	}, "")
	defer idp.Close()

	pc := oauthTestProvider()
	pc.TokenURL = idp.URL + "/token"
	pc.UserInfoURL = idp.URL + "/userinfo"
	pc.JITEnabled = true
	pc.JITDefaultOrganization = "Acme"
	pc.OrgClaim = "groups"
	pc.OrgMap = []config.ClaimMapEntry{{Value: "team-network", Mapped: "Acme"}}

	cfg := &config.Config{OAuth: config.OAuthConfig{Providers: map[string]config.OAuthProviderConfig{"sso": pc}}}
	registry, err := oauth.NewProviderRegistry(cfg)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	OAuthCallbackHandler(s, cfg, registry).ServeHTTP(rr, oauthCallbackRequest(t, "sso"))
	loc := rr.Header().Get("Location")
	if !strings.Contains(loc, "error=") {
		t.Fatalf("Location = %s, want error", loc)
	}
	if _, err := s.GetUserByEmail("unknown@example.com"); err == nil {
		t.Fatal("user should not be created")
	}
}

func TestOAuthCallbackHandler_JITDisabledRequiresInvite(t *testing.T) {
	s := store.NewStore()
	idp := newOAuthTestIdP(t, map[string]any{
		"sub":            "user-4",
		"email":          "nojit@example.com",
		"email_verified": true,
	}, "")
	defer idp.Close()

	pc := oauthTestProvider()
	pc.TokenURL = idp.URL + "/token"
	pc.UserInfoURL = idp.URL + "/userinfo"

	cfg := &config.Config{OAuth: config.OAuthConfig{Providers: map[string]config.OAuthProviderConfig{"sso": pc}}}
	registry, err := oauth.NewProviderRegistry(cfg)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	OAuthCallbackHandler(s, cfg, registry).ServeHTTP(rr, oauthCallbackRequest(t, "sso"))
	loc := rr.Header().Get("Location")
	if !strings.Contains(loc, url.QueryEscape("Use a signup link or ask an admin to create your account")) &&
		!strings.Contains(loc, "signup") {
		t.Errorf("Location = %s", loc)
	}
	if _, err := s.GetUserByEmail("nojit@example.com"); err == nil {
		t.Fatal("user should not be created")
	}
}

func TestOAuthCallbackHandler_JITDoesNotBindUnrelatedFirstAdmin(t *testing.T) {
	org := &store.Organization{Name: "Acme"}
	s := store.NewStore()
	if err := s.CreateOrganization(org); err != nil {
		t.Fatal(err)
	}
	admin := &store.User{
		Email:          "admin@ipam.local",
		Role:           store.RoleAdmin,
		OrganizationID: uuid.Nil,
	}
	if err := s.CreateUser(admin); err != nil {
		t.Fatal(err)
	}
	idp := newOAuthTestIdP(t, map[string]any{
		"sub":            "alice-sub",
		"email":          "alice@acme.com",
		"email_verified": true,
		"groups":         []any{"ipam-admins", "team-network"},
	}, "")
	defer idp.Close()

	pc := oauthTestProvider()
	pc.TokenURL = idp.URL + "/token"
	pc.UserInfoURL = idp.URL + "/userinfo"
	pc.JITEnabled = true
	pc.JITDefaultOrganization = "Acme"
	pc.RoleClaim = "groups"
	pc.RoleMap = []config.ClaimMapEntry{{Value: "ipam-admins", Mapped: "admin"}}
	pc.OrgClaim = "groups"
	pc.OrgMap = []config.ClaimMapEntry{{Value: "team-network", Mapped: "Acme"}}

	cfg := &config.Config{OAuth: config.OAuthConfig{Providers: map[string]config.OAuthProviderConfig{"sso": pc}}}
	registry, err := oauth.NewProviderRegistry(cfg)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	OAuthCallbackHandler(s, cfg, registry).ServeHTTP(rr, oauthCallbackRequest(t, "sso"))
	if strings.Contains(rr.Header().Get("Location"), "error=") {
		t.Fatalf("Location = %s", rr.Header().Get("Location"))
	}
	alice, err := s.GetUserByEmail("alice@acme.com")
	if err != nil {
		t.Fatal(err)
	}
	if alice.ID == admin.ID {
		t.Fatal("alice was bound to the bootstrap global admin")
	}
	if alice.OrganizationID != org.ID {
		t.Errorf("alice org = %v, want Acme %v", alice.OrganizationID, org.ID)
	}
	if alice.Role != store.RoleAdmin {
		t.Errorf("alice role = %q, want admin", alice.Role)
	}
	reloaded, err := s.GetUser(admin.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.OAuthProviderUserID != "" {
		t.Errorf("bootstrap admin OAuth bound to %q", reloaded.OAuthProviderUserID)
	}
}

func TestOAuthCallbackHandler_ExistingUserRoleNotRemapped(t *testing.T) {
	org := &store.Organization{Name: "Acme"}
	s := store.NewStore()
	if err := s.CreateOrganization(org); err != nil {
		t.Fatal(err)
	}
	existing := &store.User{
		Email:               "existing@example.com",
		Role:                store.RoleUser,
		OrganizationID:      org.ID,
		OAuthProvider:       "sso",
		OAuthProviderUserID: "user-5",
	}
	if err := s.CreateUser(existing); err != nil {
		t.Fatal(err)
	}
	idp := newOAuthTestIdP(t, map[string]any{
		"sub":            "user-5",
		"email":          "existing@example.com",
		"email_verified": true,
		"groups":         []any{"ipam-admins"},
	}, "")
	defer idp.Close()

	pc := oauthTestProvider()
	pc.TokenURL = idp.URL + "/token"
	pc.UserInfoURL = idp.URL + "/userinfo"
	pc.JITEnabled = true
	pc.JITDefaultOrganization = "Acme"
	pc.RoleClaim = "groups"
	pc.RoleMap = []config.ClaimMapEntry{{Value: "ipam-admins", Mapped: "admin"}}

	cfg := &config.Config{OAuth: config.OAuthConfig{Providers: map[string]config.OAuthProviderConfig{"sso": pc}}}
	registry, err := oauth.NewProviderRegistry(cfg)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	OAuthCallbackHandler(s, cfg, registry).ServeHTTP(rr, oauthCallbackRequest(t, "sso"))
	if strings.Contains(rr.Header().Get("Location"), "error=") {
		t.Fatalf("Location = %s", rr.Header().Get("Location"))
	}
	user, err := s.GetUserByEmail("existing@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if user.Role != store.RoleUser {
		t.Errorf("role = %q, want user (unchanged)", user.Role)
	}
}

func TestOAuthCallbackHandler_JITUsesIDTokenClaims(t *testing.T) {
	org := &store.Organization{Name: "Acme"}
	s := store.NewStore()
	if err := s.CreateOrganization(org); err != nil {
		t.Fatal(err)
	}
	idToken := fakeHandlerIDToken(map[string]any{
		"groups": []any{"ipam-admins"},
	})
	idp := newOAuthTestIdP(t, map[string]any{
		"sub":            "user-6",
		"email":          "jwt@example.com",
		"email_verified": true,
	}, idToken)
	defer idp.Close()

	pc := oauthTestProvider()
	pc.TokenURL = idp.URL + "/token"
	pc.UserInfoURL = idp.URL + "/userinfo"
	pc.JITEnabled = true
	pc.JITDefaultOrganization = "Acme"
	pc.RoleClaim = "groups"
	pc.RoleMap = []config.ClaimMapEntry{{Value: "ipam-admins", Mapped: "admin"}}

	cfg := &config.Config{OAuth: config.OAuthConfig{Providers: map[string]config.OAuthProviderConfig{"sso": pc}}}
	registry, err := oauth.NewProviderRegistry(cfg)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	OAuthCallbackHandler(s, cfg, registry).ServeHTTP(rr, oauthCallbackRequest(t, "sso"))
	if strings.Contains(rr.Header().Get("Location"), "error=") {
		t.Fatalf("Location = %s", rr.Header().Get("Location"))
	}
	user, err := s.GetUserByEmail("jwt@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if user.Role != store.RoleAdmin {
		t.Errorf("role = %q, want admin from id_token", user.Role)
	}
}

func TestOAuthCallbackHandler_JITDomainMapAssignsOrg(t *testing.T) {
	org := &store.Organization{Name: "Acme"}
	s := store.NewStore()
	if err := s.CreateOrganization(org); err != nil {
		t.Fatal(err)
	}
	idp := newOAuthTestIdP(t, map[string]any{
		"sub":            "user-7",
		"email":          "jane@acme.com",
		"email_verified": true,
	}, "")
	defer idp.Close()

	pc := oauthTestProvider()
	pc.TokenURL = idp.URL + "/token"
	pc.UserInfoURL = idp.URL + "/userinfo"
	pc.JITEnabled = true
	pc.JITDomainOrgMap = []config.ClaimMapEntry{{Value: "acme.com", Mapped: "Acme"}}

	cfg := &config.Config{OAuth: config.OAuthConfig{Providers: map[string]config.OAuthProviderConfig{"sso": pc}}}
	registry, err := oauth.NewProviderRegistry(cfg)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	OAuthCallbackHandler(s, cfg, registry).ServeHTTP(rr, oauthCallbackRequest(t, "sso"))
	if strings.Contains(rr.Header().Get("Location"), "error=") {
		t.Fatalf("Location = %s", rr.Header().Get("Location"))
	}
	user, err := s.GetUserByEmail("jane@acme.com")
	if err != nil {
		t.Fatal(err)
	}
	if user.OrganizationID != org.ID {
		t.Errorf("org = %v, want %v", user.OrganizationID, org.ID)
	}
}

func TestOAuthCallbackHandler_JITDomainAllowlistDenies(t *testing.T) {
	s := store.NewStore()
	if err := s.CreateOrganization(&store.Organization{Name: "Acme"}); err != nil {
		t.Fatal(err)
	}
	idp := newOAuthTestIdP(t, map[string]any{
		"sub":            "user-8",
		"email":          "bob@gmail.com",
		"email_verified": true,
	}, "")
	defer idp.Close()

	pc := oauthTestProvider()
	pc.TokenURL = idp.URL + "/token"
	pc.UserInfoURL = idp.URL + "/userinfo"
	pc.JITEnabled = true
	pc.JITDefaultOrganization = "Acme"
	pc.JITAllowedEmailDomains = []string{"acme.com"}

	cfg := &config.Config{OAuth: config.OAuthConfig{Providers: map[string]config.OAuthProviderConfig{"sso": pc}}}
	registry, err := oauth.NewProviderRegistry(cfg)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	OAuthCallbackHandler(s, cfg, registry).ServeHTTP(rr, oauthCallbackRequest(t, "sso"))
	if !strings.Contains(rr.Header().Get("Location"), "error=") {
		t.Fatalf("Location = %s, want error", rr.Header().Get("Location"))
	}
	if _, err := s.GetUserByEmail("bob@gmail.com"); err == nil {
		t.Fatal("user should not be created")
	}
}

func TestLookupOrganization_ByUUID(t *testing.T) {
	s := store.NewStore()
	org := &store.Organization{Name: "Acme"}
	if err := s.CreateOrganization(org); err != nil {
		t.Fatal(err)
	}
	got, err := lookupOrganization(s, org.ID.String())
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != org.ID {
		t.Errorf("ID = %v, want %v", got.ID, org.ID)
	}
	if _, err := lookupOrganization(s, uuid.New().String()); err == nil {
		t.Fatal("expected missing UUID to fail")
	}
}

func newOAuthTestIdP(t *testing.T, userinfo map[string]any, idToken string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/token"):
			w.Header().Set("Content-Type", "application/json")
			body := map[string]any{
				"access_token": "access-token",
				"token_type":   "bearer",
			}
			if idToken != "" {
				body["id_token"] = idToken
			}
			_ = json.NewEncoder(w).Encode(body)
		default:
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(userinfo)
		}
	}))
}

func oauthCallbackRequest(t *testing.T, provider string) *http.Request {
	t.Helper()
	nonce := "test-nonce"
	req := httptest.NewRequest(http.MethodGet, "/api/auth/oauth/"+provider+"/callback?code=abc&state="+nonce, nil)
	rec := httptest.NewRecorder()
	if err := auth.SetOAuthStateCookie(rec, auth.OAuthStatePayload{Nonce: nonce, Provider: provider}, false); err != nil {
		t.Fatal(err)
	}
	for _, c := range rec.Result().Cookies() {
		req.AddCookie(c)
	}
	return req
}

func fakeHandlerIDToken(claims map[string]any) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload, err := json.Marshal(claims)
	if err != nil {
		panic(err)
	}
	return header + "." + base64.RawURLEncoding.EncodeToString(payload) + ".sig"
}
