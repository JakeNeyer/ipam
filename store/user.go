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
type User struct {
	ID            uuid.UUID
	Email         string
	PasswordHash  string
	Role          string // "user" or "admin"
	TourCompleted bool   // true after user has completed or skipped the onboarding tour
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
