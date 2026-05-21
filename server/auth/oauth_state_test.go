package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOAuthStateCookie_RoundTrip(t *testing.T) {
	nonce, err := NewOAuthStateNonce()
	if err != nil {
		t.Fatal(err)
	}
	payload := OAuthStatePayload{Nonce: nonce, Provider: "keycloak", InviteToken: "invite-abc"}

	rr := httptest.NewRecorder()
	if err := SetOAuthStateCookie(rr, payload, false); err != nil {
		t.Fatal(err)
	}
	res := rr.Result()
	defer res.Body.Close()
	cookies := res.Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies = %d", len(cookies))
	}

	req := httptest.NewRequest(http.MethodGet, "/api/auth/oauth/keycloak/callback", nil)
	req.AddCookie(cookies[0])
	got, ok := OAuthStateFromRequest(req)
	if !ok {
		t.Fatal("OAuthStateFromRequest = false")
	}
	if got.Nonce != nonce || got.Provider != "keycloak" || got.InviteToken != "invite-abc" {
		t.Errorf("payload = %+v", got)
	}
}
