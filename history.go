package bubblecomplete

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type historyFileJson struct {
	History []string `json:"history"`
}

// SetHistoryFilePath sets the file path to the file used for persisting the command history.
//
// It creates the file if it doesn't exist. The directory of the file path must exist.
// The file path must be a valid JSON file with a .json extension.
func (m *Model) SetHistoryFilePath(path string) {
	// Reset so Error() reflects the result of this call, not a prior failure.
	m.err = nil

	cleanPath := filepath.Clean(path)

	file := filepath.Base(cleanPath)
	if file == "" {
		m.err = errors.New("invalid history file path")
		return
	}
	if filepath.Ext(file) != ".json" {
		m.err = errors.New("history file must be a JSON file")
		return
	}

	dir := filepath.Dir(cleanPath)
	if _, err := os.Stat(dir); err != nil {
		// Surface any stat error (missing dir, permission denied, etc.)
		// immediately rather than letting later file ops fail confusingly.
		m.err = err
		return
	}

	m.historyFilePath = cleanPath

	// saveHistoryToFile creates the file via atomic rename, so we no longer
	// pre-create with os.Create (which previously leaked its returned handle).
	if _, err := os.Stat(cleanPath); os.IsNotExist(err) {
		if err := m.saveHistoryToFile(); err != nil {
			m.err = err
			return
		}
	}

	if err := m.loadHistoryFromFile(); err != nil {
		m.err = err
	}
}

// ClearHistory clears the command history. If a history file path is set, the
// empty history is persisted to it; failures surface via Model.Error.
func (m *Model) ClearHistory() {
	m.History = []string{}
	if m.historyFilePath != "" {
		m.err = m.saveHistoryToFile()
	}
}

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
