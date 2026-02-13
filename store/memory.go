package store

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/JakeNeyer/ipam/network"
	"github.com/google/uuid"
)

const apiTokenPrefix = "ipam_"
const apiTokenSecretBytes = 32

const signupInviteTokenPrefix = "invite_"
const signupInviteSecretBytes = 32

// Store manages all IPAM data
type Store struct {
	organizations  map[uuid.UUID]*Organization
	environments   map[uuid.UUID]*network.Environment
	pools          map[uuid.UUID]*network.Pool
	blocks         map[uuid.UUID]*network.Block
	allocations    map[uuid.UUID]*network.Allocation
	reservedBlocks map[uuid.UUID]*ReservedBlock
	users          map[uuid.UUID]*User
	usersByEmail   map[string]uuid.UUID
	sessions       map[string]*Session
	tokens         map[uuid.UUID]*APIToken
	tokenByHash    map[string]uuid.UUID
	signupInvites  map[uuid.UUID]*SignupInvite
	inviteByHash   map[string]uuid.UUID
	mu             sync.RWMutex
}

// NewStore creates a new store
func NewStore() *Store {
	return &Store{
		organizations:  make(map[uuid.UUID]*Organization),
		environments:   make(map[uuid.UUID]*network.Environment),
		pools:          make(map[uuid.UUID]*network.Pool),
		blocks:         make(map[uuid.UUID]*network.Block),
		allocations:    make(map[uuid.UUID]*network.Allocation),
		reservedBlocks: make(map[uuid.UUID]*ReservedBlock),
		users:          make(map[uuid.UUID]*User),
		usersByEmail:   make(map[string]uuid.UUID),
		sessions:       make(map[string]*Session),
		tokens:         make(map[uuid.UUID]*APIToken),
		tokenByHash:    make(map[string]uuid.UUID),
		signupInvites:  make(map[uuid.UUID]*SignupInvite),
		inviteByHash:   make(map[string]uuid.UUID),
	}
}

// GenerateID generates a unique ID
func (s *Store) GenerateID() uuid.UUID {
	return uuid.New()
}

// Organization operations
func (s *Store) CreateOrganization(org *Organization) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if org.ID == uuid.Nil {
		org.ID = s.GenerateID()
	}
	if org.CreatedAt.IsZero() {
		org.CreatedAt = time.Now()
	}
	s.organizations[org.ID] = org
	return nil
}

func (s *Store) GetOrganization(id uuid.UUID) (*Organization, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	org, exists := s.organizations[id]
	if !exists {
		return nil, fmt.Errorf("organization not found")
	}
	return org, nil
}

func (s *Store) ListOrganizations() ([]*Organization, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*Organization
	for _, o := range s.organizations {
		out = append(out, o)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *Store) UpdateOrganization(org *Organization) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, exists := s.organizations[org.ID]
	if !exists {
		return fmt.Errorf("organization not found")
	}
	existing.Name = org.Name
	return nil
}

func (s *Store) DeleteOrganization(id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.organizations[id]; !exists {
		return fmt.Errorf("organization not found")
	}
	// Cascade: environments (and their blocks, allocations) → reserved blocks → signup invites → api tokens → users → organization
	var orgEnvIDs []uuid.UUID
	for _, env := range s.environments {
		if env.OrganizationID == id {
			orgEnvIDs = append(orgEnvIDs, env.Id)
		}
	}
	for _, envID := range orgEnvIDs {
		for pid, pool := range s.pools {
			if pool.EnvironmentID == envID {
				delete(s.pools, pid)
			}
		}
		for bid, block := range s.blocks {
			if block.EnvironmentID != envID {
				continue
			}
			blockName := strings.TrimSpace(block.Name)
			for aid, alloc := range s.allocations {
				if strings.EqualFold(strings.TrimSpace(alloc.Block.Name), blockName) {
					delete(s.allocations, aid)
				}
			}
			delete(s.blocks, bid)
		}
		delete(s.environments, envID)
	}
	// Orphan blocks scoped to this org
	var orphanBlockIDs []uuid.UUID
	for bid, block := range s.blocks {
		if block.EnvironmentID == uuid.Nil && block.OrganizationID == id {
			orphanBlockIDs = append(orphanBlockIDs, bid)
		}
	}
	for _, bid := range orphanBlockIDs {
		block := s.blocks[bid]
		if block == nil {
			continue
		}
		blockName := strings.TrimSpace(block.Name)
		for aid, alloc := range s.allocations {
			if strings.EqualFold(strings.TrimSpace(alloc.Block.Name), blockName) {
				delete(s.allocations, aid)
			}
		}
		delete(s.blocks, bid)
	}
	var reservedIDsToDelete []uuid.UUID
	for rid, r := range s.reservedBlocks {
		if r.OrganizationID == id {
			reservedIDsToDelete = append(reservedIDsToDelete, rid)
		}
	}
	for _, rid := range reservedIDsToDelete {
		delete(s.reservedBlocks, rid)
	}
	var inviteIDsToDelete []uuid.UUID
	for invID, inv := range s.signupInvites {
		if inv != nil && inv.OrganizationID == id {
			inviteIDsToDelete = append(inviteIDsToDelete, invID)
		}
	}
	for _, invID := range inviteIDsToDelete {
		if inv := s.signupInvites[invID]; inv != nil {
			delete(s.inviteByHash, inv.TokenHash)
		}
		delete(s.signupInvites, invID)
	}
	var orgUserIDs []uuid.UUID
	for uid, u := range s.users {
		if u.OrganizationID == id {
			orgUserIDs = append(orgUserIDs, uid)
		}
	}
	orgUserIDSet := make(map[uuid.UUID]bool)
	for _, uid := range orgUserIDs {
		orgUserIDSet[uid] = true
	}
	var tokenIDsToDelete []uuid.UUID
	var sessionIDsToDelete []string
	for tokenID, tok := range s.tokens {
		if tok != nil && orgUserIDSet[tok.UserID] {
			tokenIDsToDelete = append(tokenIDsToDelete, tokenID)
		}
	}
	for sid, sess := range s.sessions {
		if sess != nil && orgUserIDSet[sess.UserID] {
			sessionIDsToDelete = append(sessionIDsToDelete, sid)
		}
	}
	for _, tokenID := range tokenIDsToDelete {
		if tok := s.tokens[tokenID]; tok != nil {
			delete(s.tokenByHash, tok.KeyHash)
		}
		delete(s.tokens, tokenID)
	}
	for _, sid := range sessionIDsToDelete {
		delete(s.sessions, sid)
	}
	for _, uid := range orgUserIDs {
		u := s.users[uid]
		if u != nil {
			delete(s.usersByEmail, strings.ToLower(strings.TrimSpace(u.Email)))
		}
		delete(s.users, uid)
	}
	delete(s.organizations, id)
	return nil
}

// Environment operations
func (s *Store) CreateEnvironment(env *network.Environment) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.environments[env.Id] = env
	return nil
}

func (s *Store) GetEnvironment(id uuid.UUID) (*network.Environment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	env, exists := s.environments[id]
	if !exists {
		return nil, fmt.Errorf("environment not found")
	}
	return env, nil
}

func (s *Store) ListEnvironments() ([]*network.Environment, error) {
	envs, _, err := s.ListEnvironmentsFiltered("", nil, 0, 0)
	return envs, err
}

// ListEnvironmentsFiltered returns environments matching name (substring, case-insensitive), optionally scoped by organizationID.
// If organizationID is nil, all environments are returned (global admin). If limit <= 0, no limit is applied. offset is 0-based.
func (s *Store) ListEnvironmentsFiltered(name string, organizationID *uuid.UUID, limit, offset int) ([]*network.Environment, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	nameLower := strings.ToLower(strings.TrimSpace(name))
	var matched []*network.Environment
	for _, env := range s.environments {
		if organizationID != nil && env.OrganizationID != *organizationID {
			continue
		}
		if nameLower == "" || strings.Contains(strings.ToLower(env.Name), nameLower) {
			matched = append(matched, env)
		}
	}
	sort.Slice(matched, func(i, j int) bool { return matched[i].Name < matched[j].Name })
	total := len(matched)
	if offset > len(matched) {
		return nil, total, nil
	}
	end := offset + limit
	if limit <= 0 || end > len(matched) {
		end = len(matched)
	}
	return matched[offset:end], total, nil
}

func (s *Store) UpdateEnvironment(id uuid.UUID, env *network.Environment) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.environments[id]; !exists {
		return fmt.Errorf("environment not found")
	}
	s.environments[id] = env
	return nil
}

// DeleteEnvironment removes the environment, its pools, all blocks that belong to it, and all allocations in those blocks.
func (s *Store) DeleteEnvironment(id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.environments[id]; !exists {
		return fmt.Errorf("environment not found")
	}
	for pid, pool := range s.pools {
		if pool.EnvironmentID == id {
			delete(s.pools, pid)
		}
	}
	for bid, block := range s.blocks {
		if block.EnvironmentID == id {
			blockName := strings.TrimSpace(block.Name)
			for aid, alloc := range s.allocations {
				if strings.EqualFold(strings.TrimSpace(alloc.Block.Name), blockName) {
					delete(s.allocations, aid)
				}
			}
			delete(s.blocks, bid)
		}
	}
	delete(s.environments, id)
	return nil
}

// Pool operations
func (s *Store) CreatePool(pool *network.Pool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if pool.ID == uuid.Nil {
		pool.ID = s.GenerateID()
	}
	s.pools[pool.ID] = pool
	return nil
}

func (s *Store) GetPool(id uuid.UUID) (*network.Pool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	pool, exists := s.pools[id]
	if !exists {
		return nil, fmt.Errorf("pool not found")
	}
	return pool, nil
}

func (s *Store) ListPoolsByEnvironment(envID uuid.UUID) ([]*network.Pool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*network.Pool
	for _, pool := range s.pools {
		if pool.EnvironmentID == envID {
			out = append(out, pool)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *Store) ListPoolsByOrganization(orgID uuid.UUID) ([]*network.Pool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*network.Pool
	for _, pool := range s.pools {
		if pool.OrganizationID == orgID {
			out = append(out, pool)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *Store) UpdatePool(id uuid.UUID, pool *network.Pool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.pools[id]; !exists {
		return fmt.Errorf("pool not found")
	}
	s.pools[id] = pool
	return nil
}

func (s *Store) DeletePool(id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.pools[id]; !exists {
		return fmt.Errorf("pool not found")
	}
	delete(s.pools, id)
	return nil
}

// Block operations
func (s *Store) CreateBlock(block *network.Block) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if block.ID == uuid.Nil {
		block.ID = s.GenerateID()
	}
	s.blocks[block.ID] = block
	return nil
}

func (s *Store) GetBlock(id uuid.UUID) (*network.Block, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	block, exists := s.blocks[id]
	if !exists {
		return nil, fmt.Errorf("block not found")
	}
	return block, nil
}

func (s *Store) ListBlocks() ([]*network.Block, error) {
	blocks, _, err := s.ListBlocksFiltered("", nil, nil, nil, false, 0, 0)
	return blocks, err
}

// ListBlocksFiltered returns blocks matching name (substring), optionally environmentID, poolID, organizationID, and orphaned only.
// If organizationID != nil, only blocks in envs belonging to that org are returned. If limit <= 0, no limit is applied.
func (s *Store) ListBlocksFiltered(name string, environmentID *uuid.UUID, poolID *uuid.UUID, organizationID *uuid.UUID, orphanedOnly bool, limit, offset int) ([]*network.Block, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	nameLower := strings.ToLower(strings.TrimSpace(name))
	var matched []*network.Block
	for _, block := range s.blocks {
		if orphanedOnly && block.EnvironmentID != uuid.Nil {
			continue
		}
		if environmentID != nil && block.EnvironmentID != *environmentID {
			continue
		}
		if poolID != nil && (block.PoolID == nil || *block.PoolID != *poolID) {
			continue
		}
		if organizationID != nil {
			if block.EnvironmentID != uuid.Nil {
				env, exists := s.environments[block.EnvironmentID]
				if !exists || env.OrganizationID != *organizationID {
					continue
				}
			} else {
				if block.OrganizationID != *organizationID {
					continue
				}
			}
		}
		if nameLower == "" || strings.Contains(strings.ToLower(block.Name), nameLower) {
			matched = append(matched, block)
		}
	}
	sort.Slice(matched, func(i, j int) bool { return matched[i].Name < matched[j].Name })
	total := len(matched)
	if offset > len(matched) {
		return nil, total, nil
	}
	end := offset + limit
	if limit <= 0 || end > len(matched) {
		end = len(matched)
	}
	return matched[offset:end], total, nil
}

// ListBlocksByEnvironment returns all blocks belonging to the given environment.
func (s *Store) ListBlocksByEnvironment(envID uuid.UUID) ([]*network.Block, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*network.Block
	for _, block := range s.blocks {
		if block.EnvironmentID == envID {
			out = append(out, block)
		}
	}
	return out, nil
}

// ListBlocksByPool returns all blocks assigned to the given pool.
func (s *Store) ListBlocksByPool(poolID uuid.UUID) ([]*network.Block, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*network.Block
	for _, block := range s.blocks {
		if block.PoolID != nil && *block.PoolID == poolID {
			out = append(out, block)
		}
	}
	return out, nil
}

func (s *Store) UpdateBlock(id uuid.UUID, block *network.Block) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.blocks[id]; !exists {
		return fmt.Errorf("block not found")
	}
	s.blocks[id] = block
	return nil
}

func (s *Store) DeleteBlock(id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.blocks[id]; !exists {
		return fmt.Errorf("block not found")
	}
	delete(s.blocks, id)
	return nil
}

// Allocation operations
func (s *Store) CreateAllocation(id uuid.UUID, alloc *network.Allocation) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.allocations[id] = alloc
	return nil
}

func (s *Store) GetAllocation(id uuid.UUID) (*network.Allocation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	alloc, exists := s.allocations[id]
	if !exists {
		return nil, fmt.Errorf("allocation not found")
	}
	return alloc, nil
}

func (s *Store) ListAllocations() ([]*network.Allocation, error) {
	allocs, _, err := s.ListAllocationsFiltered("", "", uuid.Nil, nil, 0, 0)
	return allocs, err
}

// ListAllocationsFiltered returns allocations matching name (substring), optionally blockName, environmentID, and organizationID.
// When organizationID != nil, only allocations in blocks belonging to that org are returned (blocks in envs in that org, or orphan blocks with that organization_id).
func (s *Store) ListAllocationsFiltered(name string, blockName string, environmentID uuid.UUID, organizationID *uuid.UUID, limit, offset int) ([]*network.Allocation, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	nameLower := strings.ToLower(strings.TrimSpace(name))
	blockLower := strings.ToLower(strings.TrimSpace(blockName))
	var blockNamesOK map[string]bool
	if environmentID != uuid.Nil {
		blockNamesOK = make(map[string]bool)
		for _, block := range s.blocks {
			if block.EnvironmentID != environmentID {
				continue
			}
			if organizationID != nil {
				env, exists := s.environments[block.EnvironmentID]
				if !exists || env.OrganizationID != *organizationID {
					continue
				}
			}
			blockNamesOK[strings.ToLower(strings.TrimSpace(block.Name))] = true
		}
	} else if organizationID != nil {
		blockNamesOK = make(map[string]bool)
		for _, block := range s.blocks {
			if block.EnvironmentID != uuid.Nil {
				env, exists := s.environments[block.EnvironmentID]
				if !exists || env.OrganizationID != *organizationID {
					continue
				}
			} else {
				if block.OrganizationID != *organizationID {
					continue
				}
			}
			blockNamesOK[strings.ToLower(strings.TrimSpace(block.Name))] = true
		}
	}
	var matched []*network.Allocation
	for _, alloc := range s.allocations {
		if blockLower != "" && strings.ToLower(strings.TrimSpace(alloc.Block.Name)) != blockLower {
			continue
		}
		if blockNamesOK != nil && !blockNamesOK[strings.ToLower(strings.TrimSpace(alloc.Block.Name))] {
			continue
		}
		if nameLower == "" || strings.Contains(strings.ToLower(alloc.Name), nameLower) {
			matched = append(matched, alloc)
		}
	}
	sort.Slice(matched, func(i, j int) bool { return matched[i].Name < matched[j].Name })
	total := len(matched)
	if offset > len(matched) {
		return nil, total, nil
	}
	end := offset + limit
	if limit <= 0 || end > len(matched) {
		end = len(matched)
	}
	return matched[offset:end], total, nil
}

func (s *Store) UpdateAllocation(id uuid.UUID, alloc *network.Allocation) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.allocations[id]; !exists {
		return fmt.Errorf("allocation not found")
	}
	s.allocations[id] = alloc
	return nil
}

func (s *Store) DeleteAllocation(id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.allocations[id]; !exists {
		return fmt.Errorf("allocation not found")
	}
	delete(s.allocations, id)
	return nil
}

// ReservedBlock operations (blacklisted CIDR ranges; cannot be used as blocks or allocations).
func (s *Store) ListReservedBlocks(organizationID *uuid.UUID) ([]*ReservedBlock, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*ReservedBlock
	for _, r := range s.reservedBlocks {
		if organizationID != nil && r.OrganizationID != *organizationID {
			continue
		}
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CIDR < out[j].CIDR })
	return out, nil
}

func (s *Store) CreateReservedBlock(r *ReservedBlock) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r.ID == uuid.Nil {
		r.ID = s.GenerateID()
	}
	if r.CreatedAt.IsZero() {
		r.CreatedAt = time.Now()
	}
	s.reservedBlocks[r.ID] = r
	return nil
}

func (s *Store) GetReservedBlock(id uuid.UUID) (*ReservedBlock, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, exists := s.reservedBlocks[id]
	if !exists {
		return nil, fmt.Errorf("reserved block not found")
	}
	return r, nil
}

func (s *Store) UpdateReservedBlock(id uuid.UUID, r *ReservedBlock) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.reservedBlocks[id]; !exists {
		return fmt.Errorf("reserved block not found")
	}
	s.reservedBlocks[id] = r
	return nil
}

func (s *Store) DeleteReservedBlock(id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.reservedBlocks[id]; !exists {
		return fmt.Errorf("reserved block not found")
	}
	delete(s.reservedBlocks, id)
	return nil
}

// OverlapsReservedBlock returns the first reserved block that overlaps the given CIDR, or nil.
func (s *Store) OverlapsReservedBlock(cidr string, organizationID *uuid.UUID) (*ReservedBlock, error) {
	list, err := s.ListReservedBlocks(organizationID)
	if err != nil {
		return nil, err
	}
	for _, r := range list {
		overlap, err := network.Overlaps(cidr, r.CIDR)
		if err != nil {
			return nil, err
		}
		if overlap {
			return r, nil
		}
	}
	return nil, nil
}

// User operations
func (s *Store) CreateUser(u *User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if u.ID == uuid.Nil {
		u.ID = s.GenerateID()
	}
	emailKey := strings.ToLower(strings.TrimSpace(u.Email))
	if _, exists := s.usersByEmail[emailKey]; exists {
		return fmt.Errorf("user with email already exists")
	}
	s.users[u.ID] = u
	s.usersByEmail[emailKey] = u.ID
	return nil
}

func (s *Store) GetUser(id uuid.UUID) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, exists := s.users[id]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}
	return u, nil
}

func (s *Store) GetUserByEmail(email string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, exists := s.usersByEmail[strings.ToLower(strings.TrimSpace(email))]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}
	return s.users[id], nil
}

func (s *Store) GetUserByOAuth(provider, providerUserID string) (*User, error) {
	if provider == "" || providerUserID == "" {
		return nil, fmt.Errorf("user not found")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, u := range s.users {
		if u.OAuthProvider == provider && u.OAuthProviderUserID == providerUserID {
			return u, nil
		}
	}
	return nil, fmt.Errorf("user not found")
}

func (s *Store) SetUserOAuth(userID uuid.UUID, provider, providerUserID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, exists := s.users[userID]
	if !exists {
		return fmt.Errorf("user not found")
	}
	u.OAuthProvider = provider
	u.OAuthProviderUserID = providerUserID
	return nil
}

func (s *Store) ListUsers(organizationID *uuid.UUID) ([]*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*User
	for _, u := range s.users {
		if organizationID != nil && u.OrganizationID != *organizationID {
			continue
		}
		out = append(out, u)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Email < out[j].Email })
	return out, nil
}

func (s *Store) DeleteUser(userID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	u, exists := s.users[userID]
	if !exists {
		return fmt.Errorf("user not found")
	}

	delete(s.users, userID)
	delete(s.usersByEmail, strings.ToLower(strings.TrimSpace(u.Email)))

	for sid, sess := range s.sessions {
		if sess != nil && sess.UserID == userID {
			delete(s.sessions, sid)
		}
	}

	for tokenID, tok := range s.tokens {
		if tok != nil && tok.UserID == userID {
			delete(s.tokenByHash, tok.KeyHash)
			delete(s.tokens, tokenID)
		}
	}

	for inviteID, inv := range s.signupInvites {
		if inv == nil {
			continue
		}
		if inv.CreatedBy == userID {
			delete(s.inviteByHash, inv.TokenHash)
			delete(s.signupInvites, inviteID)
			continue
		}
		if inv.UsedByUserID != nil && *inv.UsedByUserID == userID {
			inv.UsedByUserID = nil
		}
	}
	return nil
}

func (s *Store) SetUserRole(userID uuid.UUID, role string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, exists := s.users[userID]
	if !exists {
		return fmt.Errorf("user not found")
	}
	if role != RoleUser && role != RoleAdmin {
		return fmt.Errorf("invalid role")
	}
	u.Role = role
	return nil
}

func (s *Store) SetUserOrganization(userID uuid.UUID, organizationID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, exists := s.users[userID]
	if !exists {
		return fmt.Errorf("user not found")
	}
	u.OrganizationID = organizationID
	return nil
}

// SetUserTourCompleted marks the onboarding tour as completed for the user.
func (s *Store) SetUserTourCompleted(userID uuid.UUID, completed bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, exists := s.users[userID]
	if !exists {
		return fmt.Errorf("user not found")
	}
	u.TourCompleted = completed
	return nil
}

// Session operations
func (s *Store) CreateSession(sessionID string, userID uuid.UUID, expiry time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[sessionID] = &Session{UserID: userID, Expiry: expiry}
}

func (s *Store) GetSession(sessionID string) (*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sess, exists := s.sessions[sessionID]
	if !exists || sess.Expired() {
		return nil, fmt.Errorf("session not found or expired")
	}
	return sess, nil
}

func (s *Store) DeleteSession(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, sessionID)
}

// hashToken returns the SHA-256 hex hash of the raw token.
func hashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

// CreateAPIToken creates a new API token for the user. Returns the raw token (only shown once).
// expiresAt is optional; nil means the token never expires.
// organizationID is optional; when set (global admin only), the token is scoped to that org.
func (s *Store) CreateAPIToken(userID uuid.UUID, name string, expiresAt *time.Time, organizationID *uuid.UUID) (token *APIToken, rawToken string, err error) {
	secret := make([]byte, apiTokenSecretBytes)
	if _, err := rand.Read(secret); err != nil {
		return nil, "", err
	}
	rawToken = apiTokenPrefix + hex.EncodeToString(secret)
	keyHash := hashToken(rawToken)

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.users[userID]; !exists {
		return nil, "", fmt.Errorf("user not found")
	}
	orgID := uuid.Nil
	if organizationID != nil {
		orgID = *organizationID
	}
	id := s.GenerateID()
	token = &APIToken{
		ID:             id,
		UserID:         userID,
		Name:           strings.TrimSpace(name),
		KeyHash:        keyHash,
		CreatedAt:      time.Now(),
		ExpiresAt:      expiresAt,
		OrganizationID: orgID,
	}
	s.tokens[id] = token
	s.tokenByHash[keyHash] = id
	return token, rawToken, nil
}

// GetUserByTokenHash returns the user for the given token hash, or nil if not found or token expired.
func (s *Store) GetUserByTokenHash(keyHash string) (*User, error) {
	tok, err := s.GetAPITokenByKeyHash(keyHash)
	if err != nil {
		return nil, err
	}
	s.mu.RLock()
	user, exists := s.users[tok.UserID]
	s.mu.RUnlock()
	if !exists {
		return nil, fmt.Errorf("user not found")
	}
	return user, nil
}

// GetAPITokenByKeyHash returns the API token for the given key hash, or error if not found or expired.
func (s *Store) GetAPITokenByKeyHash(keyHash string) (*APIToken, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tokenID, exists := s.tokenByHash[keyHash]
	if !exists {
		return nil, fmt.Errorf("token not found")
	}
	tok, exists := s.tokens[tokenID]
	if !exists {
		return nil, fmt.Errorf("token not found")
	}
	if tok.ExpiresAt != nil && time.Now().After(*tok.ExpiresAt) {
		return nil, fmt.Errorf("token expired")
	}
	return tok, nil
}

// ListAPITokens returns all API tokens for the user (without secret).
func (s *Store) ListAPITokens(userID uuid.UUID) ([]*APIToken, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*APIToken
	for _, t := range s.tokens {
		if t.UserID == userID {
			out = append(out, t)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

// DeleteAPIToken removes the token. Returns error if token not found or not owned by user.
func (s *Store) DeleteAPIToken(tokenID, userID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tok, exists := s.tokens[tokenID]
	if !exists {
		return fmt.Errorf("token not found")
	}
	if tok.UserID != userID {
		return fmt.Errorf("token not found")
	}
	delete(s.tokens, tokenID)
	delete(s.tokenByHash, tok.KeyHash)
	return nil
}

// GetAPIToken returns the token by ID (for ownership check).
func (s *Store) GetAPIToken(tokenID uuid.UUID) (*APIToken, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tok, exists := s.tokens[tokenID]
	if !exists {
		return nil, fmt.Errorf("token not found")
	}
	return tok, nil
}

// CreateSignupInvite creates a time-bound signup invite. Returns the invite and raw token (only shown once).
func (s *Store) CreateSignupInvite(createdBy uuid.UUID, expiresAt time.Time, organizationID uuid.UUID, role string) (*SignupInvite, string, error) {
	secret := make([]byte, signupInviteSecretBytes)
	if _, err := rand.Read(secret); err != nil {
		return nil, "", err
	}
	rawToken := signupInviteTokenPrefix + hex.EncodeToString(secret)
	tokenHash := hashToken(rawToken)

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.users[createdBy]; !exists {
		return nil, "", fmt.Errorf("user not found")
	}
	now := time.Now()
	if expiresAt.Before(now) {
		return nil, "", fmt.Errorf("expires_at must be in the future")
	}
	inv := &SignupInvite{
		ID:             s.GenerateID(),
		TokenHash:      tokenHash,
		CreatedBy:      createdBy,
		ExpiresAt:      expiresAt,
		CreatedAt:      now,
		OrganizationID: organizationID,
		Role:           role,
	}
	s.signupInvites[inv.ID] = inv
	s.inviteByHash[tokenHash] = inv.ID
	return inv, rawToken, nil
}

// GetSignupInviteByToken returns the invite for the given raw token if valid and not expired.
func (s *Store) GetSignupInviteByToken(rawToken string) (*SignupInvite, error) {
	if rawToken == "" || !strings.HasPrefix(rawToken, signupInviteTokenPrefix) {
		return nil, fmt.Errorf("invalid token")
	}
	tokenHash := hashToken(rawToken)
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, exists := s.inviteByHash[tokenHash]
	if !exists {
		return nil, fmt.Errorf("invite not found")
	}
	inv, exists := s.signupInvites[id]
	if !exists {
		return nil, fmt.Errorf("invite not found")
	}
	if inv.UsedAt != nil {
		return nil, fmt.Errorf("invite already used")
	}
	if time.Now().After(inv.ExpiresAt) {
		return nil, fmt.Errorf("invite expired")
	}
	return inv, nil
}

// MarkSignupInviteUsed marks the invite as used by the given user (on signup).
func (s *Store) MarkSignupInviteUsed(inviteID, userID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	inv, exists := s.signupInvites[inviteID]
	if !exists {
		return fmt.Errorf("invite not found")
	}
	now := time.Now()
	inv.UsedAt = &now
	inv.UsedByUserID = &userID
	return nil
}

// DeleteSignupInvite removes the invite (revoke).
func (s *Store) DeleteSignupInvite(id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	inv, exists := s.signupInvites[id]
	if !exists {
		return fmt.Errorf("invite not found")
	}
	delete(s.signupInvites, id)
	delete(s.inviteByHash, inv.TokenHash)
	return nil
}

// ListSignupInvites returns all signup invites created by the given user (for admin UI).
func (s *Store) ListSignupInvites(createdBy uuid.UUID) ([]*SignupInvite, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*SignupInvite
	for _, inv := range s.signupInvites {
		if inv.CreatedBy == createdBy {
			out = append(out, inv)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}
