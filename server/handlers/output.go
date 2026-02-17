package handlers

import (
	"encoding/json"

	"github.com/google/uuid"
)

// Environment Output Types
type environmentOutput struct {
	Id             uuid.UUID   `json:"id" format:"uuid"`
	Name           string      `json:"name" minLength:"1" maxLength:"255"`
	InitialPoolID  *uuid.UUID  `json:"initial_pool_id,omitempty" format:"uuid"` // first pool created (backward compat)
	PoolIDs        []uuid.UUID `json:"pool_ids,omitempty" format:"uuid"`        // all pools created with the environment
	_              struct{}    `additionalProperties:"false"`
}

type environmentListOutput struct {
	Environments []*environmentOutput `json:"environments"`
	Total        int                  `json:"total" minimum:"0"`
	_            struct{}             `additionalProperties:"false"`
}

type blockListOutput struct {
	Blocks []*blockOutput `json:"blocks"`
	Total  int            `json:"total" minimum:"0"`
	_      struct{}       `additionalProperties:"false"`
}

// environmentDetailOutput is used for GET /environments/:id (includes blocks).
type environmentDetailOutput struct {
	Id     uuid.UUID      `json:"id" format:"uuid"`
	Name   string         `json:"name" minLength:"1" maxLength:"255"`
	Blocks []*blockOutput `json:"blocks"`
	_      struct{}       `additionalProperties:"false"`
}

// Pool Output Types
type poolOutput struct {
	ID             uuid.UUID  `json:"id" format:"uuid"`
	OrganizationID  uuid.UUID  `json:"organization_id" format:"uuid"`
	EnvironmentID  uuid.UUID  `json:"environment_id" format:"uuid"`
	Name           string     `json:"name" minLength:"1" maxLength:"255"`
	CIDR           string     `json:"cidr" minLength:"9" maxLength:"50"`
	Provider       string     `json:"provider,omitempty" minLength:"0" maxLength:"32"`        // "native", "aws", etc.; omitted if native
	ExternalID     string     `json:"external_id,omitempty" minLength:"0" maxLength:"255"`   // provider resource ID
	ConnectionID   *uuid.UUID `json:"connection_id,omitempty" format:"uuid"`                 // cloud connection used to sync
	ParentPoolID   *uuid.UUID `json:"parent_pool_id,omitempty" format:"uuid"`                // for sub-pools (e.g. AWS IPAM nested pools)
	_              struct{}   `additionalProperties:"false"`
}

type poolListOutput struct {
	Pools []*poolOutput `json:"pools"`
	_     struct{}      `additionalProperties:"false"`
}

// Block Output Types (total_ips, used_ips, available_ips are derived from CIDR; string supports IPv6 /64 etc.)
type blockOutput struct {
	ID             uuid.UUID  `json:"id" format:"uuid"`
	Name           string     `json:"name" minLength:"1" maxLength:"255"`
	CIDR           string     `json:"cidr" minLength:"9" maxLength:"50"`
	TotalIPs      string     `json:"total_ips"`
	UsedIPs       string     `json:"used_ips"`
	Available     string     `json:"available_ips"`
	EnvironmentID  uuid.UUID  `json:"environment_id,omitempty" format:"uuid"`
	OrganizationID uuid.UUID  `json:"organization_id,omitempty" format:"uuid"` // for orphan blocks
	PoolID         *uuid.UUID `json:"pool_id,omitempty" format:"uuid"`          // optional
	Provider       string     `json:"provider,omitempty" minLength:"0" maxLength:"32"`      // "native", "aws", etc.; omitted if native
	ExternalID     string     `json:"external_id,omitempty" minLength:"0" maxLength:"255"`  // provider resource ID
	ConnectionID   *uuid.UUID `json:"connection_id,omitempty" format:"uuid"`               // cloud connection used to sync
	_              struct{}   `additionalProperties:"false"`
}

type suggestBlockCIDROutput struct {
	CIDR string   `json:"cidr" minLength:"9" maxLength:"50"`
	_    struct{} `additionalProperties:"false"`
}

type blockUsageOutput struct {
	Name      string  `json:"name" minLength:"1" maxLength:"255"`
	CIDR      string  `json:"cidr" minLength:"9" maxLength:"50"`
	TotalIPs  string  `json:"total_ips"`
	UsedIPs   string  `json:"used_ips"`
	Available string  `json:"available_ips"`
	Utilized  float64 `json:"utilization_percent" minimum:"0" maximum:"100"`
	_         struct{} `additionalProperties:"false"`
}

// Allocation Output Types
type allocationOutput struct {
	Id           uuid.UUID  `json:"id" format:"uuid"`
	Name         string     `json:"name" minLength:"1" maxLength:"255"`
	BlockName    string     `json:"block_name" minLength:"1" maxLength:"255"`
	CIDR         string     `json:"cidr" minLength:"9" maxLength:"50"`
	Provider     string     `json:"provider,omitempty" maxLength:"32"`
	ExternalID   string     `json:"external_id,omitempty" maxLength:"255"`
	ConnectionID *uuid.UUID `json:"connection_id,omitempty" format:"uuid"`
	_            struct{}   `additionalProperties:"false"`
}

type allocationListOutput struct {
	Allocations []*allocationOutput `json:"allocations"`
	Total       int                 `json:"total" minimum:"0"`
	_           struct{}            `additionalProperties:"false"`
}

// Reserved block output types
type reservedBlockOutput struct {
	ID        string   `json:"id" format:"uuid"`
	Name      string   `json:"name" maxLength:"255"`
	CIDR      string   `json:"cidr" minLength:"9" maxLength:"50"`
	Reason    string   `json:"reason,omitempty" maxLength:"500"`
	CreatedAt string   `json:"created_at" format:"date-time"`
	_         struct{} `additionalProperties:"false"`
}

type reservedBlockListOutput struct {
	ReservedBlocks []*reservedBlockOutput `json:"reserved_blocks"`
	_              struct{}               `additionalProperties:"false"`
}

// Integration (cloud connection) output types
type integrationOutput struct {
	ID                  uuid.UUID       `json:"id" format:"uuid"`
	OrganizationID      uuid.UUID       `json:"organization_id" format:"uuid"`
	Provider            string          `json:"provider" minLength:"1" maxLength:"32"`
	Name                string          `json:"name" minLength:"1" maxLength:"255"`
	Config              json.RawMessage `json:"config"`
	SyncIntervalMinutes int             `json:"sync_interval_minutes"` // 0=off; default 5
	SyncMode            string          `json:"sync_mode"`               // "read_only" | "read_write"
	ConflictResolution  string          `json:"conflict_resolution"`    // "cloud" | "ipam"
	LastSyncAt          *string         `json:"last_sync_at,omitempty" format:"date-time"`
	LastSyncStatus      *string         `json:"last_sync_status,omitempty"`
	LastSyncError       *string         `json:"last_sync_error,omitempty"`
	CreatedAt           string          `json:"created_at" format:"date-time"`
	UpdatedAt           string          `json:"updated_at" format:"date-time"`
	_                   struct{}        `additionalProperties:"false"`
}

type integrationListOutput struct {
	Integrations []*integrationOutput `json:"integrations"`
	_            struct{}             `additionalProperties:"false"`
}
