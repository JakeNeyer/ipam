package network

import (
	"time"

	"github.com/google/uuid"
)

// Block represents a network block with a CIDR notation.
// When EnvironmentID is nil (orphan block), OrganizationID scopes the block to that organization.
// PoolID optionally links the block to an environment pool; the block's CIDR must be contained in the pool's CIDR.
// Provider/ExternalID/ConnectionID support cloud integrations (e.g. AWS allocation -> Block for a VPC).
// DeletedAt is set when the block is soft-deleted (IPAM conflict); sync will delete it in the cloud then remove the row.
type Block struct {
	ID             uuid.UUID  `json:"id"`
	Name           string     `json:"name"`
	CIDR           string     `json:"cidr"`
	Usage          Usage      `json:"usage"`
	Children       []Block    `json:"children,omitempty"`
	EnvironmentID  uuid.UUID  `json:"environment_id,omitempty"`
	OrganizationID uuid.UUID  `json:"organization_id,omitempty"` // for orphan blocks; blocks in envs get org via environment
	PoolID         *uuid.UUID `json:"pool_id,omitempty"`         // optional; block CIDR must be contained in pool's CIDR
	Provider       string     `json:"provider,omitempty"`        // "native", "aws", "azure", "gcp"; default "native"
	ExternalID     string     `json:"external_id,omitempty"`     // provider resource ID (e.g. vpc-xxxx)
	ConnectionID   *uuid.UUID `json:"connection_id,omitempty"`   // cloud connection used to sync
	DeletedAt      *time.Time `json:"deleted_at,omitempty"`      // set when soft-deleted (pending cloud delete on next sync)
}

type Usage struct {
	TotalIPs     int `json:"total_ips"`
	UsedIPs      int `json:"used_ips"`
	AvailableIPs int `json:"available_ips"`
}
