package bubblecomplete

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestStripQuotes(t *testing.T) {
	cases := []struct {
		name     string
		in       string
		wantOut  string
		wantQuot rune
	}{
		{"empty", "", "", 0},
		{"no quote", "~/Doc", "~/Doc", 0},
		{"matched double", `"~/Doc"`, "~/Doc", '"'},
		{"matched single", `'~/Doc'`, "~/Doc", '\''},
		{"unclosed double", `"~/Doc`, "~/Doc", '"'},
		{"unclosed single", `'~/Doc`, "~/Doc", '\''},
		{"only opener double", `"`, "", '"'},
		{"only opener single", `'`, "", '\''},
		{"pair just quotes", `""`, "", '"'},
		{"mismatched closers stay literal", `"foo'`, "foo'", '"'},
		{"outer strip with embedded quote", `"foo"bar"`, `foo"bar`, '"'},
		{"leading non-quote is no-op", `x"foo"`, `x"foo"`, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, q := stripQuotes(c.in)
			if out != c.wantOut {
				t.Errorf("unquoted = %q, want %q", out, c.wantOut)
			}
			if q != c.wantQuot {
				t.Errorf("opener = %q (%d), want %q (%d)", q, q, c.wantQuot, c.wantQuot)
			}
		})
	}
}

func TestResolvePath(t *testing.T) {
	// Canonical fixtures. cwd and home are *strings* — the resolver is pure,
	// so we don't need real directories.
	const (
		cwd  = "/work"
		home = "/Users/jane"
	)

	cases := []struct {
		name           string
		typed          string
		cwd            string
		home           string
		expandTilde    bool
		wantFullClean  string
		wantParent     string
		wantBase       string
		wantUserPrefix string
	}{
		{
			name:           "empty",
			typed:          "",
			cwd:            cwd,
			home:           home,
			expandTilde:    true,
			wantFullClean:  "",
			wantParent:     "",
			wantBase:       "",
			wantUserPrefix: "",
		},
		{
			name:           "relative no separator",
			typed:          "Doc",
			cwd:            cwd,
			home:           home,
			expandTilde:    true,
			wantFullClean:  "/work/Doc",
			wantParent:     "/work/",
			wantBase:       "Doc",
			wantUserPrefix: "",
		},
		{
			name:           "relative with separator",
			typed:          "sub/Doc",
			cwd:            cwd,
			home:           home,
			expandTilde:    true,
			wantFullClean:  "/work/sub/Doc",
			wantParent:     "/work/sub/",
			wantBase:       "Doc",
			wantUserPrefix: "sub/",
		},
		{
			name:           "dot-prefix relative",
			typed:          "./Doc",
			cwd:            cwd,
			home:           home,
			expandTilde:    true,
			wantFullClean:  "/work/Doc",
			wantParent:     "/work/",
			wantBase:       "Doc",
			wantUserPrefix: "./",
		},
		{
			name:           "absolute file",
			typed:          "/abs/path/file.txt",
			cwd:            cwd,
			home:           home,
			expandTilde:    true,
			wantFullClean:  "/abs/path/file.txt",
			wantParent:     "/abs/path/",
			wantBase:       "file.txt",
			wantUserPrefix: "/abs/path/",
		},
		{
			name:           "absolute trailing slash",
			typed:          "/abs/path/",
			cwd:            cwd,
			home:           home,
			expandTilde:    true,
			wantFullClean:  "/abs/path",
			wantParent:     "/abs/path/",
			wantBase:       "",
			wantUserPrefix: "/abs/path/",
		},
		{
			name:           "bare tilde with expansion",
			typed:          "~",
			cwd:            cwd,
			home:           home,
			expandTilde:    true,
			wantFullClean:  "/Users/jane",
			wantParent:     "/Users/jane/",
			wantBase:       "",
			wantUserPrefix: "~/",
		},
		{
			// Bare ~ is normalised to ~/ regardless of expandTilde, so even
			// the no-expand path produces userPrefix "~/" and a trailing-slash
			// classification. The resolved fullClean is cwd/~ (literal),
			// because expandTilde=false leaves ~ alone.
			name:           "bare tilde without expansion",
			typed:          "~",
			cwd:            cwd,
			home:           home,
			expandTilde:    false,
			wantFullClean:  "/work/~",
			wantParent:     "/work/~/",
			wantBase:       "",
			wantUserPrefix: "~/",
		},
		{
			name:           "tilde slash with expansion",
			typed:          "~/",
			cwd:            cwd,
			home:           home,
			expandTilde:    true,
			wantFullClean:  "/Users/jane",
			wantParent:     "/Users/jane/",
			wantBase:       "",
			wantUserPrefix: "~/",
		},
		{
			name:           "tilde prefix with subpath",
			typed:          "~/Doc",
			cwd:            cwd,
			home:           home,
			expandTilde:    true,
			wantFullClean:  "/Users/jane/Doc",
			wantParent:     "/Users/jane/",
			wantBase:       "Doc",
			wantUserPrefix: "~/",
		},
		{
			name:           "tilde prefix deep",
			typed:          "~/Documents/report.md",
			cwd:            cwd,
			home:           home,
			expandTilde:    true,
			wantFullClean:  "/Users/jane/Documents/report.md",
			wantParent:     "/Users/jane/Documents/",
			wantBase:       "report.md",
			wantUserPrefix: "~/Documents/",
		},
		{
			name:           "tilde no expansion stays literal",
			typed:          "~/Doc",
			cwd:            cwd,
			home:           home,
			expandTilde:    false,
			wantFullClean:  "/work/~/Doc",
			wantParent:     "/work/~/",
			wantBase:       "Doc",
			wantUserPrefix: "~/",
		},
		{
			name:           "tilde with empty home stays literal",
			typed:          "~/Doc",
			cwd:            cwd,
			home:           "",
			expandTilde:    true,
			wantFullClean:  "/work/~/Doc",
			wantParent:     "/work/~/",
			wantBase:       "Doc",
			wantUserPrefix: "~/",
		},
		{
			name:           "tilde user form treated literally",
			typed:          "~root/Doc",
			cwd:            cwd,
			home:           home,
			expandTilde:    true,
			wantFullClean:  "/work/~root/Doc",
			wantParent:     "/work/~root/",
			wantBase:       "Doc",
			wantUserPrefix: "~root/",
		},
		{
			name:           "trailing separator after tilde",
			typed:          "~/Documents/",
			cwd:            cwd,
			home:           home,
			expandTilde:    true,
			wantFullClean:  "/Users/jane/Documents",
			wantParent:     "/Users/jane/Documents/",
			wantBase:       "",
			wantUserPrefix: "~/Documents/",
		},
		{
			name:           "dot-dot normalisation",
			typed:          "~/a/../b",
			cwd:            cwd,
			home:           home,
			expandTilde:    true,
			wantFullClean:  "/Users/jane/b",
			wantParent:     "/Users/jane/",
			wantBase:       "b",
			wantUserPrefix: "~/a/../",
		},
		{
			name:           "empty cwd keeps relative",
			typed:          "Doc",
			cwd:            "",
			home:           home,
			expandTilde:    true,
			wantFullClean:  "Doc",
			wantParent:     "",
			wantBase:       "Doc",
			wantUserPrefix: "",
		},
		{
			name:           "empty cwd with subpath",
			typed:          "sub/Doc",
			cwd:            "",
			home:           home,
			expandTilde:    true,
			wantFullClean:  "sub/Doc",
			wantParent:     "sub/",
			wantBase:       "Doc",
			wantUserPrefix: "sub/",
		},
		{
			name:           "root path",
			typed:          "/",
			cwd:            cwd,
			home:           home,
			expandTilde:    true,
			wantFullClean:  "/",
			wantParent:     "/",
			wantBase:       "",
			wantUserPrefix: "/",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fullClean, parent, base, userPrefix := resolvePath(c.typed, c.cwd, c.home, c.expandTilde)
			if fullClean != c.wantFullClean {
				t.Errorf("fullClean = %q, want %q", fullClean, c.wantFullClean)
			}
			if parent != c.wantParent {
				t.Errorf("parent = %q, want %q", parent, c.wantParent)
			}
			if base != c.wantBase {
				t.Errorf("base = %q, want %q", base, c.wantBase)
			}
			if userPrefix != c.wantUserPrefix {
				t.Errorf("userPrefix = %q, want %q", userPrefix, c.wantUserPrefix)
			}
		})
	}
}

// TestResolvePath_ParentBaseInvariant asserts that parent and base together
// reconstruct fullClean modulo trailing-separator handling. The classifier
// and candidate generator both rely on parent+base being a faithful split.
func TestResolvePath_ParentBaseInvariant(t *testing.T) {
	const (
		cwd  = "/work"
		home = "/Users/jane"
	)
	inputs := []string{
		"Doc", "sub/Doc", "/abs/path/file.txt", "/abs/path/",
		"~", "~/", "~/Doc", "~/Documents/report.md", "~/a/../b", "/",
	}
	for _, in := range inputs {
		t.Run(in, func(t *testing.T) {
			fullClean, parent, base, _ := resolvePath(in, cwd, home, true)
			if fullClean == "" {
				return
			}
			// parent may end with a separator; base may be empty.
			joined := filepath.Clean(filepath.Join(parent, base))
			if joined != fullClean {
				t.Errorf("Join(parent=%q, base=%q) = %q, want %q",
					parent, base, joined, fullClean)
			}
		})
	}
}

// TestResolvePath_UserPrefixPreservesStyle confirms userPrefix is taken
// verbatim from the typed string (post bare-tilde fixup), not from the
// expanded path. Important for Tab-accept insertion strings.
func TestResolvePath_UserPrefixPreservesStyle(t *testing.T) {
	const home = "/Users/jane"
	_, _, _, prefix := resolvePath("~/Documents/report.md", "/work", home, true)
	if !strings.HasPrefix(prefix, "~/") {
		t.Errorf("userPrefix = %q; expected to start with %q (preserve user's tilde style)", prefix, "~/")
	}
}
