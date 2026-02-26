package tui

import (
	"fmt"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/leejooy96/azad/internal/protocol"
	"github.com/leejooy96/azad/internal/serverstore"
	"github.com/leejooy96/azad/internal/subscription"
)

// inputMode determines which input modal is shown.
type inputMode int

const (
	inputAddServer inputMode = iota
	inputAddSubscription
	inputAddSplitRule
)

// inputModel wraps a text input for server URI and subscription URL entry.
type inputModel struct {
	textInput textinput.Model
	mode      inputMode
	err       error
	width     int
}

// newInputModel creates an input model with default settings.
func newInputModel() inputModel {
	ti := textinput.New()
	ti.Placeholder = "Paste server URI (vless://, vmess://, trojan://, ss://)"
	ti.CharLimit = 2048
	ti.SetWidth(50)

	return inputModel{
		textInput: ti,
		mode:      inputAddServer,
	}
}

// SetMode switches the input modal between add-server and add-subscription,
// clearing any previous value and error, and focusing the input.
func (m *inputModel) SetMode(mode inputMode) tea.Cmd {
	m.mode = mode
	m.err = nil
	m.textInput.SetValue("")

	switch mode {
	case inputAddServer:
		m.textInput.Placeholder = "Paste server URI (vless://, vmess://, trojan://, ss://)"
	case inputAddSubscription:
		m.textInput.Placeholder = "Paste subscription URL (https://...)"
	case inputAddSplitRule:
		m.textInput.Placeholder = "IP, CIDR, domain, or *.domain"
	}

	return m.textInput.Focus()
}

// Update handles messages for the input modal.
// Key routing (enter/esc) is handled by the root model; this only processes
// text input updates (character input, cursor movement, etc.).
func (m inputModel) Update(msg tea.Msg) (inputModel, tea.Cmd) {
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

// Value returns the current text input value.
func (m inputModel) Value() string {
	return m.textInput.Value()
}

// View renders the input modal as a bordered box with title, input field,
// error display, and hint text.
func (m inputModel) View(width, height int) string {
	title := "Add Server"
	switch m.mode {
	case inputAddSubscription:
		title = "Add Subscription"
	case inputAddSplitRule:
		title = "Add Split Tunnel Rule"
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(DefaultTheme.Accent.Dark)

	hintStyle := lipgloss.NewStyle().
		Foreground(DefaultTheme.Muted.Dark)

	content := titleStyle.Render(title) + "\n\n"
	content += m.textInput.View() + "\n"

	if m.err != nil {
		errStyle := lipgloss.NewStyle().
			Foreground(DefaultTheme.Error.Dark)
		content += "\n" + errStyle.Render(m.err.Error()) + "\n"
	}

	content += "\n" + hintStyle.Render("Enter to submit, Esc to cancel")

	modalWidth := 60
	if width > 0 && width < modalWidth+4 {
		modalWidth = width - 4
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(DefaultTheme.Accent.Dark).
		Padding(1, 2).
		Width(modalWidth).
		Render(content)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}

// addServerCmd parses a URI and adds the server to the store.
// It runs as a tea.Cmd in a goroutine and must not access model state.
func addServerCmd(uri string, store *serverstore.Store) tea.Cmd {
	return func() tea.Msg {
		srv, err := protocol.ParseURI(uri)
		if err != nil {
			return errMsg{Err: fmt.Errorf("invalid URI: %w", err)}
		}
		if err := store.Add(*srv); err != nil {
			return errMsg{Err: fmt.Errorf("adding server: %w", err)}
		}
		return serverAddedMsg{Server: *srv}
	}
}

// addSubscriptionCmd fetches servers from a subscription URL and stores them.
// It runs as a tea.Cmd in a goroutine and must not access model state.
func addSubscriptionCmd(url string, store *serverstore.Store) tea.Cmd {
	return func() tea.Msg {
		servers, err := subscription.Fetch(url)
		if err != nil {
			return subscriptionFetchedMsg{Err: err}
		}
		// Convert []*protocol.Server to []protocol.Server for store.
		values := make([]protocol.Server, len(servers))
		for i, srv := range servers {
			values[i] = *srv
		}
		if err := store.ReplaceBySource(url, values); err != nil {
			return subscriptionFetchedMsg{Err: fmt.Errorf("storing servers: %w", err)}
		}
		// Return the value-type servers for the message.
		result := make([]protocol.Server, len(servers))
		for i, srv := range servers {
			result[i] = *srv
		}
		return subscriptionFetchedMsg{Servers: result}
	}
}

// refreshSubscriptionsCmd re-fetches all known subscription sources
// and replaces their servers in the store.
// It runs as a tea.Cmd in a goroutine and must not access model state.
func refreshSubscriptionsCmd(store *serverstore.Store) tea.Cmd {
	return func() tea.Msg {
		servers := store.List()
		sources := make(map[string]bool)
		for _, srv := range servers {
			if srv.SubscriptionSource != "" {
				sources[srv.SubscriptionSource] = true
			}
		}
		if len(sources) == 0 {
			return errMsg{Err: fmt.Errorf("no subscriptions to refresh")}
		}
		total := 0
		for source := range sources {
			fetched, err := subscription.Fetch(source)
			if err != nil {
				continue // skip failed sources
			}
			values := make([]protocol.Server, len(fetched))
			for i, srv := range fetched {
				values[i] = *srv
			}
			_ = store.ReplaceBySource(source, values)
			total += len(fetched)
		}
		return serversReplacedMsg{Count: total}
	}
}
