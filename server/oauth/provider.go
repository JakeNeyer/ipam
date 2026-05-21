package oauth

import (
	"context"

	"github.com/JakeNeyer/ipam/server/config"
	"golang.org/x/oauth2"
)

type ProviderRegistry struct {
	endpoints map[string]oauth2.Endpoint
	userInfos map[string]UserInfoFetcher
}

type UserInfoFetcher interface {
	FetchUser(ctx context.Context, token *oauth2.Token) (providerUserID, email string, err error)
}

func NewProviderRegistry(cfg *config.Config) *ProviderRegistry {
	r := &ProviderRegistry{
		endpoints: make(map[string]oauth2.Endpoint),
		userInfos: make(map[string]UserInfoFetcher),
	}
	if cfg == nil || cfg.OAuth.Providers == nil {
		return r
	}
	for id, pc := range cfg.OAuth.Providers {
		id = config.NormalizeOAuthProviderID(id)
		if !pc.Enabled() {
			continue
		}
		r.Register(id, oauth2.Endpoint{
			AuthURL:  pc.AuthURL,
			TokenURL: pc.TokenURL,
		}, newOAuthUserInfoFromConfig(pc))
	}
	return r
}

func (r *ProviderRegistry) Register(providerID string, endpoint oauth2.Endpoint, fetcher UserInfoFetcher) {
	r.endpoints[providerID] = endpoint
	r.userInfos[providerID] = fetcher
}

func (r *ProviderRegistry) Endpoint(providerID string) (oauth2.Endpoint, bool) {
	e, ok := r.endpoints[providerID]
	return e, ok
}

func (r *ProviderRegistry) UserInfo(ctx context.Context, providerID string, token *oauth2.Token) (providerUserID, email string, err error) {
	f, ok := r.userInfos[providerID]
	if !ok || f == nil {
		return "", "", nil
	}
	return f.FetchUser(ctx, token)
}
