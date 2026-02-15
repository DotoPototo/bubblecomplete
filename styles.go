package bubblecomplete

import "github.com/charmbracelet/lipgloss"

// Colors — Charm-inspired palette with semantic naming.
// Pink accent, proper text hierarchy, subtle alternating rows.
var (
	accentColor          = lipgloss.AdaptiveColor{Light: "#D6116B", Dark: "#F5639A"}
	accentBgColor        = lipgloss.AdaptiveColor{Light: "#FDE8F0", Dark: "#3B1D2E"}
	rowBgColor           = lipgloss.AdaptiveColor{Light: "#E6E9EF", Dark: "#1E1E2E"}
	altRowBgColor        = lipgloss.AdaptiveColor{Light: "#EFF1F5", Dark: "#181825"}
	defaultTextColor     = lipgloss.AdaptiveColor{Light: "#4C4F69", Dark: "#CDD6F4"}
	mutedTextColor       = lipgloss.AdaptiveColor{Light: "#7C7F93", Dark: "#6C7086"}
	borderColor          = lipgloss.AdaptiveColor{Light: "#CCD0DA", Dark: "#313244"}
	scrollIndicatorColor = lipgloss.AdaptiveColor{Light: "#9CA0B0", Dark: "#45475A"}
	validColor           = lipgloss.AdaptiveColor{Light: "#40A02B", Dark: "#A6E3A1"}
)

// Styles
var lg = lipgloss.NewStyle()
