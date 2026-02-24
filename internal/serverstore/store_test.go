package serverstore

import (
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/leejooy96/azad/internal/protocol"
)

func makeServer(name, addr string, port int, proto protocol.Protocol) protocol.Server {
	return protocol.Server{
		ID:       protocol.NewID(),
		Name:     name,
		Protocol: proto,
		Address:  addr,
		Port:     port,
		AddedAt:  time.Now(),
	}
}

func TestSaveLoad_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "servers.json")

	s1 := New(path)

	srv1 := protocol.Server{
		ID:                 protocol.NewID(),
		Name:               "Tokyo VLESS",
		Protocol:           protocol.ProtocolVLESS,
		Address:            "tokyo.example.com",
		Port:               443,
		UUID:               "550e8400-e29b-41d4-a716-446655440000",
		Network:            "ws",
		TLS:                "tls",
		SNI:                "tokyo.example.com",
		SubscriptionSource: "https://sub.example.com/api",
		AddedAt:            time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC),
	}
	srv2 := protocol.Server{
		ID:       protocol.NewID(),
		Name:     "Berlin VMess",
		Protocol: protocol.ProtocolVMess,
		Address:  "berlin.example.com",
		Port:     8080,
		UUID:     "660e8400-e29b-41d4-a716-446655440000",
		AlterID:  0,
		Security: "auto",
		Network:  "tcp",
		AddedAt:  time.Date(2026, 2, 1, 8, 0, 0, 0, time.UTC),
	}

	if err := s1.Add(srv1); err != nil {
		t.Fatalf("Add srv1: %v", err)
	}
	if err := s1.Add(srv2); err != nil {
		t.Fatalf("Add srv2: %v", err)
	}

	// Create new store and Load
	s2 := New(path)
	if err := s2.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	list := s2.List()
	if len(list) != 2 {
		t.Fatalf("expected 2 servers, got %d", len(list))
	}

	// Verify first server fields survived round-trip
	var found bool
	for _, s := range list {
		if s.Name == "Tokyo VLESS" {
			found = true
			if s.Protocol != protocol.ProtocolVLESS {
				t.Errorf("protocol: got %q, want %q", s.Protocol, protocol.ProtocolVLESS)
			}
			if s.Address != "tokyo.example.com" {
				t.Errorf("address: got %q, want %q", s.Address, "tokyo.example.com")
			}
			if s.Port != 443 {
				t.Errorf("port: got %d, want %d", s.Port, 443)
			}
			if s.UUID != "550e8400-e29b-41d4-a716-446655440000" {
				t.Errorf("uuid: got %q", s.UUID)
			}
			if s.Network != "ws" {
				t.Errorf("network: got %q, want %q", s.Network, "ws")
			}
			if s.TLS != "tls" {
				t.Errorf("tls: got %q, want %q", s.TLS, "tls")
			}
			if s.SNI != "tokyo.example.com" {
				t.Errorf("sni: got %q", s.SNI)
			}
			if s.SubscriptionSource != "https://sub.example.com/api" {
				t.Errorf("subscription_source: got %q", s.SubscriptionSource)
			}
			if !s.AddedAt.Equal(time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)) {
				t.Errorf("added_at: got %v", s.AddedAt)
			}
		}
	}
	if !found {
		t.Error("Tokyo VLESS server not found after round-trip")
	}
}

func TestAdd(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "servers.json")

	s := New(path)
	srv := makeServer("Test Server", "1.2.3.4", 443, protocol.ProtocolVLESS)

	if err := s.Add(srv); err != nil {
		t.Fatalf("Add: %v", err)
	}

	list := s.List()
	if len(list) != 1 {
		t.Fatalf("expected 1 server, got %d", len(list))
	}
	if list[0].Name != "Test Server" {
		t.Errorf("name: got %q, want %q", list[0].Name, "Test Server")
	}

	// Verify file exists on disk
	s2 := New(path)
	if err := s2.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if s2.Count() != 1 {
		t.Errorf("expected 1 server on disk, got %d", s2.Count())
	}
}

func TestRemove(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "servers.json")

	s := New(path)
	srv1 := makeServer("Server A", "1.1.1.1", 443, protocol.ProtocolVLESS)
	srv2 := makeServer("Server B", "2.2.2.2", 8080, protocol.ProtocolVMess)

	if err := s.Add(srv1); err != nil {
		t.Fatalf("Add srv1: %v", err)
	}
	if err := s.Add(srv2); err != nil {
		t.Fatalf("Add srv2: %v", err)
	}

	if err := s.Remove(srv1.ID); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	list := s.List()
	if len(list) != 1 {
		t.Fatalf("expected 1 server, got %d", len(list))
	}
	if list[0].Name != "Server B" {
		t.Errorf("remaining server: got %q, want %q", list[0].Name, "Server B")
	}
}

func TestClear(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "servers.json")

	s := New(path)
	for i := 0; i < 3; i++ {
		srv := makeServer("Server", "1.1.1.1", 443, protocol.ProtocolVLESS)
		if err := s.Add(srv); err != nil {
			t.Fatalf("Add: %v", err)
		}
	}

	if err := s.Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}

	if len(s.List()) != 0 {
		t.Error("expected empty list after Clear")
	}

	// Verify file on disk has empty array
	s2 := New(path)
	if err := s2.Load(); err != nil {
		t.Fatalf("Load after clear: %v", err)
	}
	if s2.Count() != 0 {
		t.Errorf("expected 0 servers on disk, got %d", s2.Count())
	}
}

func TestFindByID(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "servers.json")

	s := New(path)
	srv := makeServer("Findable", "5.5.5.5", 443, protocol.ProtocolTrojan)

	if err := s.Add(srv); err != nil {
		t.Fatalf("Add: %v", err)
	}

	found, ok := s.FindByID(srv.ID)
	if !ok {
		t.Fatal("expected FindByID to return true")
	}
	if found.Name != "Findable" {
		t.Errorf("name: got %q, want %q", found.Name, "Findable")
	}

	_, ok = s.FindByID("nonexistent-id")
	if ok {
		t.Error("expected FindByID to return false for unknown ID")
	}
}

func TestLoadEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.json")

	s := New(path)
	if err := s.Load(); err != nil {
		t.Fatalf("Load from non-existent file should not error, got: %v", err)
	}

	if len(s.List()) != 0 {
		t.Error("expected empty list for non-existent file")
	}
}

func TestReplaceBySource(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "servers.json")

	s := New(path)

	// Add 2 servers from "sub1"
	sub1a := makeServer("Sub1 A", "1.1.1.1", 443, protocol.ProtocolVLESS)
	sub1a.SubscriptionSource = "sub1"
	sub1b := makeServer("Sub1 B", "2.2.2.2", 443, protocol.ProtocolVLESS)
	sub1b.SubscriptionSource = "sub1"

	// Add 1 server from "manual"
	manual := makeServer("Manual", "3.3.3.3", 8080, protocol.ProtocolVMess)
	manual.SubscriptionSource = "manual"

	for _, srv := range []protocol.Server{sub1a, sub1b, manual} {
		if err := s.Add(srv); err != nil {
			t.Fatalf("Add: %v", err)
		}
	}

	// Replace sub1 servers with new ones
	newSub1 := []protocol.Server{
		makeServer("New Sub1 X", "10.10.10.10", 443, protocol.ProtocolTrojan),
		makeServer("New Sub1 Y", "20.20.20.20", 443, protocol.ProtocolTrojan),
	}
	for i := range newSub1 {
		newSub1[i].SubscriptionSource = "sub1"
	}

	if err := s.ReplaceBySource("sub1", newSub1); err != nil {
		t.Fatalf("ReplaceBySource: %v", err)
	}

	list := s.List()
	if len(list) != 3 {
		t.Fatalf("expected 3 servers (1 manual + 2 new sub1), got %d", len(list))
	}

	// Verify manual server untouched
	var manualFound bool
	var sub1Count int
	for _, srv := range list {
		if srv.Name == "Manual" {
			manualFound = true
		}
		if srv.SubscriptionSource == "sub1" {
			sub1Count++
		}
	}
	if !manualFound {
		t.Error("manual server was removed during ReplaceBySource")
	}
	if sub1Count != 2 {
		t.Errorf("expected 2 sub1 servers, got %d", sub1Count)
	}
}

func TestConcurrentAccess(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "servers.json")

	s := New(path)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			srv := makeServer("Concurrent", "1.1.1.1", 443, protocol.ProtocolVLESS)
			_ = s.Add(srv)
		}()
	}
	wg.Wait()

	if s.Count() != 10 {
		t.Errorf("expected 10 servers after concurrent adds, got %d", s.Count())
	}
}
