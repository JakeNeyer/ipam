package store

import (
	"time"

	"github.com/google/uuid"
)

const (
	RoleUser  = "user"
	RoleAdmin = "admin"
)

// OrganizationID is uuid.Nil for the global admin (created at setup); otherwise the user belongs to that organization.
type User struct {
	ID                  uuid.UUID
	Email               string
	PasswordHash        string
	Role                string
	TourCompleted       bool
	OrganizationID      uuid.UUID
	OAuthProvider       string
	OAuthProviderUserID string
}

type Session struct {
	UserID uuid.UUID
	Expiry time.Time
}

func (s *Session) Expired() bool {
	return time.Now().After(s.Expiry)
}
