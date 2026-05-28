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
	invalidPathColor     = compat.AdaptiveColor{Light: lipgloss.Color("#D20F39"), Dark: lipgloss.Color("#F38BA8")}
)

var lg = lipgloss.NewStyle()

// InputStyles styles the text inside the input field based on validation state.
//
// Valid and Invalid apply to the whole input value. PathValid, PathPartial,
// and PathInvalid apply as a range overlay on the typed value of a
// FileArgument / DirArgument / FileDirArgument when WithFilesystemCompletions
// is enabled. Zero-value styles produce no overlay; hosts wanting defaults
// should start from DefaultStyles and modify.
type InputStyles struct {
	Valid   lipgloss.Style
	Invalid lipgloss.Style
	// PathValid styles the typed value when it resolves to an entry of the
	// expected kind.
	PathValid lipgloss.Style
	// PathPartial styles the typed value when it is a strict prefix of one
	// or more existing entries (mid-navigation).
	PathPartial lipgloss.Style
	// PathInvalid styles the typed value when it cannot resolve or has the
	// wrong kind.
	PathInvalid lipgloss.Style
}

// CompletionStyles styles the completion list rows, border, match highlight, descriptions and icons.
type CompletionStyles struct {
	Match       lipgloss.Style
	SelectedRow lipgloss.Style
	Row         lipgloss.Style
	AltRow      lipgloss.Style
	Border      lipgloss.Style
	Description lipgloss.Style
	Icon        lipgloss.Style
}

// ScrollbarStyles styles the vertical scrollbar thumb and track.
type ScrollbarStyles struct {
	Thumb lipgloss.Style
	Track lipgloss.Style
}

// Styles bundles every lipgloss style the component renders with. Customize via Model.SetStyles.
type Styles struct {
	Input      InputStyles
	Completion CompletionStyles
	Scrollbar  ScrollbarStyles
}

// DefaultStyles returns the styles applied by New.
func DefaultStyles() Styles {
	return Styles{
		Input: InputStyles{
			Valid:       lg.Foreground(validColor),
			Invalid:     lg.Foreground(mutedTextColor),
			PathValid:   lg.Foreground(validColor),
			PathPartial: lg.Underline(true),
			PathInvalid: lg.Foreground(invalidPathColor),
		},
		Completion: CompletionStyles{
			Match:       lg.Bold(true).Foreground(accentColor),
			SelectedRow: lg.Foreground(defaultTextColor).Background(accentBgColor).Bold(true),
			Row:         lg.Background(rowBgColor).Foreground(defaultTextColor),
			AltRow:      lg.Background(altRowBgColor).Foreground(defaultTextColor),
			Border:      lg.Border(lipgloss.RoundedBorder()).BorderForeground(borderColor),
			Description: lg.Foreground(mutedTextColor),
			Icon:        lg.Foreground(accentColor),
		},
		Scrollbar: ScrollbarStyles{
			Thumb: lg.Foreground(accentColor),
			Track: lg.Foreground(borderColor),
		},
	}
}
