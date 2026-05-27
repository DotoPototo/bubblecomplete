package bubblecomplete

import (
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
	// With empty prefix, no completion name should have the MatchHighlightStyle applied
	styledTest := m.Styles().Completion.Match.Render("x")
	// Extract the ANSI prefix (everything before the content character)
	highlightPrefix := styledTest[:len(styledTest)-len("x\033[0m")]
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
	m.CommandIcon = "🐛" // 2 cells
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
			name: "indent disabled returns configured offset",
			width: 80, input: "git", completionsOffset: 5, indent: false,
			want: 5,
		},
		{
			name: "empty input returns zero",
			width: 80, input: "", indent: true,
			want: 0,
		},
		{
			name: "mid-token: aligns to start of active token after prompt",
			width: 80, input: "git c", indent: true,
			want: 2 + len("git "),
		},
		{
			name: "trailing space: aligns to end of input after prompt",
			width: 80, input: "git ", indent: true,
			want: 2 + len("git "),
		},
		{
			name: "nonzero completionsOffset adds on top of base",
			width: 80, input: "git c", completionsOffset: 3, indent: true,
			want: 2 + len("git ") + 3,
		},
		{
			name: "wraps modulo terminal width on long input",
			width: 10, input: "git commit ", indent: true,
			// "> git commit " is 13 cells; the new token starts at col (13 % 10) = 3 on the wrapped line
			want: 3,
		},
		{
			name: "quoted argument with spaces aligns at start of quoted token",
			width: 80, input: `cat "my file"`, indent: true,
			// LastIndex of `"my file"` is byte 4 → trimmedInput "cat " (4 cells)
			want: 2 + 4,
		},
		{
			name: "wide characters before active token use display width",
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
			name: "tiny terminal overflow currently yields negative offset",
			// Documents current behavior — offset can go negative when completions
			// exceed the terminal width by more than the prompt + token can absorb.
			width: 10, input: "g", completions: strings.Repeat("x", 15), indent: true,
			want: 10 - 15 - 2,
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
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			lower := strings.ToLower(c.prefix)
			runeLen := len([]rune(c.prefix))
			start, end := findMatchRange(c.displayName, lower, runeLen)
			if start != c.expectedStart || end != c.expectedEnd {
				t.Errorf("findMatchRange(%q, %q) = (%d, %d), want (%d, %d)",
					c.displayName, c.prefix, start, end, c.expectedStart, c.expectedEnd)
			}
		})
	}
}
