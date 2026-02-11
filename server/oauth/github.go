package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"
	gh "golang.org/x/oauth2/github"
)

const (
	githubUserURL   = "https://api.github.com/user"
	githubEmailsURL = "https://api.github.com/user/emails"
)

func RegisterGitHub(r *ProviderRegistry) {
	r.Register("github", gh.Endpoint, &githubUserInfo{client: &http.Client{Timeout: 15 * time.Second}})
}

type githubUserInfo struct {
	client *http.Client
}

type githubUserResp struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
}

type githubEmailEntry struct {
	Email   string `json:"email"`
	Primary bool   `json:"primary"`
}

func (g *githubUserInfo) FetchUser(ctx context.Context, token *oauth2.Token) (providerUserID, email string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubUserURL, nil)
	if err != nil {
		return "", "", err
	}
	token.SetAuthHeader(req)
	req.Header.Set("Accept", "application/json")
	resp, err := g.client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("github user api: %s", resp.Status)
	}
	var u githubUserResp
	if err := json.Unmarshal(data, &u); err != nil {
		return "", "", err
	}
	providerUserID = fmt.Sprintf("%d", u.ID)
	email = strings.TrimSpace(strings.ToLower(u.Email))
	if email == "" {
		email, err = g.fetchPrimaryEmail(ctx, token)
		if err != nil || email == "" {
			return providerUserID, "", err
		}
	}
	return providerUserID, email, nil
}

func (g *githubUserInfo) fetchPrimaryEmail(ctx context.Context, token *oauth2.Token) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubEmailsURL, nil)
	if err != nil {
		return "", err
	}
	token.SetAuthHeader(req)
	req.Header.Set("Accept", "application/json")
	resp, err := g.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var emails []githubEmailEntry
	if err := json.Unmarshal(data, &emails); err != nil {
		return "", err
	}
	for _, e := range emails {
		if e.Primary && e.Email != "" {
			return strings.TrimSpace(strings.ToLower(e.Email)), nil
		}
	}
	for _, e := range emails {
		if e.Email != "" {
			return strings.TrimSpace(strings.ToLower(e.Email)), nil
		}
	}
	return "", nil
}
