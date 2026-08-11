package setup

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/JakeNeyer/ipam/store"
	"github.com/google/uuid"
)

var adminEmail string = "admin@example.com"

func testTokenHash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func TestInitialAdminAPITokenExpiration(t *testing.T) {
	tests := []struct {
		name    string
		ttl     string
		wantErr bool
	}{
		{
			name:    "empty ttl",
			ttl:     "",
			wantErr: true,
		},
		{
			name: "valid ttl",
			ttl:  "24h",
		},
		{
			name:    "invalid ttl",
			ttl:     "not-a-duration",
			wantErr: true,
		},
		{
			name:    "zero ttl",
			ttl:     "0s",
			wantErr: true,
		},
		{
			name:    "negative ttl",
			ttl:     "-1h",
			wantErr: true,
		},
		{
			name: "30 days",
			ttl:  "30d",
		},
		{
			name: "8 weeks",
			ttl:  "8w",
		},
		{
			name: "1 year",
			ttl:  "1y",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(initialAdminAPITokenEnv, "ipam_test_token")
			t.Setenv(initialAdminAPITokenTTLEnv, tt.ttl)

			got, err := initialAdminAPITokenExpiration()

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got == nil {
				t.Fatal("expected expiration, got nil")
			}
		})
	}
}

func TestEnsureInitialAdminAPIToken(t *testing.T) {
	tests := []struct {
		name         string
		rawToken     string
		ttl          string
		existingUser bool
		wantToken    bool
	}{
		{
			name:      "no token configured",
			rawToken:  "",
			wantToken: false,
			ttl:       "",
		},
		{
			name:      "creates token",
			rawToken:  "ipam_test_bootstrap_token",
			ttl:       "1d",
			wantToken: true,
		},
		{
			name:      "creates token with ttl",
			rawToken:  "ipam_test_bootstrap_token",
			ttl:       "24h",
			wantToken: true,
		},
		{
			name:         "does not create token for existing install",
			rawToken:     "ipam_test_bootstrap_token",
			ttl:          "1y",
			existingUser: true,
			wantToken:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("INITIAL_ADMIN_EMAIL", adminEmail)
			t.Setenv("INITIAL_ADMIN_PASSWORD", "TestPassword123!")
			t.Setenv("INITIAL_ADMIN_API_TOKEN", tt.rawToken)
			t.Setenv("INITIAL_ADMIN_API_TOKEN_TTL", tt.ttl)

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

			EnsureInitialAdmin(st, false)

			users, err := st.ListUsers(nil)
			if err != nil {
				t.Fatalf("ListUsers() error = %v", err)
			}

			if len(users) != 1 {
				t.Fatalf("got %d users, want 1", len(users))
			}

			if tt.existingUser {
				if users[0].Email != "existing@example.com" {
					t.Fatalf(
						"existing user changed: got %q",
						users[0].Email,
					)
				}
			} else {
				if users[0].Role != store.RoleAdmin {
					t.Fatalf(
						"admin role = %q, want %q",
						users[0].Role,
						store.RoleAdmin,
					)
				}

				if users[0].OrganizationID != uuid.Nil {
					t.Fatalf(
						"organization id = %v, want uuid.Nil",
						users[0].OrganizationID,
					)
				}
			}

			if tt.rawToken == "" {
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
				t.Fatalf(
					"token.UserID = %v, want %v",
					token.UserID,
					users[0].ID,
				)
			}

			if token.Name != "initial_admin" {
				t.Fatalf(
					"token.Name = %q, want %q",
					token.Name,
					"initial_admin",
				)
			}

			if token.OrganizationID != uuid.Nil {
				t.Fatalf(
					"token.OrganizationID = %v, want uuid.Nil",
					token.OrganizationID,
				)
			}
		})
	}
}
