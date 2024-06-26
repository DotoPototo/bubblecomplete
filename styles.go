package bubblecomplete

import "github.com/charmbracelet/lipgloss"

// Colors
var (
	green        = lipgloss.AdaptiveColor{Light: "#02BA84", Dark: "#02BF87"}
	pink         = lipgloss.AdaptiveColor{Light: "#FF2C70", Dark: "#FF2C70"}
	pinkBg       = lipgloss.AdaptiveColor{Light: "#19040b", Dark: "#19040b"}
	bluegray     = lipgloss.AdaptiveColor{Light: "#5C6773", Dark: "#1f262d"}
	darkBluegray = lipgloss.AdaptiveColor{Light: "#3D4852", Dark: "#12161B"}
	textColor    = lipgloss.AdaptiveColor{Light: "#000000", Dark: "#ffffff"}
	scrollColor  = lipgloss.AdaptiveColor{Light: "#3f0d1d", Dark: "#3f0d1d"}
)

// Styles
var (
	lg                            = lipgloss.NewStyle()
	highlightedCompletionStyle    = lg.Foreground(pink).Background(pinkBg).Bold(true)
	completionRowStyle            = lg.Background(bluegray)
	altCompletionRowStyle         = lg.Background(darkBluegray)
	completionsBoxStyle           = lg.Border(lipgloss.RoundedBorder()).BorderStyle(lipgloss.ThickBorder()).BorderForeground(bluegray)
	completionsBoxScrollStyle     = completionsBoxStyle.BorderTopForeground(scrollColor).BorderBottomForeground(scrollColor)
	completionsBoxScrollDownStyle = completionsBoxStyle.BorderBottomForeground(scrollColor)
	completionsBoxScrollUpStyle   = completionsBoxStyle.BorderTopForeground(scrollColor)
)
