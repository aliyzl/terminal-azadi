package tui

import (
	"charm.land/bubbles/v2/key"
)

// keyMap defines all keybindings for the TUI.
// It implements the help.KeyMap interface via ShortHelp and FullHelp.
type keyMap struct {
	Up         key.Binding
	Down       key.Binding
	Select     key.Binding
	Back       key.Binding
	Quit       key.Binding
	Help       key.Binding
	Filter     key.Binding
	PingAll    key.Binding
	AddServer  key.Binding
	AddSub     key.Binding
	RefreshSub key.Binding
	Delete     key.Binding
	ClearAll key.Binding
	Connect  key.Binding
	Menu     key.Binding
}

// defaultKeyMap returns the default set of keybindings.
func defaultKeyMap() keyMap {
	return keyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("j/k", "navigate"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("j/k", "navigate"),
		),
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "connect"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter"),
		),
		PingAll: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "ping all"),
		),
		AddServer: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "add server"),
		),
		AddSub: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "add subscription"),
		),
		RefreshSub: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh subscriptions"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete"),
		),
		ClearAll: key.NewBinding(
			key.WithKeys("D"),
			key.WithHelp("D", "clear all"),
		),
		Connect: key.NewBinding(
			key.WithKeys("enter", "c"),
			key.WithHelp("enter/c", "connect"),
		),
		Menu: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "menu"),
		),
	}
}

// ShortHelp returns a condensed set of key bindings for the short help view.
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit, k.Select, k.Filter}
}

// FullHelp returns the complete set of key bindings organized by group.
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Select, k.Back},
		{k.Filter, k.PingAll, k.Connect},
		{k.AddServer, k.AddSub, k.RefreshSub},
		{k.Delete, k.ClearAll, k.Menu, k.Quit, k.Help},
	}
}
