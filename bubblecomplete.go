package bubblecomplete

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

// MARK: Types and Vars

type SelectedCommandMsg struct {
	Command string
	Err     error
}

type historyFileJson struct {
	History []string `json:"history"`
}

// MARK: Public Functions

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	// Handle key presses
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.keymap.NextCompletion):
			m, cmd = m.keyTab(true)
		case key.Matches(msg, m.keymap.PrevCompletion):
			m, cmd = m.keyTab(false)
		case key.Matches(msg, m.keymap.Submit):
			m, cmd = m.keyEnter()
		case key.Matches(msg, m.keymap.HistoryPrev):
			m, cmd = m.keyUp()
		case key.Matches(msg, m.keymap.HistoryNext):
			m, cmd = m.keyDown()
		case key.Matches(msg, m.keymap.AcceptCompletion):
			m, cmd = m.keyRight()
		case msg.String() == "backspace":
			m, cmd = m.keyBackspace()
		default:
			m, cmd = m.keyDefault(msg)
		}
	}
	cmds = append(cmds, cmd)

	// Update the text input
	m.input, cmd = m.input.Update(msg)
	cmds = append(cmds, cmd)

	// If the input has changed, update the completions and validate the input
	if m.input.Value() != "" && m.input.Value() != m.lastInput && m.completionHolder == "" && !m.showAll {
		m.lastInput = m.input.Value()
		m.completions, m.matchPrefix = m.getCompletions()
		m.validationErr = m.validateInput()
		m.applyInputValidationStyle()
	} else if m.input.Value() == "" && m.validationErr != nil {
		m.validationErr = nil
		m.lastInput = ""
		m.applyInputValidationStyle()
	}

	// If not loaded, start the blinking cursor
	if !m.loaded {
		m.loaded = true
		cmds = append(cmds, textinput.Blink)
	}

	return m, tea.Batch(cmds...)
}

// SetHistoryFilePath sets the file path to the file used for persisting the command history.
//
// It creates the file if it doesn't exist. The directory of the file path must exist.
// The file path must be a valid JSON file with a .json extension.
func (m *Model) SetHistoryFilePath(path string) {
	cleanPath := filepath.Clean(path)

	file := filepath.Base(cleanPath)
	if file == "" {
		m.Err = errors.New("invalid history file path")
		return
	}
	if filepath.Ext(file) != ".json" {
		m.Err = errors.New("history file must be a JSON file")
		return
	}

	dir := filepath.Dir(cleanPath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		m.Err = err
		return
	}

	m.historyFilePath = cleanPath

	// saveHistoryToFile creates the file via atomic rename, so we no longer
	// pre-create with os.Create (which previously leaked its returned handle).
	if _, err := os.Stat(cleanPath); os.IsNotExist(err) {
		if err := m.saveHistoryToFile(); err != nil {
			m.Err = err
			return
		}
	}

	if err := m.loadHistoryFromFile(); err != nil {
		m.Err = err
	}
}

// ClearHistory clears the command history from all previous commands.
//
// If the history file path is set, it also clears the history on file.
func (m *Model) ClearHistory() {
	m.History = []string{}
	if m.historyFilePath != "" {
		m.Err = m.saveHistoryToFile()
	}
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
	m.completions = []Completion{}
	if m.completionHolder != "" || m.showAll {
		m.input.SetValue(m.completionHolder)
		m.completionHolder = ""
	}
	m.completionIndex = -1
	m.matchPrefix = ""
	m.showAll = false
}

// MARK: Private Functions

// saveHistoryToFile writes the in-memory history to the configured file
// atomically: write to a temp file in the same directory then rename, so a
// crash mid-write can never produce a corrupted history file. Returns nil
// when no history file is configured.
func (m *Model) saveHistoryToFile() error {
	if m.historyFilePath == "" {
		return nil
	}
	data := historyFileJson{History: m.History}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	dir := filepath.Dir(m.historyFilePath)
	tmp, err := os.CreateTemp(dir, "history-*.json.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(jsonData); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return os.Rename(tmpPath, m.historyFilePath)
}

func (m *Model) loadHistoryFromFile() error {
	data, err := os.ReadFile(m.historyFilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	// Tolerate an empty file as empty history.
	if len(data) == 0 {
		m.History = nil
		m.input.SetSuggestions(nil)
		return nil
	}

	jsonData := historyFileJson{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return err
	}

	m.History = jsonData.History
	if m.HistoryLimit > 0 && len(m.History) > m.HistoryLimit {
		m.History = m.History[:m.HistoryLimit]
	}
	if m.HistoryLimit <= 0 {
		m.History = nil
	}
	m.input.SetSuggestions(m.History)
	return nil
}

func (m Model) resetModel() Model {
	m.input.SetValue("")
	m.completions = []Completion{}
	m.completionIndex = -1
	m.matchPrefix = ""
	m.historyIndex = -1
	m.validationErr = nil
	m.applyInputValidationStyle()
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
	if err := m.saveHistoryToFile(); err != nil {
		m.Err = err
	}
	m.input.SetSuggestions(m.History)

	return m, func() tea.Msg {
		return SelectedCommandMsg{Command: command, Err: validationErr}
	}
}

func (m Model) keyBackspace() (Model, tea.Cmd) {
	m.completionHolder = ""
	m.completionIndex = -1
	m.historyIndex = -1
	m.filteredHistory = []string{}
	m.showAll = false
	return m, nil
}

func (m Model) keyDefault(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	if msg.Text == "" {
		return m, nil
	}

	m.completionHolder = ""
	m.completionIndex = -1
	m.historyIndex = -1
	m.filteredHistory = []string{}
	m.showAll = false
	return m, nil
}
