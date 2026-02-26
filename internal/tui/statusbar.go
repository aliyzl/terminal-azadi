package tui

import (
	"fmt"
	"time"

	"github.com/leejooy96/azad/internal/engine"
)

// statusBarModel renders the bottom status bar showing connection state.
type statusBarModel struct {
	status      engine.ConnectionStatus
	serverName  string
	socksPort   int
	connectedAt time.Time
	width       int
	styles      Styles
	killSwitch  bool
	splitTunnel bool
}

// Update refreshes the displayed connection state.
func (m *statusBarModel) Update(status engine.ConnectionStatus, name string, port int) {
	m.status = status
	m.serverName = name
	m.socksPort = port
}

// SetConnectedAt marks the connection start time for uptime calculation.
func (m *statusBarModel) SetConnectedAt(t time.Time) {
	m.connectedAt = t
}

// SetSize updates the status bar width.
func (m *statusBarModel) SetSize(w int) {
	m.width = w
}

// SetKillSwitch sets the kill switch indicator state.
func (m *statusBarModel) SetKillSwitch(active bool) {
	m.killSwitch = active
}

// SetSplitTunnel sets the split tunnel indicator state.
func (m *statusBarModel) SetSplitTunnel(active bool) {
	m.splitTunnel = active
}

// SetStyles updates the styles used for rendering.
func (m *statusBarModel) SetStyles(s Styles) {
	m.styles = s
}

// View renders the status bar as a single styled line.
func (m statusBarModel) View() string {
	// Connection status indicator with color
	var statusIndicator string
	switch m.status {
	case engine.StatusConnected:
		statusIndicator = m.styles.Success.Bold(true).Render("● Connected")
	case engine.StatusConnecting:
		statusIndicator = m.styles.Warning.Bold(true).Render("◌ Connecting")
	case engine.StatusError:
		statusIndicator = m.styles.Error.Bold(true).Render("✕ Error")
	default:
		statusIndicator = m.styles.Dim.Render("○ Disconnected")
	}

	// Server name
	serverDisplay := "No server"
	if m.serverName != "" {
		serverDisplay = m.serverName
	}

	// Proxy port
	portDisplay := fmt.Sprintf("SOCKS5:%d", m.socksPort)

	// Uptime (only when connected)
	var uptimeDisplay string
	if m.status == engine.StatusConnected && !m.connectedAt.IsZero() {
		d := time.Since(m.connectedAt).Truncate(time.Second)
		hours := int(d.Hours())
		minutes := int(d.Minutes()) % 60
		seconds := int(d.Seconds()) % 60
		uptimeDisplay = fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	}

	// Kill switch indicator
	var killSwitchDisplay string
	if m.killSwitch {
		killSwitchDisplay = m.styles.Error.Bold(true).Render(" ⛨ KILL SWITCH ON ")
	}

	// Split tunnel indicator
	var splitTunnelDisplay string
	if m.splitTunnel {
		splitTunnelDisplay = m.styles.Accent.Bold(true).Render(" SPLIT ")
	}

	// Compose sections
	sections := statusIndicator + "  " + serverDisplay + "  " + portDisplay
	if uptimeDisplay != "" {
		sections += "  " + uptimeDisplay
	}
	if killSwitchDisplay != "" {
		sections += "  " + killSwitchDisplay
	}
	if splitTunnelDisplay != "" {
		sections += "  " + splitTunnelDisplay
	}

	return m.styles.StatusBar.
		Width(m.width).
		Render(sections)
}

// statusBarHeight is the number of terminal rows the status bar occupies.
const statusBarHeight = 1
