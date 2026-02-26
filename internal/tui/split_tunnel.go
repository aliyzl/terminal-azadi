package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/leejooy96/azad/internal/config"
	"github.com/leejooy96/azad/internal/splittunnel"
)

// renderSplitTunnelView renders the split tunnel management overlay.
func renderSplitTunnelView(m model) string {
	titleStyle := lipgloss.NewStyle().Foreground(DefaultTheme.Accent.Dark).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(DefaultTheme.Border.Dark)
	successStyle := lipgloss.NewStyle().Foreground(DefaultTheme.Success.Dark).Bold(true)
	selectedStyle := lipgloss.NewStyle().Foreground(DefaultTheme.Accent.Dark).Bold(true)

	title := titleStyle.Render("Split Tunneling")

	// Status
	var statusStr string
	if m.cfg.SplitTunnel.Enabled {
		statusStr = "Status: " + successStyle.Render("ENABLED")
	} else {
		statusStr = "Status: " + dimStyle.Render("DISABLED")
	}

	// Mode
	mode := m.cfg.SplitTunnel.Mode
	if mode == "" {
		mode = "exclusive"
	}
	var modeStr string
	if mode == "inclusive" {
		modeStr = "Mode:   Inclusive (VPN-only list)"
	} else {
		modeStr = "Mode:   Exclusive (bypass list)"
	}

	// Rules
	var rulesSection string
	if len(m.cfg.SplitTunnel.Rules) == 0 {
		rulesSection = dimStyle.Render("  (No rules configured)")
	} else {
		var lines []string
		for i, rule := range m.cfg.SplitTunnel.Rules {
			prefix := "  "
			style := lipgloss.NewStyle()
			if i == m.splitTunnelIdx {
				prefix = "> "
				style = selectedStyle
			}
			typeTag := dimStyle.Render(fmt.Sprintf("[%s]", rule.Type))
			line := style.Render(fmt.Sprintf("%s%d. %-20s", prefix, i+1, rule.Value)) + " " + typeTag
			lines = append(lines, line)
		}
		rulesSection = strings.Join(lines, "\n")
	}

	// Hints
	hint := dimStyle.Render("a \u00b7 add rule   d \u00b7 delete   e \u00b7 enable/disable\nt \u00b7 toggle mode   esc \u00b7 back")

	inner := title + "\n\n" + statusStr + "\n" + modeStr + "\n\nRules:\n" + rulesSection + "\n\n" + hint

	menuBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(DefaultTheme.Accent.Dark).
		Padding(1, 3).
		Width(48).
		Align(lipgloss.Left).
		Render(inner)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, menuBox)
}

// buildSplitTunnelConfig converts config.SplitTunnelConfig to a runtime
// splittunnel.Config. Returns nil if split tunneling is disabled or has no rules.
func buildSplitTunnelConfig(cfg *config.Config) *splittunnel.Config {
	if !cfg.SplitTunnel.Enabled || len(cfg.SplitTunnel.Rules) == 0 {
		return nil
	}

	mode := splittunnel.ModeExclusive
	if cfg.SplitTunnel.Mode == "inclusive" {
		mode = splittunnel.ModeInclusive
	}

	rules := make([]splittunnel.Rule, len(cfg.SplitTunnel.Rules))
	for i, r := range cfg.SplitTunnel.Rules {
		rules[i] = splittunnel.Rule{
			Value: r.Value,
			Type:  splittunnel.RuleType(r.Type),
		}
	}

	return &splittunnel.Config{
		Enabled: true,
		Mode:    mode,
		Rules:   rules,
	}
}

// saveSplitTunnelCmd saves the split tunnel configuration to disk.
func saveSplitTunnelCmd(cfg *config.Config) tea.Cmd {
	return func() tea.Msg {
		configPath, err := config.FilePath()
		if err == nil {
			_ = config.Save(cfg, configPath)
		}
		return splitTunnelSavedMsg{}
	}
}
