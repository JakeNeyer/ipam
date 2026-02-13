package network

import "github.com/google/uuid"

// Pool is a range of CIDRs that network blocks in an environment can draw from.
// Pools are scoped to an organization (via environment). Hierarchy: Organization -> Environment -> Pool(s) -> Network blocks -> Allocations
type Pool struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	EnvironmentID  uuid.UUID `json:"environment_id"`
	Name           string    `json:"name"`
	CIDR           string    `json:"cidr"`
}
