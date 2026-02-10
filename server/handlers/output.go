package handlers

import "github.com/google/uuid"

// Environment Output Types
type environmentOutput struct {
	Id   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type environmentListOutput struct {
	Environments []*environmentOutput `json:"environments"`
	Total        int                  `json:"total"`
}

type blockListOutput struct {
	Blocks []*blockOutput `json:"blocks"`
	Total  int            `json:"total"`
}

// environmentDetailOutput is used for GET /environments/:id (includes blocks).
type environmentDetailOutput struct {
	Id     uuid.UUID      `json:"id"`
	Name   string         `json:"name"`
	Blocks []*blockOutput `json:"blocks"`
}

// Block Output Types
type blockOutput struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	CIDR          string    `json:"cidr"`
	TotalIPs      int       `json:"total_ips"`
	UsedIPs       int       `json:"used_ips"`
	Available     int       `json:"available_ips"`
	EnvironmentID uuid.UUID `json:"environment_id,omitempty"`
}


type suggestBlockCIDROutput struct {
	CIDR string `json:"cidr"`
}

type blockUsageOutput struct {
	Name      string  `json:"name"`
	CIDR      string  `json:"cidr"`
	TotalIPs  int     `json:"total_ips"`
	UsedIPs   int     `json:"used_ips"`
	Available int     `json:"available_ips"`
	Utilized  float64 `json:"utilization_percent"`
}

// Allocation Output Types
type allocationOutput struct {
	Id        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	BlockName string    `json:"block_name"`
	CIDR      string    `json:"cidr"`
}

type allocationListOutput struct {
	Allocations []*allocationOutput `json:"allocations"`
	Total       int                 `json:"total"`
}

// Reserved block output types
type reservedBlockOutput struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CIDR      string `json:"cidr"`
	Reason    string `json:"reason,omitempty"`
	CreatedAt string `json:"created_at"`
}

type reservedBlockListOutput struct {
	ReservedBlocks []*reservedBlockOutput `json:"reserved_blocks"`
}
