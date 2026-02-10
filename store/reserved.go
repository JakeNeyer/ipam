package store

import (
	"time"

	"github.com/google/uuid"
)

// ReservedBlock is a CIDR range that cannot be used as a network block or allocation
// (blacklisted). Used to preserve ranges for future use or other systems.
type ReservedBlock struct {
	ID        uuid.UUID
	Name      string    // short label (e.g. "DMZ")
	CIDR      string
	Reason    string    // optional description
	CreatedAt time.Time
}
