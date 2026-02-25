package tui

import (
	"fmt"

	"charm.land/bubbles/v2/list"
	"github.com/leejooy96/azad/internal/protocol"
)

// serverItem wraps a protocol.Server for display in a bubbles list.
// It implements the list.DefaultItem interface.
type serverItem struct {
	server protocol.Server
}

// Title returns the server name for display in the list.
func (s serverItem) Title() string {
	return s.server.Name
}

// Description returns the protocol badge and latency if measured.
func (s serverItem) Description() string {
	proto := string(s.server.Protocol)
	if s.server.LatencyMs > 0 {
		return fmt.Sprintf("%s | %dms", proto, s.server.LatencyMs)
	}
	return proto
}

// FilterValue returns a searchable string combining name, address, and protocol
// to enable fuzzy filtering by any of these fields.
func (s serverItem) FilterValue() string {
	return s.server.Name + " " + s.server.Address + " " + string(s.server.Protocol)
}

// serversToItems converts a slice of servers into list items.
func serversToItems(servers []protocol.Server) []list.Item {
	items := make([]list.Item, len(servers))
	for i, srv := range servers {
		items[i] = serverItem{server: srv}
	}
	return items
}

// newServerList creates a configured list model with filtering enabled.
func newServerList(items []list.Item, width, height int) list.Model {
	delegate := list.NewDefaultDelegate()
	l := list.New(items, delegate, width, height)
	l.Title = "Servers"
	l.SetFilteringEnabled(true)
	l.DisableQuitKeybindings()
	return l
}
