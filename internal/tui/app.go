package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/leejooy96/azad/internal/config"
	"github.com/leejooy96/azad/internal/engine"
	"github.com/leejooy96/azad/internal/killswitch"
	"github.com/leejooy96/azad/internal/serverstore"
	"github.com/leejooy96/azad/internal/splittunnel"
)

// viewState represents the current view mode of the TUI.
type viewState int

const (
	viewNormal viewState = iota
	viewHelp
	viewAddServer
	viewAddSubscription
	viewConfirmDelete
	viewMenu
	viewConfirmKillSwitch
	viewSplitTunnel
	viewAddSplitRule
)

// Minimum terminal dimensions for usable layout.
const (
	minWidth  = 60
	minHeight = 20
)

// hintBarHeight is the number of terminal rows the hint bar occupies.
const hintBarHeight = 1

// model is the root Bubble Tea model composing all child components.
type model struct {
	// Child models
	serverList list.Model
	detail     detailModel
	statusBar  statusBarModel
	help       helpModel
	input      inputModel
	keys       keyMap

	// State
	view             viewState
	width            int
	height           int
	ready            bool
	pinging          bool
	pingTotal        int
	pingDone         int
	pingLatencies    map[string]int // serverID -> latencyMs (-1 = error)
	confirmDelete    bool
	killSwitchActive bool
	splitTunnelIdx   int // selected rule index in split tunnel view

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

	// Check if kill switch is active from a previous session (recovery).
	ksActive := killswitch.IsActive()
	if ksActive {
		sb.SetKillSwitch(true)
	}

	// Check if split tunneling is enabled in config.
	if cfg.SplitTunnel.Enabled && len(cfg.SplitTunnel.Rules) > 0 {
		sb.SetSplitTunnel(true)
	}

	return model{
		serverList:       serverList,
		detail:           detail,
		statusBar:        sb,
		help:             help,
		input:            newInputModel(),
		keys:             keys,
		view:             viewNormal,
		pingLatencies:    make(map[string]int),
		killSwitchActive: ksActive,
		store:            store,
		engine:           eng,
		cfg:              cfg,
	}
}

// tickCmd returns a command that sends a tickMsg every second for uptime display.
func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// Init starts the uptime ticker and fires auto-connect on startup.
func (m model) Init() tea.Cmd {
	sc := buildSplitTunnelConfig(m.cfg)
	return tea.Batch(tickCmd(), autoConnectCmd(m.store, m.engine, m.cfg, sc))
}

// Update handles messages and routes them to appropriate handlers.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m.resizeContent()
		return m, nil

	case tickMsg:
		// Refresh status bar for uptime counter and poll traffic log.
		status, _, _ := m.engine.Status()
		if status == engine.StatusConnected {
			entries := m.engine.TrafficLog(m.detail.height)
			m.detail.SetTrafficLog(entries)
		}
		return m, tickCmd()

	case pingResultMsg:
		m.pingDone++
		m.pingLatencies[msg.ServerID] = msg.LatencyMs
		if m.pingDone >= m.pingTotal {
			// All pings complete: sort servers by latency and rebuild list
			m.pinging = false
			m.serverList.Title = "Servers"
			m.rebuildListSortedByLatency()
		} else {
			m.serverList.Title = fmt.Sprintf("Servers (pinging %d/%d...)", m.pingDone, m.pingTotal)
		}
		return m, nil

	case allPingsCompleteMsg:
		return m, nil

	case serverAddedMsg:
		m.view = viewNormal
		m.reloadServers()
		return m, nil

	case serverRemovedMsg:
		m.reloadServers()
		return m, nil

	case subscriptionFetchedMsg:
		if msg.Err != nil {
			m.input.err = msg.Err
			return m, nil
		}
		m.view = viewNormal
		m.reloadServers()
		return m, nil

	case serversReplacedMsg:
		m.reloadServers()
		return m, nil

	case autoConnectMsg:
		if msg.ServerID == "" || msg.Err != nil {
			// Skipped (empty store) or failed -- do nothing.
			return m, nil
		}
		// Auto-connect succeeded: update status bar and select the connected server.
		m.statusBar.Update(engine.StatusConnected, m.engine.ServerName(), m.cfg.Proxy.SOCKSPort)
		m.statusBar.SetConnectedAt(time.Now())
		m.selectServerByID(msg.ServerID)
		m.syncDetail()
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
		// Auto-select the connected server in the list and sync detail.
		if srv != nil {
			m.selectServerByID(srv.ID)
		}
		m.syncDetail()
		return m, nil

	case disconnectMsg:
		m.statusBar.Update(engine.StatusDisconnected, "", m.cfg.Proxy.SOCKSPort)
		m.statusBar.SetConnectedAt(time.Time{})
		m.detail.ClearTrafficLog()
		m.killSwitchActive = false
		m.statusBar.SetKillSwitch(false)
		if m.ready {
			m.resizeContent()
		}
		return m, nil

	case killSwitchResultMsg:
		if msg.Err != nil {
			m.input.err = msg.Err
			return m, nil
		}
		m.killSwitchActive = msg.Enabled
		m.statusBar.SetKillSwitch(msg.Enabled)
		if m.ready {
			m.resizeContent()
		}
		return m, nil

	case splitTunnelSavedMsg:
		// Config saved silently; update status bar to reflect new state.
		m.statusBar.SetSplitTunnel(m.cfg.SplitTunnel.Enabled && len(m.cfg.SplitTunnel.Rules) > 0)
		return m, nil

	case errMsg:
		// Show error on input modal if active, otherwise ignore for now
		if m.view == viewAddServer || m.view == viewAddSubscription {
			m.input.err = msg.Err
		}
		return m, nil

	case tea.PasteMsg:
		if m.view == viewAddServer || m.view == viewAddSubscription || m.view == viewAddSplitRule {
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKeyPress(msg)
	}

	// Pass other messages to the active child model based on view state
	var cmd tea.Cmd
	if m.view == viewAddServer || m.view == viewAddSubscription || m.view == viewAddSplitRule {
		m.input, cmd = m.input.Update(msg)
	} else {
		m.serverList, cmd = m.serverList.Update(msg)
	}
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

	case viewAddServer:
		switch key {
		case "esc":
			m.view = viewNormal
			return m, nil
		case "enter":
			value := m.input.Value()
			if value == "" {
				return m, nil
			}
			return m, addServerCmd(value, m.store)
		default:
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}

	case viewAddSubscription:
		switch key {
		case "esc":
			m.view = viewNormal
			return m, nil
		case "enter":
			value := m.input.Value()
			if value == "" {
				return m, nil
			}
			return m, addSubscriptionCmd(value, m.store)
		default:
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}

	case viewConfirmDelete:
		switch key {
		case "y", "enter":
			m.view = viewNormal
			return m, clearAllCmd(m.store)
		case "n", "esc":
			m.view = viewNormal
			return m, nil
		}
		return m, nil

	case viewMenu:
		switch key {
		case "enter", " ":
			if m.killSwitchActive {
				m.view = viewNormal
				return m, disableKillSwitchCmd()
			}
			m.view = viewConfirmKillSwitch
			return m, nil
		case "t":
			m.view = viewSplitTunnel
			m.splitTunnelIdx = 0
			return m, nil
		case "esc", "m":
			m.view = viewNormal
			return m, nil
		}
		return m, nil

	case viewSplitTunnel:
		switch key {
		case "esc":
			m.view = viewMenu
			return m, nil
		case "j", "down":
			ruleCount := len(m.cfg.SplitTunnel.Rules)
			if ruleCount > 0 && m.splitTunnelIdx < ruleCount-1 {
				m.splitTunnelIdx++
			}
			return m, nil
		case "k", "up":
			if m.splitTunnelIdx > 0 {
				m.splitTunnelIdx--
			}
			return m, nil
		case "a":
			m.view = viewAddSplitRule
			cmd := m.input.SetMode(inputAddSplitRule)
			return m, cmd
		case "d":
			ruleCount := len(m.cfg.SplitTunnel.Rules)
			if ruleCount > 0 && m.splitTunnelIdx < ruleCount {
				m.cfg.SplitTunnel.Rules = append(
					m.cfg.SplitTunnel.Rules[:m.splitTunnelIdx],
					m.cfg.SplitTunnel.Rules[m.splitTunnelIdx+1:]...,
				)
				if m.splitTunnelIdx >= len(m.cfg.SplitTunnel.Rules) && m.splitTunnelIdx > 0 {
					m.splitTunnelIdx--
				}
				return m, saveSplitTunnelCmd(m.cfg)
			}
			return m, nil
		case "e":
			m.cfg.SplitTunnel.Enabled = !m.cfg.SplitTunnel.Enabled
			m.statusBar.SetSplitTunnel(m.cfg.SplitTunnel.Enabled && len(m.cfg.SplitTunnel.Rules) > 0)
			return m, saveSplitTunnelCmd(m.cfg)
		case "t":
			if m.cfg.SplitTunnel.Mode == "inclusive" {
				m.cfg.SplitTunnel.Mode = "exclusive"
			} else {
				m.cfg.SplitTunnel.Mode = "inclusive"
			}
			return m, saveSplitTunnelCmd(m.cfg)
		}
		return m, nil

	case viewAddSplitRule:
		switch key {
		case "esc":
			m.view = viewSplitTunnel
			return m, nil
		case "enter":
			value := m.input.Value()
			if value == "" {
				return m, nil
			}
			rule, err := splittunnel.ParseRule(value)
			if err != nil {
				m.input.err = err
				return m, nil
			}
			m.cfg.SplitTunnel.Rules = append(m.cfg.SplitTunnel.Rules, config.SplitTunnelRule{
				Value: rule.Value,
				Type:  string(rule.Type),
			})
			m.view = viewSplitTunnel
			return m, saveSplitTunnelCmd(m.cfg)
		default:
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}

	case viewConfirmKillSwitch:
		switch key {
		case "y", "enter":
			m.view = viewNormal
			bypass := extractBypassIPs(m.cfg)
			return m, enableKillSwitchCmd(m.engine, m.cfg, bypass)
		case "n", "esc":
			m.view = viewMenu
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
			status, _, _ := m.engine.Status()
			if status == engine.StatusConnected || status == engine.StatusConnecting {
				return m, tea.Sequence(disconnectCmd(m.engine, m.killSwitchActive), tea.Quit)
			}
			if m.killSwitchActive {
				return m, tea.Sequence(disableKillSwitchCmd(), tea.Quit)
			}
			return m, tea.Quit

		case "?":
			m.view = viewHelp
			return m, nil

		case "a":
			m.view = viewAddServer
			cmd := m.input.SetMode(inputAddServer)
			return m, cmd

		case "s":
			m.view = viewAddSubscription
			cmd := m.input.SetMode(inputAddSubscription)
			return m, cmd

		case "r":
			return m, refreshSubscriptionsCmd(m.store)

		case "d":
			if item, ok := m.serverList.SelectedItem().(serverItem); ok {
				return m, removeServerCmd(m.store, item.server.ID)
			}
			return m, nil

		case "D":
			m.view = viewConfirmDelete
			return m, nil

		case "m":
			m.view = viewMenu
			return m, nil

		case "p":
			if !m.pinging {
				servers := m.store.List()
				if len(servers) > 0 {
					m.pinging = true
					m.pingTotal = len(servers)
					m.pingDone = 0
					m.pingLatencies = make(map[string]int)
					m.serverList.Title = fmt.Sprintf("Servers (pinging 0/%d...)", m.pingTotal)
					return m, pingAllCmd(servers)
				}
			}
			return m, nil

		case "enter", "c":
			if item, ok := m.serverList.SelectedItem().(serverItem); ok {
				sc := buildSplitTunnelConfig(m.cfg)
				status, _, _ := m.engine.Status()
				if status == engine.StatusConnected || status == engine.StatusConnecting {
					// Already connected: disconnect first, then reconnect.
					return m, tea.Sequence(disconnectCmd(m.engine), connectServerCmd(item.server, m.engine, m.cfg, m.store, sc))
				}
				m.statusBar.Update(engine.StatusConnecting, item.server.Name, m.cfg.Proxy.SOCKSPort)
				return m, connectServerCmd(item.server, m.engine, m.cfg, m.store, sc)
			}
			return m, nil

		case "x":
			status, _, _ := m.engine.Status()
			if status == engine.StatusConnected || status == engine.StatusConnecting {
				return m, tea.Sequence(disconnectCmd(m.engine, m.killSwitchActive), tea.ClearScreen)
			}
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

// selectServerByID finds the server with the given ID in the list and selects it.
func (m *model) selectServerByID(id string) {
	items := m.serverList.Items()
	for i, item := range items {
		if si, ok := item.(serverItem); ok && si.server.ID == id {
			m.serverList.Select(i)
			return
		}
	}
}

// syncDetail updates the detail panel with the currently selected server.
func (m *model) syncDetail() {
	if item, ok := m.serverList.SelectedItem().(serverItem); ok {
		m.detail.SetServer(&item.server)
	}
}

// resizeContent recalculates child model sizes based on current dimensions
// and kill switch banner visibility.
func (m *model) resizeContent() {
	bannerH := 0
	if m.killSwitchActive {
		bannerH = 1
	}
	listWidth := m.width / 3
	contentHeight := m.height - statusBarHeight - hintBarHeight - bannerH
	m.serverList.SetSize(listWidth, contentHeight)
	detailWidth := m.width - listWidth - 1
	m.detail.SetSize(detailWidth, contentHeight)
	m.statusBar.SetSize(m.width)
}

// reloadServers refreshes the list items from the store, preserving the
// selected index when possible.
func (m *model) reloadServers() {
	idx := m.serverList.Index()
	servers := m.store.List()
	items := serversToItems(servers)
	m.serverList.SetItems(items)
	if idx >= len(items) && len(items) > 0 {
		idx = len(items) - 1
	}
	if len(items) > 0 {
		m.serverList.Select(idx)
	}
	m.syncDetail()
}

// rebuildListSortedByLatency rebuilds the server list sorted by ping latency
// (ascending). Servers with errors or no result are placed last.
func (m *model) rebuildListSortedByLatency() {
	servers := m.store.List()

	sort.Slice(servers, func(i, j int) bool {
		li, oki := m.pingLatencies[servers[i].ID]
		lj, okj := m.pingLatencies[servers[j].ID]

		// Servers without results go last
		if !oki && !okj {
			return false
		}
		if !oki {
			return false
		}
		if !okj {
			return true
		}

		// Errored pings (latency -1) go last among pinged servers
		if li < 0 && lj < 0 {
			return false
		}
		if li < 0 {
			return false
		}
		if lj < 0 {
			return true
		}

		return li < lj
	})

	// Update LatencyMs on the server items for display
	for i := range servers {
		if latency, ok := m.pingLatencies[servers[i].ID]; ok && latency >= 0 {
			servers[i].LatencyMs = latency
		}
	}

	items := serversToItems(servers)
	m.serverList.SetItems(items)
	if len(items) > 0 {
		m.serverList.Select(0)
	}
	m.syncDetail()
}

// hintBar returns a context-aware shortcut hint line based on the current view.
func (m model) hintBar() string {
	keyStyle := lipgloss.NewStyle().Foreground(DefaultTheme.Accent.Dark).Bold(true)
	sepStyle := lipgloss.NewStyle().Foreground(DefaultTheme.Border.Dark)
	descStyle := lipgloss.NewStyle().Foreground(DefaultTheme.Border.Dark)
	sep := sepStyle.Render(" · ")

	hint := func(k, desc string) string {
		return keyStyle.Render(k) + descStyle.Render(" "+desc)
	}

	var hints string
	switch m.view {
	case viewNormal:
		status, _, _ := m.engine.Status()
		if status == engine.StatusConnected || status == engine.StatusConnecting {
			hints = hint("↑↓", "navigate") + sep +
				hint("enter", "switch") + sep +
				hint("x", "disconnect") + sep +
				hint("m", "menu") + sep +
				hint("a", "add") + sep +
				hint("?", "help") + sep +
				hint("q", "quit")
		} else {
			hints = hint("↑↓", "navigate") + sep +
				hint("enter", "connect") + sep +
				hint("m", "menu") + sep +
				hint("a", "add") + sep +
				hint("p", "ping") + sep +
				hint("?", "help") + sep +
				hint("q", "quit")
		}
	case viewMenu:
		hints = hint("enter", "toggle KS") + sep +
			hint("t", "split tunnel") + sep +
			hint("esc", "close")
	case viewSplitTunnel:
		hints = hint("a", "add") + sep +
			hint("d", "delete") + sep +
			hint("e", "enable/disable") + sep +
			hint("t", "toggle mode") + sep +
			hint("esc", "back")
	case viewConfirmDelete:
		hints = hint("y", "confirm") + sep + hint("n", "cancel")
	case viewConfirmKillSwitch:
		hints = hint("y", "enable") + sep + hint("n", "cancel")
	default:
		return ""
	}

	return lipgloss.NewStyle().
		Width(m.width).
		Align(lipgloss.Center).
		Render(hints)
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

	// Kill switch active banner
	var bannerHeight int
	var banner string
	if m.killSwitchActive {
		bannerHeight = 1
		banner = lipgloss.NewStyle().
			Background(DefaultTheme.Error.Dark).
			Foreground(lipgloss.Color("255")).
			Bold(true).
			Width(m.width).
			Align(lipgloss.Center).
			Render("⛨  KILL SWITCH ACTIVE — All non-VPN traffic blocked")
	}

	listWidth := m.width / 3
	detailWidth := m.width - listWidth - 1
	contentHeight := m.height - statusBarHeight - hintBarHeight - bannerHeight

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

	// Compose vertical layout with optional kill switch banner and hint bar
	hintLine := m.hintBar()
	var content string
	if banner != "" {
		content = lipgloss.JoinVertical(lipgloss.Left, banner, main, hintLine, m.statusBar.View())
	} else {
		content = lipgloss.JoinVertical(lipgloss.Left, main, hintLine, m.statusBar.View())
	}

	// Overlay modals based on view state
	switch m.view {
	case viewHelp:
		content = m.help.Render(content, m.width, m.height)

	case viewAddServer, viewAddSubscription, viewAddSplitRule:
		content = m.input.View(m.width, m.height)

	case viewMenu:
		titleStyle := lipgloss.NewStyle().Foreground(DefaultTheme.Accent.Dark).Bold(true)
		dimStyle := lipgloss.NewStyle().Foreground(DefaultTheme.Border.Dark)

		title := titleStyle.Render("Settings")

		ksIcon := "\u26E8  Kill Switch"
		var ksStatus string
		if m.killSwitchActive {
			ksStatus = lipgloss.NewStyle().Foreground(DefaultTheme.Success.Dark).Bold(true).Render("\u25CF ACTIVE")
		} else {
			ksStatus = dimStyle.Render("\u25CB OFF")
		}

		innerWidth := 36
		padLen := innerWidth - lipgloss.Width(ksIcon) - lipgloss.Width(ksStatus)
		if padLen < 2 {
			padLen = 2
		}
		ksRow := ksIcon + strings.Repeat(" ", padLen) + ksStatus

		// Split tunnel row
		stIcon := "\u21C4  Split Tunnel"
		var stStatus string
		if m.cfg.SplitTunnel.Enabled && len(m.cfg.SplitTunnel.Rules) > 0 {
			stStatus = lipgloss.NewStyle().Foreground(DefaultTheme.Success.Dark).Bold(true).Render("\u25CF ENABLED")
		} else {
			stStatus = dimStyle.Render("\u25CB OFF")
		}
		stPadLen := innerWidth - lipgloss.Width(stIcon) - lipgloss.Width(stStatus)
		if stPadLen < 2 {
			stPadLen = 2
		}
		stRow := stIcon + strings.Repeat(" ", stPadLen) + stStatus

		hint := dimStyle.Render("enter \u00b7 toggle KS    t \u00b7 split tunnel    esc \u00b7 close")

		inner := title + "\n\n" + ksRow + "\n" + stRow + "\n\n" + hint
		menuBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(DefaultTheme.Accent.Dark).
			Padding(1, 3).
			Width(52).
			Align(lipgloss.Center).
			Render(inner)
		content = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, menuBox)

	case viewConfirmDelete:
		confirmBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(DefaultTheme.Warning.Dark).
			Padding(1, 2).
			Width(40).
			Render("Clear all servers?\n\n(y) Yes  (n) No")
		content = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, confirmBox)

	case viewConfirmKillSwitch:
		warnStyle := lipgloss.NewStyle().Foreground(DefaultTheme.Warning.Dark).Bold(true)
		dimStyle := lipgloss.NewStyle().Foreground(DefaultTheme.Border.Dark)
		successStyle := lipgloss.NewStyle().Foreground(DefaultTheme.Success.Dark).Bold(true)

		title := warnStyle.Render("KILL SWITCH")
		body := "When enabled, ALL non-VPN traffic\nis blocked at the firewall level.\n\nIf VPN drops or terminal closes,\nnothing leaks — internet stays blocked\nuntil you reconnect or run cleanup."
		note := dimStyle.Render("Requires administrator password.")
		buttons := successStyle.Render("[Y] Enable") + "    " + dimStyle.Render("[N] Cancel")
		inner := title + "\n\n" + body + "\n\n" + note + "\n\n" + buttons
		confirmBox := lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(DefaultTheme.Warning.Dark).
			Padding(1, 3).
			Width(46).
			Align(lipgloss.Center).
			Render(inner)
		content = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, confirmBox)

	case viewSplitTunnel:
		content = renderSplitTunnelView(m)
	}

	v := tea.NewView(content)
	v.AltScreen = true
	return v
}
