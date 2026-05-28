package bubblecomplete

import (
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

// MARK: Types and Vars

// SelectedCommandMsg is the message [Model.Update] returns when the user
// submits a command via the Submit binding (Enter by default). Command is the
// raw submitted string (trimmed if Autotrim is true); Err is the result of
// validating it — non-nil for invalid input.
type SelectedCommandMsg struct {
	Command string
	Err     error
}

// MARK: Public Functions

// Update advances the component in response to a Bubble Tea message. Host
// models should forward every received message here and use the returned
// model and command.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	// Handle key presses
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(keyMsg, m.keymap.NextCompletion):
			m, cmd = m.keyTab(true)
		case key.Matches(keyMsg, m.keymap.PrevCompletion):
			m, cmd = m.keyTab(false)
		case key.Matches(keyMsg, m.keymap.Submit):
			m, cmd = m.keyEnter()
		case key.Matches(keyMsg, m.keymap.HistoryPrev):
			m, cmd = m.keyUp()
		case key.Matches(keyMsg, m.keymap.HistoryNext):
			m, cmd = m.keyDown()
		case key.Matches(keyMsg, m.keymap.AcceptCompletion):
			m, cmd = m.keyRight()
		case keyMsg.String() == "backspace":
			m, cmd = m.keyBackspace()
		default:
			m, cmd = m.keyDefault(keyMsg)
		}
	}
	cmds = append(cmds, cmd)

	// Update the text input
	m.input, cmd = m.input.Update(msg)
	cmds = append(cmds, cmd)

	// If the input has changed, update the completions and validate the input
	if m.input.Value() != "" && m.input.Value() != m.lastInput && m.completionHolder == "" && !m.showAll {
		m.lastInput = m.input.Value()
		m.recomputePathState()
		m.completions, m.matchPrefix = m.getCompletions()
		m.validationErr = m.validateInput()
		m.applyInputValidationStyle()
	} else if m.input.Value() == "" && m.lastInput != "" {
		// Input cleared. Always reset lastInput and pathState so a stale
		// path-completion overlay can't survive a delete-all; clear the
		// validation style only when there's an error to clear.
		m.lastInput = ""
		m.pathState = pathState{}
		if m.validationErr != nil {
			m.validationErr = nil
			m.applyInputValidationStyle()
		}
	}

	// If not loaded, start the blinking cursor
	if !m.loaded {
		m.loaded = true
		cmds = append(cmds, textinput.Blink)
	}

	return m, tea.Batch(cmds...)
}

// ShowingCompletions returns true if the completions are currently visible
func (m *Model) ShowingCompletions() bool {
	return len(m.completions) > 0 && m.historyIndex == -1 && (m.input.Value() != "" || m.showAll)
}

// CloseCompletions hides the list of completions so it's no longer visible
//
// Sets the input back to what the user had typed, if completions were being cycled through
func (m *Model) CloseCompletions() {
	if m.completions == nil {
		return
	}
	m.completions = []completion{}
	if m.completionHolder != "" || m.showAll {
		m.input.SetValue(m.completionHolder)
		m.completionHolder = ""
	}
	m.completionIndex = -1
	m.matchPrefix = ""
	m.showAll = false
}

// MARK: Private Functions

func (m Model) resetModel() Model {
	m.input.SetValue("")
	m.completions = []completion{}
	m.matchPrefix = ""
	m.lastInput = ""
	m.validationErr = nil
	m.pathState = pathState{}
	m = m.clearTransientCompletionState()
	m.applyInputValidationStyle()
	return m
}

// clearTransientCompletionState wipes the per-keystroke completion-cycling
// and history-cycling state. Shared by resetModel (full submit reset) and
// the keyBackspace / keyDefault paths, which need the same wipe without
// touching the input value or validation state.
func (m Model) clearTransientCompletionState() Model {
	m.completionHolder = ""
	m.completionIndex = -1
	m.historyIndex = -1
	m.filteredHistory = nil
	m.showAll = false
	return m
}

func splitInput(input string) []string {
	tokens := tokenize(input)
	if len(tokens) == 0 {
		return nil
	}
	out := make([]string, len(tokens))
	for i, tk := range tokens {
		out[i] = tk.Raw
	}
	return out
}

// MARK: Key Handlers

func (m Model) keyUp() (Model, tea.Cmd) {
	if len(m.History) == 0 || m.completionIndex != -1 {
		return m, nil
	}

	if len(m.filteredHistory) == 0 {
		m.filteredHistory = append(m.filteredHistory, m.input.Value())
		for _, h := range m.History {
			if strings.HasPrefix(h, m.input.Value()) {
				m.filteredHistory = append(m.filteredHistory, h)
			}
		}
	}
	if len(m.filteredHistory) < 2 {
		return m, nil
	}

	if m.historyIndex == -1 {
		m.historyIndex = 1
	} else if m.historyIndex < len(m.filteredHistory)-1 {
		m.historyIndex++
	}

	m.input.SetValue(m.filteredHistory[m.historyIndex])
	m.input.CursorEnd()
	return m, nil
}

func (m Model) keyDown() (Model, tea.Cmd) {
	if len(m.filteredHistory) < 2 || m.completionIndex != -1 {
		return m, nil
	}

	if m.historyIndex == -1 {
		m.historyIndex = len(m.filteredHistory) - 2
	} else if m.historyIndex > 0 {
		m.historyIndex--
	}

	m.input.SetValue(m.filteredHistory[m.historyIndex])
	m.input.CursorEnd()
	return m, nil
}

func (m Model) keyRight() (Model, tea.Cmd) {
	// If a tab completion is currently selected, accept it.
	// Completions and validation are recalculated by Update() since
	// clearing completionHolder makes its guard condition true.
	if m.completionIndex >= 0 {
		m.completionHolder = ""
		m.completionIndex = -1
		m.showAll = false
		return m, nil
	}

	if m.input.Value() == "" {
		return m, nil
	}

	// Accept history inline suggestion
	if len(m.input.MatchedSuggestions()) == 0 {
		return m, nil
	}
	m.input.SetValue(m.input.CurrentSuggestion())
	m.input.CursorEnd()
	m.completionIndex = -1
	m.completionHolder = ""
	m.showAll = false
	return m, nil
}

func (m Model) keyTab(forward bool) (Model, tea.Cmd) {
	trimmedInput := strings.TrimSpace(m.input.Value())

	// If the input is empty, show all completions
	if trimmedInput == "" && !m.showAll {
		m.showAll = true
		m.completions, m.matchPrefix = m.getCompletions()
		return m, nil
	}

	// If there are no completions, do nothing
	if len(m.completions) == 0 {
		return m, nil
	}

	// If there is only one completion and it matches the input, do nothing
	if len(m.completions) == 1 {
		if strings.HasSuffix(trimmedInput, m.completions[0].getName()) {
			return m, nil
		}
	}

	// Cycle and update the completion index
	if forward {
		if m.completionIndex < len(m.completions)-1 {
			m.completionIndex++
		} else {
			m.completionIndex = -1
		}
	} else {
		if m.completionIndex > -1 {
			m.completionIndex--
		} else {
			m.completionIndex = len(m.completions) - 1
		}
	}

	// Save the current input if we haven't already
	if m.completionHolder == "" && !m.showAll {
		m.completionHolder = m.input.Value()
	}

	// If the completion index is -1, reset the input to the completion holder
	if m.completionIndex == -1 {
		m.input.SetValue(m.completionHolder)
		m.completionHolder = ""
		// Refresh pathState against the restored value. The input-changed
		// branch later in Update won't help here: the restored value
		// equals m.lastInput (set before cycling began), so the branch
		// skips. Without this refresh, pathState would still belong to
		// the last-cycled preview value — stale offsets, wrong validity,
		// or (after a cycled file's trailing space made it inactive) no
		// overlay at all on a path that should now show partial/invalid.
		m.recomputePathState()
		return m, nil
	}

	// If the completion is empty (aka positional arg), don't update the input
	if m.completions[m.completionIndex].getAutocomplete() == "" {
		return m, nil
	}

	pretext := m.completionHolder
	parts := splitInput(m.completionHolder)
	// If the pretext doesn't end in a space, add one
	if !strings.HasSuffix(pretext, " ") && len(parts) > 0 {
		pretext = strings.Join(parts[:len(parts)-1], " ")
		if len(parts) > 1 {
			pretext += " "
		}
	}

	// Update the input with the current completion
	m.input.SetValue(pretext + m.completions[m.completionIndex].getAutocomplete())
	m.input.CursorEnd()

	if len(m.completions) == 1 {
		// Single-match acceptance: treat Tab as a final accept rather than
		// entering cycling state. Clearing completionHolder here means the
		// input-changed branch later in this same Update call (after
		// textinput.Update) will recompute against the new input value —
		// populating fresh completions and pathState for the just-accepted
		// text. The next Tab then cycles among those new candidates, which
		// for a directory match enables the shell-style drill-down: Tab
		// on "cd Do" with a unique "Documents/" candidate accepts it AND
		// repopulates with Documents/'s children, so the second Tab
		// descends one level.
		m.completionHolder = ""
		m.completionIndex = -1
		m.showAll = false
	} else {
		// Multi-match cycling: completionHolder stays set so the user can
		// continue cycling or revert to the original via Shift+Tab past
		// the start. The input-changed branch is therefore skipped on the
		// next Update tick. Refresh pathState here so the render overlay
		// reflects the *cycled* value's validity — without this, the
		// frozen offsets and validity would mis-style the preview (the
		// old cycling-bypass behaviour in renderedInput). m.completions
		// is intentionally NOT recomputed: the cycling list is the
		// candidates we entered cycling with.
		m.recomputePathState()
	}
	return m, nil
}

func (m Model) keyEnter() (Model, tea.Cmd) {
	command := m.input.Value()
	if m.Autotrim {
		command = strings.TrimSpace(m.input.Value())
	}

	// Re-validate against the current input to avoid stale errors from
	// before tab completion changed the value.
	m.validationErr = m.validateInput()
	validationErr := m.validationErr

	// HistoryLimit <= 0 disables history entirely.
	if m.HistoryLimit > 0 && command != "" && (len(m.History) == 0 || m.History[0] != command) {
		m.History = append([]string{command}, m.History...)
		if len(m.History) > m.HistoryLimit {
			m.History = m.History[:m.HistoryLimit]
		}
	}
	m = m.resetModel()
	// When history is disabled, skip the save entirely — otherwise a
	// configured history file would get rewritten to {"history":null} on
	// every Enter. Clear any prior error since no operation was attempted.
	if m.HistoryLimit > 0 {
		m.err = m.saveHistoryToFile()
	} else {
		m.err = nil
	}
	m.input.SetSuggestions(m.History)

	return m, func() tea.Msg {
		return SelectedCommandMsg{Command: command, Err: validationErr}
	}
}

func (m Model) keyBackspace() (Model, tea.Cmd) {
	return m.clearTransientCompletionState(), nil
}

func (m Model) keyDefault(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	if msg.Text == "" {
		return m, nil
	}
	return m.clearTransientCompletionState(), nil
}
