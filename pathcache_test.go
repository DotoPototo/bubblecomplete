package bubblecomplete

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestDirCache_ReadsAndCaches(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "a.txt"))
	mustWriteFile(t, filepath.Join(dir, "b.txt"))
	mustMkdir(t, filepath.Join(dir, "sub"))

	c := newDirCache()
	e := c.read(dir)
	if e.err != nil {
		t.Fatalf("read: %v", e.err)
	}
	if len(e.names) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(e.names))
	}
	wantKinds := map[string]entryKind{"a.txt": kindFile, "b.txt": kindFile, "sub": kindDir}
	for i, n := range e.names {
		if e.kinds[i] != wantKinds[n] {
			t.Errorf("%s: kind = %d, want %d", n, e.kinds[i], wantKinds[n])
		}
	}

	// Second read within TTL must reuse the same entry pointer.
	e2 := c.read(dir)
	if e2 != e {
		t.Errorf("second read returned a different entry pointer (TTL miss?)")
	}
}

func TestDirCache_EmptyParentReturnsEmpty(t *testing.T) {
	c := newDirCache()
	e := c.read("")
	if e == nil {
		t.Fatal("read(\"\") returned nil")
	}
	if e.err != nil {
		t.Errorf("empty-parent err = %v, want nil", e.err)
	}
	if len(e.names) != 0 {
		t.Errorf("empty-parent names = %v, want empty", e.names)
	}
}

func TestDirCache_StickyError(t *testing.T) {
	c := newDirCache()
	missing := filepath.Join(t.TempDir(), "does-not-exist")
	e1 := c.read(missing)
	if e1.err == nil {
		t.Fatal("expected error reading missing dir")
	}

	// Create the dir AFTER the first read. Within TTL the cached error
	// should still be returned — no retry storm.
	if err := os.Mkdir(missing, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	e2 := c.read(missing)
	if e2.err == nil {
		t.Error("expected cached error to stick within TTL")
	}
}

func TestDirCache_TTLRefresh(t *testing.T) {
	dir := t.TempDir()
	c := newDirCache()
	c.ttl = 1 * time.Millisecond

	e1 := c.read(dir)
	if e1.err != nil {
		t.Fatal(e1.err)
	}

	// Wait past the TTL, then re-read — should produce a fresh entry.
	time.Sleep(5 * time.Millisecond)
	e2 := c.read(dir)
	if e2 == e1 {
		t.Errorf("expected fresh entry after TTL, got same pointer")
	}
}

func TestDirCache_LRUEviction(t *testing.T) {
	c := newDirCache()
	c.maxDirs = 3

	// Populate four distinct dirs.
	dirs := make([]string, 4)
	for i := range 4 {
		dirs[i] = t.TempDir()
		c.read(dirs[i])
	}

	// The first one inserted (dirs[0]) was least-recently-used → evicted.
	if _, ok := c.entries[dirs[0]]; ok {
		t.Errorf("dirs[0] should have been evicted")
	}
	for i := 1; i < 4; i++ {
		if _, ok := c.entries[dirs[i]]; !ok {
			t.Errorf("dirs[%d] missing from cache", i)
		}
	}
}

func TestDirCache_LRUTouchOrder(t *testing.T) {
	c := newDirCache()
	c.maxDirs = 3

	a := t.TempDir()
	b := t.TempDir()
	cdir := t.TempDir()
	d := t.TempDir()

	c.read(a)
	c.read(b)
	c.read(cdir)
	// Touch a so it's most-recently-used.
	c.read(a)
	// Insert d — least-recently-used (now b) should be evicted.
	c.read(d)

	if _, ok := c.entries[b]; ok {
		t.Errorf("b should have been evicted as LRU")
	}
	if _, ok := c.entries[a]; !ok {
		t.Errorf("a should remain after touch")
	}
}

func TestDirCache_ConcurrentReads(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("temp dir cleanup races on windows under -race; covered by other tests")
	}
	dir := t.TempDir()
	for i := range 5 {
		mustWriteFile(t, filepath.Join(dir, fmt.Sprintf("f%d.txt", i)))
	}
	c := newDirCache()

	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			for range 20 {
				e := c.read(dir)
				if e.err != nil {
					t.Errorf("concurrent read: %v", e.err)
				}
			}
		})
	}
	wg.Wait()
}

func mustWriteFile(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatal(err)
	}
}
