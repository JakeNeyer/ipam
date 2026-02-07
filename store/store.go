package store

import (
	"fmt"
	"sync"

	"github.com/JakeNeyer/ipam/network"
	"github.com/google/uuid"
)

// Store manages all IPAM data
type Store struct {
	environments map[uuid.UUID]*network.Environment
	blocks       map[uuid.UUID]*network.Block
	allocations  map[uuid.UUID]*network.Allocation
	mu           sync.RWMutex
}

// NewStore creates a new store
func NewStore() *Store {
	return &Store{
		environments: make(map[uuid.UUID]*network.Environment),
		blocks:       make(map[uuid.UUID]*network.Block),
		allocations:  make(map[uuid.UUID]*network.Allocation),
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
	s.mu.RLock()
	defer s.mu.RUnlock()
	envs := make([]*network.Environment, 0, len(s.environments))
	for _, env := range s.environments {
		envs = append(envs, env)
	}
	return envs, nil
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

func (s *Store) DeleteEnvironment(id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.environments[id]; !exists {
		return fmt.Errorf("environment not found")
	}
	delete(s.environments, id)
	return nil
}

// Block operations
func (s *Store) CreateBlock(block *network.Block) error {
	s.mu.Lock()
	defer s.mu.Unlock()
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
	s.mu.RLock()
	defer s.mu.RUnlock()
	blocks := make([]*network.Block, 0, len(s.blocks))
	for _, block := range s.blocks {
		blocks = append(blocks, block)
	}
	return blocks, nil
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
	s.mu.RLock()
	defer s.mu.RUnlock()
	allocs := make([]*network.Allocation, 0, len(s.allocations))
	for _, alloc := range s.allocations {
		allocs = append(allocs, alloc)
	}
	return allocs, nil
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
