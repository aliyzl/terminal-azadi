package tui

import (
	"time"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/leejooy96/azad/internal/config"
	"github.com/leejooy96/azad/internal/engine"
	"github.com/leejooy96/azad/internal/serverstore"
)

// viewState represents the current view mode of the TUI.
type viewState int

const (
	viewNormal viewState = iota
	viewHelp
	viewAddServer
	viewAddSubscription
	viewConfirmDelete
)

// Minimum terminal dimensions for usable layout.
const (
	minWidth  = 60
	minHeight = 20
)

// model is the root Bubble Tea model composing all child components.
type model struct {
	// Child models
	serverList list.Model
	detail     detailModel
	statusBar  statusBarModel
	help       helpModel
	keys       keyMap

	// State
	view   viewState
	width  int
	height int
	ready  bool

	// Shared references
	store  *serverstore.Store
	engine *engine.Engine
	cfg    *config.Config
}

// New creates the root TUI model with all child components initialized.
func New(store *serverstore.Store, eng *engine.Engine, cfg *config.Config) model {
	servers := store.List()
	items := serversToItems(servers)
	serverList := newServerList(items, 0, 0)

	keys := defaultKeyMap()
	styles := NewStyles(DefaultTheme, true) // default to dark; updated on BackgroundColorMsg

	detail := detailModel{styles: styles}
	sb := statusBarModel{styles: styles, socksPort: cfg.Proxy.SOCKSPort}
	help := newHelpModel(keys)

	// Initialize engine status
	status, srv, _ := eng.Status()
	sb.Update(status, eng.ServerName(), cfg.Proxy.SOCKSPort)
	if srv != nil {
		detail.SetServer(srv)
	}

	return model{
		serverList: serverList,
		detail:     detail,
		statusBar:  sb,
		help:       help,
		keys:       keys,
		view:       viewNormal,
		store:      store,
		engine:     eng,
		cfg:        cfg,
	}
}

// tickCmd returns a command that sends a tickMsg every second for uptime display.
func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// Init starts the uptime ticker.
func (m model) Init() tea.Cmd {
	return tickCmd()
}

// Update handles messages and routes them to appropriate handlers.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

		listWidth := m.width / 3
		contentHeight := m.height - statusBarHeight

		// Size the server list panel
		m.serverList.SetSize(listWidth, contentHeight)

		// Size the detail panel (account for border taking 1 column)
		detailWidth := m.width - listWidth - 1
		m.detail.SetSize(detailWidth, contentHeight)

		// Size the status bar
		m.statusBar.SetSize(m.width)

		return m, nil

	case tickMsg:
		// Refresh status bar for uptime counter
		status, _, _ := m.engine.Status()
		if status == engine.StatusConnected {
			// Status bar will recalculate uptime in its View
		}
		return m, tickCmd()

	case pingResultMsg:
		// Update latency in store and refresh list items
		return m, nil

	case allPingsCompleteMsg:
		return m, nil

	case connectResultMsg:
		status, srv, _ := m.engine.Status()
		m.statusBar.Update(status, m.engine.ServerName(), m.cfg.Proxy.SOCKSPort)
		if msg.Err == nil {
			m.statusBar.SetConnectedAt(time.Now())
		}
		if srv != nil {
			m.detail.SetServer(srv)
		}
		return m, nil

	case disconnectMsg:
		m.statusBar.Update(engine.StatusDisconnected, "", m.cfg.Proxy.SOCKSPort)
		m.statusBar.SetConnectedAt(time.Time{})
		return m, nil

	case errMsg:
		// Could show in status bar as notification
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKeyPress(msg)
	}

	// Pass other messages to the server list
	var cmd tea.Cmd
	m.serverList, cmd = m.serverList.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// handleKeyPress routes key presses based on current view state.
func (m model) handleKeyPress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.Keystroke()

	switch m.view {
	case viewHelp:
		switch key {
		case "?", "esc", "q":
			m.view = viewNormal
			return m, nil
		}
		return m, nil

	case viewAddServer, viewAddSubscription:
		// Placeholder: delegate to input model (implemented in Plan 03)
		if key == "esc" {
			m.view = viewNormal
			return m, nil
		}
		return m, nil

	case viewConfirmDelete:
		if key == "esc" {
			m.view = viewNormal
			return m, nil
		}
		return m, nil

	case viewNormal:
		// If the list is actively filtering, let it handle most keys
		if m.serverList.FilterState() == list.Filtering {
			var cmd tea.Cmd
			m.serverList, cmd = m.serverList.Update(msg)
			m.syncDetail()
			return m, cmd
		}

		switch key {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "?":
			m.view = viewHelp
			return m, nil

		case "enter":
			// Get selected server and update detail panel
			m.syncDetail()
			return m, nil

		case "esc":
			// If filter is applied, let list handle clearing it
			if m.serverList.FilterState() == list.FilterApplied {
				var cmd tea.Cmd
				m.serverList, cmd = m.serverList.Update(msg)
				m.syncDetail()
				return m, cmd
			}
			return m, nil

		default:
			// Route all other keys to the server list (j/k, /, etc.)
			var cmd tea.Cmd
			m.serverList, cmd = m.serverList.Update(msg)
			m.syncDetail()
			return m, cmd
		}
	}

	return m, nil
}

// syncDetail updates the detail panel with the currently selected server.
func (m *model) syncDetail() {
	if item, ok := m.serverList.SelectedItem().(serverItem); ok {
		m.detail.SetServer(&item.server)
	}
}

// View renders the split-pane TUI layout.
func (m model) View() tea.View {
	if !m.ready {
		v := tea.NewView("Initializing...")
		v.AltScreen = true
		return v
	}

	if m.width < minWidth || m.height < minHeight {
		v := tea.NewView("Terminal too small. Resize to at least 60x20.")
		v.AltScreen = true
		return v
	}

	listWidth := m.width / 3
	detailWidth := m.width - listWidth - 1
	contentHeight := m.height - statusBarHeight

	// Style the list panel with a right border
	listStyle := lipgloss.NewStyle().
		Width(listWidth).
		Height(contentHeight).
		BorderRight(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(DefaultTheme.Border.Dark)

	// Style the detail panel
	detailStyle := lipgloss.NewStyle().
		Width(detailWidth).
		Height(contentHeight)

	// Render panels
	listPanel := listStyle.Render(m.serverList.View())
	detailPanel := detailStyle.Render(m.detail.View())

	// Compose horizontal layout
	main := lipgloss.JoinHorizontal(lipgloss.Top, listPanel, detailPanel)

	// Add status bar below
	content := lipgloss.JoinVertical(lipgloss.Left, main, m.statusBar.View())

	// Overlay help if active
	if m.view == viewHelp {
		content = m.help.Render(content, m.width, m.height)
	}

	v := tea.NewView(content)
	v.AltScreen = true
	return v
}
