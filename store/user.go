package store

import (
	"time"

	"github.com/google/uuid"
)

const (
	RoleUser  = "user"
	RoleAdmin = "admin"
)

// User represents a user account.
// OrganizationID is uuid.Nil for the global admin (created at setup); otherwise the user belongs to that organization.
// OAuthProvider and OAuthProviderUserID are set when the user signs in via OAuth (e.g. "github", "12345"). PasswordHash may be empty for OAuth-only users.
type User struct {
	ID                  uuid.UUID
	Email               string
	PasswordHash        string
	Role                string
	TourCompleted       bool
	OrganizationID      uuid.UUID
	OAuthProvider       string
	OAuthProviderUserID  string
}

// Session represents an active session.
type Session struct {
	UserID uuid.UUID
	Expiry time.Time
}

// Expired returns true if the session has expired.
func (s *Session) Expired() bool {
	return time.Now().After(s.Expiry)
}
