package tui

import (
	"fmt"
	"net"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/leejooy96/azad/internal/protocol"
	"github.com/leejooy96/azad/internal/serverstore"
)

// pingAllCmd creates a batch of concurrent ping commands, one per server.
// Each command dials a TCP connection and reports latency or error.
func pingAllCmd(servers []protocol.Server) tea.Cmd {
	cmds := make([]tea.Cmd, len(servers))
	for i, srv := range servers {
		srv := srv // capture for closure
		cmds[i] = func() tea.Msg {
			start := time.Now()
			addr := fmt.Sprintf("%s:%d", srv.Address, srv.Port)
			conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
			if err != nil {
				return pingResultMsg{
					ServerID:  srv.ID,
					LatencyMs: -1,
					Err:       err,
				}
			}
			conn.Close()
			return pingResultMsg{
				ServerID:  srv.ID,
				LatencyMs: int(time.Since(start).Milliseconds()),
			}
		}
	}
	return tea.Batch(cmds...)
}

// removeServerCmd removes a server by ID from the store.
// It runs as a tea.Cmd in a goroutine and must not access model state.
func removeServerCmd(store *serverstore.Store, id string) tea.Cmd {
	return func() tea.Msg {
		if err := store.Remove(id); err != nil {
			return errMsg{Err: err}
		}
		return serverRemovedMsg{ServerID: id}
	}
}

// clearAllCmd removes all servers from the store.
// It runs as a tea.Cmd in a goroutine and must not access model state.
func clearAllCmd(store *serverstore.Store) tea.Cmd {
	return func() tea.Msg {
		if err := store.Clear(); err != nil {
			return errMsg{Err: err}
		}
		return serversReplacedMsg{Count: 0}
	}
}
