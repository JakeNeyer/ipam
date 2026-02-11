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

// Block Output Types
type blockOutput struct {
	ID            uuid.UUID `json:"id" format:"uuid"`
	Name          string    `json:"name" minLength:"1" maxLength:"255"`
	CIDR          string    `json:"cidr" minLength:"9" maxLength:"50"`
	TotalIPs      int       `json:"total_ips" minimum:"0"`
	UsedIPs       int       `json:"used_ips" minimum:"0"`
	Available     int       `json:"available_ips" minimum:"0"`
	EnvironmentID uuid.UUID `json:"environment_id,omitempty" format:"uuid"`
	_             struct{}  `additionalProperties:"false"`
}

type suggestBlockCIDROutput struct {
	CIDR string   `json:"cidr" minLength:"9" maxLength:"50"`
	_    struct{} `additionalProperties:"false"`
}

type blockUsageOutput struct {
	Name      string   `json:"name" minLength:"1" maxLength:"255"`
	CIDR      string   `json:"cidr" minLength:"9" maxLength:"50"`
	TotalIPs  int      `json:"total_ips" minimum:"0"`
	UsedIPs   int      `json:"used_ips" minimum:"0"`
	Available int      `json:"available_ips" minimum:"0"`
	Utilized  float64  `json:"utilization_percent" minimum:"0" maximum:"100"`
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
