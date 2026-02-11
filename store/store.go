package store

import (
	"time"

	"github.com/JakeNeyer/ipam/network"
	"github.com/google/uuid"
)

type IDGenerator interface {
	GenerateID() uuid.UUID
}

type EnvironmentStore interface {
	CreateEnvironment(env *network.Environment) error
	GetEnvironment(id uuid.UUID) (*network.Environment, error)
	ListEnvironments() ([]*network.Environment, error)
	ListEnvironmentsFiltered(name string, limit, offset int) ([]*network.Environment, int, error)
	UpdateEnvironment(id uuid.UUID, env *network.Environment) error
	DeleteEnvironment(id uuid.UUID) error
}

type BlockStore interface {
	CreateBlock(block *network.Block) error
	GetBlock(id uuid.UUID) (*network.Block, error)
	ListBlocks() ([]*network.Block, error)
	ListBlocksFiltered(name string, environmentID *uuid.UUID, orphanedOnly bool, limit, offset int) ([]*network.Block, int, error)
	ListBlocksByEnvironment(envID uuid.UUID) ([]*network.Block, error)
	UpdateBlock(id uuid.UUID, block *network.Block) error
	DeleteBlock(id uuid.UUID) error
}

type AllocationStore interface {
	CreateAllocation(id uuid.UUID, alloc *network.Allocation) error
	GetAllocation(id uuid.UUID) (*network.Allocation, error)
	ListAllocations() ([]*network.Allocation, error)
	ListAllocationsFiltered(name string, blockName string, environmentID uuid.UUID, limit, offset int) ([]*network.Allocation, int, error)
	UpdateAllocation(id uuid.UUID, alloc *network.Allocation) error
	DeleteAllocation(id uuid.UUID) error
}

type ReservedBlockStore interface {
	ListReservedBlocks() ([]*ReservedBlock, error)
	CreateReservedBlock(r *ReservedBlock) error
	GetReservedBlock(id uuid.UUID) (*ReservedBlock, error)
	UpdateReservedBlock(id uuid.UUID, r *ReservedBlock) error
	DeleteReservedBlock(id uuid.UUID) error
	OverlapsReservedBlock(cidr string) (*ReservedBlock, error)
}

type UserStore interface {
	CreateUser(u *User) error
	GetUser(id uuid.UUID) (*User, error)
	GetUserByEmail(email string) (*User, error)
	ListUsers() ([]*User, error)
	DeleteUser(userID uuid.UUID) error
	SetUserRole(userID uuid.UUID, role string) error
	SetUserTourCompleted(userID uuid.UUID, completed bool) error
}

type SessionStore interface {
	CreateSession(sessionID string, userID uuid.UUID, expiry time.Time)
	GetSession(sessionID string) (*Session, error)
	DeleteSession(sessionID string)
}

type APITokenStore interface {
	CreateAPIToken(userID uuid.UUID, name string, expiresAt *time.Time) (token *APIToken, rawToken string, err error)
	GetUserByTokenHash(keyHash string) (*User, error)
	ListAPITokens(userID uuid.UUID) ([]*APIToken, error)
	DeleteAPIToken(tokenID, userID uuid.UUID) error
	GetAPIToken(tokenID uuid.UUID) (*APIToken, error)
}

type SignupInviteStore interface {
	CreateSignupInvite(createdBy uuid.UUID, expiresAt time.Time) (*SignupInvite, string, error)
	GetSignupInviteByToken(rawToken string) (*SignupInvite, error)
	MarkSignupInviteUsed(inviteID, userID uuid.UUID) error
	DeleteSignupInvite(id uuid.UUID) error
	ListSignupInvites(createdBy uuid.UUID) ([]*SignupInvite, error)
}

// Storer is the full IPAM persistence interface, composed from smaller store interfaces.
// Implemented by the in-memory Store and PostgresStore.
type Storer interface {
	IDGenerator
	EnvironmentStore
	BlockStore
	AllocationStore
	ReservedBlockStore
	UserStore
	SessionStore
	APITokenStore
	SignupInviteStore
}
