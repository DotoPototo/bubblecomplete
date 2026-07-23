package bubblecomplete

import (
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

func (m Model) getCompletions() ([]completion, string) {
	if m.input.Value() == "" && !m.showAll {
		return []completion{}, ""
	}

	// Modal path-completion context: when editing a file/dir argument value
	// and we have candidates, those replace the entire completion list.
	// Command and flag rows are intentionally hidden in this mode — the
	// user is doing filesystem navigation, not flag exploration. To
	// re-surface them, the user moves the cursor out of the value position.
	if m.pathState.active && len(m.pathState.candidates) > 0 {
		out := make([]completion, len(m.pathState.candidates))
		for i := range m.pathState.candidates {
			out[i] = m.pathState.candidates[i]
		}
		// generateCandidates already produces a deduped, dirs-first
		// case-fold-alphabetic list; skip sort/unique.
		return out, m.pathState.base
	}

	var allCompletions []completion
	var matchPrefix string

	if strings.TrimSpace(m.input.Value()) == "" && m.showAll {
		for _, c := range m.Commands {
			allCompletions = append(allCompletions, c)
		}
	} else {
		allCompletions, matchPrefix = getCompletions(m.input.Value(), m.Commands)
	}

	sortCompletions(&allCompletions)
	uniqueCompletions(&allCompletions)
	return allCompletions, matchPrefix
}

func sortCompletions(completions *[]completion) {
	slices.SortFunc(*completions, func(a, b completion) int {
		nameA := a.getName()
		nameB := b.getName()

		// Decode the first rune properly — indexing [0] would misclassify
		// names starting with a multi-byte rune (the lead byte casts to an
		// unrelated code point).
		firstA, _ := utf8.DecodeRuneInString(nameA)
		firstB, _ := utf8.DecodeRuneInString(nameB)
		isPunctA := unicode.IsPunct(firstA)
		isPunctB := unicode.IsPunct(firstB)

		if isPunctA && !isPunctB {
			return 1
		}
		if !isPunctA && isPunctB {
			return -1
		}
		if cmp := strings.Compare(strings.ToLower(nameA), strings.ToLower(nameB)); cmp != 0 {
			return cmp
		}
		return strings.Compare(nameA, nameB)
	})
}

func uniqueCompletions(completions *[]completion) {
	seen := make(map[string]struct{})
	list := []completion{}
	for _, entry := range *completions {
		if _, exists := seen[entry.getName()]; !exists {
			seen[entry.getName()] = struct{}{}
			list = append(list, entry)
		}
	}
	*completions = list
}

func getCompletions(input string, commands []*Command) ([]completion, string) {
	var completions []completion

	if strings.TrimSpace(input) == "" {
		return []completion{}, ""
	}

	parts := splitInput(input)
	if len(parts) == 0 {
		return []completion{}, ""
	}

	// Still typing the first command word.
	if len(parts) == 1 && !strings.HasSuffix(input, " ") {
		for _, c := range commands {
			if strings.HasPrefix(c.Command, parts[0]) {
				completions = append(completions, c)
			}
		}
		return completions, parts[0]
	}

	finalCommand, commandDepth, globalFlags := walkToFinalCommand(input, parts, commands)
	if finalCommand == nil {
		return []completion{}, ""
	}

	argParts := parts[commandDepth:]
	posArgs, flagArgs := splitPositionArgsAndFlags(argParts, finalCommand, globalFlags)

	matchPrefix := ""
	if !strings.HasSuffix(input, " ") && len(argParts) > 0 {
		matchPrefix = argParts[len(argParts)-1]
	}

	if len(finalCommand.SubCommands) > 0 {
		completions = handleSubCommandCompletions(finalCommand, parts, commandDepth, input, flagArgs, globalFlags)
		return completions, matchPrefix
	}

	if len(finalCommand.PositionalArguments) > 0 {
		completions = handlePositionalArgumentCompletions(finalCommand, posArgs, flagArgs, input, argParts, globalFlags)
		return completions, matchPrefix
	}

	flagCompletions, _ := getFlagCompletions(input, finalCommand, flagArgs, globalFlags)
	completions = append(completions, flagCompletions...)
	return completions, matchPrefix
}

func handleSubCommandCompletions(
	cmd *Command,
	parts []string,
	depth int,
	input string,
	flagArgs []string,
	globalFlags []*Flag,
) []completion {
	var completions []completion

	// Hide subcommands once parts have moved past the subcommand position
	// (extra tokens mean the user is typing flags / args for the parent).
	if len(parts) <= depth || (len(parts) == depth+1 && !strings.HasSuffix(input, " ")) {
		completions = append(completions, getSubCommandCompletions(input, cmd, parts)...)
	}

	flagCompletions, solo := getFlagCompletions(input, cmd, flagArgs, globalFlags)
	if solo {
		return flagCompletions
	}
	completions = append(completions, flagCompletions...)
	return completions
}

func handlePositionalArgumentCompletions(
	cmd *Command,
	posArgs, flagArgs []string,
	input string,
	argParts []string,
	globalFlags []*Flag,
) []completion {
	var completions []completion

	if len(posArgs) == 0 {
		flagCompletions, solo := getFlagCompletions(input, cmd, flagArgs, globalFlags)
		if solo {
			return flagCompletions
		}
		completions = append(completions, flagCompletions...)
	}

	// Handle positional argument completions. Both flags read off argParts'
	// last token, which is only safe when argParts is non-empty — short-
	// circuiting on len(argParts) > 0 keeps this panic-free for inputs that
	// end immediately after the command word (e.g. "git ").
	startedTyping := len(posArgs) > 0 || strings.HasSuffix(input, " ")
	posArgsLeft := len(posArgs) < len(cmd.PositionalArguments)
	posArgsFull := len(posArgs) == len(cmd.PositionalArguments)
	lastIsPositional := len(argParts) > 0 && !strings.HasPrefix(argParts[len(argParts)-1], "-")
	enteringPosArg := startedTyping && posArgsLeft
	enteringLastPosArg := posArgsFull && lastIsPositional
	if enteringPosArg || enteringLastPosArg {
		completions = append(completions, getPositionalArgumentCompletions(input, cmd, posArgs)...)
		return completions
	}

	return completions
}

func getSubCommandCompletions(input string, finalCommand *Command, parts []string) []completion {
	completions := []completion{}

	if !strings.HasSuffix(input, " ") {
		for _, command := range finalCommand.SubCommands {
			if strings.HasPrefix(command.Command, parts[len(parts)-1]) {
				if !inputContainsCompletedToken(input, command.Command) {
					completions = append(completions, command)
				}
			}
		}
		return completions
	}

	for _, command := range finalCommand.SubCommands {
		if !inputContainsCompletedToken(input, command.Command) {
			completions = append(completions, command)
		}
	}

	return completions
}

func getPositionalArgumentCompletions(input string, finalCommand *Command, posArgParts []string) []completion {
	completions := []completion{}

	if len(posArgParts) == 0 {
		return []completion{finalCommand.PositionalArguments[0]}
	}

	if yes, arg := isEnteringPosArgValue(input, finalCommand, posArgParts); yes {
		return []completion{arg}
	}

	if len(posArgParts) < len(finalCommand.PositionalArguments) {
		return []completion{finalCommand.PositionalArguments[len(posArgParts)]}
	}

	return completions
}

func isEnteringPosArgValue(input string, finalCommand *Command, posArgParts []string) (bool, *PositionalArgument) {
	if len(posArgParts) == 0 {
		return false, nil
	}

	lastArg := posArgParts[len(posArgParts)-1]
	positionalArgument := finalCommand.PositionalArguments[len(posArgParts)-1]

	// Unclosed quoted value → still entering.
	if strings.HasPrefix(lastArg, "\"") || strings.HasPrefix(lastArg, "'") {
		quote := lastArg[0:1]
		if !strings.HasSuffix(lastArg, quote) {
			return true, positionalArgument
		}
	}

	if !strings.HasSuffix(input, " ") {
		return true, positionalArgument
	}

	return false, nil
}

// getFlagCompletions returns flag completions for the current input. The
// second return is true when the result should replace the completion list
// entirely (e.g. when the user is mid-value for a known flag).
func getFlagCompletions(
	input string,
	finalCommand *Command,
	flagArgParts []string,
	globalFlags []*Flag,
) ([]completion, bool) {
	completions := []completion{}

	allFlags := slices.Concat(finalCommand.Flags, globalFlags)

	if len(flagArgParts) == 0 {
		for _, a := range allFlags {
			completions = append(completions, a)
		}
		return completions, false
	}

	if yes, flag := isEnteringFlagValue(input, finalCommand, flagArgParts, globalFlags); yes {
		return []completion{flag}, true
	}

	if yes, flag := needToEnterFlagValue(finalCommand, flagArgParts, globalFlags); yes {
		return []completion{flag}, true
	}

	if strings.HasSuffix(input, " ") {
		for _, flag := range allFlags {
			if !containsFlag(input, flag) {
				completions = append(completions, flag)
			}
		}
		return completions, false
	}

	finalPart := flagArgParts[len(flagArgParts)-1]
	completions = append(completions, filterFlagsByPrefix(input, finalPart, allFlags)...)
	return completions, false
}

// filterFlagsByPrefix returns flags whose form starts with prefix. Already-
// entered flags are filtered out unless the prefix exactly matches them (so
// the user can finish typing a flag they've already entered).
func filterFlagsByPrefix(input, prefix string, allFlags []*Flag) []completion {
	var out []completion
	for _, flag := range allFlags {
		if flag.PsFlag != "" && strings.HasPrefix(flag.PsFlag, prefix) {
			// PsFlag-style: keep if not yet entered, or if the user is
			// finishing the exact flag.
			if !containsFlag(input, flag) || prefix == flag.PsFlag {
				out = append(out, flag)
			}
			continue
		}
		if !strings.HasPrefix(flag.ShortFlag, prefix) && !strings.HasPrefix(flag.LongFlag, prefix) {
			continue
		}
		// Combined short flags: compare against the LAST char only so the
		// completion list reflects what would actually be added.
		flagToCompare := prefix
		if !strings.HasPrefix(prefix, "--") && len(prefix) > 2 {
			flagToCompare = "-" + prefix[len(prefix)-1:]
		}
		if !containsFlag(input, flag) || flagToCompare == flag.ShortFlag || flagToCompare == flag.LongFlag {
			out = append(out, flag)
		}
	}
	return out
}

func isEnteringFlagValue(
	input string,
	finalCommand *Command,
	flagArgParts []string,
	globalFlags []*Flag,
) (bool, *Flag) {
	if len(flagArgParts) == 0 {
		return false, nil
	}

	lastArg := flagArgParts[len(flagArgParts)-1]
	allFlags := slices.Concat(finalCommand.Flags, globalFlags)

	// If the input ends with a space AND the last token is closed, the user
	// has committed past the value — they're no longer entering it. This
	// applies uniformly to all forms: `-m 'msg' `, `-m "msg" `, `-m hello `,
	// `--message=value `, etc. The `Closed` field on the tokenizer's last
	// token correctly distinguishes a fully-closed quoted value from an
	// unclosed one whose body happens to end in a space (e.g. `-m "hi `,
	// which is still entering the -m value because the quote never closed).
	if strings.HasSuffix(input, " ") {
		tokens := tokenize(input)
		if len(tokens) > 0 && tokens[len(tokens)-1].Closed {
			return false, nil
		}
	}

	// Space-separated flag value (e.g. `--name foo`).
	if len(flagArgParts) >= 2 {
		lastFlag := flagArgParts[len(flagArgParts)-2]
		lastValue := lastArg

		if strings.HasPrefix(lastFlag, "-") && !inputContainsUnquotedTokenBeforeLast(input, lastValue) {
			flagValueToCompare := lastFlag
			for _, flag := range allFlags {
				// Combined short flag: the value belongs to the last char.
				if flag.PsFlag == "" && !strings.HasPrefix(lastFlag, "--") && len(lastFlag) > 2 {
					flagValueToCompare = "-" + lastFlag[len(lastFlag)-1:]
				}
				// looksLikeFlagValue lets numeric flags accept negative numbers
				// (e.g., --depth -1) while still rejecting leading-dash tokens
				// for non-numeric types.
				if containsFlag(flagValueToCompare, flag) && flag.Type != BoolArgument && looksLikeFlagValue(flag, lastValue) {
					return true, flag
				}
			}
		}
	}

	// Equals-form flag value (e.g. `--name=foo`).
	if strings.Contains(lastArg, "=") {
		// Skip when the token is a fully-closed quoted value already followed
		// by a space — that means we've moved past it, not into it.
		if !stringEndsInQuoteWithoutEquals(lastArg) || !strings.HasSuffix(input, " ") {
			for _, flag := range allFlags {
				if flag.PsFlag == "" && !strings.HasPrefix(lastArg, "--") {
					continue
				}
				if containsFlag(lastArg, flag) && flag.Type != BoolArgument {
					return true, flag
				}
			}
		}
	}

	return false, nil
}

func needToEnterFlagValue(finalCommand *Command, flagArgParts []string, globalFlags []*Flag) (bool, *Flag) {
	lastArgument := flagArgParts[len(flagArgParts)-1]
	allFlags := slices.Concat(finalCommand.Flags, globalFlags)

	for _, flag := range allFlags {
		// Combined short flag: the value belongs to the last char.
		flagToCompare := lastArgument
		if flag.PsFlag == "" && !strings.HasPrefix(lastArgument, "--") && len(lastArgument) > 2 {
			flagToCompare = "-" + lastArgument[len(lastArgument)-1:]
		}

		// Only treat the last token as a flag that's waiting on a value if
		// it actually references this flag AND hasn't already taken a value
		// via long-form or PsFlag "=value" syntax.
		hasFlag := containsFlag(flagToCompare, flag)
		hasLongEquals := strings.Contains(lastArgument, flag.LongFlag+"=")
		hasPsEquals := strings.Contains(lastArgument, flag.PsFlag+"=")
		if hasFlag && !hasLongEquals && !hasPsEquals && flag.Type != BoolArgument {
			return true, flag
		}
	}

	return false, nil
}

func stringEndsInQuoteWithoutEquals(s string) bool {
	if strings.HasSuffix(s, "\"") && !strings.HasSuffix(s, "=\"") {
		return true
	}
	if strings.HasSuffix(s, "'") && !strings.HasSuffix(s, "='") {
		return true
	}
	return false
}

func splitPositionArgsAndFlags(argParts []string, command *Command, globalFlags []*Flag) ([]string, []string) {
	if len(argParts) == 0 {
		return []string{}, []string{}
	}

	effectiveFlags := slices.Concat(command.Flags, globalFlags)

	if len(effectiveFlags) == 0 {
		return argParts, []string{}
	}
	if len(command.PositionalArguments) == 0 {
		return []string{}, argParts
	}

	var positionalArgs []string
	var flags []string
	for i := 0; i < len(argParts); i++ {
		if strings.HasPrefix(argParts[i], "-") {
			matched := findMatchingFlag(argParts[i], effectiveFlags)
			if matched != nil {
				flags = append(flags, argParts[i])
				// Bool flags don't consume the next token.
				if matched.Type != BoolArgument && i+1 < len(argParts) {
					flags = append(flags, argParts[i+1])
					i++
				}
			} else {
				// Unknown flags are still recorded so completion can filter them out.
				flags = append(flags, argParts[i])
			}
		} else {
			positionalArgs = append(positionalArgs, argParts[i])
		}
	}

	return positionalArgs, flags
}

// findMatchingFlag returns the effective flag whose form matches the given
// argument token, or nil if none does. Single-pass classification avoids the
// duplicate-append bug the old inner loop had.
//
// For combined short flags like "-fm", the LAST character determines the
// matching flag — validation enforces that only the last char in a combined
// group may be non-bool (and therefore value-taking), so attributing the
// token to that flag is consistent with how the command would actually parse.
//
// PsFlag and LongFlag matches (exact-token OR equals-form, since
// containsLongFlag / containsPowerShellFlag also recognise "--flag=value"
// and "-Path=value") are checked BEFORE the combined-short-flag heuristic.
// A multi-letter PsFlag like "-Path" is indistinguishable token-shape-wise
// from a combined short flag "-Path" (the body is all ASCII letters,
// length > 1), so the combined-short-flag branch must not preempt a real
// PsFlag match.
func findMatchingFlag(arg string, effectiveFlags []*Flag) *Flag {
	for _, f := range effectiveFlags {
		if f.PsFlag != "" && containsPowerShellFlag(arg, f.PsFlag) {
			return f
		}
		if f.LongFlag != "" && containsLongFlag(arg, f.LongFlag) {
			return f
		}
	}
	if isCombinedShortFlag(arg) {
		lastChar := "-" + arg[len(arg)-1:]
		for _, f := range effectiveFlags {
			if containsFlag(lastChar, f) {
				return f
			}
		}
		return nil
	}
	for _, f := range effectiveFlags {
		if containsFlag(arg, f) {
			return f
		}
	}
	return nil
}

// isCombinedShortFlag reports whether arg is a short-flag token whose body
// is more than one ASCII letter (e.g., "-xyz", but not "-x", "--foo", or "-1").
func isCombinedShortFlag(arg string) bool {
	body, ok := shortFlagBody(arg)
	return ok && len(body) > 1
}

// walkToFinalCommand walks the parsed input parts down the command tree and
// returns the deepest matched command, how many parts were consumed by command
// words, and the persistent flags accumulated from ancestor commands. Returns
// finalCmd == nil when no command word matched.
//
// Persistent flags propagate to every subcommand. Downstream completion paths
// dedupe via [containsFlag], so we must not pre-filter here — value-detection
// needs the persistent flag in scope even when the user is actively entering
// it.
func walkToFinalCommand(input string, parts []string, commands []*Command) (finalCmd *Command, commandDepth int, globalFlags []*Flag) {
	for _, enteredInput := range parts {
		for _, c := range commands {
			if c.Command == enteredInput && inputContainsCompletedToken(input, c.Command) {
				finalCmd = c
				commands = c.SubCommands
				commandDepth++
				for _, flag := range c.Flags {
					if flag.Persistent {
						globalFlags = append(globalFlags, flag)
					}
				}
				break
			}
		}
	}
	return finalCmd, commandDepth, globalFlags
}

// inputContainsCompletedToken returns true if input contains an unquoted token
// equal to s that has been moved past — there is either another token after it
// or the input ends with whitespace. Used to ask "has this command word been
// committed?" while ignoring matches inside quoted arguments.
func inputContainsCompletedToken(input, s string) bool {
	tokens := tokenize(input)
	for i, tok := range tokens {
		if tok.Quoted || tok.Unquoted != s {
			continue
		}
		if i < len(tokens)-1 || strings.HasSuffix(input, " ") {
			return true
		}
	}
	return false
}

// inputContainsUnquotedTokenBeforeLast returns true if input contains an
// unquoted token equal to s at any position before the final token. Used to
// distinguish a value being typed (the final token) from an earlier
// occurrence of the same value elsewhere in the input.
func inputContainsUnquotedTokenBeforeLast(input, s string) bool {
	tokens := tokenize(input)
	for i := range len(tokens) - 1 {
		if !tokens[i].Quoted && tokens[i].Unquoted == s {
			return true
		}
	}
	return false
}

// containsFlag returns true if the input contains an unquoted reference to the
// given flag. Token-aware: quoted segments do not contribute matches and
// end-of-input flags are detected correctly.
func containsFlag(command string, flag *Flag) bool {
	if flag.PsFlag != "" && containsPowerShellFlag(command, flag.PsFlag) {
		return true
	}
	if flag.ShortFlag != "" && containsShortFlag(command, flag.ShortFlag) {
		return true
	}
	if flag.LongFlag != "" && containsLongFlag(command, flag.LongFlag) {
		return true
	}
	return false
}

// containsShortFlag returns true if any unquoted short-flag-shaped token in
// command contains the single-character body of flag. Combined short flags
// like "-xyz" match each of their characters; tokens whose body is not purely
// ASCII letters are ignored (matches the historical detection rule).
func containsShortFlag(command string, flag string) bool {
	if len(flag) > 0 && flag[0] == '-' {
		flag = flag[1:]
	}
	if len(flag) != 1 {
		return false
	}
	flagChar := flag[0]

	for _, tok := range tokenize(command) {
		if tok.Quoted {
			continue
		}
		body, ok := shortFlagBody(tok.Unquoted)
		if !ok {
			continue
		}
		for i := range len(body) {
			if body[i] == flagChar {
				return true
			}
		}
	}
	return false
}

// containsLongFlag returns true if command contains an unquoted token that
// equals flag, or flag immediately followed by "=" (long-flag value form).
func containsLongFlag(command string, flag string) bool {
	for _, tok := range tokenize(command) {
		if tok.Quoted {
			continue
		}
		text := tok.Unquoted
		if text == flag {
			return true
		}
		if before, _, found := strings.Cut(text, "="); found && before == flag {
			return true
		}
	}
	return false
}

// containsPowerShellFlag reports whether command contains a token referencing
// the given PsFlag. Flag.Validate guarantees PsFlag bodies are at least two
// runes, so detection reuses the long-flag exact/equals-form matcher.
func containsPowerShellFlag(command string, flag string) bool {
	return containsLongFlag(command, flag)
}

// shortFlagBody returns the body of a short-flag-shaped token (e.g., "xyz" for
// "-xyz") if the token is a candidate for short-flag detection. Tokens that
// start with "--" or have a non-alpha body are rejected.
func shortFlagBody(text string) (string, bool) {
	if !strings.HasPrefix(text, "-") || strings.HasPrefix(text, "--") || len(text) < 2 {
		return "", false
	}
	body := text[1:]
	for i := range len(body) {
		if !isASCIILetter(body[i]) {
			return "", false
		}
	}
	return body, true
}
