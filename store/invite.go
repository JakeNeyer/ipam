package store

import (
	"time"

	"github.com/google/uuid"
)

// SignupInvite represents a time-bound invite link for new user signup.
// The token is hashed in storage; the raw token is only returned at creation.
// UsedAt/UsedByUserID are set when someone signs up with the invite.
// OrganizationID and Role are set when creating the invite (global admin can set; org admin gets their org and user role).
// If OrganizationID is uuid.Nil at use time, the inviter's org is used (backward compat).
type SignupInvite struct {
	ID             uuid.UUID
	TokenHash      string
	CreatedBy      uuid.UUID
	ExpiresAt      time.Time
	CreatedAt      time.Time
	UsedAt         *time.Time
	UsedByUserID   *uuid.UUID
	OrganizationID uuid.UUID
	Role           string
}
