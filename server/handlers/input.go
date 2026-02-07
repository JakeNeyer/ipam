package handlers

import "github.com/google/uuid"

// Environment Input Types
type createEnvironmentInput struct {
	Name string   `json:"name" minLength:"1" maxLength:"255"`
	_    struct{} `additionalProperties:"false"`
}

type getEnvironmentInput struct {
	ID uuid.UUID `json:"id" minLength:"1" maxLength:"255" path:"id"`
	_  struct{}  `additionalProperties:"false"`
}

type updateEnvironmentInput struct {
	ID   uuid.UUID `json:"id" minLength:"1" maxLength:"255" path:"id"`
	Name string    `json:"name" minLength:"1" maxLength:"255"`
	_    struct{}  `additionalProperties:"false"`
}

// Block Input Types
type createBlockInput struct {
	Name string   `json:"name" minLength:"1" maxLength:"255"`
	CIDR string   `json:"cidr" minLength:"9" maxLength:"18"` // e.g., "10.0.0.0/8"
	_    struct{} `additionalProperties:"false"`
}

type getBlockInput struct {
	ID uuid.UUID `json:"id" minLength:"1" maxLength:"255" path:"id"`
	_  struct{}  `additionalProperties:"false"`
}

type updateBlockInput struct {
	ID   uuid.UUID `json:"id" minLength:"1" maxLength:"255" path:"id"`
	Name string    `json:"name" minLength:"1" maxLength:"255"`
	_    struct{}  `additionalProperties:"false"`
}

// Allocation Input Types
type createAllocationInput struct {
	Name      string   `json:"name" minLength:"1" maxLength:"255"`
	BlockName string   `json:"block_name" minLength:"1" maxLength:"255"`
	CIDR      string   `json:"cidr" minLength:"9" maxLength:"18"`
	_         struct{} `additionalProperties:"false"`
}

type getAllocationInput struct {
	ID uuid.UUID `json:"id" minLength:"1" maxLength:"255" path:"id"`
	_  struct{}  `additionalProperties:"false"`
}

type updateAllocationInput struct {
	ID   uuid.UUID `json:"id" minLength:"1" maxLength:"255" path:"id"`
	Name string    `json:"name" minLength:"1" maxLength:"255"`
	_    struct{}  `additionalProperties:"false"`
}
