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
// [Model.Render] and the validation-style suppression.
//
// Refreshed on every input change (the input-changed branch in Update)
// AND on every Tab cycle and cycle-revert (keyTab), so the offsets and
// validity always reflect the value currently in [Model.input]. Files
// cycled-to carry a trailing space which makes activeFileArgument
// inactive — overlay correctly absent for that preview. Dirs cycle with
// the overlay active and the colour reflects the cycled-to dir.
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
	description string // "file" / "dir" — or "symlink → file" / "symlink → dir" when the underlying DirEntry was a symlink
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

// activeArg describes the user's active typing position when it's a value
// for a file/dir-typed argument. Populated by [activeFileArgument] and
// consumed by [Model.recomputePathState].
type activeArg struct {
	// kind is one of FileArgument, DirArgument, FileDirArgument.
	kind ArgumentType
	// name is the active argument's display name as produced by
	// argument.getName(). Used by isPartialPathMidType to verify that a
	// PathNotFound validation error belongs to this argument before
	// suppressing whole-input red.
	name string
	// valueStart and valueEnd are byte offsets in the input string that
	// bound the *unquoted* value text — used by the render overlay.
	valueStart int
	valueEnd   int
	// tokenPrefix is the leading bytes of the active token before the
	// value: flag prefix (`--path=`) and/or opening quote.
	tokenPrefix string
	// openingQuote is 0 if no quote, else '"' or '\''.
	openingQuote rune
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
func activeFileArgument(input string, commands []*Command) (activeArg, bool) {
	if strings.HasSuffix(input, " ") {
		return activeArg{}, false
	}
	tokens := tokenize(input)
	if len(tokens) == 0 {
		return activeArg{}, false
	}
	parts := splitInput(input)
	if len(parts) == 0 {
		return activeArg{}, false
	}

	finalCmd, depth, globalFlags := walkToFinalCommand(input, parts, commands)
	if finalCmd == nil {
		return activeArg{}, false
	}

	argParts := parts[depth:]
	if len(argParts) == 0 {
		return activeArg{}, false
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
				return activeArg{}, false
			}
			return equalsFormActiveArg(input, activeToken, tokenRaw, f.Type, f.getName())
		}
		// Flag prefix didn't match any known flag — not active.
		return activeArg{}, false
	}

	// Case 2: space-separated flag value. isEnteringFlagValue inspects the
	// PRECEDING token to see whether it was a flag waiting for a value.
	if yes, flag := isEnteringFlagValue(input, finalCmd, flagArgs, globalFlags); yes {
		if !isFileLikeArg(flag.Type) {
			return activeArg{}, false
		}
		return tokenActiveArg(activeToken, flag.Type, flag.getName())
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
				return activeArg{}, false
			}
			return tokenActiveArg(activeToken, pos.Type, pos.getName())
		}
	}

	return activeArg{}, false
}

// equalsFormActiveArg builds an activeArg for the equals-form flag value
// case (`--flag=value`, `--flag="value"`, etc.). Splits the active token on
// the first '=' and detects an inline opener after it.
func equalsFormActiveArg(input string, tok token, tokenRaw string, kind ArgumentType, name string) (activeArg, bool) {
	eqIdx := strings.IndexByte(tokenRaw, '=')
	a := activeArg{
		kind:        kind,
		name:        name,
		valueStart:  tok.Start + eqIdx + 1,
		valueEnd:    tok.End,
		tokenPrefix: tokenRaw[:eqIdx+1],
	}
	if a.valueStart < a.valueEnd {
		first := input[a.valueStart]
		if first == '"' || first == '\'' {
			a.openingQuote = rune(first)
			a.tokenPrefix += string(a.openingQuote)
			a.valueStart++
			if a.valueEnd > a.valueStart && input[a.valueEnd-1] == first {
				a.valueEnd--
			}
		}
	}
	if a.valueStart >= a.valueEnd {
		return activeArg{}, false
	}
	return a, true
}

// tokenActiveArg builds an activeArg for a positional or space-separated
// flag value (Cases 2 and 3 above). The active token's bounds come from
// [tokenize]; quote handling reflects whether the token opened with a
// quote and whether that quote was closed.
func tokenActiveArg(tok token, kind ArgumentType, name string) (activeArg, bool) {
	a := activeArg{
		kind:       kind,
		name:       name,
		valueStart: tok.Start,
		valueEnd:   tok.End,
	}
	if tok.Quoted {
		a.openingQuote = tok.Quote
		a.tokenPrefix = string(a.openingQuote)
		a.valueStart++
		if tok.Closed {
			a.valueEnd--
		}
	}
	if a.valueStart >= a.valueEnd {
		return activeArg{}, false
	}
	return a, true
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
	names, needle := entry.matchKey(base)
	return slices.Contains(names, needle)
}

func prefixMatch(entry *dirCacheEntry, base string) bool {
	names, needle := entry.matchKey(base)
	for _, n := range names {
		if strings.HasPrefix(n, needle) {
			return true
		}
	}
	return false
}

// candidateRequest groups the inputs to [generateCandidates] so the
// function signature stays manageable. Callers should construct the
// struct deliberately rather than relying on field defaults — several
// fields have meaningful zero values (e.g. an empty tokenPrefix or
// userPrefix, openingQuote == 0 meaning "no quote", hiddenFiles ==
// false), so a partial literal must be intentional, not accidental.
type candidateRequest struct {
	// kind is the active argument's type (FileArgument / DirArgument /
	// FileDirArgument) — drives the inclusion filter.
	kind ArgumentType
	// tokenPrefix is the leading bytes of the active token that precede
	// the value (flag prefix and/or opening quote). Used to build
	// insertion strings that preserve the user's token shape.
	tokenPrefix string
	// userPrefix is the leading bytes of the typed value up to the last
	// separator before base ("~/", "./", "/abs/path/", ""). Preserved
	// verbatim in insertions so completion keeps the user's style.
	userPrefix string
	// openingQuote is 0 if no quote, else '"' or '\'' — drives insertion
	// quoting and auto-quote-on-space logic in [buildCompletion].
	openingQuote rune
	// base is the basename prefix to match against. Empty when the typed
	// value ends in a separator (list-all-children case).
	base string
	// entry is the parent directory's cached ReadDir result.
	entry *dirCacheEntry
	// limit caps the candidate count after sort. Clamped to >= 1.
	limit int
	// hiddenFiles surfaces dotfiles even when base does not start with ".".
	hiddenFiles bool
	// parent is the absolute parent directory. Used to construct stat
	// paths when resolving symlink/unknown entry kinds.
	parent string
}

// generateCandidates produces the [pathCompletion] list for the active
// value. Iteration order is the ReadDir result (sorted lex). Survivors are
// re-sorted dirs-first, case-fold alphabetic, then truncated to limit.
//
// Symlink target kinds are resolved via [os.Stat] for entries that survive
// the prefix and hidden filters; [statBudget] bounds the worst-case stat
// count per pass.
func generateCandidates(req candidateRequest) []pathCompletion {
	if req.entry.err != nil {
		return nil
	}
	limit := max(req.limit, 1)

	type prelim struct {
		name      string
		isDir     bool
		isSymlink bool // original DirEntry kind was kindSymlink (description hint)
	}

	matchNames, needle := req.entry.matchKey(req.base)
	statsRemaining := statBudget(limit)
	survivors := make([]prelim, 0, min(limit, len(req.entry.names)))

	for i, candidateKey := range matchNames {
		if !strings.HasPrefix(candidateKey, needle) {
			continue
		}
		// Display always uses the raw filesystem name regardless of
		// case-insensitive matching — case is preserved in the UI.
		name := req.entry.names[i]
		if !req.hiddenFiles && strings.HasPrefix(name, ".") && !strings.HasPrefix(req.base, ".") {
			continue
		}

		k := req.entry.kinds[i]
		wasSymlink := k == kindSymlink
		if k == kindSymlink || k == kindUnknown {
			if statsRemaining <= 0 {
				continue // budget exhausted; drop
			}
			statsRemaining--
			info, err := os.Stat(filepath.Join(req.parent, name))
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

		if !includeKind(req.kind, k) {
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
		out[i] = buildCompletion(s.name, s.isDir, s.isSymlink, req.tokenPrefix, req.userPrefix, req.openingQuote)
	}
	return out
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

	a, ok := activeFileArgument(m.input.Value(), m.Commands)
	if !ok {
		return
	}

	if m.pathCache == nil {
		m.pathCache = newDirCache()
	}

	cwd, _ := os.Getwd()
	home, _ := os.UserHomeDir()
	typed := m.input.Value()[a.valueStart:a.valueEnd]
	expandTilde := a.openingQuote != '\''
	fullClean, parent, base, userPrefix := resolvePath(typed, cwd, home, expandTilde)

	entries := m.pathCache.read(parent)

	limit := max(m.FilesystemCompletionLimit, 1)

	m.pathState = pathState{
		active:     true,
		kind:       a.kind,
		argName:    a.name,
		valueStart: a.valueStart,
		valueEnd:   a.valueEnd,
		base:       base,
		validity:   classify(a.kind, fullClean, base, entries),
		candidates: generateCandidates(candidateRequest{
			kind:         a.kind,
			tokenPrefix:  a.tokenPrefix,
			userPrefix:   userPrefix,
			openingQuote: a.openingQuote,
			base:         base,
			entry:        entries,
			limit:        limit,
			hiddenFiles:  m.HiddenFiles,
			parent:       parent,
		}),
	}
}
