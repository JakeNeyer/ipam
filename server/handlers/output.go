package handlers

import "github.com/google/uuid"

// Environment Output Types
type environmentOutput struct {
	Id   uuid.UUID `json:"id" format:"uuid"`
	Name string    `json:"name" minLength:"1" maxLength:"255"`
	_    struct{}  `additionalProperties:"false"`
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

// Block Output Types (total_ips, used_ips, available_ips are derived from CIDR; string supports IPv6 /64 etc.)
type blockOutput struct {
	ID             uuid.UUID `json:"id" format:"uuid"`
	Name           string    `json:"name" minLength:"1" maxLength:"255"`
	CIDR           string    `json:"cidr" minLength:"9" maxLength:"50"`
	TotalIPs      string    `json:"total_ips"`
	UsedIPs       string    `json:"used_ips"`
	Available     string    `json:"available_ips"`
	EnvironmentID  uuid.UUID `json:"environment_id,omitempty" format:"uuid"`
	OrganizationID uuid.UUID `json:"organization_id,omitempty" format:"uuid"` // for orphan blocks
	_              struct{}  `additionalProperties:"false"`
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
	Id        uuid.UUID `json:"id" format:"uuid"`
	Name      string    `json:"name" minLength:"1" maxLength:"255"`
	BlockName string    `json:"block_name" minLength:"1" maxLength:"255"`
	CIDR      string    `json:"cidr" minLength:"9" maxLength:"50"`
	_         struct{}  `additionalProperties:"false"`
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
