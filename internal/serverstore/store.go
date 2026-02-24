package serverstore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

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
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			s.servers = nil
			return nil
		}
		return fmt.Errorf("reading server store: %w", err)
	}

	var servers []protocol.Server
	if err := json.Unmarshal(data, &servers); err != nil {
		return fmt.Errorf("parsing server store: %w", err)
	}
	s.servers = servers
	return nil
}

// Save writes the server list to disk atomically (temp file + rename).
func (s *Store) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.save()
}

// save is the internal unlocked save, called by methods that already hold the lock.
func (s *Store) save() error {
	data, err := json.MarshalIndent(s.servers, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling servers: %w", err)
	}

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating store directory: %w", err)
	}

	tmp, err := os.CreateTemp(dir, "servers-*.json.tmp")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	defer os.Remove(tmp.Name()) // clean up on error

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("writing temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing temp file: %w", err)
	}

	if err := os.Rename(tmp.Name(), s.path); err != nil {
		return fmt.Errorf("renaming temp to store: %w", err)
	}
	return nil
}

// Add appends a server to the store and saves.
// If the server has no ID, one is generated.
// If AddedAt is zero, it is set to the current time.
func (s *Store) Add(srv protocol.Server) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if srv.ID == "" {
		srv.ID = protocol.NewID()
	}
	if srv.AddedAt.IsZero() {
		srv.AddedAt = time.Now()
	}

	s.servers = append(s.servers, srv)
	return s.save()
}

// Remove deletes a server by ID and saves.
func (s *Store) Remove(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	found := false
	filtered := make([]protocol.Server, 0, len(s.servers))
	for _, srv := range s.servers {
		if srv.ID == id {
			found = true
			continue
		}
		filtered = append(filtered, srv)
	}

	if !found {
		return fmt.Errorf("server with ID %q not found", id)
	}

	s.servers = filtered
	return s.save()
}

// List returns a copy of all servers in the store.
func (s *Store) List() []protocol.Server {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.servers == nil {
		return nil
	}

	copy := make([]protocol.Server, len(s.servers))
	for i, srv := range s.servers {
		copy[i] = srv
	}
	return copy
}

// Clear removes all servers and saves.
func (s *Store) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.servers = nil
	return s.save()
}

// FindByID looks up a server by its ID.
func (s *Store) FindByID(id string) (*protocol.Server, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for i := range s.servers {
		if s.servers[i].ID == id {
			srv := s.servers[i] // copy
			return &srv, true
		}
	}
	return nil, false
}

// Count returns the number of servers in the store.
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.servers)
}

// ReplaceBySource removes all servers with matching SubscriptionSource
// and adds the new servers in their place, then saves.
func (s *Store) ReplaceBySource(source string, servers []protocol.Server) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Filter out servers from the given source
	filtered := make([]protocol.Server, 0, len(s.servers))
	for _, srv := range s.servers {
		if srv.SubscriptionSource != source {
			filtered = append(filtered, srv)
		}
	}

	// Add new servers with metadata defaults
	for _, srv := range servers {
		if srv.ID == "" {
			srv.ID = protocol.NewID()
		}
		if srv.AddedAt.IsZero() {
			srv.AddedAt = time.Now()
		}
		filtered = append(filtered, srv)
	}

	s.servers = filtered
	return s.save()
}
