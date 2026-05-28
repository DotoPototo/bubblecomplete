package bubblecomplete

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
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
	var got historyFileJson
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
	seed := historyFileJson{History: []string{"a", "b", "c", "d", "e"}}
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
	seed := historyFileJson{History: []string{"a", "b", "c", "d", "e"}}
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
