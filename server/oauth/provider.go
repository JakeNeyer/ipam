package oauth

import (
	"context"
	"net/http"

	"github.com/JakeNeyer/ipam/server/config"
	"golang.org/x/oauth2"
)

type ProviderRegistry struct {
	endpoints map[string]oauth2.Endpoint
	userInfos map[string]UserInfoFetcher
	clients   map[string]*http.Client
}

type UserInfoFetcher interface {
	FetchUser(ctx context.Context, token *oauth2.Token) (*UserProfile, error)
}

func NewProviderRegistry(cfg *config.Config) (*ProviderRegistry, error) {
	r := &ProviderRegistry{
		endpoints: make(map[string]oauth2.Endpoint),
		userInfos: make(map[string]UserInfoFetcher),
		clients:   make(map[string]*http.Client),
	}
	if cfg == nil || cfg.OAuth.Providers == nil {
		return r, nil
	}
	for id, pc := range cfg.OAuth.Providers {
		id = config.NormalizeOAuthProviderID(id)
		if !pc.Enabled() {
			continue
		}

		httpClient, err := pc.BuildHTTPClient()
		if err != nil {
			return nil, err
		}

		r.Register(id, oauth2.Endpoint{
			AuthURL:  pc.AuthURL,
			TokenURL: pc.TokenURL,
		}, newOAuthUserInfoFromConfig(pc, httpClient), httpClient)
	}
	return r, nil
}

func (r *ProviderRegistry) HTTPClient(providerID string) *http.Client {
	return r.clients[config.NormalizeOAuthProviderID(providerID)]
}

func (r *ProviderRegistry) Register(providerID string, endpoint oauth2.Endpoint, fetcher UserInfoFetcher, client *http.Client) {
	r.endpoints[providerID] = endpoint
	r.userInfos[providerID] = fetcher
	r.clients[providerID] = client
}

func (r *ProviderRegistry) Endpoint(providerID string) (oauth2.Endpoint, bool) {
	e, ok := r.endpoints[providerID]
	return e, ok
}

func (r *ProviderRegistry) UserInfo(ctx context.Context, providerID string, token *oauth2.Token) (*UserProfile, error) {
	f, ok := r.userInfos[providerID]
	if !ok || f == nil {
		return nil, nil
	}
	return f.FetchUser(ctx, token)
}
