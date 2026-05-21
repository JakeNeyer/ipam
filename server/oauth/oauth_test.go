package oauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/oauth2"
)

func TestOAuthUserInfo_FetchUser(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer access-token" {
			t.Errorf("Authorization = %q", r.Header.Get("Authorization"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"sub":            "user-123",
			"email":          "User@Example.com",
			"email_verified": true,
		})
	}))
	defer srv.Close()

	f := newOAuthUserInfo(srv.URL, "sub", "email", "", "", "email_verified", "")
	id, email, err := f.FetchUser(context.Background(), &oauth2.Token{AccessToken: "access-token"})
	if err != nil {
		t.Fatal(err)
	}
	if id != "user-123" {
		t.Errorf("providerUserID = %q", id)
	}
	if email != "user@example.com" {
		t.Errorf("email = %q", email)
	}
}

func TestOAuthUserInfo_FetchUser_RejectsUnverifiedEmail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"sub":            "user-123",
			"email":          "user@example.com",
			"email_verified": false,
		})
	}))
	defer srv.Close()

	f := newOAuthUserInfo(srv.URL, "sub", "email", "", "", "email_verified", "")
	_, _, err := f.FetchUser(context.Background(), &oauth2.Token{AccessToken: "access-token"})
	if err == nil {
		t.Fatal("expected error for unverified email")
	}
}

func TestOAuthUserInfo_FetchUser_EmailsListFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/user":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 42})
		case "/user/emails":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"email": "other@example.com", "primary": false, "verified": true},
				{"email": "User@Example.com", "primary": true, "verified": true},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	f := newOAuthUserInfo(srv.URL+"/user", "id", "email", srv.URL+"/user/emails", "primary", "", "verified")
	id, email, err := f.FetchUser(context.Background(), &oauth2.Token{AccessToken: "access-token"})
	if err != nil {
		t.Fatal(err)
	}
	if id != "42" {
		t.Errorf("providerUserID = %q", id)
	}
	if email != "user@example.com" {
		t.Errorf("email = %q", email)
	}
}

func TestOAuthUserInfo_FetchUser_SkipsUnverifiedListEntry(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/user":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 42})
		case "/user/emails":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"email": "bad@example.com", "primary": true, "verified": false},
				{"email": "good@example.com", "primary": false, "verified": true},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	f := newOAuthUserInfo(srv.URL+"/user", "id", "email", srv.URL+"/user/emails", "primary", "", "verified")
	_, email, err := f.FetchUser(context.Background(), &oauth2.Token{AccessToken: "access-token"})
	if err != nil {
		t.Fatal(err)
	}
	if email != "good@example.com" {
		t.Errorf("email = %q", email)
	}
}

func TestClaimString(t *testing.T) {
	claims := map[string]any{"sub": "abc", "n": float64(42)}
	if got := claimString(claims, "sub"); got != "abc" {
		t.Errorf("sub = %q", got)
	}
	if got := claimString(claims, "n"); got != "42" {
		t.Errorf("n = %q", got)
	}
}

func TestClaimBool(t *testing.T) {
	if !claimBool(map[string]any{"primary": true}, "primary") {
		t.Error("primary true = false")
	}
	if claimBool(map[string]any{"primary": false}, "primary") {
		t.Error("primary false = true")
	}
}
