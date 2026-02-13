package network

import "github.com/google/uuid"

// Block represents a network block with a CIDR notation.
// When EnvironmentID is nil (orphan block), OrganizationID scopes the block to that organization.
// PoolID optionally links the block to an environment pool; the block's CIDR must be contained in the pool's CIDR.
type Block struct {
	ID             uuid.UUID  `json:"id"`
	Name           string     `json:"name"`
	CIDR           string     `json:"cidr"`
	Usage          Usage      `json:"usage"`
	Children       []Block    `json:"children,omitempty"`
	EnvironmentID  uuid.UUID  `json:"environment_id,omitempty"`
	OrganizationID uuid.UUID  `json:"organization_id,omitempty"` // for orphan blocks; blocks in envs get org via environment
	PoolID         *uuid.UUID `json:"pool_id,omitempty"`         // optional; block CIDR must be contained in pool's CIDR
}

type Usage struct {
	TotalIPs     int `json:"total_ips"`
	UsedIPs      int `json:"used_ips"`
	AvailableIPs int `json:"available_ips"`
}
