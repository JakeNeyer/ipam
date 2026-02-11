package network

import "github.com/google/uuid"

// Block represents a network block with a CIDR notation.
type Block struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	CIDR          string    `json:"cidr"`
	Usage         Usage     `json:"usage"`
	Children      []Block   `json:"children,omitempty"`
	EnvironmentID uuid.UUID `json:"environment_id,omitempty"`
}

type Usage struct {
	TotalIPs     int `json:"total_ips"`
	UsedIPs      int `json:"used_ips"`
	AvailableIPs int `json:"available_ips"`
}
