package bubblecomplete

import (
	"path/filepath"
	"strings"
)

// stripQuotes strips a recognised outer quote (" or '): the opening byte is
// always removed, the trailing byte too when it equals the opener. Returns
// the stripped text and the opener (0 if none). Embedded quotes
// (`"foo"bar"`) are outer-stripped only — unreachable from tokenizer
// output, and submit-time callers run [checkUnclosedQuote] first.
func stripQuotes(s string) (unquoted string, opener rune) {
	if s == "" {
		return s, 0
	}
	first := s[0]
	if first != '"' && first != '\'' {
		return s, 0
	}
	opener = rune(first)
	body := s[1:]
	if len(body) >= 1 && body[len(body)-1] == first {
		return body[:len(body)-1], opener
	}
	return body, opener
}

// resolvePath canonicalises an already-unquoted path value. Pure given the
// inputs. Callers set expandTilde true when the value is unquoted or
// double-quoted, false inside single quotes (shell behaviour); a bare "~"
// is normalised to "~/" regardless. fullClean is Clean'd and absolute
// unless cwd == "" and the typed path is relative (os.Stat then falls back
// to the process CWD). parent ends with a separator when base is empty so
// consumers can ReadDir(parent) uniformly. userPrefix is the typed bytes up
// to and including the last separator, preserved verbatim so insertions
// keep the user's style ("~/" rather than the expanded home).
func resolvePath(typed, cwd, home string, expandTilde bool) (fullClean, parent, base, userPrefix string) {
	if typed == "" {
		return "", "", "", ""
	}

	// A user typing just "~" expects to navigate into home, not to complete
	// a sibling of home.
	if typed == "~" {
		typed = "~/"
	}

	// userPrefix is derived from the (possibly normalised) typed string,
	// before any tilde expansion or CWD joining.
	if i := lastSeparatorIndex(typed); i >= 0 {
		userPrefix = typed[:i+1]
	}

	endsWithSep := endsInSeparator(typed)

	expanded := typed
	if expandTilde && home != "" && strings.HasPrefix(typed, "~") {
		switch {
		case typed == "~/":
			expanded = home
		case len(typed) >= 2 && (typed[1] == '/' || typed[1] == filepath.Separator):
			expanded = filepath.Join(home, typed[2:])
		}
	}

	joined := expanded
	if !filepath.IsAbs(expanded) && cwd != "" {
		joined = filepath.Join(cwd, expanded)
	}

	fullClean = filepath.Clean(joined)

	if endsWithSep {
		parent = fullClean
		if !endsInSeparator(parent) {
			parent += string(filepath.Separator)
		}
		base = ""
	} else {
		parent, base = filepath.Split(fullClean)
	}

	return fullClean, parent, base, userPrefix
}

// endsInSeparator reports whether s ends with '/' or filepath.Separator —
// on Windows both forms are accepted, matching Go's filepath package.
func endsInSeparator(s string) bool {
	if s == "" {
		return false
	}
	c := s[len(s)-1]
	return c == '/' || c == filepath.Separator
}

func lastSeparatorIndex(s string) int {
	i := strings.LastIndexByte(s, '/')
	if filepath.Separator != '/' {
		if j := strings.LastIndexByte(s, byte(filepath.Separator)); j > i {
			i = j
		}
	}
	return i
}
