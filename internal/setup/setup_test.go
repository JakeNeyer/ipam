package setup

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/JakeNeyer/ipam/store"
	"github.com/google/uuid"
)

const testAdminEmail = "admin@example.com"
const testAdminPassword = "TestPassword123!"
const testBootstrapToken = "ipam_test_bootstrap_token_ok"

func testTokenHash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func TestParseAPITokenTTL(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{name: "empty", raw: "", wantErr: true},
		{name: "hours", raw: "24h"},
		{name: "minutes", raw: "30m"},
		{name: "days", raw: "30d"},
		{name: "weeks", raw: "8w"},
		{name: "years", raw: "1y"},
		{name: "invalid words", raw: "8 weeks", wantErr: true},
		{name: "zero", raw: "0s", wantErr: false}, // ParseDuration accepts; expiration rejects separately
		{name: "garbage", raw: "not-a-duration", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseAPITokenTTL(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestInitialAdminAPITokenExpiration(t *testing.T) {
	tests := []struct {
		name    string
		ttl     string
		wantErr bool
	}{
		{name: "empty ttl", ttl: "", wantErr: true},
		{name: "valid ttl", ttl: "24h"},
		{name: "invalid ttl", ttl: "not-a-duration", wantErr: true},
		{name: "zero ttl", ttl: "0s", wantErr: true},
		{name: "negative ttl", ttl: "-1h", wantErr: true},
		{name: "30 days", ttl: "30d"},
		{name: "8 weeks", ttl: "8w"},
		{name: "1 year", ttl: "1y"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(initialAdminAPITokenTTLEnv, tt.ttl)
			got, err := initialAdminAPITokenExpiration()
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got == nil {
				t.Fatal("expected expiration")
			}
		})
	}
}

func TestEnsureInitialAdmin_Bootstrap(t *testing.T) {
	tests := []struct {
		name         string
		email        string
		password     string
		orgName      string
		rawToken     string
		ttl          string
		existingUser bool
		wantErr      string
		wantOrg      bool
		wantToken    bool
	}{
		{
			name:     "admin only",
			email:    testAdminEmail,
			password: testAdminPassword,
		},
		{
			name:      "admin and organization",
			email:     testAdminEmail,
			password:  testAdminPassword,
			orgName:   "Acme",
			wantOrg:   true,
		},
		{
			name:      "org-scoped token",
			email:     testAdminEmail,
			password:  testAdminPassword,
			orgName:   "Acme",
			rawToken:  testBootstrapToken,
			ttl:       "24h",
			wantOrg:   true,
			wantToken: true,
		},
		{
			name:     "token without org fails before create",
			email:    testAdminEmail,
			password: testAdminPassword,
			rawToken: testBootstrapToken,
			ttl:      "24h",
			wantErr:  initialOrganizationNameEnv,
		},
		{
			name:     "token without ttl fails",
			email:    testAdminEmail,
			password: testAdminPassword,
			orgName:  "Acme",
			rawToken: testBootstrapToken,
			wantErr:  initialAdminAPITokenTTLEnv,
		},
		{
			name:     "token without email fails",
			orgName:  "Acme",
			rawToken: testBootstrapToken,
			ttl:      "1d",
			wantErr:  "INITIAL_ADMIN_EMAIL",
		},
		{
			name:     "short token fails",
			email:    testAdminEmail,
			password: testAdminPassword,
			orgName:  "Acme",
			rawToken: "too-short",
			ttl:      "1d",
			wantErr:  "at least",
		},
		{
			name:         "existing install is no-op",
			email:        testAdminEmail,
			password:     testAdminPassword,
			orgName:      "Acme",
			rawToken:     testBootstrapToken,
			ttl:          "1y",
			existingUser: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("INITIAL_ADMIN_EMAIL", tt.email)
			t.Setenv("INITIAL_ADMIN_PASSWORD", tt.password)
			t.Setenv(initialOrganizationNameEnv, tt.orgName)
			t.Setenv(initialAdminAPITokenEnv, tt.rawToken)
			t.Setenv(initialAdminAPITokenTTLEnv, tt.ttl)

			st := store.NewStore()
			if tt.existingUser {
				existing := &store.User{
					ID:             uuid.New(),
					Email:          "existing@example.com",
					Role:           store.RoleAdmin,
					OrganizationID: uuid.Nil,
				}
				if err := st.CreateUser(existing); err != nil {
					t.Fatalf("CreateUser() error = %v", err)
				}
			}

			err := EnsureInitialAdmin(st, false)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatal("expected error")
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error %q does not contain %q", err.Error(), tt.wantErr)
				}
				users, listErr := st.ListUsers(nil)
				if listErr != nil {
					t.Fatalf("ListUsers() error = %v", listErr)
				}
				if !tt.existingUser && len(users) != 0 {
					t.Fatalf("expected no users after failed bootstrap, got %d", len(users))
				}
				return
			}
			if err != nil {
				t.Fatalf("EnsureInitialAdmin() error = %v", err)
			}

			users, err := st.ListUsers(nil)
			if err != nil {
				t.Fatalf("ListUsers() error = %v", err)
			}
			if len(users) != 1 {
				t.Fatalf("got %d users, want 1", len(users))
			}
			if tt.existingUser {
				if users[0].Email != "existing@example.com" {
					t.Fatalf("existing user changed: got %q", users[0].Email)
				}
			} else if users[0].Role != store.RoleAdmin || users[0].OrganizationID != uuid.Nil {
				t.Fatalf("unexpected admin user: %+v", users[0])
			}

			orgs, err := st.ListOrganizations()
			if err != nil {
				t.Fatalf("ListOrganizations() error = %v", err)
			}
			if tt.wantOrg {
				if len(orgs) != 1 || orgs[0].Name != tt.orgName {
					t.Fatalf("got orgs %#v, want one named %q", orgs, tt.orgName)
				}
			} else if len(orgs) != 0 {
				t.Fatalf("got %d orgs, want 0", len(orgs))
			}

			if tt.rawToken == "" || tt.existingUser {
				tokens, err := st.ListAPITokens(users[0].ID)
				if err != nil {
					t.Fatalf("ListAPITokens() error = %v", err)
				}
				if len(tokens) != 0 {
					t.Fatalf("got %d tokens, want 0", len(tokens))
				}
				return
			}

			token, err := st.GetAPITokenByKeyHash(testTokenHash(tt.rawToken))
			if !tt.wantToken {
				if err == nil {
					t.Fatal("expected token lookup to fail")
				}
				return
			}
			if err != nil {
				t.Fatalf("GetAPITokenByKeyHash() error = %v", err)
			}
			if token.UserID != users[0].ID {
				t.Fatalf("token.UserID = %v, want %v", token.UserID, users[0].ID)
			}
			if token.Name != initialAdminAPITokenName {
				t.Fatalf("token.Name = %q, want %q", token.Name, initialAdminAPITokenName)
			}
			if token.OrganizationID != orgs[0].ID {
				t.Fatalf("token.OrganizationID = %v, want %v", token.OrganizationID, orgs[0].ID)
			}
			if token.ExpiresAt == nil {
				t.Fatal("expected token expiration")
			}
		})
	}
}
