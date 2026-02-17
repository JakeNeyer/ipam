package integrations

import (
	"fmt"
	"sync"
)

var (
	registry   = make(map[string]CloudProvider)
	registryMu sync.RWMutex
)

// Register registers a cloud provider by its ProviderID().
// It is typically called from init() of provider packages (e.g. internal/integrations/aws).
func Register(p CloudProvider) {
	registryMu.Lock()
	defer registryMu.Unlock()
	id := p.ProviderID()
	if id == "" {
		panic("integrations: provider ID must be non-empty")
	}
	if _, exists := registry[id]; exists {
		panic(fmt.Sprintf("integrations: provider %q already registered", id))
	}
	registry[id] = p
}

// Get returns the provider for the given ID, or nil if not registered.
func Get(id string) CloudProvider {
	registryMu.RLock()
	defer registryMu.RUnlock()
	return registry[id]
}

// List returns all registered provider IDs.
func List() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	ids := make([]string, 0, len(registry))
	for id := range registry {
		ids = append(ids, id)
	}
	return ids
}
