package serverstore

import (
	"fmt"
	"sync"

	"github.com/leejooy96/azad/internal/protocol"
)

// Store manages a collection of proxy servers with atomic persistence.
type Store struct {
	mu      sync.RWMutex
	servers []protocol.Server
	path    string
}

// New creates a new Store that persists to the given file path.
func New(path string) *Store {
	return &Store{path: path}
}

// Load reads the server list from disk.
// If the file does not exist, the store is empty (not an error).
func (s *Store) Load() error {
	return fmt.Errorf("not implemented")
}

// Save writes the server list to disk atomically (temp file + rename).
func (s *Store) Save() error {
	return fmt.Errorf("not implemented")
}

// Add appends a server to the store and saves.
// If the server has no ID, one is generated.
// If AddedAt is zero, it is set to the current time.
func (s *Store) Add(srv protocol.Server) error {
	return fmt.Errorf("not implemented")
}

// Remove deletes a server by ID and saves.
func (s *Store) Remove(id string) error {
	return fmt.Errorf("not implemented")
}

// List returns a copy of all servers in the store.
func (s *Store) List() []protocol.Server {
	return nil
}

// Clear removes all servers and saves.
func (s *Store) Clear() error {
	return fmt.Errorf("not implemented")
}

// FindByID looks up a server by its ID.
func (s *Store) FindByID(id string) (*protocol.Server, bool) {
	return nil, false
}

// Count returns the number of servers in the store.
func (s *Store) Count() int {
	return 0
}

// ReplaceBySource removes all servers with matching SubscriptionSource
// and adds the new servers in their place, then saves.
func (s *Store) ReplaceBySource(source string, servers []protocol.Server) error {
	return fmt.Errorf("not implemented")
}
