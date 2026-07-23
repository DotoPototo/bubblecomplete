package bubblecomplete

import (
	"fmt"
	"strings"

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
		output = m.renderedInput()
	} else {
		output = m.showCompletionsRender()
	}
	return lg.Width(m.width).Render(output)
}

// renderedInput returns m.input.View() with the path-validity overlay
// applied when m.pathState is active. The overlay is skipped when:
//   - the rendered input width exceeds m.width (textinput's internal
//     scroll window obscures cell offsets and we'd paint garbage)
//   - the stored offsets are out of range for the current value
//     (defensive — shouldn't fire under correct lifecycle but cheap to
//     check)
//
// During Tab cycling the overlay does apply: keyTab refreshes pathState
// after every multi-match SetValue so the offsets and validity match the
// cycled preview value. Cycling between candidates of different validity
// (e.g. a valid file vs a directory under FileArgument) therefore shows
// different colours per Tab.
func (m Model) renderedInput() string {
	base := m.input.View()
	if !m.pathState.active {
		return base
	}
	if m.pathState.valueStart >= m.pathState.valueEnd {
		return base
	}
	value := m.input.Value()
	if m.pathState.valueEnd > len(value) {
		return base
	}
	promptCells := lipgloss.Width(m.input.Prompt)
	if promptCells+lipgloss.Width(value) > m.width {
		return base
	}
	beforeCells := lipgloss.Width(value[:m.pathState.valueStart])
	tokenCells := lipgloss.Width(value[m.pathState.valueStart:m.pathState.valueEnd])
	if tokenCells == 0 {
		return base
	}

	var style lipgloss.Style
	switch m.pathState.validity {
	case pathValid:
		style = m.styles.Input.PathValid
	case pathPartial:
		style = m.styles.Input.PathPartial
	default:
		style = m.styles.Input.PathInvalid
	}
	rangeStart := promptCells + beforeCells
	rangeEnd := rangeStart + tokenCells
	return lipgloss.StyleRanges(base, lipgloss.NewRange(rangeStart, rangeEnd, style))
}

// MARK: completion Row Model

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

func kindOf(c completion) completionKind {
	switch c.(type) {
	case *Command:
		return commandKind
	case *PositionalArgument:
		return argumentKind
	case *Flag:
		return flagKind
	case pathCompletion:
		// Path candidates are filesystem entries surfaced for an argument
		// value — map to argumentKind so ShowIcons uses ArgumentIcon.
		return argumentKind
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
		return m.renderedInput()
	}

	rows := m.completionRows()
	titleWidth, lineWidth := m.completionBoxWidth(rows)
	completionsWidth := m.getCompletionsWidth(lineWidth)
	// On narrow terminals the widest name can exceed the clamped box width.
	// Cap the title column at the box too, or renderCompletionRow's descPad
	// pads the (truncated) name back out past the clamp and rows overflow.
	titleWidth = min(titleWidth, completionsWidth)

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

	// When path completions were dropped — either truncated post-sort or
	// skipped because the symlink stat budget ran out — append a subtle
	// footer line BELOW the box describing what's missing. The footer is
	// render-only (never added to m.completions), so Tab cycling cannot
	// select or accept it as a completion row. Left-aligned with the box.
	if m.pathState.active && (m.pathState.droppedSorted > 0 || m.pathState.unresolvedEntries > 0) {
		footer := m.renderTruncationFooter(offset)
		renderedBox = lipgloss.JoinVertical(lipgloss.Left, renderedBox, footer)
	}

	if m.CompletionsPosition == PositionAbove {
		return renderedBox + "\n" + m.renderedInput()
	}
	return m.renderedInput() + "\n" + renderedBox
}

// renderTruncationFooter formats the hint that appears below the
// completion box when generateCandidates dropped some matches. Two
// distinct counts get distinct wording:
//
//   - droppedSorted is verified — those entries passed the kind and prefix
//     filters and would be reachable by narrowing the prefix. Surfaced as
//     "+ N more".
//   - unresolvedEntries is unverified — symlinks the stat budget skipped.
//     Some might be valid, some might be broken or wrong-kind. Surfaced
//     with weaker "N unresolved" wording so we don't over-promise results.
//
// Combined wording when both counts are non-zero. Styled subtly (muted
// foreground, italic) so it doesn't compete with the active completion
// rows. Left margin matches the box.
func (m Model) renderTruncationFooter(leftMargin int) string {
	var text string
	switch {
	case m.pathState.droppedSorted > 0 && m.pathState.unresolvedEntries > 0:
		text = fmt.Sprintf("+ %d more (%d unresolved) — type to narrow",
			m.pathState.droppedSorted, m.pathState.unresolvedEntries)
	case m.pathState.droppedSorted > 0:
		text = fmt.Sprintf("+ %d more — type to narrow", m.pathState.droppedSorted)
	default: // only unresolvedEntries > 0
		text = fmt.Sprintf("%d symlinks unresolved — narrow to filter",
			m.pathState.unresolvedEntries)
	}
	return m.styles.Completion.Description.
		Italic(true).
		MarginLeft(leftMargin).
		Render(text)
}

func (m Model) renderCompletionRow(row completionRow, titleWidth, completionsWidth int, selected, alt bool) string {
	name := row.Name
	if start, end := findMatchRange(name, m.matchPrefix); start >= 0 {
		name = lipgloss.StyleRanges(name, lipgloss.NewRange(start, end, m.styles.Completion.Match))
	}
	if m.ShowIcons {
		if icon := m.iconFor(row.Kind); icon != "" {
			name = m.styles.Completion.Icon.Render(icon) + " " + name
		}
	}

	// Cap the rendered name at the box width minus the row's side padding
	// (one space each side in the JoinHorizontal below). Without this, a
	// long path completion like "/Users/jane/very/long/.../file.md" would
	// overflow the box on a narrow terminal. ansi.Truncate is style-aware
	// — it preserves any match-highlight or icon ANSI prefix.
	maxNameWidth := max(0, completionsWidth-2)
	if lipgloss.Width(name) > maxNameWidth {
		name = ansi.Truncate(name, maxNameWidth, "…")
	}

	nameWidth := lipgloss.Width(name)

	var rowText string
	if m.ShowDescriptions {
		descText := truncateDescription(row.Description, max(0, completionsWidth-titleWidth))
		if !selected {
			descText = m.styles.Completion.Description.Render(descText)
		}
		// Clamp width and padding to >= 0: a name wider than the title
		// column (very narrow terminal, long completion name) would
		// otherwise pass negative values to lipgloss.
		descWidth := max(0, completionsWidth-nameWidth)
		descPad := max(0, titleWidth-nameWidth)
		rowText = lipgloss.JoinHorizontal(
			lipgloss.Left,
			" ",
			name,
			lg.Width(descWidth).PaddingLeft(descPad).Render(descText),
			" ",
		)
	} else {
		rowText = lipgloss.JoinHorizontal(
			lipgloss.Left,
			" ",
			lg.Width(max(0, titleWidth)).Render(name),
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

	hasItemsAbove := startCompletionsIndex > 0
	hasItemsBelow := endCompletionsIndex < rows
	switch {
	case hasItemsAbove && hasItemsBelow:
		return border.BorderTopForeground(scrollIndicatorColor).BorderBottomForeground(scrollIndicatorColor)
	case hasItemsAbove:
		return border.BorderTopForeground(scrollIndicatorColor)
	case hasItemsBelow:
		return border.BorderBottomForeground(scrollIndicatorColor)
	}
	return border
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

	// Offset anchors to the end of the input when we're about to start a
	// fresh part (trailing space and the previous part is either a bare
	// word or a closed-quote token); otherwise anchor to the start of the
	// last part so completions sit under what's actively being typed.
	lastPart := parts[len(parts)-1]
	endsInSpace := strings.HasSuffix(input, " ")
	lastPartIsBareWord := !strings.Contains(lastPart, " ")
	startingNewPart := endsInSpace && (lastPartIsBareWord || stringEndsInQuote(lastPart))
	anchor := input
	if !startingNewPart {
		anchor = input[:strings.LastIndex(input, lastPart)]
	}
	offset := (promptWidth + lipgloss.Width(anchor)) % m.width

	offset += m.CompletionsOffset
	if offset+lipgloss.Width(completions) > m.width {
		offset = m.width - lipgloss.Width(completions) - 2
	}

	// Clamp to >= 0 so a too-wide completions box on a tiny terminal doesn't
	// hand a negative margin to lipgloss (would render off-screen).
	if offset < 0 {
		offset = 0
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

// findMatchRange finds the CELL-based start and end positions within name
// where matchPrefix (case-insensitively) matches the start of any
// space-separated word. Cell-based so the returned range plugs directly
// into [lipgloss.NewRange], which is column/cell-indexed — not rune-indexed.
//
// This handles flag display names like "-m --message" where the prefix
// "--me" should highlight the "--me" portion of "--message". It also
// handles wide-rune names ("日本.txt" with prefix "日") correctly: lipgloss
// measures "日" as 2 cells, so the highlight covers both columns rather
// than just one — the bug a naive rune-position implementation would hit.
//
// Best-effort caveat: the returned range uses lipgloss.Width(matchPrefix)
// for the highlight length, which assumes case folding preserves cell
// width. That's true for ASCII and for typical Latin/CJK casing, but a
// few obscure Unicode codepoints (e.g. the Turkish dotless-İ pair, some
// digraphs) fold to substrings whose cell width differs from the source.
// In those rare cases the highlight may be off by one cell at the
// trailing edge. Acceptable for the completion-row use case; revisit if
// it surfaces in real filenames.
//
// Returns (-1, -1) if no match is found.
func findMatchRange(name, matchPrefix string) (int, int) {
	if matchPrefix == "" {
		return -1, -1
	}
	lowerPrefix := strings.ToLower(matchPrefix)
	prefixCells := lipgloss.Width(matchPrefix)
	cellPos := 0
	for word := range strings.SplitSeq(name, " ") {
		if strings.HasPrefix(strings.ToLower(word), lowerPrefix) {
			return cellPos, cellPos + prefixCells
		}
		cellPos += lipgloss.Width(word) + 1 // +1 for the space
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
	thumbSize := min(height, max(1, height*height/totalItems))
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
