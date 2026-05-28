package bubblecomplete

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestHistory_TrimKeepsNewest(t *testing.T) {
	m, err := New(TestCommands, 100, WithHistoryLimit(3))
	if err != nil {
		t.Fatal(err)
	}

	for _, c := range []string{"git status", "git stash list", "git stash apply", "git commit --amend"} {
		m = simulateTyping(t, m, c)
		m, _ = pressEnter(t, m)
	}

	if len(m.History) != 3 {
		t.Fatalf("History length = %d, want 3", len(m.History))
	}
	if m.History[0] != "git commit --amend" {
		t.Errorf("History[0] = %q, want %q", m.History[0], "git commit --amend")
	}
	if m.History[2] != "git stash list" {
		t.Errorf("History[2] = %q, want %q (oldest within limit)", m.History[2], "git stash list")
	}
}

func TestClearHistory_ResetsDerivedState(t *testing.T) {
	m, err := New(TestCommands, 100, WithHistoryLimit(10))
	if err != nil {
		t.Fatal(err)
	}

	for _, c := range []string{"git status", "git commit --amend"} {
		m = simulateTyping(t, m, c)
		m, _ = pressEnter(t, m)
	}

	// Prime filteredHistory and historyIndex by walking the history with
	// the up arrow — this is the state that would go stale after a clear.
	m = simulateKey(t, m, tea.KeyUp)
	if len(m.filteredHistory) == 0 || m.historyIndex == -1 {
		t.Fatalf("setup failed: expected nav state to be active, filteredHistory=%v historyIndex=%d", m.filteredHistory, m.historyIndex)
	}

	m.ClearHistory()

	if len(m.History) != 0 {
		t.Errorf("History not cleared: %v", m.History)
	}
	if m.filteredHistory != nil {
		t.Errorf("filteredHistory should be nil after clear, got %v", m.filteredHistory)
	}
	if m.historyIndex != -1 {
		t.Errorf("historyIndex should be -1 after clear, got %d", m.historyIndex)
	}

	// After clear, Up should be a no-op — there's no history to navigate
	// to, so the input value should remain whatever the user had typed.
	m.input.SetValue("fresh input")
	before := m.input.Value()
	m = simulateKey(t, m, tea.KeyUp)
	if m.input.Value() != before {
		t.Errorf("Up navigated stale history after ClearHistory: value changed from %q to %q", before, m.input.Value())
	}
}

func TestHistory_ZeroLimitDisablesStorage(t *testing.T) {
	for _, n := range []int{0, -3} {
		t.Run("", func(t *testing.T) {
			m, err := New(TestCommands, 100, WithHistoryLimit(n))
			if err != nil {
				t.Fatal(err)
			}
			m = simulateTyping(t, m, "git status")
			m, _ = pressEnter(t, m)
			if len(m.History) != 0 {
				t.Errorf("HistoryLimit=%d should disable storage, got %v", n, m.History)
			}
		})
	}
}

func TestHistory_FileSaveSurvivesEnter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.json")

	m, err := New(TestCommands, 100, WithHistoryFilePath(path), WithHistoryLimit(5))
	if err != nil {
		t.Fatal(err)
	}
	if m.Error() != nil {
		t.Fatalf("unexpected Err after construction: %v", m.Error())
	}

	for _, c := range []string{"git status", "git stash list"} {
		m = simulateTyping(t, m, c)
		m, _ = pressEnter(t, m)
	}
	if m.Error() != nil {
		t.Fatalf("unexpected Err after enters: %v", m.Error())
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got historyFileJSON
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("history file is not valid JSON: %v\nraw: %s", err, raw)
	}
	if len(got.History) != 2 || got.History[0] != "git stash list" {
		t.Errorf("file contents = %v, want first entry 'git stash list'", got.History)
	}
}

func TestHistory_OptionOrderDoesNotMatter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.json")

	// Seed five entries on disk so loading would exceed any non-default limit.
	seed := historyFileJSON{History: []string{"a", "b", "c", "d", "e"}}
	raw, _ := json.Marshal(seed)
	if err := os.WriteFile(path, raw, 0644); err != nil {
		t.Fatal(err)
	}

	// Limit applied AFTER file path: must still cap loaded history to 2.
	m, err := New(TestCommands, 100,
		WithHistoryFilePath(path),
		WithHistoryLimit(2),
	)
	if err != nil {
		t.Fatal(err)
	}
	if m.Error() != nil {
		t.Fatalf("unexpected Err: %v", m.Error())
	}
	if len(m.History) != 2 {
		t.Errorf("limit-after-load: history length = %d, want 2 (%v)", len(m.History), m.History)
	}

	// Same arguments, opposite option order — must produce the same result.
	m2, err := New(TestCommands, 100,
		WithHistoryLimit(2),
		WithHistoryFilePath(path),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(m2.History) != 2 {
		t.Errorf("limit-before-load: history length = %d, want 2 (%v)", len(m2.History), m2.History)
	}
}

func TestHistory_LoadCapsAtHistoryLimit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.json")

	// Seed the file with five entries.
	seed := historyFileJSON{History: []string{"a", "b", "c", "d", "e"}}
	raw, _ := json.Marshal(seed)
	if err := os.WriteFile(path, raw, 0644); err != nil {
		t.Fatal(err)
	}

	m, err := New(TestCommands, 100, WithHistoryLimit(2), WithHistoryFilePath(path))
	if err != nil {
		t.Fatal(err)
	}
	if m.Error() != nil {
		t.Fatalf("unexpected Err: %v", m.Error())
	}
	if len(m.History) != 2 {
		t.Errorf("expected capped to 2, got %d (%v)", len(m.History), m.History)
	}
	if m.History[0] != "a" || m.History[1] != "b" {
		t.Errorf("expected first two entries, got %v", m.History)
	}
}

func TestHistory_LoadEmptyFileIsNotError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.json")
	if err := os.WriteFile(path, nil, 0644); err != nil {
		t.Fatal(err)
	}

	m, err := New(TestCommands, 100, WithHistoryFilePath(path))
	if err != nil {
		t.Fatal(err)
	}
	if m.Error() != nil {
		t.Errorf("expected no Err for empty file, got %v", m.Error())
	}
	if len(m.History) != 0 {
		t.Errorf("expected empty history, got %v", m.History)
	}
}

func TestHistory_LoadRejectsOversizedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.json")

	// One byte over the cap is enough to trigger rejection. Use a JSON
	// shape that would otherwise unmarshal cleanly so any future failure
	// proves the size check fired (not malformed JSON).
	prefix := []byte(`{"history":["`)
	suffix := []byte(`"]}`)
	padding := bytes.Repeat([]byte("a"), (10<<20)-len(prefix)-len(suffix)+1)
	payload := append(append(prefix, padding...), suffix...)
	if err := os.WriteFile(path, payload, 0644); err != nil {
		t.Fatal(err)
	}

	m, err := New(TestCommands, 100, WithHistoryFilePath(path))
	if err != nil {
		t.Fatal(err)
	}
	if m.Error() == nil {
		t.Fatal("expected Error() to report oversized file, got nil")
	}
	if !strings.Contains(m.Error().Error(), "exceeds") {
		t.Errorf("expected size-cap error, got %v", m.Error())
	}
}

func TestHistory_SaveFailureSurfacesAsErr(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.json")

	m, err := New(TestCommands, 100, WithHistoryFilePath(path))
	if err != nil {
		t.Fatal(err)
	}

	// Make the parent directory unwritable so saveHistoryToFile fails on rename.
	if err := os.Chmod(dir, 0555); err != nil {
		t.Skip("cannot chmod tmpdir read-only on this platform:", err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0755) })

	m = simulateTyping(t, m, "git status")
	m, _ = pressEnter(t, m)

	if m.Error() == nil {
		t.Error("expected save failure to surface as m.Error(), got nil")
	}
}

func TestError_ClearsAfterSuccessfulOperation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.json")

	m, err := New(TestCommands, 100, WithHistoryFilePath(path))
	if err != nil {
		t.Fatal(err)
	}

	// Trigger a save failure by making the dir read-only.
	if err := os.Chmod(dir, 0555); err != nil {
		t.Skip("cannot chmod tmpdir read-only:", err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0755) })

	m = simulateTyping(t, m, "git status")
	m, _ = pressEnter(t, m)
	if m.Error() == nil {
		t.Fatal("expected non-nil Error after failed save")
	}

	// Restore writability and submit a successful command.
	if err := os.Chmod(dir, 0755); err != nil {
		t.Fatal(err)
	}
	m = simulateTyping(t, m, "git status")
	m, _ = pressEnter(t, m)
	if m.Error() != nil {
		t.Errorf("Error should clear after a successful save, got: %v", m.Error())
	}
}

func TestHistory_DisabledLimitSkipsFileSave(t *testing.T) {
	// With HistoryLimit <= 0 history is disabled. If a file path is still
	// configured, Enter must NOT rewrite the file to {"history":null} on
	// every submit — the on-disk content from before the disable should
	// remain intact.
	dir := t.TempDir()
	path := filepath.Join(dir, "history.json")

	// Pre-populate the file with a known content using a normal-limit model.
	m, err := New(TestCommands, 100, WithHistoryFilePath(path), WithHistoryLimit(5))
	if err != nil {
		t.Fatal(err)
	}
	m = simulateTyping(t, m, "git status")
	m, _ = pressEnter(t, m)

	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	// Now disable history at runtime and submit again.
	m.HistoryLimit = 0
	m = simulateTyping(t, m, "git stash list")
	m, _ = pressEnter(t, m)

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Errorf("history file was rewritten while history disabled:\nbefore: %s\nafter:  %s", before, after)
	}
	if m.Error() != nil {
		t.Errorf("Error() should be nil after disabled-history Enter, got %v", m.Error())
	}
}

func TestSetHistoryFilePath_SurfacesNonIsNotExistStatErrors(t *testing.T) {
	// Create a regular file then point WithHistoryFilePath at a path that
	// would require traversing it as a directory. os.Stat on that parent
	// returns ENOTDIR (or the Windows equivalent), which is NOT
	// os.IsNotExist — the old gate would have silently fallen through.
	tmp := t.TempDir()
	blocker := filepath.Join(tmp, "blocker")
	if err := os.WriteFile(blocker, nil, 0644); err != nil {
		t.Fatal(err)
	}
	// blocker is a file; treating it as a parent dir must fail at Stat.
	historyPath := filepath.Join(blocker, "subdir", "history.json")

	m, err := New(TestCommands, 100, WithHistoryFilePath(historyPath))
	if err != nil {
		t.Fatal(err)
	}
	if m.Error() == nil {
		t.Error("expected Error() to surface the non-IsNotExist stat failure, got nil")
	}
}

func TestHistory_RenameFailureCleansUpTempFile(t *testing.T) {
	// Force the rename to fail by making the destination a read-only
	// directory after the file is configured. The temp file created by
	// saveHistoryToFile must be removed on rename failure.
	dir := t.TempDir()
	path := filepath.Join(dir, "history.json")

	m, err := New(TestCommands, 100, WithHistoryFilePath(path))
	if err != nil {
		t.Fatal(err)
	}

	// Make the directory read-only so rename can't replace the destination.
	if err := os.Chmod(dir, 0555); err != nil {
		t.Skip("cannot chmod tmpdir read-only on this platform:", err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0755) })

	m = simulateTyping(t, m, "git status")
	m, _ = pressEnter(t, m)
	if m.Error() == nil {
		t.Fatal("expected save failure to surface as Error")
	}

	// Restore writability so we can list the directory.
	if err := os.Chmod(dir, 0755); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "history-") && strings.HasSuffix(e.Name(), ".json.tmp") {
			t.Errorf("temp file %q was not cleaned up after rename failure", e.Name())
		}
	}
}

func TestHistory_AtomicSaveLeavesNoTempFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.json")

	m, err := New(TestCommands, 100, WithHistoryFilePath(path))
	if err != nil {
		t.Fatal(err)
	}
	m = simulateTyping(t, m, "git status")
	m, _ = pressEnter(t, m)
	if m.Error() != nil {
		t.Fatal(m.Error())
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() != "history.json" {
			t.Errorf("found leftover file %q after save; expected only history.json", e.Name())
		}
	}
}
