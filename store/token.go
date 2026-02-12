package store

import (
	"time"

	"github.com/google/uuid"
)

// APIToken represents an API key for a user. The secret is hashed; the raw token
// is only returned once at creation. ExpiresAt is optional; nil means never expires.
// OrganizationID, when set, scopes the token to that org (global admin only); uuid.Nil means full access.
type APIToken struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	Name           string
	KeyHash        string
	CreatedAt      time.Time
	ExpiresAt      *time.Time
	OrganizationID uuid.UUID // optional; when set, token is scoped to this org (global admin only)
}
