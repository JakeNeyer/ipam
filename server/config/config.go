package config

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
