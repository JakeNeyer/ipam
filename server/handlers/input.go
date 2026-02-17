package handlers

import (
	"encoding/json"

	"github.com/google/uuid"
)

const defaultListLimit = 50
const maxListLimit = 500

// Environment Input Types
type createEnvironmentInput struct {
	Name           string         `json:"name" required:"true" minLength:"1" maxLength:"255"`
	OrganizationID uuid.UUID      `json:"organization_id,omitempty" format:"uuid"`
	Pools          []poolItemInput `json:"pools" minItems:"0"` // optional; empty for integration-only envs (pools added by sync)
	_              struct{}       `additionalProperties:"false"`
}

type poolItemInput struct {
	Name string `json:"name" required:"true" minLength:"1" maxLength:"255"`
	CIDR string `json:"cidr" required:"true" minLength:"9" maxLength:"50"`
}

type getEnvironmentInput struct {
	ID uuid.UUID `json:"id" path:"id" required:"true" format:"uuid"`
	_  struct{}  `additionalProperties:"false"`
}

type listEnvironmentsInput struct {
	Limit          int       `query:"limit" minimum:"1" maximum:"500"`
	Offset         int       `query:"offset" minimum:"0"`
	Name           string   `query:"name" maxLength:"255"`
	OrganizationID uuid.UUID `query:"organization_id" format:"uuid"` // optional; global admin uses this to scope to one org
	_              struct{} `additionalProperties:"false"`
}

type listBlocksInput struct {
	Limit          int       `query:"limit" minimum:"1" maximum:"500"`
	Offset         int       `query:"offset" minimum:"0"`
	Name           string    `query:"name" maxLength:"255"`
	EnvironmentID  uuid.UUID `query:"environment_id" format:"uuid"`
	PoolID         uuid.UUID `query:"pool_id" format:"uuid"`          // optional; filter blocks by pool (can be used with or without environment_id)
	OrganizationID uuid.UUID `query:"organization_id" format:"uuid"` // optional; global admin uses this to scope to one org
	OrphanedOnly   bool      `query:"orphaned_only"`
	Provider       string    `query:"provider" maxLength:"32"`      // optional; filter by provider (e.g. "aws", "native")
	ConnectionID   uuid.UUID `query:"connection_id" format:"uuid"`  // optional; filter by cloud connection
	_              struct{}  `additionalProperties:"false"`
}

type listAllocationsInput struct {
	Limit          int       `query:"limit" minimum:"1" maximum:"500"`
	Offset         int       `query:"offset" minimum:"0"`
	Name           string    `query:"name" maxLength:"255"`
	BlockName      string    `query:"block_name" maxLength:"255"`
	EnvironmentID  uuid.UUID `query:"environment_id" format:"uuid"`
	OrganizationID uuid.UUID `query:"organization_id" format:"uuid"` // optional; global admin uses this to scope to one org
	Provider       string    `query:"provider" maxLength:"32"`       // optional; filter by provider (e.g. "aws", "native")
	ConnectionID   uuid.UUID `query:"connection_id" format:"uuid"` // optional; filter by cloud connection
	_              struct{}  `additionalProperties:"false"`
}

type updateEnvironmentInput struct {
	ID   uuid.UUID `json:"id" path:"id" required:"true" format:"uuid"`
	Name string    `json:"name" required:"true" minLength:"1" maxLength:"255"`
	_    struct{}  `additionalProperties:"false"`
}

// Pool Input Types
type createPoolInput struct {
	EnvironmentID  uuid.UUID  `json:"environment_id" required:"true" format:"uuid"`
	Name           string     `json:"name" required:"true" minLength:"1" maxLength:"255"`
	CIDR           string     `json:"cidr" required:"true" minLength:"9" maxLength:"50"`
	ParentPoolID   *uuid.UUID `json:"parent_pool_id,omitempty" format:"uuid"` // optional; when set, creates a child pool under this parent (same environment)
	ConnectionID   *uuid.UUID `json:"connection_id,omitempty" format:"uuid"`  // optional; when set and connection is read_write, push pool to cloud
	_              struct{}   `additionalProperties:"false"`
}

type getPoolInput struct {
	ID uuid.UUID `json:"id" path:"id" required:"true" format:"uuid"`
	_  struct{}  `additionalProperties:"false"`
}

type suggestPoolBlockCIDRInput struct {
	ID     uuid.UUID `path:"id" required:"true" format:"uuid"`
	Prefix int       `query:"prefix" minimum:"1" maximum:"32"`
	_      struct{}  `additionalProperties:"false"`
}

type listPoolsInput struct {
	EnvironmentID  uuid.UUID `query:"environment_id" format:"uuid"`
	OrganizationID uuid.UUID `query:"organization_id" format:"uuid"` // optional; when set, list all pools in org (for dashboard)
	Provider       string    `query:"provider" maxLength:"32"`       // optional; filter by provider (e.g. "aws", "native")
	ConnectionID   uuid.UUID `query:"connection_id" format:"uuid"`  // optional; filter by cloud connection
	_              struct{}  `additionalProperties:"false"`
}

type updatePoolInput struct {
	ID   uuid.UUID `json:"id" path:"id" required:"true" format:"uuid"`
	Name string    `json:"name" required:"true" minLength:"1" maxLength:"255"`
	CIDR string    `json:"cidr" required:"true" minLength:"9" maxLength:"50"`
	_    struct{}  `additionalProperties:"false"`
}

// Block Input Types
type createBlockInput struct {
	Name           string     `json:"name" required:"true" minLength:"1" maxLength:"255"`
	CIDR           string     `json:"cidr" required:"true" minLength:"9" maxLength:"18"`
	EnvironmentID  uuid.UUID  `json:"environment_id,omitempty" format:"uuid"`
	OrganizationID uuid.UUID  `json:"organization_id,omitempty" format:"uuid"` // required for orphan blocks (no environment)
	PoolID         *uuid.UUID `json:"pool_id,omitempty" format:"uuid"`          // optional; block CIDR must be contained in pool's CIDR
	_              struct{}   `additionalProperties:"false"`
}

type getBlockInput struct {
	ID uuid.UUID `json:"id" path:"id" required:"true" format:"uuid"`
	_  struct{}  `additionalProperties:"false"`
}

type suggestBlockCIDRInput struct {
	ID     uuid.UUID `path:"id" required:"true" format:"uuid"`
	Prefix int       `query:"prefix" minimum:"1" maximum:"32"`
	_      struct{}  `additionalProperties:"false"`
}

type updateBlockInput struct {
	ID             uuid.UUID  `json:"id" path:"id" required:"true" format:"uuid"`
	Name           string     `json:"name" required:"true" minLength:"1" maxLength:"255"`
	EnvironmentID  *uuid.UUID `json:"environment_id,omitempty" format:"uuid"`
	OrganizationID *uuid.UUID `json:"organization_id,omitempty" format:"uuid"` // for orphan blocks
	PoolID         *uuid.UUID `json:"pool_id,omitempty" format:"uuid"`       // optional; block CIDR must be contained in pool's CIDR
	_              struct{}   `additionalProperties:"false"`
}

// Allocation Input Types
type createAllocationInput struct {
	Name      string   `json:"name" required:"true" minLength:"1" maxLength:"255"`
	BlockName string   `json:"block_name" required:"true" minLength:"1" maxLength:"255"`
	CIDR      string   `json:"cidr" required:"true" minLength:"9" maxLength:"18"`
	_         struct{} `additionalProperties:"false"`
}

type autoAllocateInput struct {
	Name         string   `json:"name" required:"true" minLength:"1" maxLength:"255"`
	BlockName    string   `json:"block_name" required:"true" minLength:"1" maxLength:"255"`
	PrefixLength int      `json:"prefix_length" required:"true" minimum:"1" maximum:"32"`
	_            struct{} `additionalProperties:"false"`
}

type getAllocationInput struct {
	ID uuid.UUID `json:"id" path:"id" required:"true" format:"uuid"`
	_  struct{}  `additionalProperties:"false"`
}

type updateAllocationInput struct {
	ID   uuid.UUID `json:"id" path:"id" required:"true" format:"uuid"`
	Name string    `json:"name" required:"true" minLength:"1" maxLength:"255"`
	_    struct{}  `additionalProperties:"false"`
}

// List reserved blocks input (admin only). Optional organization_id for global admin to scope to one org.
type listReservedBlocksInput struct {
	OrganizationID uuid.UUID `query:"organization_id" format:"uuid"`
	_              struct{}  `additionalProperties:"false"`
}

// Reserved block input types (admin only)
type createReservedBlockInput struct {
	Name           string    `json:"name" maxLength:"255"`
	CIDR           string    `json:"cidr" required:"true" minLength:"9" maxLength:"50"`
	Reason         string    `json:"reason,omitempty" maxLength:"500"`
	OrganizationID uuid.UUID `json:"organization_id,omitempty" format:"uuid"`
	_              struct{}  `additionalProperties:"false"`
}

type getReservedBlockInput struct {
	ID uuid.UUID `path:"id" required:"true" format:"uuid"`
	_  struct{}  `additionalProperties:"false"`
}

type updateReservedBlockInput struct {
	ID   uuid.UUID `json:"id" path:"id" required:"true" format:"uuid"`
	Name string    `json:"name" maxLength:"255"`
	_    struct{}  `additionalProperties:"false"`
}

// Integration (cloud connection) input types
type createIntegrationInput struct {
	OrganizationID       uuid.UUID       `json:"organization_id,omitempty" format:"uuid"` // optional; resolved from auth when nil (org user or global admin with selected org)
	Provider             string          `json:"provider" required:"true" minLength:"1" maxLength:"32"`
	Name                 string          `json:"name" required:"true" minLength:"1" maxLength:"255"`
	Config              json.RawMessage `json:"config"` // provider-specific (e.g. aws: region, environment_id)
	SyncIntervalMinutes  *int            `json:"sync_interval_minutes,omitempty"` // 0=off; 1-1440=minutes; default 5
	SyncMode             string          `json:"sync_mode,omitempty"`            // "read_only" | "read_write"; default "read_only"
	ConflictResolution   string          `json:"conflict_resolution,omitempty"`  // "cloud" | "ipam"; default "cloud"
	_                    struct{}        `additionalProperties:"false"`
}

type getIntegrationInput struct {
	ID uuid.UUID `path:"id" required:"true" format:"uuid"`
	_  struct{}  `additionalProperties:"false"`
}

type updateIntegrationInput struct {
	ID                  uuid.UUID       `path:"id" required:"true" format:"uuid"`
	Name                string          `json:"name" required:"true" minLength:"1" maxLength:"255"`
	Config              json.RawMessage `json:"config"`
	SyncIntervalMinutes *int            `json:"sync_interval_minutes,omitempty"` // 0=off; 1-1440=minutes
	SyncMode            string          `json:"sync_mode,omitempty"`            // "read_only" | "read_write"
	ConflictResolution  string          `json:"conflict_resolution,omitempty"`  // "cloud" | "ipam"
	_                   struct{}        `additionalProperties:"false"`
}

type listIntegrationsInput struct {
	OrganizationID uuid.UUID `query:"organization_id" format:"uuid"` // optional; when set, list connections for that org
	_              struct{}  `additionalProperties:"false"`
}

type syncIntegrationInput struct {
	ID uuid.UUID `path:"id" required:"true" format:"uuid"`
	_  struct{}  `additionalProperties:"false"`
}
