package bubblecomplete

import (
	"strings"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)


// MARK: Public Functions

// View returns the component as a tea.View so [Model] satisfies the Bubble
// Tea v2 model contract directly. Prefer [Model.Render] when composing the
// component into a larger view.
func (m Model) View() tea.View {
	return tea.NewView(m.Render())
}

// Render returns the component as a styled string. Use this when composing
// Bubblecomplete with other content inside the host model's View.
func (m Model) Render() string {
	var output string
	if m.historyIndex != -1 {
		output = m.input.View()
	} else {
		output = m.showCompletionsRender()
	}
	return lg.Width(m.width).Render(output)
}

// MARK: Completion Row Model

type completionKind int

const (
	commandKind completionKind = iota
	argumentKind
	flagKind
)

type completionRow struct {
	Name        string
	Description string
	Kind        completionKind
}

func kindOf(c Completion) completionKind {
	switch c.(type) {
	case *Command, Command:
		return commandKind
	case *PositionalArgument, PositionalArgument:
		return argumentKind
	case *Flag, Flag:
		return flagKind
	}
	return commandKind
}

func (m Model) completionRows() []completionRow {
	rows := make([]completionRow, 0, len(m.completions))
	for _, c := range m.completions {
		rows = append(rows, completionRow{
			Name:        c.getName(),
			Description: c.getDescription(),
			Kind:        kindOf(c),
		})
	}
	return rows
}

func (m Model) iconFor(k completionKind) string {
	switch k {
	case commandKind:
		return m.CommandIcon
	case argumentKind:
		return m.ArgumentIcon
	case flagKind:
		return m.FlagIcon
	}
	return ""
}

// completionBoxWidth returns the column widths used to lay out the completion box.
// titleWidth is the title column budget (max name + padding + widest icon if enabled).
// lineWidth is titleWidth + max description width.
func (m Model) completionBoxWidth(rows []completionRow) (titleWidth, lineWidth int) {
	const titlePadding = 3
	maxTitle, maxDesc := 0, 0
	for _, r := range rows {
		if w := lipgloss.Width(r.Name); w > maxTitle {
			maxTitle = w
		}
		if m.ShowDescriptions {
			if w := lipgloss.Width(r.Description); w > maxDesc {
				maxDesc = w
			}
		}
	}
	titleWidth = maxTitle + titlePadding
	if m.ShowIcons {
		if iconW := m.maxIconWidth(rows); iconW > 0 {
			titleWidth += iconW + 1
		}
	}
	return titleWidth, titleWidth + maxDesc
}

// maxIconWidth returns the largest display width across icons used by the given rows.
// Returns 0 if no row's kind has a non-empty icon.
func (m Model) maxIconWidth(rows []completionRow) int {
	seen := map[completionKind]bool{}
	for _, r := range rows {
		seen[r.Kind] = true
	}
	maxW := 0
	for kind := range seen {
		icon := m.iconFor(kind)
		if icon == "" {
			continue
		}
		if w := lipgloss.Width(icon); w > maxW {
			maxW = w
		}
	}
	return maxW
}

// visibleWindow returns the [start, end) slice indices of rows to display given
// the currently selected row, total row count, and visible-row budget.
// Safe for selected == -1 (no selection) and zero total/rows.
func visibleWindow(selected, total, rows int) (start, end int) {
	if rows <= 0 || total <= 0 {
		return 0, 0
	}
	if selected < 0 {
		selected = 0
	}
	start = max(0, selected-(rows-1))
	end = min(total, start+rows)
	return start, end
}

// MARK: Private Functions

func (m Model) showCompletionsRender() string {
	if len(m.completions) == 0 || (len(m.input.Value()) == 0 && !m.showAll) {
		return m.input.View()
	}

	rows := m.completionRows()
	titleWidth, lineWidth := m.completionBoxWidth(rows)
	completionsWidth := m.getCompletionsWidth(lineWidth)

	rendered := make([]string, len(rows))
	for i, row := range rows {
		rendered[i] = m.renderCompletionRow(row, titleWidth, completionsWidth, i == m.completionIndex, i%2 == 0)
	}

	rowsToShow := max(1, m.CompletionRows)
	start, end := visibleWindow(m.completionIndex, len(rendered), rowsToShow)
	visible := make([]string, end-start)
	copy(visible, rendered[start:end])

	if len(rendered) > rowsToShow && m.ShowScrollbar {
		scrollbar := renderScrollbar(len(visible), len(rendered), start, m.styles.Scrollbar)
		for j := range visible {
			visible[j] = lipgloss.JoinHorizontal(lipgloss.Left, visible[j], scrollbar[j])
		}
	}

	box := lipgloss.JoinVertical(lipgloss.Left, visible...)
	offset := m.calculateCompletionsOffset(box)
	style := m.getCompletionsStyle(start, end, len(rendered))
	renderedBox := style.Margin(0, 0, 0, offset).Render(box)

	if m.CompletionsPosition == PositionAbove {
		return renderedBox + "\n" + m.input.View()
	}
	return m.input.View() + "\n" + renderedBox
}

func (m Model) renderCompletionRow(row completionRow, titleWidth, completionsWidth int, selected, alt bool) string {
	name := row.Name
	lowerPrefix := strings.ToLower(m.matchPrefix)
	prefixRuneLen := utf8.RuneCountInString(m.matchPrefix)
	if start, end := findMatchRange(name, lowerPrefix, prefixRuneLen); start >= 0 {
		name = lipgloss.StyleRanges(name, lipgloss.NewRange(start, end, m.styles.Completion.Match))
	}
	if m.ShowIcons {
		if icon := m.iconFor(row.Kind); icon != "" {
			name = m.styles.Completion.Icon.Render(icon) + " " + name
		}
	}

	nameWidth := lipgloss.Width(name)

	var rowText string
	if m.ShowDescriptions {
		descText := truncateDescription(row.Description, completionsWidth-titleWidth)
		if !selected {
			descText = m.styles.Completion.Description.Render(descText)
		}
		rowText = lipgloss.JoinHorizontal(
			lipgloss.Left,
			" ",
			name,
			lg.Width(completionsWidth-nameWidth).PaddingLeft(titleWidth-nameWidth).Render(descText),
			" ",
		)
	} else {
		rowText = lipgloss.JoinHorizontal(
			lipgloss.Left,
			" ",
			lg.Width(titleWidth).Render(name),
			" ",
		)
	}

	switch {
	case selected:
		return m.styles.Completion.SelectedRow.Render(rowText)
	case alt:
		return m.styles.Completion.AltRow.Render(rowText)
	default:
		return m.styles.Completion.Row.Render(rowText)
	}
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

	// The textinput prompt ("> " by default) shifts every column right by its
	// display width. Without it the wrap modulo lands two cells short on lines
	// that started with the prompt.
	promptWidth := lipgloss.Width(m.input.Prompt)

	offset := 0

	// If we're about to start typing a new part, set the offset to the end of the string
	if strings.HasSuffix(input, " ") && (!strings.Contains(parts[len(parts)-1], " ") || stringEndsInQuote(parts[len(parts)-1])) {
		offset = (promptWidth + lipgloss.Width(input)) % m.width
	} else {
		// If we're typing, set the offset to the end of the last part
		trimmedInput := input[:strings.LastIndex(input, parts[len(parts)-1])]
		offset = (promptWidth + lipgloss.Width(trimmedInput)) % m.width
	}

	offset += m.CompletionsOffset
	if offset+lipgloss.Width(completions) > m.width {
		offset = m.width - lipgloss.Width(completions) - 2
	}

	return offset
}

// getCompletionsWidth caps the completion box at the terminal width minus a
// small border reserve. Narrow terminals are respected — we never return a
// width larger than the visible area.
func (m Model) getCompletionsWidth(maxLineLength int) int {
	maxTermWidth := max(1, m.width-8)
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

func stringEndsInQuote(s string) bool {
	return strings.HasSuffix(s, "\"") || strings.HasSuffix(s, "'")
}

func truncateDescription(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= maxWidth {
		return s
	}
	return ansi.Truncate(s, maxWidth, "…")
}

// renderScrollbar lays out a vertical scrollbar of length height for a list of
// totalItems items, with the viewport's top at offset. Returns nil when the
// scrollbar cannot be drawn (zero height or total).
func renderScrollbar(height, totalItems, offset int, s ScrollbarStyles) []string {
	if height <= 0 || totalItems <= 0 {
		return nil
	}
	thumbSize := max(1, height*height/totalItems)
	if thumbSize > height {
		thumbSize = height
	}
	scrollRange := totalItems - height
	trackRange := height - thumbSize
	thumbStart := 0
	if scrollRange > 0 && trackRange > 0 {
		thumbStart = offset * trackRange / scrollRange
	}
	if thumbStart+thumbSize > height {
		thumbStart = height - thumbSize
	}
	result := make([]string, height)
	for i := range height {
		if i >= thumbStart && i < thumbStart+thumbSize {
			result[i] = s.Thumb.Render("┃")
		} else {
			result[i] = s.Track.Render("│")
		}
	}
	return result
}
