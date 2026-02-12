package handlers

import "github.com/google/uuid"

const defaultListLimit = 50
const maxListLimit = 500

// Environment Input Types
type createEnvironmentInput struct {
	Name             string             `json:"name" required:"true" minLength:"1" maxLength:"255"`
	OrganizationID   uuid.UUID          `json:"organization_id,omitempty" format:"uuid"`
	InitialBlock     *initialBlockInput `json:"initial_block,omitempty"`
	_                struct{}           `additionalProperties:"false"`
}

type initialBlockInput struct {
	Name string `json:"name" required:"true" minLength:"1" maxLength:"255"`
	CIDR string `json:"cidr" required:"true" minLength:"9" maxLength:"18"`
}

type getEnvironmentInput struct {
	ID uuid.UUID `json:"id" path:"id" required:"true" format:"uuid"`
	_  struct{}  `additionalProperties:"false"`
}

type listEnvironmentsInput struct {
	Limit          int       `query:"limit" minimum:"1" maximum:"500"`
	Offset         int       `query:"offset" minimum:"0"`
	Name           string   `query:"name" maxLength:"255"`
	OrganizationID uuid.UUID `query:"organization_id" format:"uuid"` // optional; global admin uses this to scope to one org
	_              struct{} `additionalProperties:"false"`
}

type listBlocksInput struct {
	Limit          int       `query:"limit" minimum:"1" maximum:"500"`
	Offset         int       `query:"offset" minimum:"0"`
	Name           string    `query:"name" maxLength:"255"`
	EnvironmentID  uuid.UUID `query:"environment_id" format:"uuid"`
	OrganizationID uuid.UUID `query:"organization_id" format:"uuid"` // optional; global admin uses this to scope to one org
	OrphanedOnly   bool      `query:"orphaned_only"`
	_              struct{}  `additionalProperties:"false"`
}

type listAllocationsInput struct {
	Limit          int       `query:"limit" minimum:"1" maximum:"500"`
	Offset         int       `query:"offset" minimum:"0"`
	Name           string    `query:"name" maxLength:"255"`
	BlockName      string    `query:"block_name" maxLength:"255"`
	EnvironmentID  uuid.UUID `query:"environment_id" format:"uuid"`
	OrganizationID uuid.UUID `query:"organization_id" format:"uuid"` // optional; global admin uses this to scope to one org
	_              struct{}  `additionalProperties:"false"`
}

type suggestEnvironmentBlockCIDRInput struct {
	ID     uuid.UUID `path:"id" required:"true" format:"uuid"`
	Prefix int       `query:"prefix" minimum:"1" maximum:"32"`
	_      struct{}  `additionalProperties:"false"`
}

type updateEnvironmentInput struct {
	ID   uuid.UUID `json:"id" path:"id" required:"true" format:"uuid"`
	Name string    `json:"name" required:"true" minLength:"1" maxLength:"255"`
	_    struct{}  `additionalProperties:"false"`
}

// Block Input Types
type createBlockInput struct {
	Name           string    `json:"name" required:"true" minLength:"1" maxLength:"255"`
	CIDR           string    `json:"cidr" required:"true" minLength:"9" maxLength:"18"`
	EnvironmentID  uuid.UUID `json:"environment_id,omitempty" format:"uuid"`
	OrganizationID uuid.UUID `json:"organization_id,omitempty" format:"uuid"` // required for orphan blocks (no environment)
	_              struct{}  `additionalProperties:"false"`
}

type getBlockInput struct {
	ID uuid.UUID `json:"id" path:"id" required:"true" format:"uuid"`
	_  struct{}  `additionalProperties:"false"`
}

type suggestBlockCIDRInput struct {
	ID     uuid.UUID `path:"id" required:"true" format:"uuid"`
	Prefix int       `query:"prefix" minimum:"1" maximum:"32"`
	_      struct{}  `additionalProperties:"false"`
}

type updateBlockInput struct {
	ID             uuid.UUID  `json:"id" path:"id" required:"true" format:"uuid"`
	Name           string     `json:"name" required:"true" minLength:"1" maxLength:"255"`
	EnvironmentID  *uuid.UUID `json:"environment_id,omitempty" format:"uuid"`
	OrganizationID *uuid.UUID `json:"organization_id,omitempty" format:"uuid"` // for orphan blocks
	_              struct{}   `additionalProperties:"false"`
}

// Allocation Input Types
type createAllocationInput struct {
	Name      string   `json:"name" required:"true" minLength:"1" maxLength:"255"`
	BlockName string   `json:"block_name" required:"true" minLength:"1" maxLength:"255"`
	CIDR      string   `json:"cidr" required:"true" minLength:"9" maxLength:"18"`
	_         struct{} `additionalProperties:"false"`
}

type autoAllocateInput struct {
	Name         string   `json:"name" required:"true" minLength:"1" maxLength:"255"`
	BlockName    string   `json:"block_name" required:"true" minLength:"1" maxLength:"255"`
	PrefixLength int      `json:"prefix_length" required:"true" minimum:"1" maximum:"32"`
	_            struct{} `additionalProperties:"false"`
}

type getAllocationInput struct {
	ID uuid.UUID `json:"id" path:"id" required:"true" format:"uuid"`
	_  struct{}  `additionalProperties:"false"`
}

type updateAllocationInput struct {
	ID   uuid.UUID `json:"id" path:"id" required:"true" format:"uuid"`
	Name string    `json:"name" required:"true" minLength:"1" maxLength:"255"`
	_    struct{}  `additionalProperties:"false"`
}

// List reserved blocks input (admin only). Optional organization_id for global admin to scope to one org.
type listReservedBlocksInput struct {
	OrganizationID uuid.UUID `query:"organization_id" format:"uuid"`
	_              struct{}  `additionalProperties:"false"`
}

// Reserved block input types (admin only)
type createReservedBlockInput struct {
	Name           string    `json:"name" maxLength:"255"`
	CIDR           string    `json:"cidr" required:"true" minLength:"9" maxLength:"50"`
	Reason         string    `json:"reason,omitempty" maxLength:"500"`
	OrganizationID uuid.UUID `json:"organization_id,omitempty" format:"uuid"`
	_              struct{}  `additionalProperties:"false"`
}

type getReservedBlockInput struct {
	ID uuid.UUID `path:"id" required:"true" format:"uuid"`
	_  struct{}  `additionalProperties:"false"`
}

type updateReservedBlockInput struct {
	ID   uuid.UUID `json:"id" path:"id" required:"true" format:"uuid"`
	Name string    `json:"name" maxLength:"255"`
	_    struct{}  `additionalProperties:"false"`
}
