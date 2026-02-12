package config

import (
	"os"
	"strings"
)

// Config holds optional server configuration (e.g. OAuth, app origin).
type Config struct {
	// OAuth holds OAuth provider configs. Key is provider ID (e.g. "github"). When present and valid, that provider is enabled.
	OAuth OAuthConfig
	// AppOrigin is the public URL of the frontend (e.g. http://localhost:5173). When set, invite URLs and OAuth redirects use it; non-API requests to this server return 401 Unauthorized.
	AppOrigin string
}

// OAuthConfig holds per-provider OAuth settings.
type OAuthConfig struct {
	Providers map[string]OAuthProviderConfig
}

// OAuthProviderConfig is the config for one OAuth provider (e.g. GitHub).
type OAuthProviderConfig struct {
	ClientID     string
	ClientSecret string
	Scopes       []string
}

// EnabledOAuthProviders returns the list of provider IDs that are configured and enabled.
func (c *Config) EnabledOAuthProviders() []string {
	if c == nil || c.OAuth.Providers == nil {
		return nil
	}
	var out []string
	for id, p := range c.OAuth.Providers {
		if p.ClientID != "" && p.ClientSecret != "" {
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
	p, ok := c.OAuth.Providers[providerID]
	if !ok || p.ClientID == "" || p.ClientSecret == "" {
		return nil
	}
	return &p
}

// IsGitHubOAuthEnabled returns true if GitHub OAuth is configured (for backward compatibility with frontend).
func (c *Config) IsGitHubOAuthEnabled() bool {
	return c.OAuthProvider("github") != nil
}

// LoadFromEnv returns Config from environment variables.
// ENABLE_GITHUB_OAUTH (true/1), GITHUB_CLIENT_ID, GITHUB_CLIENT_SECRET enable GitHub OAuth.
// APP_ORIGIN sets the public app URL (OAuth redirects, signup links).
func LoadFromEnv() *Config {
	enabled := strings.ToLower(strings.TrimSpace(os.Getenv("ENABLE_GITHUB_OAUTH"))) == "true" ||
		strings.TrimSpace(os.Getenv("ENABLE_GITHUB_OAUTH")) == "1"
	clientID := strings.TrimSpace(os.Getenv("GITHUB_CLIENT_ID"))
	clientSecret := strings.TrimSpace(os.Getenv("GITHUB_CLIENT_SECRET"))
	var cfg Config
	if enabled && clientID != "" && clientSecret != "" {
		cfg.OAuth = OAuthConfig{
			Providers: map[string]OAuthProviderConfig{
				"github": {
					ClientID:     clientID,
					ClientSecret: clientSecret,
					Scopes:       []string{"user:email"},
				},
			},
		}
	}
	if origin := strings.TrimSpace(os.Getenv("APP_ORIGIN")); origin != "" {
		cfg.AppOrigin = origin
	}
	return &cfg
}
