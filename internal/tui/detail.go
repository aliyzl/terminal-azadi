package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/leejooy96/azad/internal/engine"
	"github.com/leejooy96/azad/internal/protocol"
)

// detailModel renders the detail panel showing information about the selected server.
type detailModel struct {
	server     *protocol.Server
	trafficLog []engine.LogEntry
	width      int
	height     int
	styles     Styles
}

// SetServer updates the server displayed in the detail panel.
func (m *detailModel) SetServer(srv *protocol.Server) {
	m.server = srv
}

// SetTrafficLog updates the traffic log entries displayed when connected.
func (m *detailModel) SetTrafficLog(entries []engine.LogEntry) {
	m.trafficLog = entries
}

// ClearTrafficLog removes all traffic log entries.
func (m *detailModel) ClearTrafficLog() {
	m.trafficLog = nil
}

// SetSize updates the dimensions available for rendering.
func (m *detailModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// SetStyles updates the styles used for rendering.
func (m *detailModel) SetStyles(s Styles) {
	m.styles = s
}

// View renders the detail panel content.
func (m detailModel) View() string {
	if m.server == nil {
		placeholder := m.styles.Dim.Render("No server selected")
		return lipgloss.NewStyle().
			Width(m.width).
			Height(m.height).
			Align(lipgloss.Center, lipgloss.Center).
			Render(placeholder)
	}

	// If we have traffic log entries, show compact header + traffic log.
	if len(m.trafficLog) > 0 {
		return m.trafficLogView()
	}

	return m.serverDetailView()
}

// serverDetailView renders the full server detail (when disconnected).
func (m detailModel) serverDetailView() string {
	srv := m.server
	var b strings.Builder

	// Server name
	b.WriteString(m.styles.Accent.Bold(true).Render(srv.Name))
	b.WriteString("\n\n")

	// Protocol badge
	b.WriteString(m.styles.ProtocolBadge.Render(string(srv.Protocol)))
	b.WriteString("\n\n")

	// Address:Port
	b.WriteString(m.labelValue("Address", fmt.Sprintf("%s:%d", srv.Address, srv.Port)))

	// Transport
	if srv.Network != "" {
		transport := srv.Network
		if srv.Path != "" {
			transport += " " + srv.Path
		}
		b.WriteString(m.labelValue("Transport", transport))
	}

	// TLS info
	if srv.TLS != "" && srv.TLS != "none" {
		tls := srv.TLS
		if srv.SNI != "" {
			tls += " (SNI: " + srv.SNI + ")"
		}
		if srv.PublicKey != "" {
			tls += " [REALITY]"
			if srv.Fingerprint != "" {
				tls += " fp:" + srv.Fingerprint
			}
		}
		b.WriteString(m.labelValue("TLS", tls))
	}

	// Flow
	if srv.Flow != "" {
		b.WriteString(m.labelValue("Flow", srv.Flow))
	}

	// Subscription source
	if srv.SubscriptionSource != "" {
		b.WriteString(m.labelValue("Source", srv.SubscriptionSource))
	}

	// Last connected
	if !srv.LastConnected.IsZero() {
		b.WriteString(m.labelValue("Last connected", srv.LastConnected.Format("2006-01-02 15:04")))
	}

	// Latency
	if srv.LatencyMs > 0 {
		latency := fmt.Sprintf("%dms", srv.LatencyMs)
		b.WriteString(m.labelValue("Latency", latency))
	}

	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		PaddingLeft(2).
		PaddingTop(1).
		Render(b.String())
}

// trafficLogView renders a compact server header followed by scrollable traffic log.
func (m detailModel) trafficLogView() string {
	srv := m.server
	var b strings.Builder

	// Compact server header: name + protocol badge
	name := m.styles.Accent.Bold(true).Render(srv.Name)
	badge := m.styles.ProtocolBadge.Render(string(srv.Protocol))
	b.WriteString(name + "  " + badge)
	b.WriteString("\n")

	// Separator
	titleStyle := lipgloss.NewStyle().Foreground(DefaultTheme.Accent.Dark).Bold(true)
	dimStyle := m.styles.Dim
	b.WriteString("\n")
	b.WriteString(titleStyle.Render("Traffic Log"))
	b.WriteString("\n")

	// Available width for content (minus padding)
	contentWidth := m.width - 4
	if contentWidth < 40 {
		contentWidth = 40
	}
	b.WriteString(dimStyle.Render(strings.Repeat("─", contentWidth)))
	b.WriteString("\n")

	// Calculate how many log entries fit.
	// Header uses 4 lines (name+badge, blank, title, separator).
	// Each entry uses 1 line.
	headerLines := 4
	maxEntries := m.height - headerLines - 2 // padding
	if maxEntries < 1 {
		maxEntries = 1
	}

	// Show the most recent entries that fit.
	entries := m.trafficLog
	if len(entries) > maxEntries {
		entries = entries[len(entries)-maxEntries:]
	}

	// Determine max domain width for alignment.
	maxDomain := 0
	for _, e := range entries {
		if len(e.Domain) > maxDomain {
			maxDomain = len(e.Domain)
		}
	}
	// Cap domain width to prevent overflow.
	domainCap := contentWidth - 18 // time(8) + spaces(4) + route(6)
	if domainCap < 20 {
		domainCap = 20
	}
	if maxDomain > domainCap {
		maxDomain = domainCap
	}

	timeStyle := dimStyle
	domainStyle := m.styles.Normal
	routeProxyStyle := m.styles.Success
	routeDirectStyle := dimStyle

	for _, e := range entries {
		domain := e.Domain
		if len(domain) > maxDomain {
			domain = domain[:maxDomain-1] + "…"
		}

		routeStyle := routeProxyStyle
		if e.Route == "direct" {
			routeStyle = routeDirectStyle
		}

		line := timeStyle.Render(e.Time) + "  " +
			domainStyle.Render(fmt.Sprintf("%-*s", maxDomain, domain)) + "  " +
			routeStyle.Render(e.Route)
		b.WriteString(line)
		b.WriteString("\n")
	}

	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		PaddingLeft(2).
		PaddingTop(1).
		Render(b.String())
}

// labelValue formats a label-value pair for the detail view.
func (m detailModel) labelValue(label, value string) string {
	l := m.styles.Dim.Render(label + ":")
	v := m.styles.Normal.Render(value)
	return l + " " + v + "\n"
}
