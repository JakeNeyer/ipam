package network

import (
	"time"

	"github.com/google/uuid"
)

// Pool is a range of CIDRs that network blocks in an environment can draw from.
// Pools are scoped to an organization (via environment). Hierarchy: Organization -> Environment -> Pool(s) -> Network blocks -> Allocations
// Provider/ExternalID/ConnectionID/ParentPoolID support cloud integrations (e.g. AWS IPAM sub-pools).
// DeletedAt is set when the pool is soft-deleted (IPAM conflict resolution); sync will delete it in the cloud then remove the row.
type Pool struct {
	ID             uuid.UUID  `json:"id"`
	OrganizationID uuid.UUID  `json:"organization_id"`
	EnvironmentID  uuid.UUID  `json:"environment_id"`
	Name           string     `json:"name"`
	CIDR           string     `json:"cidr"`
	Provider       string     `json:"provider,omitempty"`        // "native", "aws", "azure", "gcp"; default "native"
	ExternalID     string     `json:"external_id,omitempty"`    // provider resource ID (e.g. ipam-pool-xxxx)
	ConnectionID   *uuid.UUID `json:"connection_id,omitempty"`   // cloud connection used to sync
	ParentPoolID   *uuid.UUID `json:"parent_pool_id,omitempty"`  // for sub-pools (e.g. AWS IPAM nested pools)
	DeletedAt      *time.Time `json:"deleted_at,omitempty"`      // set when soft-deleted (pending cloud delete on next sync)
}
