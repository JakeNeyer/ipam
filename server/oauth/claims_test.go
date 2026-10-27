package oauth

import (
	"encoding/base64"
	"encoding/json"
	"reflect"
	"testing"

	"golang.org/x/oauth2"
)

func TestClaimValues(t *testing.T) {
	claims := map[string]any{
		"groups": []any{"ipam-admins", "team-network"},
		"role":   "user",
		"realm_access": map[string]any{
			"roles": []any{"ipam-user", "offline_access"},
		},
		"empty": []any{},
	}
	tests := []struct {
		path string
		want []string
	}{
		{"groups", []string{"ipam-admins", "team-network"}},
		{"role", []string{"user"}},
		{"realm_access.roles", []string{"ipam-user", "offline_access"}},
		{"missing", nil},
		{"realm_access.missing", nil},
		{"empty", nil},
		{"", nil},
	}
	for _, tt := range tests {
		got := ClaimValues(claims, tt.path)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("ClaimValues(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestDecodeIDTokenClaims(t *testing.T) {
	token := fakeIDToken(map[string]any{"sub": "abc", "groups": []any{"admins"}})
	got, err := DecodeIDTokenClaims(token)
	if err != nil {
		t.Fatal(err)
	}
	if got["sub"] != "abc" {
		t.Errorf("sub = %v", got["sub"])
	}
	groups, ok := got["groups"].([]any)
	if !ok || len(groups) != 1 || groups[0] != "admins" {
		t.Errorf("groups = %v", got["groups"])
	}
}

func TestMergeTokenClaims_IDTokenOverlaysUserinfo(t *testing.T) {
	userinfo := map[string]any{
		"sub":    "from-userinfo",
		"email":  "a@example.com",
		"groups": []any{"userinfo-group"},
	}
	tok := (&oauth2.Token{AccessToken: "tok"}).WithExtra(map[string]any{
		"id_token": fakeIDToken(map[string]any{
			"sub":    "from-id-token",
			"groups": []any{"jwt-group"},
		}),
	})
	got := MergeTokenClaims(userinfo, tok)
	if got["sub"] != "from-id-token" {
		t.Errorf("sub = %v, want id_token overlay", got["sub"])
	}
	if got["email"] != "a@example.com" {
		t.Errorf("email = %v, want userinfo value", got["email"])
	}
	if !reflect.DeepEqual(got["groups"], []any{"jwt-group"}) {
		t.Errorf("groups = %v", got["groups"])
	}
}

func TestMergeTokenClaims_NoIDToken(t *testing.T) {
	userinfo := map[string]any{"sub": "abc"}
	got := MergeTokenClaims(userinfo, &oauth2.Token{AccessToken: "tok"})
	if got["sub"] != "abc" {
		t.Errorf("sub = %v", got["sub"])
	}
}

func fakeIDToken(claims map[string]any) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload, err := json.Marshal(claims)
	if err != nil {
		panic(err)
	}
	body := base64.RawURLEncoding.EncodeToString(payload)
	return header + "." + body + ".sig"
}
