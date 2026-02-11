package store

import (
	"time"

	"github.com/google/uuid"
)

// ReservedBlock is a CIDR range that cannot be used as a network block or allocation
// (blacklisted). Used to preserve ranges for future use or other systems.
// Scoped to an organization; overlap checks use the org's reserved list (or all orgs when nil).
type ReservedBlock struct {
	ID             uuid.UUID
	Name           string
	CIDR           string
	Reason         string
	CreatedAt      time.Time
	OrganizationID uuid.UUID
}
