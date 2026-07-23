package bubblecomplete

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
)

// maxHistoryFileSize caps loadHistoryFromFile reads, protecting hosts from a
// tampered or accidentally enormous history file pinning memory on startup.
// 10 MiB is orders of magnitude beyond any realistic history.
const maxHistoryFileSize = 10 << 20

type historyFileJSON struct {
	History []string `json:"history"`
}

// SetHistoryFilePath configures the file used for persisting the command
// history. The path must end in ".json" and its parent directory must exist.
// If the file doesn't exist it is created with an empty history. Existing
// content is loaded and capped to HistoryLimit. Errors surface via Model.Error.
//
// The path is trusted. The host is responsible for choosing a location the
// current user controls — entries are loaded verbatim and fed to the input's
// suggestion list, so a tampered file could surface unexpected strings
// (including terminal-control sequences) to the renderer. Loads reject files
// larger than 10 MiB to bound memory use.
func (m *Model) SetHistoryFilePath(path string) {
	// Reset so Error() reflects the result of this call, not a prior failure.
	m.err = nil

	cleanPath := filepath.Clean(path)
	if filepath.Ext(cleanPath) != ".json" {
		m.err = errors.New("history file must have a .json extension")
		return
	}

	dir := filepath.Dir(cleanPath)
	if _, err := os.Stat(dir); err != nil {
		// Surface stat errors now rather than letting later file ops fail confusingly.
		m.err = err
		return
	}

	m.historyFilePath = cleanPath

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

// ClearHistory clears the command history and state derived from it (the
// suggestion list and the history-navigation cursor). If a history file path
// is set, the empty history is persisted; failures surface via Model.Error.
func (m *Model) ClearHistory() {
	m.History = []string{}
	m.syncHistoryDerivedState()
	if m.historyFilePath != "" {
		m.err = m.saveHistoryToFile()
	}
}

// syncHistoryDerivedState refreshes everything that hangs off m.History.
// Call after any wholesale replacement so a stale filteredHistory or
// historyIndex can't survive the change.
func (m *Model) syncHistoryDerivedState() {
	m.input.SetSuggestions(m.History)
	m.filteredHistory = nil
	m.historyIndex = -1
}

// saveHistoryToFile writes the in-memory history to the configured file
// atomically: write to a temp file in the same directory then rename, so a
// crash mid-write can never produce a corrupted history file. Returns nil
// when no history file is configured.
func (m *Model) saveHistoryToFile() error {
	if m.historyFilePath == "" {
		return nil
	}
	data := historyFileJSON{History: m.History}
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
	// Cleanup errors below are discarded: best-effort temp removal must not
	// shadow the primary failure.
	if _, err := tmp.Write(jsonData); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, m.historyFilePath); err != nil {
		// Rename can fail (Windows dest-exists, permissions, cross-fs); clean
		// up so repeated failures don't leave a trail of tmp files.
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}

func (m *Model) loadHistoryFromFile() error {
	f, err := os.Open(m.historyFilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	defer f.Close()

	// Read at most maxHistoryFileSize+1 so we can distinguish "exactly the
	// cap" (valid) from "exceeds the cap" (rejected) without a separate
	// Stat call.
	data, err := io.ReadAll(io.LimitReader(f, maxHistoryFileSize+1))
	if err != nil {
		return err
	}
	if len(data) > maxHistoryFileSize {
		return fmt.Errorf("history file exceeds %d bytes", maxHistoryFileSize)
	}
	// Tolerate an empty file as empty history.
	if len(data) == 0 {
		m.History = nil
		m.syncHistoryDerivedState()
		return nil
	}

	jsonData := historyFileJSON{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return err
	}

	m.History = jsonData.History
	if m.HistoryLimit > 0 && len(m.History) > m.HistoryLimit {
		// Clone rather than re-slice so the unreachable tail of the
		// unmarshalled array can be GC'd.
		m.History = slices.Clone(m.History[:m.HistoryLimit])
	}
	if m.HistoryLimit <= 0 {
		m.History = nil
	}
	m.syncHistoryDerivedState()
	return nil
}
