package bubblecomplete

import (
	"strings"
	"unicode/utf8"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var minCompletionsSize = 60

// MARK: Public Functions

func (m Model) View() tea.View {
	return tea.NewView(m.Render())
}

func (m Model) Render() string {
	var output string
	if m.historyIndex != -1 {
		output = m.input.View()
	} else {
		output = m.showCompletionsRender()
	}
	return lg.Width(m.width).Render(output)
}

// MARK: Private Functions

func (m Model) showCompletionsRender() string {
	if m.validationErr == nil {
		setInputTextStyle(&m.input, m.styles.Input.Valid)
	} else {
		setInputTextStyle(&m.input, m.styles.Input.Invalid)
	}

	completionTitles := []string{}
	completionDescriptions := []string{}
	maxTitleLength := 0
	maxDescriptionLength := 0
	titlePadding := 3

	// Get each completion text
	lowerPrefix := strings.ToLower(m.matchPrefix)
	prefixRuneLen := utf8.RuneCountInString(m.matchPrefix)

	if len(m.completions) > 0 && (len(m.input.Value()) > 0 || m.showAll) {
		for _, comp := range m.completions {
			name := comp.getName()
			description := comp.getDescription()

			if len(name) > maxTitleLength {
				maxTitleLength = len(name)
			}
			if len(description) > maxDescriptionLength {
				maxDescriptionLength = len(description)
			}

			// Apply match highlighting to the matched portion of the name
			if start, end := findMatchRange(name, lowerPrefix, prefixRuneLen); start >= 0 {
				name = lipgloss.StyleRanges(name, lipgloss.NewRange(start, end, m.styles.Completion.Match))
			}

			// Prepend type indicator icon
			if m.ShowIcons {
				var icon string
				switch comp.(type) {
				case *Command, Command:
					icon = m.CommandIcon
				case *PositionalArgument, PositionalArgument:
					icon = m.ArgumentIcon
				case *Flag, Flag:
					icon = m.FlagIcon
				}
				if icon != "" {
					name = m.styles.Completion.Icon.Render(icon) + " " + name
				}
			}

			completionTitles = append(completionTitles, name)
			completionDescriptions = append(completionDescriptions, description)
		}
	}
	maxTitleLength += titlePadding
	if m.ShowIcons {
		maxTitleLength += 2 // icon char + space
	}
	maxLineLength := maxTitleLength + maxDescriptionLength

	// Create each completion row
	completionsRow := make([]string, 0, len(completionTitles))
	completionsWidth := m.getCompletionsWidth(maxLineLength)
	descMaxWidth := completionsWidth - maxTitleLength
	for i := 0; i < len(completionTitles); i++ {
		descText := truncateDescription(completionDescriptions[i], descMaxWidth)
		if i != m.completionIndex {
			descText = m.styles.Completion.Description.Render(descText)
		}

		titleWidth := lipgloss.Width(completionTitles[i])
		rowText := lipgloss.JoinHorizontal(
			lipgloss.Left,
			" ",
			completionTitles[i],
			lg.
				Width(completionsWidth-titleWidth).
				PaddingLeft(maxTitleLength-titleWidth).
				Render(descText),
			" ",
		)

		if i == m.completionIndex {
			completionsRow = append(
				completionsRow,
				m.styles.Completion.SelectedRow.Render(rowText),
			)
		} else if i%2 == 0 {
			completionsRow = append(
				completionsRow,
				m.styles.Completion.AltRow.Render(rowText),
			)
		} else {
			completionsRow = append(
				completionsRow,
				m.styles.Completion.Row.Render(rowText),
			)
		}
	}

	// Render the completions
	if len(completionsRow) != 0 {
		startCompletionsIndex := max(0, m.completionIndex-(m.CompletionRows-1))
		endCompletionsIndex := min(len(completionsRow), startCompletionsIndex+m.CompletionRows)

		visibleRows := make([]string, endCompletionsIndex-startCompletionsIndex)
		copy(visibleRows, completionsRow[startCompletionsIndex:endCompletionsIndex])

		if len(completionsRow) > m.CompletionRows && m.ShowScrollbar {
			scrollbar := m.renderScrollbar(len(visibleRows), len(completionsRow), startCompletionsIndex)
			for j := range visibleRows {
				visibleRows[j] = lipgloss.JoinHorizontal(lipgloss.Left, visibleRows[j], scrollbar[j])
			}
		}

		completions := lipgloss.JoinVertical(
			lipgloss.Left,
			visibleRows...,
		)

		offset := m.calculateCompletionsOffset(completions)

		completionsStyle := m.getCompletionsStyle(startCompletionsIndex, endCompletionsIndex, len(completionsRow))
		completionsRender := completionsStyle.Margin(0, 0, 0, offset).Render(completions)

		if m.CompletionsPosition == PositionAbove {
			return completionsRender + "\n" + m.input.View()
		}
		if m.CompletionsPosition == PositionBelow {
			return m.input.View() + "\n" + completionsRender
		}
	}

	return m.input.View()
}

func (m Model) getCompletionsStyle(startCompletionsIndex int, endCompletionsIndex int, rows int) lipgloss.Style {
	border := m.styles.Completion.Border
	if !m.ShowBorderScroll {
		return border
	}

	if startCompletionsIndex > 0 && endCompletionsIndex < rows {
		return border.BorderTopForeground(scrollIndicatorColor).BorderBottomForeground(scrollIndicatorColor)
	} else if startCompletionsIndex > 0 {
		return border.BorderTopForeground(scrollIndicatorColor)
	} else if endCompletionsIndex < rows {
		return border.BorderBottomForeground(scrollIndicatorColor)
	} else {
		return border
	}
}

func (m Model) calculateCompletionsOffset(completions string) int {
	if !m.IndentCompletions {
		return m.CompletionsOffset
	}

	input := m.input.Value()
	parts := splitInput(input)

	if len(parts) == 0 {
		return 0
	}

	offset := 0

	// TODO: Offset is slightly off on each new line

	// If we're about to start typing a new part, set the offset to the end of the string
	if strings.HasSuffix(input, " ") && (!strings.Contains(parts[len(parts)-1], " ") || stringEndsInQuote(parts[len(parts)-1])) {
		offset = (lipgloss.Width(input) % m.width)
	} else {
		// If we're typing, set the offset to the end of the last part
		trimmedInput := input[:strings.LastIndex(input, parts[len(parts)-1])]
		offset = lipgloss.Width(trimmedInput) % m.width
	}

	offset += m.CompletionsOffset
	if offset+lipgloss.Width(completions) > m.width {
		offset = m.width - lipgloss.Width(completions) - 2
	}

	return offset
}

func (m Model) getCompletionsWidth(maxLineLength int) int {
	maxTermWidth := m.width - 8
	if maxTermWidth < minCompletionsSize {
		maxTermWidth = minCompletionsSize
	}

	if maxLineLength > maxTermWidth {
		return maxTermWidth
	}
	return maxLineLength
}

// findMatchRange finds the rune-based start and end positions where lowerPrefix
// matches the beginning of any space-separated word in name.
// This handles flag display names like "-m --message" where the prefix "--me"
// should match the "--message" portion starting at rune position 3.
// Returns (-1, -1) if no match is found.
func findMatchRange(name, lowerPrefix string, prefixRuneLen int) (int, int) {
	if lowerPrefix == "" {
		return -1, -1
	}
	runePos := 0
	for _, word := range strings.Split(name, " ") {
		if strings.HasPrefix(strings.ToLower(word), lowerPrefix) {
			return runePos, runePos + prefixRuneLen
		}
		runePos += utf8.RuneCountInString(word) + 1 // +1 for the space
	}
	return -1, -1
}

func setInputTextStyle(input *textinput.Model, style lipgloss.Style) {
	styles := input.Styles()
	styles.Focused.Text = style
	styles.Blurred.Text = style
	input.SetStyles(styles)
}

func stringEndsInQuote(s string) bool {
	return strings.HasSuffix(s, "\"") || strings.HasSuffix(s, "'")
}

func truncateDescription(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= maxWidth {
		return s
	}
	runes := []rune(s)
	return string(runes[:maxWidth-1]) + "\u2026"
}

func (m Model) renderScrollbar(height, totalItems, offset int) []string {
	thumbSize := max(1, height*height/totalItems)
	scrollRange := totalItems - height
	trackRange := height - thumbSize
	thumbStart := 0
	if scrollRange > 0 {
		thumbStart = offset * trackRange / scrollRange
	}
	result := make([]string, height)
	for i := range height {
		if i >= thumbStart && i < thumbStart+thumbSize {
			result[i] = m.styles.Scrollbar.Thumb.Render("\u2503")
		} else {
			result[i] = m.styles.Scrollbar.Track.Render("\u2502")
		}
	}
	return result
}
