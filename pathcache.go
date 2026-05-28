package bubblecomplete

import (
	"io/fs"
	"os"
	"strings"
	"sync"
	"time"
)

// entryKind classifies a directory entry by file type. Captured up-front by
// the cache so the classifier and candidate generator can route entries
// without re-statting. kindSymlink and kindUnknown are left unresolved here;
// the candidate generator resolves target kinds lazily for survivors that
// need it.
type entryKind uint8

const (
	// kindOther covers anything that's neither file, dir, nor symlink —
	// devices, sockets, pipes, irregular. Excluded from candidates by
	// generateCandidates regardless of ArgumentType.
	kindOther entryKind = iota
	kindFile
	kindDir
	kindSymlink
	// kindUnknown means the OS did not report a type (DT_UNKNOWN sentinel
	// leaked through Go's lstat fallback, or a custom fs.FS DirEntry that
	// doesn't follow os.ReadDir's resolution contract). Treated like a
	// symlink: candidate generation resolves via os.Stat within the budget.
	kindUnknown
)

// kindFromDirEntry maps an [fs.DirEntry]'s Type() to an [entryKind].
//
// os.ReadDir resolves DT_UNKNOWN entries by calling lstat, so under standard
// use a mode with no type bits set (Type() == 0) means "regular file". The
// ^FileMode(0) sentinel guard catches custom DirEntry implementations that
// return the unresolved value. Devices, sockets, pipes, and other irregular
// modes fall into kindOther and are filtered out of candidates.
func kindFromDirEntry(d fs.DirEntry) entryKind {
	mode := d.Type()
	if mode == ^fs.FileMode(0) {
		return kindUnknown
	}
	switch {
	case mode.IsDir():
		return kindDir
	case mode&fs.ModeSymlink != 0:
		return kindSymlink
	case mode == 0:
		// No type bits set after the readdir+lstat fallback → regular file.
		return kindFile
	default:
		return kindOther
	}
}

// dirCacheEntry holds one cached ReadDir result. Slices are read-only by
// contract; consumers must not mutate them.
type dirCacheEntry struct {
	// names, foldNames, and kinds are parallel slices indexed by entry
	// position. names holds the raw filesystem case; foldNames is the
	// lowercase form for case-insensitive matching; kinds is the type as
	// reported by [fs.DirEntry.Type] at read time.
	names     []string
	foldNames []string
	kinds     []entryKind

	// fetchedAt is when the entry was populated; used for TTL eviction.
	fetchedAt time.Time

	// err is the ReadDir error, if any. Sticky for the TTL window so
	// repeated keystrokes on a missing or permission-denied directory
	// don't trigger retry storms.
	err error
}

// matchKey returns the (names slice, lookup needle) pair to use for
// case-aware matching of base against this entry. On case-insensitive
// platforms (macOS, Windows heuristic) it returns the folded names and a
// lowercased base; on Linux it returns the raw names and base unchanged.
// Consumers iterate the returned slice in parallel with entry.names by
// index — names is always the source of truth for display.
func (e *dirCacheEntry) matchKey(base string) (names []string, needle string) {
	if caseInsensitiveFS() {
		return e.foldNames, strings.ToLower(base)
	}
	return e.names, base
}

// dirCache is a bounded LRU + TTL cache of [os.ReadDir] results, keyed by
// cleaned absolute parent path.
//
// The mutex makes the cache safe under future tea.Cmd async paths. Production
// callers run on Bubble Tea's single Update goroutine where it's
// uncontended.
type dirCache struct {
	mu      sync.Mutex
	entries map[string]*dirCacheEntry
	order   []string // LRU order: front (index 0) is most-recently-used
	maxDirs int
	ttl     time.Duration
}

const (
	// defaultDirCacheMaxDirs caps how many parent directories are kept in
	// memory. 64 is far more than a typical session needs.
	defaultDirCacheMaxDirs = 64
	// defaultDirCacheTTL is how long a cached entry stays valid before
	// re-fetch. Short enough that filesystem changes surface within a
	// reasonable time; long enough that a typing burst hits cache.
	defaultDirCacheTTL = 2 * time.Second
)

// newDirCache constructs a cache with library defaults.
func newDirCache() *dirCache {
	return &dirCache{
		entries: make(map[string]*dirCacheEntry),
		maxDirs: defaultDirCacheMaxDirs,
		ttl:     defaultDirCacheTTL,
	}
}

// read returns the cached entry for parent, refreshing if missing or stale.
// An empty parent yields a synthetic empty entry — callers don't need to
// special-case it.
func (c *dirCache) read(parent string) *dirCacheEntry {
	if parent == "" {
		return &dirCacheEntry{}
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	if e, ok := c.entries[parent]; ok {
		if time.Since(e.fetchedAt) < c.ttl {
			c.touchLocked(parent)
			return e
		}
		// stale — fall through to refresh
	}

	entry := fetchDir(parent)
	c.entries[parent] = entry
	c.touchLocked(parent)
	c.evictIfFullLocked()
	return entry
}

// fetchDir performs an [os.ReadDir] and packages the result into a
// [dirCacheEntry]. Errors are captured into entry.err; callers check it.
//
// Note: os.ReadDir is unbounded — it reads every entry in the directory and
// returns them sorted. The TTL cache amortises the cost across keystrokes.
func fetchDir(parent string) *dirCacheEntry {
	e := &dirCacheEntry{fetchedAt: time.Now()}
	entries, err := os.ReadDir(parent)
	if err != nil {
		e.err = err
		return e
	}
	e.names = make([]string, len(entries))
	e.foldNames = make([]string, len(entries))
	e.kinds = make([]entryKind, len(entries))
	for i, de := range entries {
		name := de.Name()
		e.names[i] = name
		e.foldNames[i] = strings.ToLower(name)
		e.kinds[i] = kindFromDirEntry(de)
	}
	return e
}

// touchLocked moves parent to the front of the LRU order. Caller holds c.mu.
func (c *dirCache) touchLocked(parent string) {
	for i, p := range c.order {
		if p == parent {
			c.order = append(c.order[:i], c.order[i+1:]...)
			break
		}
	}
	c.order = append([]string{parent}, c.order...)
}

// evictIfFullLocked drops least-recently-used entries until the cache is
// within maxDirs. Caller holds c.mu.
func (c *dirCache) evictIfFullLocked() {
	for len(c.order) > c.maxDirs {
		victim := c.order[len(c.order)-1]
		c.order = c.order[:len(c.order)-1]
		delete(c.entries, victim)
	}
}
