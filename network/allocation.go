package network

import (
	"time"

	"github.com/google/uuid"
)

// Allocation represents an allocation of a network block.
// Provider/ExternalID/ConnectionID support cloud-synced allocations (e.g. AWS VPC subnets).
// DeletedAt is set when the allocation is soft-deleted (IPAM conflict); sync will delete it in the cloud then remove the row.
type Allocation struct {
	Id           uuid.UUID  `json:"id"`
	Name         string     `json:"name"`
	Block        Block      `json:"block"`
	Provider     string     `json:"provider,omitempty"`      // "native", "aws", "azure", "gcp"; default "native"
	ExternalID   string     `json:"external_id,omitempty"`  // provider resource ID (e.g. subnet-xxxx)
	ConnectionID *uuid.UUID `json:"connection_id,omitempty"` // cloud connection used to sync
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`    // set when soft-deleted (pending cloud delete on next sync)
}
