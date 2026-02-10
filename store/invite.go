package store

import (
	"time"

	"github.com/google/uuid"
)

// SignupInvite represents a time-bound invite link for new user signup.
// The token is hashed in storage; the raw token is only returned at creation.
// UsedAt/UsedByUserID are set when someone signs up with the invite.
type SignupInvite struct {
	ID            uuid.UUID
	TokenHash     string
	CreatedBy     uuid.UUID
	ExpiresAt     time.Time
	CreatedAt     time.Time
	UsedAt        *time.Time // when the invite was used; nil if still active
	UsedByUserID  *uuid.UUID // user who signed up with this invite
}
