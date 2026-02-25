package tui

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// ColorPair holds light and dark terminal variants of a color.
type ColorPair struct {
	Light color.Color
	Dark  color.Color
}

// Theme defines the complete color palette for the TUI.
// All colors are specified as pairs for light and dark terminal backgrounds.
type Theme struct {
	Primary       ColorPair
	Secondary     ColorPair
	Accent        ColorPair
	Muted         ColorPair
	Success       ColorPair
	Warning       ColorPair
	Error         ColorPair
	Border        ColorPair
	StatusBar     ColorPair
	StatusBarText ColorPair
}

// DefaultTheme provides the default color palette using ANSI 256 colors
// for broad terminal compatibility.
var DefaultTheme = Theme{
	Primary:       ColorPair{Light: lipgloss.Color("235"), Dark: lipgloss.Color("252")},
	Secondary:     ColorPair{Light: lipgloss.Color("241"), Dark: lipgloss.Color("245")},
	Accent:        ColorPair{Light: lipgloss.Color("63"), Dark: lipgloss.Color("86")},
	Muted:         ColorPair{Light: lipgloss.Color("250"), Dark: lipgloss.Color("238")},
	Success:       ColorPair{Light: lipgloss.Color("34"), Dark: lipgloss.Color("78")},
	Warning:       ColorPair{Light: lipgloss.Color("208"), Dark: lipgloss.Color("214")},
	Error:         ColorPair{Light: lipgloss.Color("196"), Dark: lipgloss.Color("204")},
	Border:        ColorPair{Light: lipgloss.Color("245"), Dark: lipgloss.Color("240")},
	StatusBar:     ColorPair{Light: lipgloss.Color("235"), Dark: lipgloss.Color("236")},
	StatusBarText: ColorPair{Light: lipgloss.Color("252"), Dark: lipgloss.Color("252")},
}

// Styles holds resolved lipgloss styles for a specific terminal background.
type Styles struct {
	Title         lipgloss.Style
	Selected      lipgloss.Style
	Normal        lipgloss.Style
	Dim           lipgloss.Style
	ProtocolBadge lipgloss.Style
	StatusBar     lipgloss.Style
	Accent        lipgloss.Style
	Success       lipgloss.Style
	Warning       lipgloss.Style
	Error         lipgloss.Style
}

// Resolve returns a color from a ColorPair based on the dark background flag.
func (cp ColorPair) Resolve(isDark bool) color.Color {
	ld := lipgloss.LightDark(isDark)
	return ld(cp.Light, cp.Dark)
}

// NewStyles creates resolved styles from a theme for the given background.
func NewStyles(theme Theme, isDark bool) Styles {
	primary := theme.Primary.Resolve(isDark)
	secondary := theme.Secondary.Resolve(isDark)
	accent := theme.Accent.Resolve(isDark)
	muted := theme.Muted.Resolve(isDark)
	statusBg := theme.StatusBar.Resolve(isDark)
	statusText := theme.StatusBarText.Resolve(isDark)
	success := theme.Success.Resolve(isDark)
	warning := theme.Warning.Resolve(isDark)
	errorColor := theme.Error.Resolve(isDark)

	return Styles{
		Title: lipgloss.NewStyle().
			Foreground(accent).
			Bold(true).
			PaddingLeft(1),

		Selected: lipgloss.NewStyle().
			Foreground(primary).
			Bold(true),

		Normal: lipgloss.NewStyle().
			Foreground(secondary),

		Dim: lipgloss.NewStyle().
			Foreground(muted),

		ProtocolBadge: lipgloss.NewStyle().
			Foreground(accent).
			Bold(true),

		StatusBar: lipgloss.NewStyle().
			Foreground(statusText).
			Background(statusBg).
			PaddingLeft(1).
			PaddingRight(1),

		Accent: lipgloss.NewStyle().
			Foreground(accent),

		Success: lipgloss.NewStyle().
			Foreground(success),

		Warning: lipgloss.NewStyle().
			Foreground(warning),

		Error: lipgloss.NewStyle().
			Foreground(errorColor),
	}
}
