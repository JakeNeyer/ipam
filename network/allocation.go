package network

import "github.com/google/uuid"

// Allocation represents an allocation of a network block.
type Allocation struct {
	Id    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Block Block     `json:"block"`
}
