package bubblecomplete

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strings"
)

// pathValidity classifies the typed path value for the input-range overlay
// and the whole-input style suppression.
type pathValidity uint8

const (
	// pathInvalid: the value does not resolve, or resolves to the wrong kind.
	pathInvalid pathValidity = iota
	// pathPartial: the value is a strict prefix of one or more existing
	// entries, OR (for FileArgument) the value ends in a separator on a
	// real directory — mid-navigation.
	pathPartial
	// pathValid: the value resolves to an entry of the expected kind.
	pathValid
)

// pathState is the per-keystroke result of [activeFileArgument] plus
// classification and candidate generation. Stored on [Model], read by
// [Model.Render] and the validation-style suppression. Frozen during
// completion cycling, same as validationErr — [Model.renderedInput]
// bypasses the overlay during cycling so the stale offsets don't mis-style
// the previewed candidate.
type pathState struct {
	// active is true when the user is currently editing a value for a
	// file/dir-typed argument and the feature is enabled.
	active bool
	// kind is the argument's [ArgumentType] (one of FileArgument,
	// DirArgument, FileDirArgument) when active.
	kind ArgumentType
	// argName is the active argument's display name as produced by
	// argument.getName() — used to verify that a PathNotFound validation
	// error actually belongs to the active token before suppressing
	// whole-input red.
	argName string
	// valueStart and valueEnd are byte offsets in m.input.Value() of the
	// unquoted value range — used by the render overlay.
	valueStart int
	valueEnd   int
	// base is the basename prefix from [resolvePath], used as the
	// matchPrefix when path candidates replace the completion list.
	base string
	// validity drives the per-token overlay style.
	validity pathValidity
	// candidates is the list rendered in place of the normal completion
	// rows when active and non-empty.
	candidates []pathCompletion
}

// pathCompletion is a [completion] backed by a filesystem entry under the
// active file/dir argument value. It carries the full active-token
// replacement in insertion so the existing keyTab pretext + getAutocomplete
// concatenation works unchanged for positional, space-separated flag, and
// equals-form values.
type pathCompletion struct {
	displayName string // basename + "/" if directory
	insertion   string // FULL active-token replacement
	description string // "file" or "dir"
	isDir       bool
}

func (p pathCompletion) getName() string         { return p.displayName }
func (p pathCompletion) getDescription() string  { return p.description }
func (p pathCompletion) getAutocomplete() string { return p.insertion }

// statBudget returns the maximum number of [os.Stat] calls
// generateCandidates will make per pass to resolve symlink / unknown-type
// target kinds. Tying the budget to limit means worst-case CPU per
// keystroke scales with the candidate cap the host configured: with the
// default limit of 200, at most 200 stats are made even in a directory of
// 100k symlinks. A floor protects the small-limit case from being
// pathologically restrictive.
//
// The budget primarily exists to cap CPU on pathological cases. For the
// typical mix of regular files and dirs (with DirEntry.Type() already
// populated) it has no effect on output. For symlink-heavy directories
// where dirs-first sorting would have promoted a late symlink-to-dir, the
// budget can cause that entry to be dropped entirely rather than sorted
// to the front — an accepted candidate-quality tradeoff against bounded
// worst-case latency.
func statBudget(limit int) int {
	const floor = 16
	if limit < floor {
		return floor
	}
	return limit
}

// activeFileArgument reports whether the user is currently editing a value
// for a [FileArgument], [DirArgument], or [FileDirArgument], and if so
// returns the metadata needed to drive completion and validity colouring.
//
// Returns ok == false when:
//   - input is empty or ends in a space (no active value token)
//   - no command word has been entered yet
//   - the active position is a command word, a flag name without value, or a
//     value for a non-file argument type
//   - the active value is empty (e.g. "--path=" with nothing after)
//
// argName is the active argument's display name (argument.getName()) — used
// to verify that a PathNotFound validation error belongs to the active
// token before suppressing whole-input red. valueStart and valueEnd are
// byte offsets in input that bound the *unquoted* value text. tokenPrefix
// is whatever leads the active token before the value (flag prefix and/or
// opening quote). openingQuote is 0 if no quote.
func activeFileArgument(input string, commands []*Command) (
	kind ArgumentType,
	argName string,
	valueStart, valueEnd int,
	tokenPrefix string,
	openingQuote rune,
	ok bool,
) {
	if strings.HasSuffix(input, " ") {
		return
	}
	tokens := tokenize(input)
	if len(tokens) == 0 {
		return
	}
	parts := splitInput(input)
	if len(parts) == 0 {
		return
	}

	finalCmd, depth, globalFlags := walkToFinalCommand(input, parts, commands)
	if finalCmd == nil {
		return
	}

	argParts := parts[depth:]
	if len(argParts) == 0 {
		return
	}
	posArgs, flagArgs := splitPositionArgsAndFlags(argParts, finalCmd, globalFlags)

	activeToken := tokens[len(tokens)-1]
	tokenRaw := input[activeToken.Start:activeToken.End]

	// Case 1: equals-form flag value. Distinguished by the active token
	// being flag-shaped (starts with '-') AND containing an '='.
	if strings.HasPrefix(activeToken.Unquoted, "-") && strings.Contains(activeToken.Unquoted, "=") {
		before, _, _ := strings.Cut(tokenRaw, "=")
		allFlags := slices.Concat(finalCmd.Flags, globalFlags)
		for _, f := range allFlags {
			if f.LongFlag != before && f.PsFlag != before {
				continue
			}
			if !isFileLikeArg(f.Type) {
				return
			}
			kind = f.Type
			argName = f.getName()
			eqIdx := strings.IndexByte(tokenRaw, '=')
			valueStart = activeToken.Start + eqIdx + 1
			valueEnd = activeToken.End
			tokenPrefix = tokenRaw[:eqIdx+1]
			// Inline opening quote after '='?
			if valueStart < valueEnd {
				first := input[valueStart]
				if first == '"' || first == '\'' {
					openingQuote = rune(first)
					tokenPrefix += string(openingQuote)
					valueStart++
					if valueEnd > valueStart && input[valueEnd-1] == first {
						valueEnd--
					}
				}
			}
			ok = valueStart < valueEnd
			if !ok {
				return ArgumentType(""), "", 0, 0, "", 0, false
			}
			return kind, argName, valueStart, valueEnd, tokenPrefix, openingQuote, true
		}
		// Flag prefix didn't match any known flag — not active.
		return
	}

	// Case 2: space-separated flag value. isEnteringFlagValue inspects the
	// PRECEDING token to see whether it was a flag waiting for a value.
	if yes, flag := isEnteringFlagValue(input, finalCmd, flagArgs, globalFlags); yes {
		if !isFileLikeArg(flag.Type) {
			return
		}
		return finalizeTokenRange(activeToken, flag.Type, flag.getName())
	}

	// Case 3: positional argument value. Two guards:
	//   - active token must not be a flag-shaped token (the user might be
	//     typing "--path" — isEnteringPosArgValue would otherwise report
	//     that as "still typing positional ." for the previous positional)
	//   - posArgs index must be in bounds for finalCmd.PositionalArguments
	//     (extra positionals beyond declared = "unexpected argument", not
	//     an active value position)
	if !strings.HasPrefix(activeToken.Unquoted, "-") &&
		len(posArgs) > 0 && len(posArgs) <= len(finalCmd.PositionalArguments) {
		if yes, pos := isEnteringPosArgValue(input, finalCmd, posArgs); yes {
			if !isFileLikeArg(pos.Type) {
				return
			}
			return finalizeTokenRange(activeToken, pos.Type, pos.getName())
		}
	}

	return
}

// finalizeTokenRange computes valueStart/valueEnd/tokenPrefix/openingQuote
// for a positional or space-separated-flag value (Case 2/3 above). The
// active token's bounds come from [tokenize]; quote handling reflects
// whether the token opened with a quote and whether that quote was closed.
func finalizeTokenRange(tok token, kind ArgumentType, name string) (
	ArgumentType, string, int, int, string, rune, bool,
) {
	valueStart := tok.Start
	valueEnd := tok.End
	var tokenPrefix string
	var openingQuote rune
	if tok.Quoted {
		openingQuote = tok.Quote
		tokenPrefix = string(openingQuote)
		valueStart++
		if tok.Closed {
			valueEnd--
		}
	}
	if valueStart >= valueEnd {
		return ArgumentType(""), "", 0, 0, "", 0, false
	}
	return kind, name, valueStart, valueEnd, tokenPrefix, openingQuote, true
}

// isFileLikeArg reports whether t is one of the filesystem-backed argument
// types that this feature targets.
func isFileLikeArg(t ArgumentType) bool {
	return t == FileArgument || t == DirArgument || t == FileDirArgument
}

// caseInsensitiveFS is the runtime-derived heuristic for matching policy.
// Linux is case-sensitive, macOS and Windows are case-insensitive. Documented
// as a heuristic — case-sensitive APFS volumes on macOS are misclassified.
func caseInsensitiveFS() bool {
	return runtime.GOOS != "linux"
}

// classify decides the path validity for the currently typed value. Stat
// semantics mirror [validatePath] (uses [os.Stat], which follows symlinks)
// so green ⇔ Enter accepts.
func classify(kind ArgumentType, fullClean, base string, entry *dirCacheEntry) pathValidity {
	if entry.err != nil {
		return pathInvalid
	}
	if base == "" {
		// Trailing separator on a real directory.
		switch kind {
		case DirArgument, FileDirArgument:
			return pathValid
		case FileArgument:
			return pathPartial
		default:
			return pathInvalid
		}
	}

	if exactMatch(entry, base) {
		info, err := os.Stat(fullClean)
		if err != nil {
			return pathInvalid
		}
		return classifyByKind(kind, info.IsDir())
	}
	if prefixMatch(entry, base) {
		return pathPartial
	}
	return pathInvalid
}

// classifyByKind applies the same IsDir() predicate validatePath uses.
func classifyByKind(kind ArgumentType, isDir bool) pathValidity {
	switch kind {
	case FileArgument:
		if isDir {
			return pathInvalid
		}
		return pathValid
	case DirArgument:
		if isDir {
			return pathValid
		}
		return pathInvalid
	case FileDirArgument:
		return pathValid
	default:
		return pathInvalid
	}
}

func exactMatch(entry *dirCacheEntry, base string) bool {
	if caseInsensitiveFS() {
		return slices.Contains(entry.foldNames, strings.ToLower(base))
	}
	return slices.Contains(entry.names, base)
}

func prefixMatch(entry *dirCacheEntry, base string) bool {
	if caseInsensitiveFS() {
		lower := strings.ToLower(base)
		for _, n := range entry.foldNames {
			if strings.HasPrefix(n, lower) {
				return true
			}
		}
		return false
	}
	for _, n := range entry.names {
		if strings.HasPrefix(n, base) {
			return true
		}
	}
	return false
}

// generateCandidates produces the [pathCompletion] list for the active
// value. Iteration order is the ReadDir result (sorted lex). Survivors are
// re-sorted dirs-first, case-fold alphabetic, then truncated to limit.
//
// Symlink target kinds are resolved via [os.Stat] for entries that survive
// the prefix and hidden filters; [statBudget] bounds the worst-case stat
// count per pass.
func generateCandidates(
	kind ArgumentType,
	tokenPrefix, userPrefix string,
	openingQuote rune,
	base string,
	entry *dirCacheEntry,
	limit int,
	hiddenFiles bool,
	parent string,
) []pathCompletion {
	if entry.err != nil {
		return nil
	}
	if limit < 1 {
		limit = 1
	}

	type prelim struct {
		name      string
		isDir     bool
		isSymlink bool // original DirEntry kind was kindSymlink (description hint)
	}

	caseInsensitive := caseInsensitiveFS()
	lowerBase := strings.ToLower(base)
	statsRemaining := statBudget(limit)
	survivors := make([]prelim, 0, min(limit, len(entry.names)))

	for i, name := range entry.names {
		if !matchesPrefix(name, entry.foldNames[i], base, lowerBase, caseInsensitive) {
			continue
		}
		if !hiddenFiles && strings.HasPrefix(name, ".") && !strings.HasPrefix(base, ".") {
			continue
		}

		k := entry.kinds[i]
		wasSymlink := k == kindSymlink
		if k == kindSymlink || k == kindUnknown {
			if statsRemaining <= 0 {
				continue // budget exhausted; drop
			}
			statsRemaining--
			info, err := os.Stat(filepath.Join(parent, name))
			if err != nil {
				continue // broken symlink or stat failure
			}
			switch {
			case info.IsDir():
				k = kindDir
			case info.Mode().IsRegular():
				k = kindFile
			default:
				k = kindOther
			}
		}

		if !includeKind(kind, k) {
			continue
		}
		survivors = append(survivors, prelim{name: name, isDir: k == kindDir, isSymlink: wasSymlink})
	}

	// Dirs first, then case-fold alphabetic within each group.
	sort.SliceStable(survivors, func(i, j int) bool {
		if survivors[i].isDir != survivors[j].isDir {
			return survivors[i].isDir
		}
		return strings.ToLower(survivors[i].name) < strings.ToLower(survivors[j].name)
	})

	if len(survivors) > limit {
		survivors = survivors[:limit]
	}

	out := make([]pathCompletion, len(survivors))
	for i, s := range survivors {
		out[i] = buildCompletion(s.name, s.isDir, s.isSymlink, tokenPrefix, userPrefix, openingQuote)
	}
	return out
}

// matchesPrefix applies the platform's case-sensitivity heuristic.
func matchesPrefix(name, foldName, base, lowerBase string, caseInsensitive bool) bool {
	if caseInsensitive {
		return strings.HasPrefix(foldName, lowerBase)
	}
	return strings.HasPrefix(name, base)
}

// includeKind applies the candidate-inclusion table:
//   - FileArgument: files + dirs (dirs for drill-down)
//   - DirArgument:  dirs only
//   - FileDirArgument: files + dirs
//
// Specials (devices, sockets, pipes — kindOther) never appear in candidates,
// regardless of ArgumentType. Submitting a typed path to a special still
// classifies and validates per kind, so the invariant holds for any
// concrete typed path.
func includeKind(arg ArgumentType, k entryKind) bool {
	switch k {
	case kindFile:
		return arg != DirArgument
	case kindDir:
		return true
	default:
		return false
	}
}

// buildCompletion assembles the displayName and the full active-token
// replacement string. Quoting:
//   - openingQuote != 0: keep the user's quote style; restore the closer.
//   - openingQuote == 0 and the name contains a space: auto-quote with '"'
//     uniformly for files and directories. The opener slots between any
//     tokenPrefix (e.g. "--path=") and the userPrefix.
//
// File completions append a trailing space to the insertion — bash-style
// "this token is done, move on." Directories deliberately omit the space
// so the next Tab can drill into the dir's children. Autotrim (default
// on) strips the trailing space at submit time.
//
// When isSymlink is true (the original DirEntry was a symlink, regardless
// of its target kind), the description carries "symlink → file/dir" so
// the user can see that the entry isn't a regular file/directory.
//
// Known limitation: filenames containing literal quote characters (" or ')
// are inserted verbatim, which the tokenizer cannot re-parse as a single
// token. Such filenames are vanishingly rare in practice and documented as
// out of scope for v1.
func buildCompletion(name string, isDir, isSymlink bool, tokenPrefix, userPrefix string, openingQuote rune) pathCompletion {
	displayName := name
	if isDir {
		displayName += "/"
	}

	insertion := tokenPrefix
	opener := openingQuote
	if opener == 0 && strings.ContainsRune(name, ' ') {
		opener = '"'
	}
	if opener != 0 && !strings.HasSuffix(tokenPrefix, string(opener)) {
		insertion += string(opener)
	}
	insertion += userPrefix + name
	if isDir {
		insertion += "/"
	}
	if opener != 0 {
		insertion += string(opener)
	}
	if !isDir {
		insertion += " "
	}

	kindLabel := "file"
	if isDir {
		kindLabel = "dir"
	}
	description := kindLabel
	if isSymlink {
		description = "symlink → " + kindLabel
	}
	return pathCompletion{
		displayName: displayName,
		insertion:   insertion,
		description: description,
		isDir:       isDir,
	}
}

// isPartialPathMidType reports whether the whole-input invalid style should
// be suppressed in favour of the path-range overlay. True only when:
//   - the active path is a strict prefix (pathPartial), AND
//   - the validation error is a PathNotFound, AND
//   - the error's Argument matches the active argument's name.
//
// The argument-name check matters for commands with multiple path
// positionals: typing `cp /missing/foo /tmp/par` would otherwise suppress
// the whole-input red even though the validation error comes from the
// FIRST (committed) path, not the partial second one. Suppression should
// only fire when the partial path IS the cause of the error.
func (m Model) isPartialPathMidType() bool {
	if !m.pathState.active || m.pathState.validity != pathPartial {
		return false
	}
	var ve *ValidationError
	if !errors.As(m.validationErr, &ve) {
		return false
	}
	if ve.Kind != PathNotFound {
		return false
	}
	return ve.Argument == m.pathState.argName
}

// recomputePathState refreshes m.pathState from the current input value.
// Called from Update's input-changed branch before getCompletions and
// validateInput, so getCompletions can wholesale-replace its result list
// with pathState.candidates when the active argument is file/dir-typed.
//
// When the feature is disabled or no file/dir value is active, pathState is
// reset to its zero value. Cost in that case is one bool check plus the
// activeFileArgument detector (tokenisation + command walk, which run for
// existing completion logic anyway).
func (m *Model) recomputePathState() {
	m.pathState = pathState{}
	if !m.FilesystemCompletions {
		return
	}

	kind, argName, valueStart, valueEnd, tokenPrefix, openingQuote, ok := activeFileArgument(m.input.Value(), m.Commands)
	if !ok {
		return
	}

	if m.pathCache == nil {
		m.pathCache = newDirCache()
	}

	cwd, _ := os.Getwd()
	home, _ := os.UserHomeDir()
	typed := m.input.Value()[valueStart:valueEnd]
	expandTilde := openingQuote != '\''
	fullClean, parent, base, userPrefix := resolvePath(typed, cwd, home, expandTilde)

	entries := m.pathCache.read(parent)

	limit := max(m.FilesystemCompletionLimit, 1)

	m.pathState = pathState{
		active:     true,
		kind:       kind,
		argName:    argName,
		valueStart: valueStart,
		valueEnd:   valueEnd,
		base:       base,
		validity:   classify(kind, fullClean, base, entries),
		candidates: generateCandidates(kind, tokenPrefix, userPrefix, openingQuote, base, entries, limit, m.HiddenFiles, parent),
	}
}
