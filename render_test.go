package bubblecomplete

import (
	"strings"
	"testing"
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

func TestCompletionBoxWidth(t *testing.T) {
	rows := []completionRow{
		{Name: "short", Description: "tiny"},
		{Name: "longer-name", Description: "a longer description"},
	}

	titleWidth, lineWidth := completionBoxWidth(rows, false)
	if titleWidth != len("longer-name")+3 {
		t.Errorf("titleWidth = %d, want %d", titleWidth, len("longer-name")+3)
	}
	if lineWidth != titleWidth+len("a longer description") {
		t.Errorf("lineWidth = %d, want %d", lineWidth, titleWidth+len("a longer description"))
	}

	titleWithIcons, _ := completionBoxWidth(rows, true)
	if titleWithIcons != titleWidth+2 {
		t.Errorf("titleWidth with icons = %d, want %d", titleWithIcons, titleWidth+2)
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
