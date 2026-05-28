package bubblecomplete

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// fileArgCmd is a minimal command used to exercise activeFileArgument for
// positional FileArgument values.
var fileArgCmd = &Command{
	Command: "cat",
	PositionalArguments: []*PositionalArgument{
		{Name: "File", Type: FileArgument, Required: true},
	},
}

// dirArgCmd has a positional DirArgument.
var dirArgCmd = &Command{
	Command: "cd",
	PositionalArguments: []*PositionalArgument{
		{Name: "Dir", Type: DirArgument, Required: true},
	},
}

// fileDirArgCmd has a positional FileDirArgument.
var fileDirArgCmd = &Command{
	Command: "open",
	PositionalArguments: []*PositionalArgument{
		{Name: "Target", Type: FileDirArgument, Required: true},
	},
}

// findCmd exercises flag-value paths. --path is a FileArgument flag.
var findCmd = &Command{
	Command: "find",
	PositionalArguments: []*PositionalArgument{
		{Name: "Dir", Type: DirArgument, Required: true},
	},
	Flags: []*Flag{
		{LongFlag: "--path", Type: FileArgument},
		{LongFlag: "--name", Type: StringArgument},
	},
}

// stringArgCmd has a string positional — should NEVER trigger active path
// completion regardless of input.
var stringArgCmd = &Command{
	Command: "echo",
	PositionalArguments: []*PositionalArgument{
		{Name: "Msg", Type: StringArgument, Required: true},
	},
}

func cmds(c ...*Command) []*Command { return c }

func TestActiveFileArgument_Inactive(t *testing.T) {
	cases := []struct {
		name  string
		input string
		cmds  []*Command
	}{
		{"empty", "", cmds(fileArgCmd)},
		{"trailing space", "cat ", cmds(fileArgCmd)},
		{"no command yet", "ca", cmds(fileArgCmd)},
		{"unknown command", "wat /tmp/x", cmds(fileArgCmd)},
		{"string arg type", "echo hello", cmds(stringArgCmd)},
		{"flag name without value", "find . --path", cmds(findCmd)},
		{"equals-form empty value", "find . --path=", cmds(findCmd)},
		{"equals-form unknown flag", "find . --bogus=foo", cmds(findCmd)},
		{"equals-form non-file flag", "find . --name=foo", cmds(findCmd)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, _, _, _, _, ok := activeFileArgument(c.input, c.cmds)
			if ok {
				t.Errorf("activeFileArgument(%q) ok=true, want false", c.input)
			}
		})
	}
}

func TestActiveFileArgument_Positional(t *testing.T) {
	const input = "cat ~/Doc"
	kind, vs, ve, tp, oq, ok := activeFileArgument(input, cmds(fileArgCmd))
	if !ok {
		t.Fatal("expected active=true")
	}
	if kind != FileArgument {
		t.Errorf("kind = %v, want FileArgument", kind)
	}
	if got := input[vs:ve]; got != "~/Doc" {
		t.Errorf("value = %q, want %q", got, "~/Doc")
	}
	if tp != "" {
		t.Errorf("tokenPrefix = %q, want \"\"", tp)
	}
	if oq != 0 {
		t.Errorf("openingQuote = %q, want 0", oq)
	}
}

func TestActiveFileArgument_PositionalQuoted(t *testing.T) {
	cases := []struct {
		name       string
		input      string
		wantVal    string
		wantPrefix string
		wantOpener rune
	}{
		{"unclosed double", `cat "~/Doc`, "~/Doc", `"`, '"'},
		{"closed double", `cat "~/Doc"`, "~/Doc", `"`, '"'},
		{"unclosed single", `cat '~/Doc`, "~/Doc", `'`, '\''},
		{"closed single", `cat '~/Doc'`, "~/Doc", `'`, '\''},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			kind, vs, ve, tp, oq, ok := activeFileArgument(c.input, cmds(fileArgCmd))
			if !ok {
				t.Fatal("expected active=true")
			}
			if kind != FileArgument {
				t.Errorf("kind = %v, want FileArgument", kind)
			}
			if got := c.input[vs:ve]; got != c.wantVal {
				t.Errorf("value = %q, want %q", got, c.wantVal)
			}
			if tp != c.wantPrefix {
				t.Errorf("tokenPrefix = %q, want %q", tp, c.wantPrefix)
			}
			if oq != c.wantOpener {
				t.Errorf("openingQuote = %q, want %q", oq, c.wantOpener)
			}
		})
	}
}

func TestActiveFileArgument_FlagSpaceSeparated(t *testing.T) {
	const input = "find . --path ~/Doc"
	kind, vs, ve, tp, oq, ok := activeFileArgument(input, cmds(findCmd))
	if !ok {
		t.Fatal("expected active=true")
	}
	if kind != FileArgument {
		t.Errorf("kind = %v, want FileArgument", kind)
	}
	if got := input[vs:ve]; got != "~/Doc" {
		t.Errorf("value = %q", got)
	}
	if tp != "" || oq != 0 {
		t.Errorf("tokenPrefix=%q opener=%q, want empty/0", tp, oq)
	}
}

func TestActiveFileArgument_FlagEqualsForm(t *testing.T) {
	cases := []struct {
		name       string
		input      string
		wantVal    string
		wantPrefix string
		wantOpener rune
	}{
		{"unquoted", "find . --path=~/Doc", "~/Doc", "--path=", 0},
		{"quoted closed", `find . --path="~/Doc"`, "~/Doc", `--path="`, '"'},
		{"quoted unclosed", `find . --path="~/Doc`, "~/Doc", `--path="`, '"'},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			kind, vs, ve, tp, oq, ok := activeFileArgument(c.input, cmds(findCmd))
			if !ok {
				t.Fatal("expected active=true")
			}
			if kind != FileArgument {
				t.Errorf("kind = %v, want FileArgument", kind)
			}
			if got := c.input[vs:ve]; got != c.wantVal {
				t.Errorf("value = %q, want %q", got, c.wantVal)
			}
			if tp != c.wantPrefix {
				t.Errorf("tokenPrefix = %q, want %q", tp, c.wantPrefix)
			}
			if oq != c.wantOpener {
				t.Errorf("openingQuote = %q, want %q", oq, c.wantOpener)
			}
		})
	}
}

func TestActiveFileArgument_DirAndFileDir(t *testing.T) {
	if kind, _, _, _, _, ok := activeFileArgument("cd /etc", cmds(dirArgCmd)); !ok || kind != DirArgument {
		t.Errorf("DirArgument: ok=%v kind=%v", ok, kind)
	}
	if kind, _, _, _, _, ok := activeFileArgument("open /etc", cmds(fileDirArgCmd)); !ok || kind != FileDirArgument {
		t.Errorf("FileDirArgument: ok=%v kind=%v", ok, kind)
	}
}

// makeTestDir builds a deterministic test layout under t.TempDir().
//
//	tmp/
//	  alpha.txt
//	  beta.txt
//	  Beta/         (case-collision on case-insensitive FS)
//	  .hidden
//	  subdir/
//	    nested.go
//	  link_alpha  → alpha.txt
//	  link_dir    → subdir/
//	  link_broken → nonexistent
func makeTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "alpha.txt"))
	mustWriteFile(t, filepath.Join(dir, "beta.txt"))
	mustMkdir(t, filepath.Join(dir, "Beta"))
	mustWriteFile(t, filepath.Join(dir, ".hidden"))
	mustMkdir(t, filepath.Join(dir, "subdir"))
	mustWriteFile(t, filepath.Join(dir, "subdir", "nested.go"))
	if err := os.Symlink(filepath.Join(dir, "alpha.txt"), filepath.Join(dir, "link_alpha")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(dir, "subdir"), filepath.Join(dir, "link_dir")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(dir, "nonexistent"), filepath.Join(dir, "link_broken")); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestClassify(t *testing.T) {
	dir := makeTestDir(t)
	c := newDirCache()
	entry := c.read(dir)

	cases := []struct {
		name string
		kind ArgumentType
		base string
		want pathValidity
	}{
		{"file under FileArg → valid", FileArgument, "alpha.txt", pathValid},
		{"dir under FileArg → invalid (wrong kind)", FileArgument, "subdir", pathInvalid},
		{"dir under DirArg → valid", DirArgument, "subdir", pathValid},
		{"file under DirArg → invalid", DirArgument, "alpha.txt", pathInvalid},
		{"file under FileDirArg → valid", FileDirArgument, "alpha.txt", pathValid},
		{"dir under FileDirArg → valid", FileDirArgument, "subdir", pathValid},
		{"prefix under FileArg → partial", FileArgument, "alph", pathPartial},
		{"no match under FileArg → invalid", FileArgument, "zzz", pathInvalid},
		{"symlink to file under FileArg → valid", FileArgument, "link_alpha", pathValid},
		{"symlink to dir under DirArg → valid", DirArgument, "link_dir", pathValid},
		{"broken symlink under FileArg → invalid", FileArgument, "link_broken", pathInvalid},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fullClean := filepath.Join(dir, tc.base)
			got := classify(tc.kind, fullClean, tc.base, entry)
			if got != tc.want {
				t.Errorf("classify = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestClassify_TrailingSeparator(t *testing.T) {
	dir := makeTestDir(t)
	c := newDirCache()
	entry := c.read(dir)
	// base="" simulates the trailing-separator case (e.g. "~/Documents/").
	if got := classify(DirArgument, dir, "", entry); got != pathValid {
		t.Errorf("DirArg trailing-sep: got %d, want pathValid", got)
	}
	if got := classify(FileDirArgument, dir, "", entry); got != pathValid {
		t.Errorf("FileDirArg trailing-sep: got %d, want pathValid", got)
	}
	if got := classify(FileArgument, dir, "", entry); got != pathPartial {
		t.Errorf("FileArg trailing-sep: got %d, want pathPartial (mid-nav)", got)
	}
}

func TestClassify_PermissionDeniedParent(t *testing.T) {
	// Use a nonexistent dir to simulate the ReadDir error path. We don't
	// rely on chmod since that's flaky cross-platform.
	missing := filepath.Join(t.TempDir(), "does-not-exist")
	c := newDirCache()
	entry := c.read(missing)
	if entry.err == nil {
		t.Fatal("expected error reading nonexistent dir")
	}
	if got := classify(FileArgument, filepath.Join(missing, "foo"), "foo", entry); got != pathInvalid {
		t.Errorf("classify on err entry = %d, want pathInvalid", got)
	}
}

func TestGenerateCandidates_InclusionRules(t *testing.T) {
	dir := makeTestDir(t)
	c := newDirCache()
	entry := c.read(dir)

	// FileArgument: files + dirs (drill-down), no specials.
	fileCands := generateCandidates(FileArgument, "", "", 0, "", entry, 50, false, dir)
	if !containsName(fileCands, "alpha.txt") || !containsName(fileCands, "subdir/") {
		t.Errorf("FileArgument candidates missing files or dirs: %v", names(fileCands))
	}

	// DirArgument: dirs only.
	dirCands := generateCandidates(DirArgument, "", "", 0, "", entry, 50, false, dir)
	for _, c := range dirCands {
		if !c.isDir {
			t.Errorf("DirArgument surfaced non-dir %q", c.displayName)
		}
	}
	if !containsName(dirCands, "subdir/") {
		t.Errorf("DirArgument missing subdir/: %v", names(dirCands))
	}

	// FileDirArgument: files + dirs.
	fdCands := generateCandidates(FileDirArgument, "", "", 0, "", entry, 50, false, dir)
	if !containsName(fdCands, "alpha.txt") || !containsName(fdCands, "subdir/") {
		t.Errorf("FileDirArgument candidates missing files or dirs: %v", names(fdCands))
	}
}

func TestGenerateCandidates_HiddenFiles(t *testing.T) {
	dir := makeTestDir(t)
	c := newDirCache()
	entry := c.read(dir)

	// Default: dotfiles hidden when base prefix doesn't start with '.'.
	cands := generateCandidates(FileArgument, "", "", 0, "", entry, 50, false, dir)
	if containsName(cands, ".hidden") {
		t.Errorf("dotfile leaked into candidates: %v", names(cands))
	}

	// Typed prefix "." surfaces dotfiles even without the option.
	cands = generateCandidates(FileArgument, "", "", 0, ".", entry, 50, false, dir)
	if !containsName(cands, ".hidden") {
		t.Errorf("dotfile not surfaced for '.' prefix: %v", names(cands))
	}

	// Option enabled: always surfaced.
	cands = generateCandidates(FileArgument, "", "", 0, "", entry, 50, true, dir)
	if !containsName(cands, ".hidden") {
		t.Errorf("dotfile not surfaced with HiddenFiles=true: %v", names(cands))
	}
}

func TestGenerateCandidates_PrefixFilter(t *testing.T) {
	dir := makeTestDir(t)
	c := newDirCache()
	entry := c.read(dir)

	cands := generateCandidates(FileArgument, "", "", 0, "alph", entry, 50, false, dir)
	if !containsName(cands, "alpha.txt") {
		t.Errorf("prefix 'alph' should match alpha.txt: %v", names(cands))
	}
	for _, c := range cands {
		stripped := strings.TrimSuffix(c.displayName, "/")
		lowerStripped := strings.ToLower(stripped)
		if !strings.HasPrefix(lowerStripped, "alph") {
			t.Errorf("candidate %q does not match prefix", c.displayName)
		}
	}
}

func TestGenerateCandidates_SymlinkResolution(t *testing.T) {
	dir := makeTestDir(t)
	c := newDirCache()
	entry := c.read(dir)

	// FileArgument: link_alpha (symlink to file) included, link_dir
	// (symlink to dir) also included (drill-down), link_broken excluded.
	cands := generateCandidates(FileArgument, "", "", 0, "link", entry, 50, false, dir)
	if !containsName(cands, "link_alpha") {
		t.Errorf("link_alpha should be included: %v", names(cands))
	}
	if !containsName(cands, "link_dir/") {
		t.Errorf("link_dir/ should be included: %v", names(cands))
	}
	if containsName(cands, "link_broken") || containsName(cands, "link_broken/") {
		t.Errorf("broken symlink should be excluded: %v", names(cands))
	}

	// DirArgument: only link_dir.
	dirCands := generateCandidates(DirArgument, "", "", 0, "link", entry, 50, false, dir)
	if containsName(dirCands, "link_alpha") {
		t.Errorf("link_alpha (→file) should NOT be in DirArgument: %v", names(dirCands))
	}
	if !containsName(dirCands, "link_dir/") {
		t.Errorf("link_dir/ should be in DirArgument: %v", names(dirCands))
	}
}

func TestGenerateCandidates_SortDirsFirst(t *testing.T) {
	dir := makeTestDir(t)
	c := newDirCache()
	entry := c.read(dir)
	cands := generateCandidates(FileArgument, "", "", 0, "", entry, 50, false, dir)

	// All directory entries should precede all file entries in the output.
	sawFile := false
	for _, c := range cands {
		if !c.isDir {
			sawFile = true
			continue
		}
		if sawFile {
			t.Errorf("dir %q appears after a file in: %v", c.displayName, names(cands))
		}
	}
}

// TestGenerateCandidates_StatBudget pins the budget contract: with more
// symlinks than statBudget(limit), the surplus symlinks are silently
// dropped rather than triggering an unbounded stat storm. The output count
// is at most limit. The test doesn't assert on stat count directly (no
// instrumentation hook exposed) — it asserts on the contract that the
// function returns bounded results under a symlink-heavy load and does so
// without timing out.
func TestGenerateCandidates_StatBudget(t *testing.T) {
	dir := t.TempDir()
	// Real file the symlinks point to.
	target := filepath.Join(dir, "target.txt")
	mustWriteFile(t, target)

	// Create symlinks well above the floor (16) so the budget bound is
	// genuinely exercised. statBudget(limit=2) returns the floor=16.
	const symlinkCount = 50
	for i := range symlinkCount {
		name := filepath.Join(dir, "ln"+padNum(i))
		if err := os.Symlink(target, name); err != nil {
			t.Fatal(err)
		}
	}

	c := newDirCache()
	entry := c.read(dir)

	cands := generateCandidates(FileArgument, "", "", 0, "ln", entry, 2, false, dir)
	if len(cands) > 2 {
		t.Errorf("candidates exceeded limit: got %d, want ≤ 2", len(cands))
	}
}

func padNum(i int) string {
	if i < 10 {
		return "0" + string(rune('0'+i))
	}
	tens := i / 10
	ones := i % 10
	return string(rune('0'+tens)) + string(rune('0'+ones))
}

func TestGenerateCandidates_UnknownTypeResolvedByStat(t *testing.T) {
	// A "real" DT_UNKNOWN scenario can't easily be forced from userspace,
	// but we can verify the kindUnknown handling by crafting a dirCacheEntry
	// manually. This locks in that kindUnknown entries go through stat
	// (same path as kindSymlink) rather than being misclassified.
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "real.txt"))

	entry := &dirCacheEntry{
		names:     []string{"real.txt"},
		foldNames: []string{"real.txt"},
		kinds:     []entryKind{kindUnknown}, // forced unknown
	}
	cands := generateCandidates(FileArgument, "", "", 0, "real", entry, 10, false, dir)
	if len(cands) != 1 || cands[0].displayName != "real.txt" {
		t.Errorf("kindUnknown entry should resolve to file via stat: got %v", names(cands))
	}
}

func TestGenerateCandidates_Truncation(t *testing.T) {
	dir := makeTestDir(t)
	c := newDirCache()
	entry := c.read(dir)

	cands := generateCandidates(FileArgument, "", "", 0, "", entry, 2, false, dir)
	if len(cands) != 2 {
		t.Errorf("limit=2 → len=%d, want 2", len(cands))
	}
}

func TestGenerateCandidates_CaseSensitivity(t *testing.T) {
	dir := makeTestDir(t)
	c := newDirCache()
	entry := c.read(dir)

	cands := generateCandidates(FileArgument, "", "", 0, "BET", entry, 50, false, dir)
	if runtime.GOOS == "linux" {
		// Case-sensitive: only "Beta/" matches "BET", not "beta.txt".
		if containsName(cands, "beta.txt") {
			t.Errorf("Linux case-sensitive: beta.txt should NOT match 'BET'")
		}
		if !containsName(cands, "Beta/") {
			t.Errorf("Linux case-sensitive: Beta/ should match 'BET'")
		}
	} else if !containsName(cands, "beta.txt") || !containsName(cands, "Beta/") {
		// Case-insensitive (macOS/Windows heuristic): both surface.
		t.Errorf("case-insensitive: expected both Beta/ and beta.txt for 'BET': %v", names(cands))
	}
}

// TestGenerateCandidates_InsertionStrings exercises the full pipeline with
// non-empty tokenPrefix and openingQuote so the candidate's insertion field
// reflects the equals-form / quoted-form variants end-to-end. (Without this,
// the unparam linter flags tokenPrefix/openingQuote as always-zero.)
func TestGenerateCandidates_InsertionStrings(t *testing.T) {
	dir := makeTestDir(t)
	c := newDirCache()
	entry := c.read(dir)

	// Equals-form unquoted: tokenPrefix="--path=", openingQuote=0.
	cands := generateCandidates(FileArgument, "--path=", "", 0, "alpha", entry, 10, false, dir)
	if len(cands) == 0 || cands[0].insertion != "--path=alpha.txt" {
		t.Errorf("equals-form insertion = %v, want --path=alpha.txt", cands)
	}

	// Equals-form quoted: tokenPrefix=`--path="`, openingQuote='"'.
	cands = generateCandidates(FileArgument, `--path="`, "", '"', "alpha", entry, 10, false, dir)
	if len(cands) == 0 || cands[0].insertion != `--path="alpha.txt"` {
		t.Errorf("equals-quoted insertion = %v, want --path=\"alpha.txt\"", cands)
	}

	// Positional quoted with userPrefix preserved.
	cands = generateCandidates(FileArgument, `"`, "~/", '"', "alpha", entry, 10, false, dir)
	if len(cands) == 0 || cands[0].insertion != `"~/alpha.txt"` {
		t.Errorf("quoted positional insertion = %v, want \"~/alpha.txt\"", cands)
	}
}

func TestBuildCompletion_Insertion(t *testing.T) {
	cases := []struct {
		name        string
		basename    string
		isDir       bool
		tokenPrefix string
		userPrefix  string
		opener      rune
		want        string
	}{
		{"plain file", "report.md", false, "", "~/", 0, "~/report.md"},
		{"plain dir", "Documents", true, "", "~/", 0, "~/Documents/"},
		{"quoted file", "report.md", false, `"`, "~/", '"', `"~/report.md"`},
		{"quoted dir", "Documents", true, `"`, "~/", '"', `"~/Documents/"`},
		{"equals-form file", "report.md", false, "--path=", "~/", 0, "--path=~/report.md"},
		{"equals-form dir", "Documents", true, "--path=", "~/", 0, "--path=~/Documents/"},
		{"equals-quoted", "Documents", true, `--path="`, "~/", '"', `--path="~/Documents/"`},
		{"auto-quote file with space", "My Doc.txt", false, "", "~/", 0, `"~/My Doc.txt"`},
		{"auto-quote dir with space", "My Documents", true, "", "~/", 0, `"~/My Documents/"`},
		{"auto-quote equals-form", "My Documents", true, "--path=", "~/", 0, `--path="~/My Documents/"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := buildCompletion(tc.basename, tc.isDir, tc.tokenPrefix, tc.userPrefix, tc.opener)
			if got.insertion != tc.want {
				t.Errorf("insertion = %q, want %q", got.insertion, tc.want)
			}
		})
	}
}

func containsName(cands []pathCompletion, name string) bool {
	for _, c := range cands {
		if c.displayName == name {
			return true
		}
	}
	return false
}

func names(cands []pathCompletion) []string {
	out := make([]string, len(cands))
	for i, c := range cands {
		out[i] = c.displayName
	}
	return out
}
