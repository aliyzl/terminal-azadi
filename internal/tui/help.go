package tui

import (
	"charm.land/bubbles/v2/help"
)

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

// Render draws the help overlay centered on top of the existing content.
func (m helpModel) Render(content string, width, height int) string {
	// Placeholder: full implementation in Task 2
	return m.help.FullHelpView(m.keys.FullHelp())
}
