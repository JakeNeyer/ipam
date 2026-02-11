package oauth

import (
	"context"

	"golang.org/x/oauth2"
)

// ProviderRegistry provides OAuth endpoint and user-info fetch for known providers.
type ProviderRegistry struct {
	endpoints  map[string]oauth2.Endpoint
	userInfos  map[string]UserInfoFetcher
}

// UserInfoFetcher fetches provider user id and email from an OAuth token.
type UserInfoFetcher interface {
	FetchUser(ctx context.Context, token *oauth2.Token) (providerUserID, email string, err error)
}

// NewProviderRegistry returns a registry with built-in providers (e.g. github).
func NewProviderRegistry() *ProviderRegistry {
	r := &ProviderRegistry{
		endpoints: make(map[string]oauth2.Endpoint),
		userInfos: make(map[string]UserInfoFetcher),
	}
	RegisterGitHub(r)
	return r
}

// Register adds a provider's endpoint and user-info fetcher.
func (r *ProviderRegistry) Register(providerID string, endpoint oauth2.Endpoint, fetcher UserInfoFetcher) {
	r.endpoints[providerID] = endpoint
	r.userInfos[providerID] = fetcher
}

// Endpoint returns the OAuth2 endpoint for the provider, or false if unknown.
func (r *ProviderRegistry) Endpoint(providerID string) (oauth2.Endpoint, bool) {
	e, ok := r.endpoints[providerID]
	return e, ok
}

// UserInfo fetches provider user id and email using the token. Returns ("", "", err) if provider unknown or fetch fails.
func (r *ProviderRegistry) UserInfo(ctx context.Context, providerID string, token *oauth2.Token) (providerUserID, email string, err error) {
	f, ok := r.userInfos[providerID]
	if !ok || f == nil {
		return "", "", nil
	}
	return f.FetchUser(ctx, token)
}
