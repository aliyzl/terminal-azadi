package tui

import (
	"charm.land/bubbles/v2/help"
	"charm.land/lipgloss/v2"
)

// maxHelpWidth is the maximum width for the help overlay box.
const maxHelpWidth = 50

// helpModel wraps the bubbles help component for overlay rendering.
type helpModel struct {
	help help.Model
	keys keyMap
}

// newHelpModel creates a help model with the given key bindings.
func newHelpModel(keys keyMap) helpModel {
	h := help.New()
	h.ShowAll = true
	return helpModel{
		help: h,
		keys: keys,
	}
}

// Render draws a centered bordered box with all keybindings on top of the
// existing content. The overlay replaces the content entirely (full repaint).
func (m helpModel) Render(_ string, width, height int) string {
	helpText := m.help.FullHelpView(m.keys.FullHelp())

	helpBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(DefaultTheme.Accent.Dark).
		Padding(1, 2).
		Width(maxHelpWidth).
		Render(helpText)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, helpBox)
}
