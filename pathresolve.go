package bubblecomplete

import (
	"path/filepath"
	"strings"
)

// stripQuotes performs simple outer-quote stripping. If s begins with a
// recognised quote character (" or '), the opening byte is always removed;
// the trailing byte is also removed when it equals the opener. Returns the
// stripped text plus the opening quote rune (0 if no quote).
//
// Behaviour:
//
//	`~/Doc`     → ("~/Doc", 0)         (no opener)
//	`"~/Doc`    → ("~/Doc", '"')       (unclosed opener)
//	`"~/Doc"`   → ("~/Doc", '"')       (matched pair)
//	`'~/Doc'`   → ("~/Doc", '\'')      (matched pair)
//	`"foo"bar"` → (`foo"bar`, '"')     (outer-strip; embedded quote is kept)
//
// Embedded quotes (`"foo"bar"`) are not produced by the tokenizer — it splits
// on the first matched closer — so the last case is only reachable when a
// caller passes a pre-assembled string. Outer-strip semantics are simple and
// adequate because submit-time callers run [checkUnclosedQuote] first, and
// editing-time callers use it on tokenizer output where the case can't arise.
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
// inputs.
//
// expandTilde controls whether a leading "~" or "~/" is expanded against
// home. By library convention, callers set expandTilde true when the value
// is unquoted or inside double quotes, and false when inside single quotes
// — matching shell behaviour. A bare "~" is normalised to "~/" for both
// resolution and userPrefix, regardless of expandTilde.
//
// fullClean is filepath.Clean'd. It is absolute when the typed path is
// absolute OR cwd is non-empty. When cwd == "" and the typed path is
// relative, fullClean stays relative — os.Stat then uses the process CWD,
// matching today's validatePath behaviour when os.Getwd errors.
//
// parent ends with a separator when base is empty (the user typed a
// trailing separator) so consumers can ReadDir(parent) uniformly. parent
// and base together reconstruct fullClean.
//
// userPrefix is the leading bytes of typed (after the bare-tilde fixup) up
// to and including the last separator before base — preserved verbatim so
// insertions keep the user's style (e.g. "~/" rather than the expanded
// "/Users/jane/").
//
// For an empty typed string, resolvePath returns four empty strings.
func resolvePath(typed, cwd, home string, expandTilde bool) (fullClean, parent, base, userPrefix string) {
	if typed == "" {
		return "", "", "", ""
	}

	// Normalise bare "~" to "~/" for both resolution and userPrefix. The
	// user typing just "~" expects to navigate into home, not to complete a
	// sibling of home (which is what the strict reading would imply).
	if typed == "~" {
		typed = "~/"
	}

	// userPrefix is derived from the (possibly normalised) typed string,
	// before any tilde expansion or CWD joining.
	if i := lastSeparatorIndex(typed); i >= 0 {
		userPrefix = typed[:i+1]
	}

	endsWithSep := endsInSeparator(typed)

	// Tilde expansion.
	expanded := typed
	if expandTilde && home != "" && strings.HasPrefix(typed, "~") {
		switch {
		case typed == "~/":
			expanded = home
		case len(typed) >= 2 && (typed[1] == '/' || typed[1] == filepath.Separator):
			expanded = filepath.Join(home, typed[2:])
		}
	}

	// Join relative with cwd.
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

// endsInSeparator reports whether s ends with '/' or filepath.Separator.
// On Linux filepath.Separator is '/' so both checks collapse; on Windows
// '\\' is also accepted because Go's filepath package treats both forms
// equivalently.
func endsInSeparator(s string) bool {
	if s == "" {
		return false
	}
	c := s[len(s)-1]
	return c == '/' || c == filepath.Separator
}

// lastSeparatorIndex returns the byte index of the last '/' or
// filepath.Separator in s, or -1 if none.
func lastSeparatorIndex(s string) int {
	i := strings.LastIndexByte(s, '/')
	if filepath.Separator != '/' {
		if j := strings.LastIndexByte(s, byte(filepath.Separator)); j > i {
			i = j
		}
	}
	return i
}
