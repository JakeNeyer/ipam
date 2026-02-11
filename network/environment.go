package network

import "github.com/google/uuid"

// Environment represents a network environment. It containers a supernet block.
type Environment struct {
	Id             uuid.UUID `json:"id"`
	Name           string    `json:"name"`
	OrganizationID uuid.UUID `json:"organization_id"`
	Block          []Block   `json:"block"`
}
