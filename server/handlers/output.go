package handlers

import "github.com/google/uuid"

// Environment Output Types
type environmentOutput struct {
	Id   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type environmentListOutput struct {
	Environments []*environmentOutput `json:"environments"`
}

// Block Output Types
type blockOutput struct {
	Name      string `json:"name"`
	CIDR      string `json:"cidr"`
	TotalIPs  int    `json:"total_ips"`
	UsedIPs   int    `json:"used_ips"`
	Available int    `json:"available_ips"`
}

type blockListOutput struct {
	Blocks []*blockOutput `json:"blocks"`
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
}
