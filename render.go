package bubblecomplete

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var scrollbarPercent float64
var minCompletionsSize = 60

// MARK: Public Functions

func (m Model) View() string {
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
	if m.validCommand == nil {
		m.input.TextStyle = m.ValidCommandStyle
	} else {
		m.input.TextStyle = m.InvalidCommandStyle
	}

	completionTitles := []string{}
	completionDescriptions := []string{}
	maxTitleLength := 0
	maxDescriptionLength := 0
	titlePadding := 3

	// Get each completion text
	if len(m.completions) > 0 && (len(m.input.Value()) > 0 || m.showAll) {
		for _, comp := range m.completions {
			name := comp.getName()
			description := comp.getDescription()

			completionTitles = append(completionTitles, name)
			completionDescriptions = append(completionDescriptions, description)

			if len(name) > maxTitleLength {
				maxTitleLength = len(name)
			}
			if len(description) > maxDescriptionLength {
				maxDescriptionLength = len(description)
			}
		}
	}
	maxTitleLength += titlePadding
	maxLineLength := maxTitleLength + maxDescriptionLength

	// Create each completion row
	completionsRow := make([]string, 0, len(completionTitles))
	completionsWidth := m.getCompletionsWidth(maxLineLength)
	for i := 0; i < len(completionTitles); i++ {
		titleWidth := lipgloss.Width(completionTitles[i])
		rowText := lipgloss.JoinHorizontal(
			lipgloss.Left,
			" ",
			completionTitles[i],
			lg.
				Width(completionsWidth-titleWidth).
				PaddingLeft(maxTitleLength-titleWidth).
				Render(completionDescriptions[i]),
			" ",
		)

		if i == m.completionIndex {
			completionsRow = append(
				completionsRow,
				highlightedCompletionStyle.Render(rowText),
			)
		} else if i%2 == 0 {
			completionsRow = append(
				completionsRow,
				altCompletionRowStyle.Render(rowText),
			)
		} else {
			completionsRow = append(
				completionsRow,
				completionRowStyle.Render(rowText),
			)
		}
	}

	// Render the completions
	if len(completionsRow) != 0 {
		startCompletionsIndex := max(0, m.completionIndex-(m.CompletionRows-1))
		endCompletionsIndex := min(len(completionsRow), startCompletionsIndex+m.CompletionRows)

		completions := lipgloss.JoinVertical(
			lipgloss.Left,
			completionsRow[startCompletionsIndex:endCompletionsIndex]...,
		)

		if len(completionsRow) > m.CompletionRows && m.ShowScrollbar {
			m.scrollbarProgress.Width = maxLineLength + titlePadding
			completions = lipgloss.JoinVertical(
				lipgloss.Left,
				completions,
				m.scrollbarProgress.ViewAs(scrollbarPercent),
			)
		}

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
	if !m.ShowBorderScroll {
		return completionsBoxStyle
	}

	if startCompletionsIndex > 0 && endCompletionsIndex < rows {
		return completionsBoxScrollStyle
	} else if startCompletionsIndex > 0 {
		return completionsBoxScrollUpStyle
	} else if endCompletionsIndex < rows {
		return completionsBoxScrollDownStyle
	} else {
		return completionsBoxStyle
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

func stringEndsInQuote(s string) bool {
	if strings.HasSuffix(s, "\"") || strings.HasSuffix(s, "'") {
		return true
	}
	return false
}
