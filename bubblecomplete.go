package bubblecomplete

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
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
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "ctrl+n", "shift+tab", "ctrl+p":
			m, cmd = m.keyTab(msg.String())
		case "backspace":
			m, cmd = m.keyBackspace()
		case "enter":
			m, cmd = m.keyEnter()
		case "up":
			m, cmd = m.keyUp()
		case "down":
			m, cmd = m.keyDown()
		case "right", "ctrl+e":
			m, cmd = m.keyRight()
		default:
			m, cmd = m.keyDefault(msg.String())
		}
	}
	cmds = append(cmds, cmd)

	// Update the text input
	m.input, cmd = m.input.Update(msg)
	cmds = append(cmds, cmd)

	// If the input has changed, update the completions and validate the input
	if m.input.Value() != "" && m.input.Value() != m.lastInput && m.completionHolder == "" && !m.showAll {
		m.lastInput = m.input.Value()
		m.completions = m.getCompletions()
		m.validCommand = m.validateInput()
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

	if _, err := os.Stat(cleanPath); os.IsNotExist(err) {
		if _, err := os.Create(cleanPath); err != nil {
			m.Err = err
			return
		}
		err = m.saveHistoryToFile()
		if err != nil {
			m.Err = err
			return
		}
	}

	err := m.loadHistoryFromFile()
	if err != nil {
		m.Err = err
		return
	}
}

// ClearHistory clears the command history from all previous commands.
//
// If the history file path is set, it also clears the history on file.
func (m *Model) ClearHistory() {
	m.History = []string{}
	m.saveHistoryToFile()
}

// ShowingCompletions returns true if the completions are currently visible
func (m *Model) ShowingCompletions() bool {
	return m.completions != nil && len(m.completions) > 0 && m.historyIndex == -1 && (m.input.Value() != "" || m.showAll)
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
	m.showAll = false
}

// MARK: Private Functions

func (m *Model) saveHistoryToFile() error {
	// For small history lengths, it's better to just write the entire history to the file every time
	if m.historyFilePath == "" {
		return errors.New("history file path not set")
	}
	data := historyFileJson{History: m.History}
	jsonData, err := json.Marshal(data)
	if err != nil {
		m.Err = err
		return err
	}
	return os.WriteFile(m.historyFilePath, jsonData, 0644)
}

func (m *Model) loadHistoryFromFile() error {
	file, err := os.Open(m.historyFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	data, err := os.ReadFile(m.historyFilePath)
	if err != nil {
		return err
	}

	jsonData := historyFileJson{}
	err = json.Unmarshal(data, &jsonData)
	if err != nil {
		return err
	}

	m.History = jsonData.History
	m.input.SetSuggestions(m.History)
	return nil
}

func (m Model) resetModel() Model {
	m.input.SetValue("")
	m.completions = []Completion{}
	m.completionIndex = -1
	m.historyIndex = -1
	return m
}

func splitInput(input string) []string {
	var result []string
	var buffer strings.Builder
	inQuotes := false
	var quoteChar rune

	flushBuffer := func() {
		if buffer.Len() > 0 {
			result = append(result, buffer.String())
			buffer.Reset()
		}
	}

	for _, char := range input {
		switch {
		case char == ' ' && !inQuotes:
			flushBuffer()
		case char == '"' || char == '\'':
			if inQuotes && char == quoteChar {
				inQuotes = false
				buffer.WriteRune(char)
				flushBuffer()
			} else if !inQuotes {
				inQuotes = true
				quoteChar = char
				buffer.WriteRune(char)
			} else {
				buffer.WriteRune(char)
			}
		default:
			buffer.WriteRune(char)
		}
	}

	flushBuffer()
	return result
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
	if m.input.Value() == "" {
		return m, nil
	}

	// TODO: Open Bubbletea issue about input supporting checking if a suggestion is available aka currentsuggestion is safe or out of length
	hasSuggestion := false
	for _, h := range m.History {
		if strings.HasPrefix(h, m.input.Value()) && h != m.input.Value() {
			hasSuggestion = true
			break
		}
	}

	if !hasSuggestion {
		return m, nil
	}
	m.input.SetValue(m.input.CurrentSuggestion())
	m.input.CursorEnd()
	m.completionIndex = -1
	m.completionHolder = ""
	m.showAll = false
	return m, nil
}

func (m Model) keyTab(input string) (Model, tea.Cmd) {
	trimmedInput := strings.TrimSpace(m.input.Value())

	// If the input is empty, show all completions
	if trimmedInput == "" && !m.showAll {
		m.showAll = true
		m.completions = m.getCompletions()
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
	if input == "tab" || input == "ctrl+n" {
		// Down
		if m.completionIndex < len(m.completions)-1 {
			m.completionIndex++
		} else {
			m.completionIndex = -1
		}
	} else {
		// Up
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

	// Update the scroll bar percent
	scrollbarPercent = (float64(m.completionIndex) + 1) / float64(len(m.completions))

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

	if (len(m.History) == 0 || m.History[0] != command) && command != "" {
		m.History = append([]string{command}, m.History...)
	}

	if len(m.History) > m.HistoryLimit {
		m.History = m.History[1:]
	}
	m = m.resetModel()
	m.saveHistoryToFile()
	m.input.SetSuggestions(m.History)

	return m, func() tea.Msg {
		return SelectedCommandMsg{Command: command, Err: m.validCommand}
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

func (m Model) keyDefault(msg string) (Model, tea.Cmd) {
	if len(msg) > 1 {
		return m, nil
	}

	m.completionHolder = ""
	m.completionIndex = -1
	m.historyIndex = -1
	m.filteredHistory = []string{}
	m.showAll = false
	return m, nil
}
