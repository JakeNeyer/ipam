package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/JakeNeyer/ipam/server/config"
	"golang.org/x/oauth2"
)

const maxOAuthResponseBytes = 1 << 20 // 1 MiB

type oauthUserInfo struct {
	userInfoURL         string
	userIDClaim         string
	emailClaim          string
	emailsURL           string
	emailsPrimaryClaim  string
	emailVerifiedClaim  string
	emailsVerifiedClaim string
	client              *http.Client
}

func newOAuthUserInfoFromConfig(pc config.OAuthProviderConfig, h *http.Client) *oauthUserInfo {
	emailVerifiedClaim := pc.EmailVerifiedClaim
	if emailVerifiedClaim == "" {
		emailVerifiedClaim = "email_verified"
	}
	emailsVerifiedClaim := pc.EmailsVerifiedClaim
	if emailsVerifiedClaim == "" && strings.TrimSpace(pc.EmailsURL) != "" {
		emailsVerifiedClaim = "verified"
	}
	return newOAuthUserInfo(
		pc.UserInfoURL, pc.UserIDClaim, pc.EmailClaim,
		pc.EmailsURL, pc.EmailsPrimaryClaim,
		emailVerifiedClaim, emailsVerifiedClaim,
		h,
	)
}

func newOAuthUserInfo(
	userInfoURL, userIDClaim, emailClaim, emailsURL, emailsPrimaryClaim,
	emailVerifiedClaim, emailsVerifiedClaim string, client *http.Client,
) *oauthUserInfo {
	if userIDClaim == "" {
		userIDClaim = "sub"
	}
	if emailClaim == "" {
		emailClaim = "email"
	}
	if emailsPrimaryClaim == "" {
		emailsPrimaryClaim = "primary"
	}
	return &oauthUserInfo{
		userInfoURL:         strings.TrimSpace(userInfoURL),
		userIDClaim:         userIDClaim,
		emailClaim:          emailClaim,
		emailsURL:           strings.TrimSpace(emailsURL),
		emailsPrimaryClaim:  emailsPrimaryClaim,
		emailVerifiedClaim:  strings.TrimSpace(emailVerifiedClaim),
		emailsVerifiedClaim: strings.TrimSpace(emailsVerifiedClaim),
		client:              client,
	}
}

func (u *oauthUserInfo) FetchUser(ctx context.Context, token *oauth2.Token) (providerUserID, email string, err error) {
	claims, err := u.fetchJSON(ctx, token, u.userInfoURL)
	if err != nil {
		return "", "", err
	}
	providerUserID = strings.TrimSpace(claimString(claims, u.userIDClaim))
	email = strings.TrimSpace(strings.ToLower(claimString(claims, u.emailClaim)))
	if providerUserID == "" {
		return "", "", fmt.Errorf("missing %q in userinfo response", u.userIDClaim)
	}
	if email != "" {
		if err := u.requireVerifiedOnClaims(claims, u.emailVerifiedClaim); err != nil {
			return "", "", err
		}
	}
	if email == "" && u.emailsURL != "" {
		email, err = u.fetchEmailFromList(ctx, token)
		if err != nil {
			return providerUserID, "", err
		}
	}
	if email == "" {
		if u.emailsURL != "" {
			return providerUserID, "", fmt.Errorf("no verified email in userinfo or emails list")
		}
		return providerUserID, "", fmt.Errorf("missing %q in userinfo response", u.emailClaim)
	}
	return providerUserID, email, nil
}

func (u *oauthUserInfo) requireVerifiedOnClaims(claims map[string]any, verifiedClaim string) error {
	if verifiedClaim == "" {
		return nil
	}
	if !claimPresent(claims, verifiedClaim) {
		return fmt.Errorf("missing %q in userinfo response", verifiedClaim)
	}
	if !claimBool(claims, verifiedClaim) {
		return fmt.Errorf("email not verified (%q is false)", verifiedClaim)
	}
	return nil
}

func (u *oauthUserInfo) fetchEmailFromList(ctx context.Context, token *oauth2.Token) (string, error) {
	data, err := u.fetchBytes(ctx, token, u.emailsURL)
	if err != nil {
		return "", err
	}
	var entries []map[string]any
	if err := json.Unmarshal(data, &entries); err != nil {
		return "", err
	}
	var fallback string
	for _, entry := range entries {
		if u.emailsVerifiedClaim != "" && !claimBool(entry, u.emailsVerifiedClaim) {
			continue
		}
		addr := strings.TrimSpace(strings.ToLower(claimString(entry, u.emailClaim)))
		if addr == "" {
			continue
		}
		if claimBool(entry, u.emailsPrimaryClaim) {
			return addr, nil
		}
		if fallback == "" {
			fallback = addr
		}
	}
	return fallback, nil
}

func (u *oauthUserInfo) fetchJSON(ctx context.Context, token *oauth2.Token, url string) (map[string]any, error) {
	data, err := u.fetchBytes(ctx, token, url)
	if err != nil {
		return nil, err
	}
	var claims map[string]any
	if err := json.Unmarshal(data, &claims); err != nil {
		return nil, err
	}
	return claims, nil
}

func (u *oauthUserInfo) fetchBytes(ctx context.Context, token *oauth2.Token, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	token.SetAuthHeader(req)
	req.Header.Set("Accept", "application/json")
	// #nosec G704 -- URL is configured by the operator for this OAuth provider.
	resp, err := u.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxOAuthResponseBytes))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("oauth api %s: %s", url, resp.Status)
	}
	return data, nil
}

func claimPresent(claims map[string]any, key string) bool {
	v, ok := claims[key]
	return ok && v != nil
}

func claimString(claims map[string]any, key string) string {
	v, ok := claims[key]
	if !ok || v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	case float64:
		if x == float64(int64(x)) {
			return fmt.Sprintf("%d", int64(x))
		}
		return fmt.Sprintf("%v", x)
	case json.Number:
		return x.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

func claimBool(claims map[string]any, key string) bool {
	v, ok := claims[key]
	if !ok || v == nil {
		return false
	}
	switch x := v.(type) {
	case bool:
		return x
	case string:
		return strings.EqualFold(strings.TrimSpace(x), "true")
	default:
		return false
	}
}
