package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/leejooy96/azad/internal/protocol"
)

// detailModel renders the detail panel showing information about the selected server.
type detailModel struct {
	server *protocol.Server
	width  int
	height int
	styles Styles
}

// SetServer updates the server displayed in the detail panel.
func (m *detailModel) SetServer(srv *protocol.Server) {
	m.server = srv
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

// labelValue formats a label-value pair for the detail view.
func (m detailModel) labelValue(label, value string) string {
	l := m.styles.Dim.Render(label + ":")
	v := m.styles.Normal.Render(value)
	return l + " " + v + "\n"
}
