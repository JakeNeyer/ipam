package store

import (
	"time"

	"github.com/google/uuid"
)

// Organization represents a tenant. Users and environments belong to an organization.
type Organization struct {
	ID        uuid.UUID
	Name      string
	CreatedAt time.Time
}
