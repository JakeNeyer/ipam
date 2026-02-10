package store

import (
	"time"

	"github.com/google/uuid"
)

// APIToken represents an API key for a user. The secret is hashed; the raw token
// is only returned once at creation. ExpiresAt is optional; nil means never expires.
type APIToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Name      string
	KeyHash   string     // SHA-256 of the raw token
	CreatedAt time.Time  // When the token was created
	ExpiresAt *time.Time // When the token expires; nil = never
}
