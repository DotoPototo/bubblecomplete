package bubblecomplete

import (
	"fmt"
	"testing"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// simulateTyping sends each character of text through the full Update cycle.
func simulateTyping(t *testing.T, m Model, text string) Model {
	t.Helper()
	for _, r := range text {
		msg := tea.KeyPressMsg{Code: r, Text: string(r)}
		m, _ = m.Update(msg)
	}
	return m
}

// simulateKey sends a single special key through the full Update cycle.
func simulateKey(t *testing.T, m Model, code rune) Model {
	t.Helper()
	m, _ = m.Update(tea.KeyPressMsg{Code: code})
	return m
}

func simulateKeyMod(t *testing.T, m Model, code rune, mod tea.KeyMod) Model {
	t.Helper()
	m, _ = m.Update(tea.KeyPressMsg{Code: code, Mod: mod})
	return m
}

// extractSelectedCommand walks the Update result for a SelectedCommandMsg.
// tea.Batch returns a single cmd directly when only one non-nil cmd exists,
// otherwise it wraps in BatchMsg, so we handle both cases.
func extractSelectedCommand(cmd tea.Cmd) (SelectedCommandMsg, bool) {
	if cmd == nil {
		return SelectedCommandMsg{}, false
	}
	raw := cmd()
	if msg, ok := raw.(SelectedCommandMsg); ok {
		return msg, true
	}
	if batch, ok := raw.(tea.BatchMsg); ok {
		for _, c := range batch {
			if c == nil {
				continue
			}
			if msg, ok := c().(SelectedCommandMsg); ok {
				return msg, true
			}
		}
	}
	return SelectedCommandMsg{}, false
}

func pressEnter(t *testing.T, m Model) (Model, SelectedCommandMsg) {
	t.Helper()
	m, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	msg, ok := extractSelectedCommand(cmd)
	if !ok {
		t.Fatal("SelectedCommandMsg not found for enter")
	}
	return m, msg
}

func newTestModel(t *testing.T) Model {
	t.Helper()
	m, err := New(TestCommands, 100)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestTabCompletionThenEnter(t *testing.T) {
	testCases := []struct {
		name          string
		typed         string
		expectedValue string
		expectErr     bool
	}{
		{
			name:          "partial subcommand completes and validates",
			typed:         "git stash a",
			expectedValue: "git stash apply",
		},
		{
			name:          "different partial subcommand",
			typed:         "git stash p",
			expectedValue: "git stash pop",
		},
		{
			name:          "top-level partial command",
			typed:         "ca",
			expectedValue: "cat",
			expectErr:     true, // "cat" without file arg is invalid
		},
		{
			name:          "from trailing space",
			typed:         "git stash ",
			expectedValue: "git stash apply",
		},
		{
			name:          "flag partial completion",
			typed:         "git commit --am",
			expectedValue: "git commit --amend",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := newTestModel(t)
			m = simulateTyping(t, m, tc.typed)
			m = simulateKey(t, m, tea.KeyTab)

			if m.input.Value() != tc.expectedValue {
				t.Fatalf("Expected %q after tab, got %q", tc.expectedValue, m.input.Value())
			}

			_, msg := pressEnter(t, m)
			if msg.Command != tc.expectedValue {
				t.Errorf("Expected command %q, got %q", tc.expectedValue, msg.Command)
			}
			if tc.expectErr && msg.Err == nil {
				t.Error("Expected validation error, got nil")
			}
			if !tc.expectErr && msg.Err != nil {
				t.Errorf("Expected no validation error, got: %v", msg.Err)
			}
		})
	}
}

func TestTabCompletionThenEnter_StaleValidationFixed(t *testing.T) {
	// This is the exact scenario from the bug report:
	// type "git stash a" (validation error), tab to "git stash apply", enter.
	// Before the fix, the stale "unexpected argument: a" error persisted.
	m := newTestModel(t)
	m = simulateTyping(t, m, "git stash a")

	if m.validationErr == nil {
		t.Fatal("Expected validation error after typing 'git stash a'")
	}

	m = simulateKey(t, m, tea.KeyTab)

	if m.input.Value() != "git stash apply" {
		t.Fatalf("Expected 'git stash apply' after tab, got %q", m.input.Value())
	}

	_, msg := pressEnter(t, m)
	if msg.Err != nil {
		t.Errorf("Expected no validation error after tab-completing to valid command, got: %v", msg.Err)
	}
}

func TestTabCycling_ReturnsToOriginal(t *testing.T) {
	m := newTestModel(t)

	m = simulateTyping(t, m, "git stash ")
	originalValue := m.input.Value()
	numCompletions := len(m.completions)

	if numCompletions == 0 {
		t.Fatal("Expected completions for 'git stash '")
	}

	// Tab through all completions and back to original
	for i := 0; i <= numCompletions; i++ {
		m = simulateKey(t, m, tea.KeyTab)
	}

	if m.input.Value() != originalValue {
		t.Errorf("Expected input to return to %q after full tab cycle, got %q", originalValue, m.input.Value())
	}
	if m.completionIndex != -1 {
		t.Errorf("Expected completionIndex -1 after full cycle, got %d", m.completionIndex)
	}
}

func TestShiftTabCycling_ReversesDirection(t *testing.T) {
	m := newTestModel(t)

	m = simulateTyping(t, m, "git stash ")

	m = simulateKey(t, m, tea.KeyTab)
	firstValue := m.input.Value()

	m = simulateKeyMod(t, m, tea.KeyTab, tea.ModShift)

	if m.completionIndex != -1 {
		t.Errorf("Expected completionIndex -1 after shift-tab from first, got %d", m.completionIndex)
	}

	m = simulateKey(t, m, tea.KeyTab)
	if m.input.Value() != firstValue {
		t.Errorf("Expected same first completion %q, got %q", firstValue, m.input.Value())
	}
}

func TestMultipleTabsThenEnter(t *testing.T) {
	m := newTestModel(t)

	m = simulateTyping(t, m, "git c")

	if len(m.completions) < 2 {
		t.Fatalf("Expected multiple completions for 'git c', got %d", len(m.completions))
	}

	m = simulateKey(t, m, tea.KeyTab)
	firstCompletion := m.input.Value()

	m = simulateKey(t, m, tea.KeyTab)
	secondCompletion := m.input.Value()

	if firstCompletion == secondCompletion {
		t.Error("Expected different completions on successive tabs")
	}

	_, msg := pressEnter(t, m)
	if msg.Command != secondCompletion {
		t.Errorf("Expected command %q, got %q", secondCompletion, msg.Command)
	}
}

func TestRightArrow_AcceptsTabCompletion(t *testing.T) {
	m := newTestModel(t)

	m = simulateTyping(t, m, "git stash a")
	m = simulateKey(t, m, tea.KeyTab)

	if m.completionIndex < 0 {
		t.Fatal("Expected completionIndex >= 0 after tab")
	}
	if m.completionHolder == "" {
		t.Fatal("Expected completionHolder to be set after tab")
	}

	expectedValue := m.input.Value()

	m = simulateKey(t, m, tea.KeyRight)

	if m.completionIndex != -1 {
		t.Errorf("Expected completionIndex -1 after right arrow accept, got %d", m.completionIndex)
	}
	if m.completionHolder != "" {
		t.Errorf("Expected empty completionHolder after right arrow accept, got %q", m.completionHolder)
	}
	if m.input.Value() != expectedValue {
		t.Errorf("Expected input to stay %q after right arrow, got %q", expectedValue, m.input.Value())
	}
}

func TestCtrlE_RoutesToKeyRight(t *testing.T) {
	// Verify ctrl+e is routed to the same handler as right arrow by checking
	// that it clears the completion holder when a completion is selected.
	m := newTestModel(t)

	m = simulateTyping(t, m, "git stash a")
	m = simulateKey(t, m, tea.KeyTab)

	if m.completionHolder == "" {
		t.Fatal("Expected completionHolder to be set after tab")
	}

	// ctrl+e should clear completionHolder (same as right arrow)
	m, _ = m.Update(tea.KeyPressMsg{Code: 'e', Mod: tea.ModCtrl})

	if m.completionHolder != "" {
		t.Errorf("Expected empty completionHolder after ctrl+e, got %q", m.completionHolder)
	}
}

func TestRightArrow_NoSelection_DoesNotChangeState(t *testing.T) {
	m := newTestModel(t)

	m = simulateTyping(t, m, "git stash ")

	originalIndex := m.completionIndex
	m = simulateKey(t, m, tea.KeyRight)

	if m.completionIndex != originalIndex {
		t.Errorf("Expected completionIndex to stay %d, got %d", originalIndex, m.completionIndex)
	}
}

func TestRightArrow_ThenEnter_ValidCommand(t *testing.T) {
	m := newTestModel(t)

	m = simulateTyping(t, m, "git stash a")
	m = simulateKey(t, m, tea.KeyTab)
	m = simulateKey(t, m, tea.KeyRight)

	_, msg := pressEnter(t, m)
	if msg.Command != "git stash apply" {
		t.Errorf("Expected 'git stash apply', got %q", msg.Command)
	}
	if msg.Err != nil {
		t.Errorf("Expected no error, got: %v", msg.Err)
	}
}

func TestEnter_DirectValidCommand(t *testing.T) {
	m := newTestModel(t)

	m = simulateTyping(t, m, "git stash apply")

	_, msg := pressEnter(t, m)
	if msg.Command != "git stash apply" {
		t.Errorf("Expected 'git stash apply', got %q", msg.Command)
	}
	if msg.Err != nil {
		t.Errorf("Expected no error, got: %v", msg.Err)
	}
}

func TestEnter_InvalidCommand_ReturnsError(t *testing.T) {
	m := newTestModel(t)

	m = simulateTyping(t, m, "git stash xyz")

	_, msg := pressEnter(t, m)
	if msg.Err == nil {
		t.Error("Expected validation error for 'git stash xyz'")
	}
}

func TestEnter_PartialSubcommand_ReturnsError(t *testing.T) {
	m := newTestModel(t)

	m = simulateTyping(t, m, "git stash a")

	_, msg := pressEnter(t, m)
	if msg.Err == nil {
		t.Error("Expected validation error for partial 'git stash a' without tab completion")
	}
}

func TestBackspace_ResetsCompletionState(t *testing.T) {
	m := newTestModel(t)

	m = simulateTyping(t, m, "git stash a")
	m = simulateKey(t, m, tea.KeyTab)

	if m.completionIndex < 0 {
		t.Fatal("Expected active completion after tab")
	}

	m = simulateKey(t, m, tea.KeyBackspace)

	if m.completionIndex != -1 {
		t.Errorf("Expected completionIndex -1 after backspace, got %d", m.completionIndex)
	}
	if m.completionHolder != "" {
		t.Errorf("Expected empty completionHolder after backspace, got %q", m.completionHolder)
	}
}

func TestTypingAfterTab_ResetsCompletionState(t *testing.T) {
	m := newTestModel(t)

	m = simulateTyping(t, m, "git stash a")
	m = simulateKey(t, m, tea.KeyTab)

	if m.completionIndex < 0 {
		t.Fatal("Expected active completion after tab")
	}

	m = simulateTyping(t, m, " ")

	if m.completionIndex != -1 {
		t.Errorf("Expected completionIndex -1 after typing, got %d", m.completionIndex)
	}
	if m.completionHolder != "" {
		t.Errorf("Expected empty completionHolder after typing, got %q", m.completionHolder)
	}
}

func TestEnter_AddsToHistory(t *testing.T) {
	m := newTestModel(t)

	m = simulateTyping(t, m, "git status")
	m, _ = pressEnter(t, m)

	if len(m.History) == 0 {
		t.Fatal("Expected history to have an entry")
	}
	if m.History[0] != "git status" {
		t.Errorf("Expected first history entry 'git status', got %q", m.History[0])
	}
}

func TestEnter_ResetsModelState(t *testing.T) {
	m := newTestModel(t)

	m = simulateTyping(t, m, "git status")
	m, _ = pressEnter(t, m)

	if m.input.Value() != "" {
		t.Errorf("Expected empty input after enter, got %q", m.input.Value())
	}
	if m.completionIndex != -1 {
		t.Errorf("Expected completionIndex -1 after enter, got %d", m.completionIndex)
	}
	if len(m.completions) != 0 {
		t.Errorf("Expected no completions after enter, got %d", len(m.completions))
	}
}

func TestValueAndValidationError(t *testing.T) {
	m := newTestModel(t)

	if m.Value() != "" {
		t.Errorf("Expected empty Value(), got %q", m.Value())
	}
	if m.ValidationError() != nil {
		t.Errorf("Expected nil ValidationError(), got %v", m.ValidationError())
	}

	m = simulateTyping(t, m, "git status")
	if m.Value() != "git status" {
		t.Errorf("Expected Value() %q, got %q", "git status", m.Value())
	}
	if m.ValidationError() != nil {
		t.Errorf("Expected nil ValidationError() for valid input, got %v", m.ValidationError())
	}

	m = newTestModel(t)
	m = simulateTyping(t, m, "git stash a")
	if m.ValidationError() == nil {
		t.Error("Expected non-nil ValidationError() for partial subcommand")
	}

	// Backspacing back to empty input should clear the validation error.
	for range "git stash a" {
		m = simulateKey(t, m, tea.KeyBackspace)
	}
	if m.Value() != "" {
		t.Fatalf("Expected empty input after backspaces, got %q", m.Value())
	}
	if m.ValidationError() != nil {
		t.Errorf("Expected nil ValidationError() after clearing input, got %v", m.ValidationError())
	}

	// Submitting any command resets the model; subsequent ValidationError must be nil.
	m = newTestModel(t)
	m = simulateTyping(t, m, "git stash xyz")
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.ValidationError() != nil {
		t.Errorf("Expected nil ValidationError() after submit reset, got %v", m.ValidationError())
	}
}

func TestRender_IsIdempotent(t *testing.T) {
	m := newTestModel(t)
	m = simulateTyping(t, m, "git stash a")

	first := m.Render()
	second := m.Render()
	if first != second {
		t.Errorf("Render not idempotent:\nfirst:  %q\nsecond: %q", first, second)
	}
}

func TestInputTextStyle_TracksValidation(t *testing.T) {
	m := newTestModel(t)
	sample := "x"

	got := m.input.Styles().Focused.Text.Render(sample)
	if got != m.Styles().Input.Valid.Render(sample) {
		t.Error("Initial input Focused.Text should be the Valid style")
	}

	m = simulateTyping(t, m, "git stash a")
	got = m.input.Styles().Focused.Text.Render(sample)
	if got != m.Styles().Input.Invalid.Render(sample) {
		t.Error("After invalid input, Focused.Text should be the Invalid style")
	}

	for range "git stash a" {
		m = simulateKey(t, m, tea.KeyBackspace)
	}
	got = m.input.Styles().Focused.Text.Render(sample)
	if got != m.Styles().Input.Valid.Render(sample) {
		t.Error("After clearing input, Focused.Text should return to the Valid style")
	}
}

func TestWithCompletionRows_ClampsToOne(t *testing.T) {
	for _, n := range []int{-5, 0} {
		m, err := New(TestCommands, 80, WithCompletionRows(n))
		if err != nil {
			t.Fatal(err)
		}
		if m.CompletionRows != 1 {
			t.Errorf("WithCompletionRows(%d) yielded %d, want 1", n, m.CompletionRows)
		}
	}
}

func TestNew_ClampsWidth(t *testing.T) {
	for _, w := range []int{-10, 0} {
		m, err := New(TestCommands, w)
		if err != nil {
			t.Fatal(err)
		}
		if m.width != 1 {
			t.Errorf("width clamped from %d to %d, want 1", w, m.width)
		}
	}
}

func TestSetWidth_Clamps(t *testing.T) {
	m := newTestModel(t)
	m.SetWidth(-5)
	if m.width != 1 {
		t.Errorf("SetWidth(-5) produced width=%d, want 1", m.width)
	}
}

func TestRender_AcrossWidths(t *testing.T) {
	// Renders must not panic across the width spectrum, including degenerate
	// cases that previously divided by zero in calculateCompletionsOffset.
	for _, w := range []int{1, 5, 20, 60, 68, 200} {
		t.Run(fmt.Sprintf("width=%d", w), func(t *testing.T) {
			m, err := New(TestCommands, w)
			if err != nil {
				t.Fatal(err)
			}
			m = simulateTyping(t, m, "git c")
			_ = m.Render()
		})
	}
}

func TestGetCompletionsWidth_RespectsNarrowTerminal(t *testing.T) {
	cases := []struct {
		name          string
		terminalWidth int
		maxLineLength int
		want          int
	}{
		{"wide terminal, narrow content", 200, 30, 30},
		{"wide terminal, overflowing content", 200, 300, 192},
		{"narrow terminal must not return 60", 20, 100, 12},
		{"border exceeds width", 8, 100, 1},
		{"single cell terminal", 1, 100, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m, err := New(TestCommands, c.terminalWidth)
			if err != nil {
				t.Fatal(err)
			}
			got := m.getCompletionsWidth(c.maxLineLength)
			if got != c.want {
				t.Errorf("width=%d, maxLine=%d: got %d, want %d",
					c.terminalWidth, c.maxLineLength, got, c.want)
			}
		})
	}
}

func TestRender_HandlesZeroCompletionRows(t *testing.T) {
	m := newTestModel(t)
	m.CompletionRows = 0
	m.ShowScrollbar = true
	m = simulateTyping(t, m, "git c")

	if len(m.completions) == 0 {
		t.Fatal("expected completions for 'git c'")
	}

	// Must not panic and must not silently render an empty completion box.
	_ = m.Render()
}

func TestNew_AppliesOptions(t *testing.T) {
	customKM := DefaultKeyMap()
	customKM.Submit = key.NewBinding(key.WithKeys("ctrl+s"))

	m, err := New(TestCommands, 80,
		WithHistoryLimit(7),
		WithCompletionRows(3),
		WithIcons(true),
		WithCompletionsPosition(PositionAbove),
		WithKeyMap(customKM),
		WithPlaceholder("type stuff"),
	)
	if err != nil {
		t.Fatal(err)
	}

	if m.HistoryLimit != 7 {
		t.Errorf("HistoryLimit = %d, want 7", m.HistoryLimit)
	}
	if m.CompletionRows != 3 {
		t.Errorf("CompletionRows = %d, want 3", m.CompletionRows)
	}
	if !m.ShowIcons {
		t.Error("ShowIcons = false, want true")
	}
	if m.CompletionsPosition != PositionAbove {
		t.Errorf("CompletionsPosition = %v, want PositionAbove", m.CompletionsPosition)
	}
	if got := m.KeyMap().Submit.Keys(); len(got) != 1 || got[0] != "ctrl+s" {
		t.Errorf("Submit keys = %v, want [ctrl+s]", got)
	}
	if m.input.Placeholder != "type stuff" {
		t.Errorf("Placeholder = %q, want %q", m.input.Placeholder, "type stuff")
	}
}

func TestSetKeyMap_RebindSubmit(t *testing.T) {
	m := newTestModel(t)

	k := m.KeyMap()
	k.Submit = key.NewBinding(key.WithKeys("ctrl+s"))
	m.SetKeyMap(k)

	m = simulateTyping(t, m, "git status")

	m, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if _, ok := extractSelectedCommand(cmd); ok {
		t.Error("Enter should not submit after Submit rebind")
	}
	if m.input.Value() != "git status" {
		t.Errorf("Expected input preserved after non-binding enter, got %q", m.input.Value())
	}

	m, cmd = m.Update(tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	msg, ok := extractSelectedCommand(cmd)
	if !ok {
		t.Fatal("Expected ctrl+s to submit after rebind")
	}
	if msg.Command != "git status" {
		t.Errorf("Expected 'git status', got %q", msg.Command)
	}
}
