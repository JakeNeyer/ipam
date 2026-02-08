package network

import "github.com/google/uuid"

const (
	// ExtraSmallBlockPrefixLength  = 26 // contains 64 IPs
	ExtraSmallBlockPrefixLength = 26

	// SmallBlockPrefixLength = 24 // contains 256 IPs
	SmallBlockPrefixLength = 24

	// MediumBlockPrefixLength = 22 // contains 1024 IPs
	MediumBlockPrefixLength = 22

	// LargeBlockPrefixLength  = 20 // contains 4096 IPs
	LargeBlockPrefixLength = 20
)

var (
	AvailableBlockSizes = []int{
		ExtraSmallBlockPrefixLength,
		SmallBlockPrefixLength,
		MediumBlockPrefixLength,
		LargeBlockPrefixLength,
	}
)

// Block represents a network block with a CIDR notation.
type Block struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	CIDR          string    `json:"cidr"`
	Usage         Usage     `json:"usage"`
	Children      []Block   `json:"children,omitempty"`
	EnvironmentID uuid.UUID `json:"environment_id,omitempty"` // uuid.Nil when not assigned
}

type Usage struct {
	TotalIPs     int `json:"total_ips"`
	UsedIPs      int `json:"used_ips"`
	AvailableIPs int `json:"available_ips"`
}
