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

	m.input, cmd = m.input.Update(msg)
	cmds = append(cmds, cmd)

	if m.input.Value() != "" && m.input.Value() != m.lastInput && m.completionHolder == "" && !m.showAll {
		m.lastInput = m.input.Value()
		m.recomputePathState()
		m.completions, m.matchPrefix = m.getCompletions()
		m.validationErr = m.validateInput()
		m.applyInputValidationStyle()
	} else if m.input.Value() == "" && m.lastInput != "" {
		// Input cleared: reset pathState so a stale overlay can't survive delete-all.
		m.lastInput = ""
		m.pathState = pathState{}
		if m.validationErr != nil {
			m.validationErr = nil
			m.applyInputValidationStyle()
		}
	}

	if !m.loaded {
		m.loaded = true
		cmds = append(cmds, textinput.Blink)
	}

	return m, tea.Batch(cmds...)
}

// ShowingCompletions returns true if the completions are currently visible.
func (m Model) ShowingCompletions() bool {
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

// clearTransientCompletionState wipes per-keystroke completion- and history-
// cycling state without touching the input value or validation error.
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
	// Clearing completionHolder makes Update's input-changed guard true, so
	// completions and validation are recalculated against the accepted value.
	if m.completionIndex >= 0 {
		m.completionHolder = ""
		m.completionIndex = -1
		m.showAll = false
		return m, nil
	}

	if m.input.Value() == "" {
		return m, nil
	}

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

	if trimmedInput == "" && !m.showAll {
		m.showAll = true
		m.completions, m.matchPrefix = m.getCompletions()
		return m, nil
	}

	if len(m.completions) == 0 {
		return m, nil
	}

	if len(m.completions) == 1 {
		if strings.HasSuffix(trimmedInput, m.completions[0].getName()) {
			return m, nil
		}
	}

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

	if m.completionHolder == "" && !m.showAll {
		m.completionHolder = m.input.Value()
	}

	if m.completionIndex == -1 {
		m.input.SetValue(m.completionHolder)
		m.completionHolder = ""
		// Restored value equals m.lastInput, so Update's input-changed branch
		// won't refresh pathState. Do it here to avoid stale cycling overlay.
		m.recomputePathState()
		return m, nil
	}

	// Positional-argument completions have no insertion text — they're info-only rows.
	if m.completions[m.completionIndex].getAutocomplete() == "" {
		return m, nil
	}

	pretext := m.completionHolder
	parts := splitInput(m.completionHolder)
	if !strings.HasSuffix(pretext, " ") && len(parts) > 0 {
		pretext = strings.Join(parts[:len(parts)-1], " ")
		if len(parts) > 1 {
			pretext += " "
		}
	}

	m.input.SetValue(pretext + m.completions[m.completionIndex].getAutocomplete())
	m.input.CursorEnd()

	if len(m.completions) == 1 {
		// Single-match: final accept (not cycling). Clearing completionHolder
		// lets Update's input-changed branch recompute against the new value,
		// enabling shell-style drill-down on directory matches (second Tab
		// descends into the just-accepted dir).
		m.completionHolder = ""
		m.completionIndex = -1
		m.showAll = false
	} else {
		// Multi-match cycling: refresh pathState so the overlay reflects the
		// cycled preview's validity. m.completions is deliberately NOT
		// recomputed — we cycle the list we entered with.
		m.recomputePathState()
	}
	return m, nil
}

func (m Model) keyEnter() (Model, tea.Cmd) {
	command := m.input.Value()
	if m.Autotrim {
		command = strings.TrimSpace(m.input.Value())
	}

	// Re-validate: tab completion may have changed the value since the last tick.
	m.validationErr = m.validateInput()
	validationErr := m.validationErr

	if m.HistoryLimit > 0 && command != "" && (len(m.History) == 0 || m.History[0] != command) {
		m.History = append([]string{command}, m.History...)
		if len(m.History) > m.HistoryLimit {
			m.History = m.History[:m.HistoryLimit]
		}
	}
	m = m.resetModel()
	// History disabled: skip save (would rewrite file to {"history":null}).
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
