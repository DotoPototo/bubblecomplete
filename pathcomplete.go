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
// classification and candidate generation. Refreshed on every input change
// AND on every Tab cycle/revert (keyTab), so offsets and validity always
// reflect the value currently in [Model.input].
type pathState struct {
	active bool
	// kind is one of FileArgument, DirArgument, FileDirArgument.
	kind ArgumentType
	// argName verifies a PathNotFound validation error belongs to this
	// argument before suppressing whole-input red.
	argName string
	// valueStart and valueEnd are byte offsets in m.input.Value() of the
	// unquoted value range — used by the render overlay.
	valueStart int
	valueEnd   int
	// base is the basename prefix from [resolvePath], used as the
	// matchPrefix when path candidates replace the completion list.
	base     string
	validity pathValidity
	// candidates is rendered in place of the normal completion rows when
	// active and non-empty.
	candidates []pathCompletion
	// droppedSorted counts verified matches truncated past
	// [Model.FilesystemCompletionLimit] — real "+ N more" entries the user
	// could reach by narrowing the prefix.
	droppedSorted int
	// unresolvedEntries counts symlink/unknown-kind entries the stat budget
	// couldn't verify — surfaced with weaker wording than droppedSorted.
	// The footer says "symlinks" because kindUnknown is vanishingly rare
	// (Go's stdlib resolves DT_UNKNOWN via lstat at readdir time).
	unresolvedEntries int
}

// pathCompletion is a [completion] backed by a filesystem entry. insertion
// carries the FULL active-token replacement so keyTab's pretext +
// getAutocomplete concatenation works unchanged for positional, flag, and
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

// statBudget caps the [os.Stat] calls generateCandidates makes per pass to
// resolve symlink/unknown target kinds, tying worst-case CPU per keystroke
// to the configured candidate limit (with a floor for tiny limits). In
// symlink-heavy dirs the budget can drop a late symlink-to-dir that sorting
// would have promoted — an accepted tradeoff for bounded latency.
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
	// name is the argument's display name, matched against a PathNotFound
	// error's Argument by isPartialPathMidType.
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
//   - input is empty or ends in a space outside quotes (no active value
//     token — a space inside an open quote is part of the value)
//   - no command word has been entered yet
//   - the active position is a command word, a flag name without value, or a
//     value for a non-file argument type
//   - the active value is empty (e.g. "--path=" with nothing after)
func activeFileArgument(input string, commands []*Command) (activeArg, bool) {
	tokens := tokenize(input)
	if len(tokens) == 0 {
		return activeArg{}, false
	}
	// A trailing space ends the active token only when it falls outside a
	// token; tokenize keeps quoted spaces inside the final token, so its End
	// reaches len(input) while the user is still typing a quoted value.
	if tokens[len(tokens)-1].End < len(input) {
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

	// Case 1: equals-form flag value.
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

// candidateRequest groups the inputs to [generateCandidates].
type candidateRequest struct {
	// kind drives the inclusion filter.
	kind ArgumentType
	// tokenPrefix is the leading bytes of the active token before the value
	// (flag prefix and/or opening quote), preserved in insertions.
	tokenPrefix string
	// userPrefix is the typed value up to the last separator before base
	// ("~/", "./", "/abs/path/", ""), preserved verbatim in insertions so
	// completion keeps the user's style.
	userPrefix string
	// openingQuote is 0 if no quote, else '"' or '\''.
	openingQuote rune
	// base is the basename prefix to match. Empty when the typed value ends
	// in a separator (list-all-children case).
	base string
	// entry is the parent directory's cached ReadDir result.
	entry *dirCacheEntry
	// limit caps the candidate count after sort. Clamped to >= 1.
	limit int
	// hiddenFiles surfaces dotfiles even when base does not start with ".".
	hiddenFiles bool
	// parent is the absolute parent directory, used for symlink stat paths.
	parent string
}

// generateCandidates produces the [pathCompletion] list for the active
// value: filter on prefix/hidden/kind, resolve symlink targets within
// [statBudget], sort dirs-first case-fold alphabetic, truncate to limit.
// The two drop counts stay separate so the renderer can phrase the footer
// honestly: droppedSorted are verified "+ N more" matches reachable by
// narrowing; unresolvedEntries are stat-budget-skipped entries that might
// not be candidates at all.
func generateCandidates(req candidateRequest) (candidates []pathCompletion, droppedSorted, unresolvedEntries int) {
	if req.entry.err != nil {
		return nil, 0, 0
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
				unresolvedEntries++
				continue // budget exhausted; drop (kind unverified)
			}
			statsRemaining--
			info, err := os.Stat(filepath.Join(req.parent, name))
			if err != nil {
				continue // broken symlink or stat failure — verified failure, not a drop
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

	sort.SliceStable(survivors, func(i, j int) bool {
		if survivors[i].isDir != survivors[j].isDir {
			return survivors[i].isDir
		}
		return strings.ToLower(survivors[i].name) < strings.ToLower(survivors[j].name)
	})

	if len(survivors) > limit {
		droppedSorted = len(survivors) - limit
		survivors = survivors[:limit]
	}

	out := make([]pathCompletion, len(survivors))
	for i, s := range survivors {
		out[i] = buildCompletion(s.name, s.isDir, s.isSymlink, req.tokenPrefix, req.userPrefix, req.openingQuote)
	}
	return out, droppedSorted, unresolvedEntries
}

// includeKind admits dirs for every kind (drill-down) and files unless
// DirArgument. Specials (devices, sockets, pipes) never appear as
// candidates, though a typed path to one still validates per kind.
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

// buildCompletion assembles the display name and full active-token
// replacement. Names containing a space are auto-quoted with '"' when the
// user hasn't opened a quote; an existing opener is kept and its closer
// restored. Files append a trailing space (bash-style "token done") —
// directories deliberately omit it so the next Tab drills in. Known
// limitation: filenames containing literal quote characters are inserted
// verbatim and won't re-parse as a single token.
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
// be suppressed in favour of the path-range overlay. The argument-name
// check matters for commands with multiple path positionals: in
// `cp /missing/foo /tmp/par` the red comes from the FIRST (committed) path,
// and suppression must only fire when the partial path IS the cause.
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
// Must run before getCompletions in Update's input-changed branch so
// getCompletions can wholesale-replace its result list with
// pathState.candidates. Resets to the zero value when the feature is
// disabled or no file/dir value is active.
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

	candidates, droppedSorted, unresolvedEntries := generateCandidates(candidateRequest{
		kind:         a.kind,
		tokenPrefix:  a.tokenPrefix,
		userPrefix:   userPrefix,
		openingQuote: a.openingQuote,
		base:         base,
		entry:        entries,
		limit:        limit,
		hiddenFiles:  m.HiddenFiles,
		parent:       parent,
	})

	m.pathState = pathState{
		active:            true,
		kind:              a.kind,
		argName:           a.name,
		valueStart:        a.valueStart,
		valueEnd:          a.valueEnd,
		base:              base,
		validity:          classify(a.kind, fullClean, base, entries),
		candidates:        candidates,
		droppedSorted:     droppedSorted,
		unresolvedEntries: unresolvedEntries,
	}
}
