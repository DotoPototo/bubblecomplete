package bubblecomplete

import (
	"reflect"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

func TestStringEndsInQuote(t *testing.T) {
	cases := []struct {
		input    string
		expected bool
	}{
		{"Hello World", false},
		{"Hello World\"", true},
		{"Hello World'", true},
		{"Hello \"World\"", true},
		{"Hello 'World'", true},
		{"Hello", false},
	}

	for _, c := range cases {
		result := stringEndsInQuote(c.input)
		if result != c.expected {
			t.Errorf("stringEndsInQuote(%q) == %t, expected %t", c.input, result, c.expected)
		}
	}
}

func TestMatchHighlightApplied(t *testing.T) {
	m, err := New(TestCommands, 100)
	if err != nil {
		t.Fatal(err)
	}

	// Simulate typing "co" — should match "commit", "clone" etc
	m.input.SetValue("git co")
	m.completions, m.matchPrefix = m.getCompletions()

	if m.matchPrefix != "co" {
		t.Fatalf("Expected matchPrefix %q, got %q", "co", m.matchPrefix)
	}
	if len(m.completions) == 0 {
		t.Fatal("Expected completions, got none")
	}

	// Render and verify the match highlight style appears around the matched prefix
	output := m.Render()
	// "co" should be styled with MatchHighlightStyle, "mmit" should not
	styledCo := m.Styles().Completion.Match.Render("co")
	if !strings.Contains(output, styledCo) {
		t.Errorf("Expected styled 'co' (%q) in output, got: %q", styledCo, output)
	}
}

func TestMatchHighlightLongFlag(t *testing.T) {
	m, err := New(TestCommands, 100)
	if err != nil {
		t.Fatal(err)
	}

	// Typing "--me" should match the "--message" portion of "-m --message"
	m.input.SetValue("git commit --me")
	m.completions, m.matchPrefix = m.getCompletions()

	if m.matchPrefix != "--me" {
		t.Fatalf("Expected matchPrefix %q, got %q", "--me", m.matchPrefix)
	}
	if len(m.completions) == 0 {
		t.Fatal("Expected completions, got none")
	}

	output := m.Render()
	// The "--me" portion of "--message" should be highlighted with MatchHighlightStyle
	styledMe := m.Styles().Completion.Match.Render("--me")
	if !strings.Contains(output, styledMe) {
		t.Errorf("Expected styled '--me' (%q) in output, got: %q", styledMe, output)
	}
}

func TestNoMatchHighlightWhenShowingAll(t *testing.T) {
	m, err := New(TestCommands, 100)
	if err != nil {
		t.Fatal(err)
	}

	// "git " with trailing space — shows all subcommands, no prefix
	m.input.SetValue("git ")
	m.completions, m.matchPrefix = m.getCompletions()

	if m.matchPrefix != "" {
		t.Errorf("Expected empty matchPrefix, got %q", m.matchPrefix)
	}

	// Rendered output should contain no match highlighting escapes
	output := m.Render()
	// With empty prefix, no completion name should have the MatchHighlightStyle applied.
	// Locate the content rune in the rendered sample to derive the prefix —
	// don't assume lipgloss always emits a "x\033[0m" suffix.
	styledTest := m.Styles().Completion.Match.Render("x")
	idx := strings.Index(styledTest, "x")
	if idx < 0 {
		t.Fatalf("Match style render %q did not contain the content rune", styledTest)
	}
	highlightPrefix := styledTest[:idx]
	if highlightPrefix == "" {
		t.Skip("Match style renders without any ANSI prefix; nothing distinctive to assert absence of")
	}
	if strings.Contains(output, highlightPrefix) {
		t.Errorf("Expected no match highlight escape when showing all completions, but found one")
	}
}

func TestCompletionRows_KindsForEachConcreteType(t *testing.T) {
	m, err := New(TestCommands, 100)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		input string
		kind  completionKind
	}{
		{"g", commandKind},
		{"git commit -", flagKind},
		{"cat ", argumentKind},
	}

	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			m.input.SetValue(c.input)
			m.completions, m.matchPrefix = m.getCompletions()
			if len(m.completions) == 0 {
				t.Fatalf("no completions for %q", c.input)
			}
			rows := m.completionRows()
			if rows[0].Kind != c.kind {
				t.Errorf("Kind = %d, want %d", rows[0].Kind, c.kind)
			}
		})
	}
}

func TestCompletionBoxWidth_AccountsForDisplayWidth(t *testing.T) {
	// CJK characters and emoji occupy two cells; accented characters one cell
	// despite being multi-byte.
	rows := []completionRow{
		{Name: "café", Description: "naïve"},
		{Name: "日本語", Description: "🐛 bug"},
	}

	m, err := New(TestCommands, 100)
	if err != nil {
		t.Fatal(err)
	}
	titleWidth, _ := m.completionBoxWidth(rows)
	// "日本語" is 3 wide chars = 6 cells, plus 3 padding = 9
	if titleWidth != 6+3 {
		t.Errorf("titleWidth = %d, want 9 (6 cells + 3 padding)", titleWidth)
	}
}

func TestCompletionBoxWidth_AccountsForWideIcons(t *testing.T) {
	rows := []completionRow{
		{Name: "git", Kind: commandKind},
		{Name: "file", Kind: argumentKind},
	}

	m, err := New(TestCommands, 100,
		WithIcons(true),
	)
	if err != nil {
		t.Fatal(err)
	}
	m.CommandIcon = "🐛"  // 2 cells
	m.ArgumentIcon = "›" // 1 cell

	titleWidth, _ := m.completionBoxWidth(rows)
	// max name = "file" (4 cells), padding 3, widest icon = 🐛 (2 cells) + 1 space = 3
	// expected: 4 + 3 + 3 = 10
	if titleWidth != 4+3+3 {
		t.Errorf("titleWidth = %d, want 10 (4 + 3 padding + 3 icon)", titleWidth)
	}
}

func TestTruncateDescription_WideChars(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		maxWidth int
		wantW    int
	}{
		{"ascii fits", "hello", 10, 5},
		{"ascii truncates", "hello world", 5, 5},
		{"accented fits", "café", 5, 4},
		{"cjk truncates", "日本語テスト", 6, 5}, // each CJK char is 2 cells; fits "日本…" (4+1=5) within 6
		{"emoji truncates", "🐛 a bug", 4, 4},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out := truncateDescription(c.input, c.maxWidth)
			if w := lipgloss.Width(out); w > c.maxWidth {
				t.Errorf("output width %d exceeds maxWidth %d: %q", w, c.maxWidth, out)
			}
			if w := lipgloss.Width(out); w != c.wantW {
				t.Errorf("output width = %d, want %d (%q)", w, c.wantW, out)
			}
		})
	}
}

func TestCompletionBoxWidth_DescriptionsHidden(t *testing.T) {
	rows := []completionRow{
		{Name: "git", Description: "a longer description"},
		{Name: "commit", Description: "another longer description"},
	}

	m, err := New(TestCommands, 100, WithDescriptions(false))
	if err != nil {
		t.Fatal(err)
	}
	titleWidth, lineWidth := m.completionBoxWidth(rows)
	wantTitle := len("commit") + 3
	if titleWidth != wantTitle {
		t.Errorf("titleWidth = %d, want %d", titleWidth, wantTitle)
	}
	if lineWidth != wantTitle {
		t.Errorf("lineWidth with descriptions off = %d, want %d (descriptions should not contribute)", lineWidth, wantTitle)
	}
}

func TestRender_HistoryActivePathRendersJustTheInput(t *testing.T) {
	// When historyIndex != -1 the user is walking the history; Render must
	// short-circuit to just the input view, not the completion box. The
	// branch was previously untested.
	m, err := New(TestCommands, 100)
	if err != nil {
		t.Fatal(err)
	}
	m.input.SetValue("git status")
	m.historyIndex = 0 // simulate active history walk

	out := m.Render()
	if out == "" {
		t.Error("Render returned empty string in history-active path")
	}
	// The completion box border (rounded) shouldn't appear when history
	// navigation is active.
	if strings.Contains(out, "╭") || strings.Contains(out, "╰") {
		t.Errorf("Render produced completion border while history navigation active: %q", out)
	}
}

func TestRenderCompletionRow_LongNameTruncates(t *testing.T) {
	// A completion name longer than the box's content width must be
	// truncated with an ellipsis so the row fits inside completionsWidth.
	// Without truncation the row overflows the box border, which is
	// especially visible for long path completions like a full filesystem
	// path on a narrow terminal.
	m, err := New(TestCommands, 100)
	if err != nil {
		t.Fatal(err)
	}
	longName := strings.Repeat("a", 60)
	row := completionRow{Name: longName, Description: "ignored", Kind: argumentKind}

	const completionsWidth = 20
	rendered := m.renderCompletionRow(row, 5, completionsWidth, false, false)

	// ANSI-stripped width must not exceed completionsWidth + 2 (one cell of
	// side padding on each end via JoinHorizontal). Without the truncation
	// fix, the full 60-cell name plus padding would render at ~62 cells.
	const sidePadding = 2
	if lipgloss.Width(rendered) > completionsWidth+sidePadding {
		t.Errorf("rendered width = %d, want ≤ %d (long name should be truncated to fit box)",
			lipgloss.Width(rendered), completionsWidth+sidePadding)
	}
	if !strings.Contains(rendered, "…") {
		t.Errorf("expected ellipsis marker in truncated name, got: %q", rendered)
	}
}

func TestRenderCompletionRow_LongNameTruncates_PreservesMatchHighlight(t *testing.T) {
	// The truncation step runs AFTER match-highlight ANSI has been embedded
	// into the name string. ansi.Truncate must therefore correctly handle
	// the ANSI sequences — not split them mid-CSI and not over-count their
	// width. This test plants a match-highlight on the first few cells of
	// a long name and confirms:
	//   1. lipgloss.Width still measures within the box budget
	//      (ANSI bytes must not inflate the displayed width)
	//   2. the leading match-highlight bytes survive truncation
	//      (the highlighted prefix is what the user uses to orient,
	//      losing it would be a regression in user experience)
	m, err := New(TestCommands, 100)
	if err != nil {
		t.Fatal(err)
	}
	// Match the first three cells so the row carries match-highlight ANSI
	// around "aaa" before truncation runs.
	m.matchPrefix = "aaa"

	longName := strings.Repeat("a", 60)
	row := completionRow{Name: longName, Description: "ignored", Kind: argumentKind}

	const completionsWidth = 20
	rendered := m.renderCompletionRow(row, 5, completionsWidth, false, false)

	const sidePadding = 2
	if lipgloss.Width(rendered) > completionsWidth+sidePadding {
		t.Errorf("rendered width with match-highlight = %d, want ≤ %d (ANSI bytes must not inflate width)",
			lipgloss.Width(rendered), completionsWidth+sidePadding)
	}
	if !strings.Contains(rendered, "…") {
		t.Errorf("expected ellipsis in truncated highlighted name, got: %q", rendered)
	}
	// The match-highlight style sets a foreground colour. After truncation
	// the leading SGR sequence must still be present — verifiable by
	// checking that the rendered string differs from an un-highlighted
	// render of the same row (proves the ANSI survived the cut).
	m.matchPrefix = ""
	unhighlighted := m.renderCompletionRow(row, 5, completionsWidth, false, false)
	if rendered == unhighlighted {
		t.Errorf("truncation stripped the match-highlight ANSI:\n  with    %q\n  without %q",
			rendered, unhighlighted)
	}
}

func TestRenderCompletionRow_NarrowTerminalDoesNotPanic(t *testing.T) {
	// completionsWidth < nameWidth would have passed negative values to
	// lipgloss.Width / PaddingLeft before the clamp.
	m, err := New(TestCommands, 4)
	if err != nil {
		t.Fatal(err)
	}
	row := completionRow{Name: "very-long-completion-name", Description: "desc", Kind: commandKind}
	// titleWidth and completionsWidth deliberately smaller than the name's
	// display width to exercise the negative-result paths.
	_ = m.renderCompletionRow(row, 3, 5, false, false)
	_ = m.renderCompletionRow(row, 3, 5, true, false)
	_ = m.renderCompletionRow(row, 3, 5, false, true)
}

func TestRender_DescriptionsHidden(t *testing.T) {
	m, err := New(TestCommands, 100, WithDescriptions(false))
	if err != nil {
		t.Fatal(err)
	}
	m.input.SetValue("git ")
	m.completions, m.matchPrefix = m.getCompletions()
	if len(m.completions) == 0 {
		t.Fatal("expected completions for 'git '")
	}

	out := m.Render()

	// With descriptions hidden, no completion's description text should appear.
	for _, c := range m.completions {
		desc := c.getDescription()
		if desc == "" {
			continue
		}
		if strings.Contains(out, desc) {
			t.Errorf("Render() unexpectedly contains description %q while ShowDescriptions=false", desc)
		}
	}
}

func TestCompletionBoxWidth(t *testing.T) {
	rows := []completionRow{
		{Name: "short", Description: "tiny"},
		{Name: "longer-name", Description: "a longer description"},
	}

	m, err := New(TestCommands, 100)
	if err != nil {
		t.Fatal(err)
	}

	titleWidth, lineWidth := m.completionBoxWidth(rows)
	if titleWidth != len("longer-name")+3 {
		t.Errorf("titleWidth = %d, want %d", titleWidth, len("longer-name")+3)
	}
	if lineWidth != titleWidth+len("a longer description") {
		t.Errorf("lineWidth = %d, want %d", lineWidth, titleWidth+len("a longer description"))
	}

	mIcons, err := New(TestCommands, 100, WithIcons(true))
	if err != nil {
		t.Fatal(err)
	}
	titleWithIcons, _ := mIcons.completionBoxWidth(rows)
	// Default icons are 1 cell wide, so titleWidth grows by 1 + 1 space = 2.
	if titleWithIcons != titleWidth+2 {
		t.Errorf("titleWidth with icons = %d, want %d", titleWithIcons, titleWidth+2)
	}
}

func TestCalculateCompletionsOffset(t *testing.T) {
	// Cases pin the offset math to known values so layout regressions surface.
	// Assumes the default textinput prompt "> " (width 2).
	cases := []struct {
		name              string
		width             int
		input             string
		completions       string
		completionsOffset int
		indent            bool
		want              int
	}{
		{
			name:  "indent disabled returns configured offset",
			width: 80, input: "git", completionsOffset: 5, indent: false,
			want: 5,
		},
		{
			name:  "empty input returns zero",
			width: 80, input: "", indent: true,
			want: 0,
		},
		{
			name:  "mid-token: aligns to start of active token after prompt",
			width: 80, input: "git c", indent: true,
			want: 2 + len("git "),
		},
		{
			name:  "trailing space: aligns to end of input after prompt",
			width: 80, input: "git ", indent: true,
			want: 2 + len("git "),
		},
		{
			name:  "nonzero completionsOffset adds on top of base",
			width: 80, input: "git c", completionsOffset: 3, indent: true,
			want: 2 + len("git ") + 3,
		},
		{
			name:  "wraps modulo terminal width on long input",
			width: 10, input: "git commit ", indent: true,
			// "> git commit " is 13 cells; the new token starts at col (13 % 10) = 3 on the wrapped line
			want: 3,
		},
		{
			name:  "quoted argument with spaces aligns at start of quoted token",
			width: 80, input: `cat "my file"`, indent: true,
			// LastIndex of `"my file"` is byte 4 → trimmedInput "cat " (4 cells)
			want: 2 + 4,
		},
		{
			name:  "wide characters before active token use display width",
			width: 80, input: "日本 file", indent: true,
			// "日本 " is 5 cells (2+2+1)
			want: 2 + 5,
		},
		{
			name: "overflow shifts box left so it stays on-screen",
			// base offset would be 2 + len("git ") = 6; completions is 30 cells
			// 6 + 30 = 36 > 32: overflow branch trips, offset = 32 - 30 - 2 = 0
			width: 32, input: "git c", completions: strings.Repeat("x", 30), indent: true,
			want: 0,
		},
		{
			name: "tiny terminal overflow clamps offset to zero",
			// When completions exceed the terminal width by more than the
			// prompt + token can absorb, offset would otherwise go negative
			// and produce a negative lipgloss Margin. Clamped to 0.
			width: 10, input: "g", completions: strings.Repeat("x", 15), indent: true,
			want: 0,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m, err := New(TestCommands, c.width,
				WithCompletionsOffset(c.completionsOffset),
				WithIndentCompletions(c.indent),
			)
			if err != nil {
				t.Fatal(err)
			}
			m.input.SetValue(c.input)
			got := m.calculateCompletionsOffset(c.completions)
			if got != c.want {
				t.Errorf("input=%q width=%d completions=%dcells: got %d, want %d",
					c.input, c.width, lipgloss.Width(c.completions), got, c.want)
			}
		})
	}
}

func TestRenderScrollbar(t *testing.T) {
	const thumb, track = "┃", "│"

	cases := []struct {
		name                  string
		height, total, offset int
		want                  []string
	}{
		{"zero height returns nil", 0, 20, 0, nil},
		{"zero total returns nil", 5, 0, 0, nil},
		{
			"first window: thumb at top", 5, 20, 0,
			[]string{thumb, track, track, track, track},
		},
		{
			"last window: thumb at bottom", 5, 20, 15,
			[]string{track, track, track, track, thumb},
		},
		{
			"middle: thumb in middle", 5, 10, 2,
			[]string{track, thumb, thumb, track, track},
		},
		{
			"thumb size never zero on huge totals", 5, 1000, 0,
			[]string{thumb, track, track, track, track},
		},
		{
			"thumb clamped to height when total <= height", 5, 5, 0,
			[]string{thumb, thumb, thumb, thumb, thumb},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := renderScrollbar(c.height, c.total, c.offset, ScrollbarStyles{})
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}

func TestVisibleWindow(t *testing.T) {
	cases := []struct {
		name                  string
		selected, total, rows int
		wantStart, wantEnd    int
	}{
		{"no selection", -1, 10, 5, 0, 5},
		{"first row selected", 0, 10, 5, 0, 5},
		{"middle row selected", 4, 10, 5, 0, 5},
		{"row 5 scrolls window", 5, 10, 5, 1, 6},
		{"last row selected", 9, 10, 5, 5, 10},
		{"rows larger than total", 0, 3, 5, 0, 3},
		{"zero rows", 0, 10, 0, 0, 0},
		{"zero total", 0, 0, 5, 0, 0},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			start, end := visibleWindow(c.selected, c.total, c.rows)
			if start != c.wantStart || end != c.wantEnd {
				t.Errorf("visibleWindow(%d, %d, %d) = (%d, %d), want (%d, %d)",
					c.selected, c.total, c.rows, start, end, c.wantStart, c.wantEnd)
			}
		})
	}
}

func TestFindMatchRange(t *testing.T) {
	cases := []struct {
		name          string
		displayName   string
		prefix        string
		expectedStart int
		expectedEnd   int
	}{
		{"simple prefix", "commit", "co", 0, 2},
		{"full match", "git", "git", 0, 3},
		{"long flag in compound name", "-m --message", "--me", 3, 7},
		{"short flag in compound name", "-m --message", "-m", 0, 2},
		{"powershell flag", "-Verbose", "-Verb", 0, 5},
		{"no match", "commit", "xx", -1, -1},
		{"empty prefix", "commit", "", -1, -1},
		{"case insensitive", "-FileDirArg", "-file", 0, 5},
		// Wide-rune cases. lipgloss measures a CJK char as 2 cells, so the
		// returned cell range must reflect that — a rune-based
		// implementation would report (0, 1) and only highlight half the
		// glyph.
		{"wide-rune prefix matches in cells", "日本.txt", "日", 0, 2},
		{"wide-rune prefix spans two chars", "日本.txt", "日本", 0, 4},
		// Wide-rune word offset: when the matched word comes after a
		// space-separated wide-rune word, the returned start must reflect
		// cells (not runes) past the leading wide-rune word.
		{"ASCII match after wide-rune word", "日 file.txt", "fi", 3, 5},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			start, end := findMatchRange(c.displayName, c.prefix)
			if start != c.expectedStart || end != c.expectedEnd {
				t.Errorf("findMatchRange(%q, %q) = (%d, %d), want (%d, %d)",
					c.displayName, c.prefix, start, end, c.expectedStart, c.expectedEnd)
			}
		})
	}
}

// TestRenderTruncationFooter_BranchWording locks each of the three footer
// wording branches: verified-only ("+ N more"), unresolved-only ("N
// symlinks unresolved"), and combined ("+ N more (M unresolved)"). The
// integration test (TestRender_TruncationFooter_AppearsWhenCandidatesDropped)
// exercises the verified-only case end-to-end through Update; this is the
// direct unit-test counterpart for the wording dispatch.
func TestRenderTruncationFooter_BranchWording(t *testing.T) {
	m, err := New(TestCommands, 100)
	if err != nil {
		t.Fatal(err)
	}
	m.pathState.active = true

	cases := []struct {
		name              string
		droppedSorted     int
		unresolvedEntries int
		wantSubstr        []string
		dontWant          []string
	}{
		{
			name:          "verified drops only",
			droppedSorted: 5,
			wantSubstr:    []string{"+ 5 more", "type to narrow"},
			dontWant:      []string{"unresolved"},
		},
		{
			name:              "unresolved only (weaker wording)",
			unresolvedEntries: 3,
			wantSubstr:        []string{"3 symlinks unresolved", "narrow to filter"},
			dontWant:          []string{"+ ", "more"},
		},
		{
			name:              "both drop kinds combined",
			droppedSorted:     5,
			unresolvedEntries: 3,
			wantSubstr:        []string{"+ 5 more", "3 unresolved", "type to narrow"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m.pathState.droppedSorted = c.droppedSorted
			m.pathState.unresolvedEntries = c.unresolvedEntries
			got := m.renderTruncationFooter(0)
			for _, want := range c.wantSubstr {
				if !strings.Contains(got, want) {
					t.Errorf("footer missing %q; got %q", want, got)
				}
			}
			for _, no := range c.dontWant {
				if strings.Contains(got, no) {
					t.Errorf("footer should NOT contain %q; got %q", no, got)
				}
			}
		})
	}
}
