package handlers

import "github.com/google/uuid"

const defaultListLimit = 50
const maxListLimit = 500

// Environment Input Types
type createEnvironmentInput struct {
	Name         string             `json:"name" minLength:"1" maxLength:"255"`
	InitialBlock *initialBlockInput `json:"initial_block,omitempty"`
	_            struct{}           `additionalProperties:"false"`
}

type initialBlockInput struct {
	Name string `json:"name" minLength:"1" maxLength:"255"`
	CIDR string `json:"cidr" minLength:"9" maxLength:"18"`
}

type getEnvironmentInput struct {
	ID uuid.UUID `json:"id" minLength:"1" maxLength:"255" path:"id"`
	_  struct{}  `additionalProperties:"false"`
}

type listEnvironmentsInput struct {
	Limit  int    `query:"limit"`
	Offset int    `query:"offset"`
	Name   string `query:"name"`
	_      struct{} `additionalProperties:"false"`
}

type listBlocksInput struct {
	Limit         int       `query:"limit"`
	Offset        int       `query:"offset"`
	Name          string    `query:"name"`
	EnvironmentID uuid.UUID `query:"environment_id"`
	OrphanedOnly  bool      `query:"orphaned_only"`
	_             struct{}  `additionalProperties:"false"`
}

type listAllocationsInput struct {
	Limit         int       `query:"limit"`
	Offset        int       `query:"offset"`
	Name          string    `query:"name"`
	BlockName     string    `query:"block_name"`
	EnvironmentID uuid.UUID `query:"environment_id"`
	_             struct{}  `additionalProperties:"false"`
}

type suggestEnvironmentBlockCIDRInput struct {
	ID     uuid.UUID `path:"id"`
	Prefix int       `query:"prefix"`
	_      struct{}  `additionalProperties:"false"`
}

type updateEnvironmentInput struct {
	ID   uuid.UUID `json:"id" minLength:"1" maxLength:"255" path:"id"`
	Name string    `json:"name" minLength:"1" maxLength:"255"`
	_    struct{}  `additionalProperties:"false"`
}

// Block Input Types
type createBlockInput struct {
	Name          string    `json:"name" minLength:"1" maxLength:"255"`
	CIDR          string    `json:"cidr" minLength:"9" maxLength:"18"` // e.g., "10.0.0.0/8"
	EnvironmentID uuid.UUID `json:"environment_id,omitempty"`
	_             struct{}  `additionalProperties:"false"`
}

type getBlockInput struct {
	ID uuid.UUID `json:"id" minLength:"1" maxLength:"255" path:"id"`
	_  struct{}  `additionalProperties:"false"`
}

type suggestBlockCIDRInput struct {
	ID     uuid.UUID `path:"id"`
	Prefix int       `query:"prefix"`
	_      struct{}  `additionalProperties:"false"`
}

type updateBlockInput struct {
	ID            uuid.UUID  `json:"id" minLength:"1" maxLength:"255" path:"id"`
	Name          string     `json:"name" minLength:"1" maxLength:"255"`
	EnvironmentID *uuid.UUID `json:"environment_id,omitempty"`
	_             struct{}   `additionalProperties:"false"`
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

// Reserved block input types (admin only)
type createReservedBlockInput struct {
	Name   string `json:"name" maxLength:"255"`
	CIDR   string `json:"cidr" minLength:"9" maxLength:"50"`
	Reason string `json:"reason,omitempty" maxLength:"500"`
	_      struct{} `additionalProperties:"false"`
}

type getReservedBlockInput struct {
	ID uuid.UUID `path:"id"`
	_  struct{}  `additionalProperties:"false"`
}
