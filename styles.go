package bubblecomplete

import (
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/compat"
)

// Colors — Charm-inspired palette with semantic naming.
// Pink accent, proper text hierarchy, subtle alternating rows.
var (
	accentColor          = compat.AdaptiveColor{Light: lipgloss.Color("#D6116B"), Dark: lipgloss.Color("#F5639A")}
	accentBgColor        = compat.AdaptiveColor{Light: lipgloss.Color("#FDE8F0"), Dark: lipgloss.Color("#3B1D2E")}
	rowBgColor           = compat.AdaptiveColor{Light: lipgloss.Color("#E6E9EF"), Dark: lipgloss.Color("#1E1E2E")}
	altRowBgColor        = compat.AdaptiveColor{Light: lipgloss.Color("#EFF1F5"), Dark: lipgloss.Color("#181825")}
	defaultTextColor     = compat.AdaptiveColor{Light: lipgloss.Color("#4C4F69"), Dark: lipgloss.Color("#CDD6F4")}
	mutedTextColor       = compat.AdaptiveColor{Light: lipgloss.Color("#7C7F93"), Dark: lipgloss.Color("#6C7086")}
	borderColor          = compat.AdaptiveColor{Light: lipgloss.Color("#CCD0DA"), Dark: lipgloss.Color("#313244")}
	scrollIndicatorColor = compat.AdaptiveColor{Light: lipgloss.Color("#9CA0B0"), Dark: lipgloss.Color("#45475A")}
	validColor           = compat.AdaptiveColor{Light: lipgloss.Color("#40A02B"), Dark: lipgloss.Color("#A6E3A1")}
)

// Styles
var lg = lipgloss.NewStyle()
