package config

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

type Config struct {
	OAuth OAuthConfig
	// AppOrigin is the public URL of the frontend (e.g. http://localhost:5173). When set, invite URLs and OAuth redirects use it; non-API requests to this server return 401 Unauthorized.
	AppOrigin string
}

type OAuthConfig struct {
	Providers map[string]OAuthProviderConfig
	TLSConfig OAuthTLSConfig
}

type OAuthProviderConfig struct {
	ClientID            string
	ClientSecret        string // #nosec G117 -- OAuth client secret from config, not logged
	Scopes              []string
	AuthURL             string
	TokenURL            string
	UserInfoURL         string
	UserIDClaim         string // JSON claim for provider user id; default "sub"
	EmailClaim          string // JSON claim for email; default "email"
	EmailsURL           string // Optional second URL returning a JSON array when email is not on userinfo (e.g. GitHub /user/emails)
	EmailsPrimaryClaim  string // Claim on array entries marking the preferred email; default "primary"
	EmailVerifiedClaim  string // Userinfo claim that must be true when email comes from userinfo (e.g. email_verified)
	EmailsVerifiedClaim string // Emails-list entry claim that must be true (e.g. verified on GitHub /user/emails)
	AllowEmailMatch     bool   // Allow signing in to an existing account by email alone; default false
	DisplayName         string // Login/signup button label; default derived from provider id
}

type OAuthTLSConfig struct {
	TLSCertFile string // Path to TLS Certificate file
	TLSKeyFile  string // Path to TLS Key file
	TLSVersion  uint16 // e.g. tls.VersionTLS12
}

func NormalizeOAuthProviderID(id string) string {
	return strings.ToLower(strings.TrimSpace(id))
}

func (p OAuthProviderConfig) hasCredentials() bool {
	return strings.TrimSpace(p.ClientID) != "" && strings.TrimSpace(p.ClientSecret) != ""
}

func (p OAuthProviderConfig) hasEndpoints() bool {
	return strings.TrimSpace(p.AuthURL) != "" &&
		strings.TrimSpace(p.TokenURL) != "" &&
		strings.TrimSpace(p.UserInfoURL) != ""
}

func (p OAuthProviderConfig) Enabled() bool {
	return p.hasCredentials() && p.hasEndpoints()
}

// EnabledOAuthProviders returns the list of provider IDs that are configured and enabled.
func (c *Config) EnabledOAuthProviders() []string {
	if c == nil || c.OAuth.Providers == nil {
		return nil
	}
	var out []string
	for id, p := range c.OAuth.Providers {
		id = NormalizeOAuthProviderID(id)
		if p.Enabled() {
			out = append(out, id)
		}
	}
	return out
}

// OAuthProvider returns the config for a provider ID, or nil if not configured.
func (c *Config) OAuthProvider(providerID string) *OAuthProviderConfig {
	if c == nil || c.OAuth.Providers == nil {
		return nil
	}
	providerID = NormalizeOAuthProviderID(providerID)
	p, ok := c.OAuth.Providers[providerID]
	if !ok || !p.Enabled() {
		return nil
	}
	return &p
}

// DefaultOAuthDisplayName returns the login button label for a provider.
func DefaultOAuthDisplayName(providerID string) string {
	providerID = NormalizeOAuthProviderID(providerID)
	if providerID == "" {
		return "Sign in"
	}
	return "Sign in with " + strings.ToUpper(providerID[:1]) + providerID[1:]
}

// BuildOAuthHTTPClient builds the http client and configures TLS if enabled
func (c *Config) BuildOAuthHTTPClient() (*http.Client, error) {
	defaultClient := &http.Client{Timeout: 15 * time.Second}
	if c == nil || c.OAuth.TLSConfig.TLSCertFile == "" || c.OAuth.TLSConfig.TLSKeyFile == "" {
		return defaultClient, nil
	}

	cert, err := tls.LoadX509KeyPair(
		c.OAuth.TLSConfig.TLSCertFile,
		c.OAuth.TLSConfig.TLSKeyFile,
	)
	if err != nil {
		err := fmt.Errorf("failed to load OAuth TLS cert %v", err)
		return defaultClient, err
	}

	tlsConfig := &tls.Config{
		MinVersion:   c.OAuth.TLSConfig.TLSVersion,
		Certificates: []tls.Certificate{cert},
	}

	return &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}, nil
}

func LoadFromEnv() *Config {
	var cfg Config
	providers := make(map[string]OAuthProviderConfig)

	for _, id := range parseOAuthProviderList(os.Getenv("OAUTH_PROVIDERS")) {
		p := loadOAuthProviderFromEnv(id)
		if !p.Enabled() {
			continue
		}
		if existing, ok := providers[id]; ok {
			providers[id] = mergeOAuthProvider(existing, p)
		} else {
			providers[id] = p
		}
	}

	if len(providers) > 0 {
		cfg.OAuth = OAuthConfig{Providers: providers}
	}
	if origin := strings.TrimSpace(os.Getenv("APP_ORIGIN")); origin != "" {
		cfg.AppOrigin = origin
	}
	if enableTLS := strings.ToLower(strings.TrimSpace(os.Getenv("OAUTH_TLS_ENABLED"))); enableTLS == "true" {
		var tlsConfig = &OAuthTLSConfig{
			TLSVersion: tls.VersionTLS12,
		}
		if tlsCertFile := strings.TrimSpace(os.Getenv("OAUTH_TLS_CERT_FILE")); tlsCertFile != "" {
			tlsConfig.TLSCertFile = tlsCertFile
		}
		if tlsKeyFile := strings.TrimSpace(os.Getenv("OAUTH_TLS_KEY_FILE")); tlsKeyFile != "" {
			tlsConfig.TLSKeyFile = tlsKeyFile
		}
		if tlsVersion := strings.TrimSpace(os.Getenv("OAUTH_TLS_VERSION")); tlsVersion != "" {
			switch tlsVersion {
			case "1.2":
				tlsConfig.TLSVersion = tls.VersionTLS12
			case "1.3":
				tlsConfig.TLSVersion = tls.VersionTLS13
			default:
				tlsConfig.TLSVersion = tls.VersionTLS12
			}
		}
		cfg.OAuth.TLSConfig = *tlsConfig
	}
	return &cfg
}

func parseOAuthProviderList(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := make(map[string]bool)
	for _, part := range parts {
		id := NormalizeOAuthProviderID(part)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

func oauthEnvPrefix(providerID string) string {
	return "OAUTH_" + strings.ToUpper(strings.ReplaceAll(providerID, "-", "_")) + "_"
}

func loadOAuthProviderFromEnv(providerID string) OAuthProviderConfig {
	prefix := oauthEnvPrefix(providerID)
	p := OAuthProviderConfig{
		ClientID:            strings.TrimSpace(os.Getenv(prefix + "CLIENT_ID")),
		ClientSecret:        strings.TrimSpace(os.Getenv(prefix + "CLIENT_SECRET")),
		AuthURL:             strings.TrimSpace(os.Getenv(prefix + "AUTH_URL")),
		TokenURL:            strings.TrimSpace(os.Getenv(prefix + "TOKEN_URL")),
		UserInfoURL:         strings.TrimSpace(os.Getenv(prefix + "USERINFO_URL")),
		UserIDClaim:         strings.TrimSpace(os.Getenv(prefix + "USER_ID_CLAIM")),
		EmailClaim:          strings.TrimSpace(os.Getenv(prefix + "EMAIL_CLAIM")),
		EmailsURL:           strings.TrimSpace(os.Getenv(prefix + "EMAILS_URL")),
		EmailsPrimaryClaim:  strings.TrimSpace(os.Getenv(prefix + "EMAILS_PRIMARY_CLAIM")),
		EmailVerifiedClaim:  strings.TrimSpace(os.Getenv(prefix + "EMAIL_VERIFIED_CLAIM")),
		EmailsVerifiedClaim: strings.TrimSpace(os.Getenv(prefix + "EMAILS_VERIFIED_CLAIM")),
		AllowEmailMatch:     envBoolDefault(prefix+"ALLOW_EMAIL_MATCH", false),
		DisplayName:         strings.TrimSpace(os.Getenv(prefix + "DISPLAY_NAME")),
	}
	if scopes := strings.TrimSpace(os.Getenv(prefix + "SCOPES")); scopes != "" {
		p.Scopes = splitScopes(scopes)
	}
	return p
}

func mergeOAuthProvider(base, overlay OAuthProviderConfig) OAuthProviderConfig {
	if overlay.ClientID != "" {
		base.ClientID = overlay.ClientID
	}
	if overlay.ClientSecret != "" {
		base.ClientSecret = overlay.ClientSecret
	}
	if len(overlay.Scopes) > 0 {
		base.Scopes = overlay.Scopes
	}
	if overlay.AuthURL != "" {
		base.AuthURL = overlay.AuthURL
	}
	if overlay.TokenURL != "" {
		base.TokenURL = overlay.TokenURL
	}
	if overlay.UserInfoURL != "" {
		base.UserInfoURL = overlay.UserInfoURL
	}
	if overlay.UserIDClaim != "" {
		base.UserIDClaim = overlay.UserIDClaim
	}
	if overlay.EmailClaim != "" {
		base.EmailClaim = overlay.EmailClaim
	}
	if overlay.EmailsURL != "" {
		base.EmailsURL = overlay.EmailsURL
	}
	if overlay.EmailsPrimaryClaim != "" {
		base.EmailsPrimaryClaim = overlay.EmailsPrimaryClaim
	}
	if overlay.EmailVerifiedClaim != "" {
		base.EmailVerifiedClaim = overlay.EmailVerifiedClaim
	}
	if overlay.EmailsVerifiedClaim != "" {
		base.EmailsVerifiedClaim = overlay.EmailsVerifiedClaim
	}
	base.AllowEmailMatch = base.AllowEmailMatch || overlay.AllowEmailMatch
	if overlay.DisplayName != "" {
		base.DisplayName = overlay.DisplayName
	}
	return base
}

// envBoolDefault parses true/1/false/0; returns defaultVal when unset.
func envBoolDefault(key string, defaultVal bool) bool {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return defaultVal
	}
	switch strings.ToLower(raw) {
	case "1", "true", "yes":
		return true
	case "0", "false", "no":
		return false
	default:
		return defaultVal
	}
}

func splitScopes(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if s := strings.TrimSpace(part); s != "" {
			out = append(out, s)
		}
	}
	return out
}
