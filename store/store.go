package store

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/JakeNeyer/ipam/network"
	"github.com/google/uuid"
)

// Store manages all IPAM data
type Store struct {
	environments map[uuid.UUID]*network.Environment
	blocks       map[uuid.UUID]*network.Block
	allocations  map[uuid.UUID]*network.Allocation
	users        map[uuid.UUID]*User
	usersByEmail map[string]uuid.UUID
	sessions     map[string]*Session
	mu           sync.RWMutex
}

// NewStore creates a new store
func NewStore() *Store {
	return &Store{
		environments: make(map[uuid.UUID]*network.Environment),
		blocks:       make(map[uuid.UUID]*network.Block),
		allocations:  make(map[uuid.UUID]*network.Allocation),
		users:        make(map[uuid.UUID]*User),
		usersByEmail: make(map[string]uuid.UUID),
		sessions:     make(map[string]*Session),
	}
}

// GenerateID generates a unique ID
func (s *Store) GenerateID() uuid.UUID {
	return uuid.New()
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
	envs, _, err := s.ListEnvironmentsFiltered("", 0, 0)
	return envs, err
}

// ListEnvironmentsFiltered returns environments matching name (substring, case-insensitive).
// If limit <= 0, no limit is applied. offset is 0-based. Returns (items, total, error).
func (s *Store) ListEnvironmentsFiltered(name string, limit, offset int) ([]*network.Environment, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	nameLower := strings.ToLower(strings.TrimSpace(name))
	var matched []*network.Environment
	for _, env := range s.environments {
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

// DeleteEnvironment removes the environment, all blocks that belong to it, and all allocations in those blocks.
func (s *Store) DeleteEnvironment(id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.environments[id]; !exists {
		return fmt.Errorf("environment not found")
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
	blocks, _, err := s.ListBlocksFiltered("", nil, false, 0, 0)
	return blocks, err
}

// ListBlocksFiltered returns blocks matching name (substring), optionally environmentID, and optionally orphaned only.
// If limit <= 0, no limit is applied. offset is 0-based. Returns (items, total, error).
func (s *Store) ListBlocksFiltered(name string, environmentID *uuid.UUID, orphanedOnly bool, limit, offset int) ([]*network.Block, int, error) {
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
	allocs, _, err := s.ListAllocationsFiltered("", "", 0, 0)
	return allocs, err
}

// ListAllocationsFiltered returns allocations matching name (substring) and optionally blockName.
// If limit <= 0, no limit is applied. offset is 0-based. Returns (items, total, error).
func (s *Store) ListAllocationsFiltered(name string, blockName string, limit, offset int) ([]*network.Allocation, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	nameLower := strings.ToLower(strings.TrimSpace(name))
	blockLower := strings.ToLower(strings.TrimSpace(blockName))
	var matched []*network.Allocation
	for _, alloc := range s.allocations {
		if blockLower != "" && strings.ToLower(strings.TrimSpace(alloc.Block.Name)) != blockLower {
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

func (s *Store) ListUsers() ([]*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*User
	for _, u := range s.users {
		out = append(out, u)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Email < out[j].Email })
	return out, nil
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
