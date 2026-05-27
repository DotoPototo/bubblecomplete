package bubblecomplete

import (
	"errors"
	"os"
	"slices"
	"strconv"
	"strings"
)

func (m *Model) validateInput() error {
	if m.input.Value() == "" {
		return nil
	}

	return validateCommandInput(m.input.Value(), m.Commands)
}

func validateCommandInput(input string, commands []*Command) error {
	parts := splitInput(input)
	if len(parts) == 0 {
		return errEmptyCommand()
	}

	var parentCmd *Command
	var globalFlags []*Flag
	currentCommands := commands
	positionalIndex := 0
	isCommand := true

	for i := 0; i < len(parts); i++ {
		part := parts[i]

		if isCommand {
			cmd, err := findCommand(currentCommands, part)
			if err != nil {
				if parentCmd == nil {
					return errInvalidCommand(part)
				}
				// If no subcommand is found, stop looking for commands
				isCommand = false
				i-- // Reprocess the current part as a flag or positional argument
				continue
			}
			parentCmd = cmd
			currentCommands = cmd.SubCommands
			isCommand = len(cmd.SubCommands) > 0
			for _, flag := range cmd.Flags {
				if flag.Persistent {
					globalFlags = append(globalFlags, flag)
				}
			}
			continue
		}

		if strings.HasPrefix(part, "--") {
			if err := validateFlag(part, parts, &i, parentCmd, globalFlags); err != nil {
				return err
			}
			continue
		}

		if strings.HasPrefix(part, "-") {
			// -x=VALUE syntax bypasses combined-short-flag parsing because
			// validateShortFlags treats every char after the dash as its own
			// flag name and has no notion of an inline value.
			if strings.Contains(part, "=") {
				if err := validateFlag(part, parts, &i, parentCmd, globalFlags); err != nil {
					return err
				}
				continue
			}

			err := validateShortFlags(part, parts, &i, parentCmd, globalFlags)
			if err != nil {
				// Only fall back to long/PsFlag parsing when the short-flag
				// parser failed because the char wasn't a known short flag.
				// Other errors (malformed token, misused position, missing
				// value) come from a successful short-flag identification
				// and must surface to the user.
				var ve *ValidationError
				if errors.As(err, &ve) && ve.Kind == UnknownFlag {
					if err := validateFlag(part, parts, &i, parentCmd, globalFlags); err != nil {
						return err
					}
				} else {
					return err
				}
			}
			continue
		}

		if positionalIndex < len(parentCmd.PositionalArguments) {
			err := validatePositionalArgument(part, &positionalIndex, parentCmd)
			if err != nil {
				return err
			}
			continue
		}

		return errUnexpectedArgument(part)
	}

	// Check if all required positional arguments are present
	expectedPositionalArgs := 0
	for _, cmd := range parentCmd.PositionalArguments {
		if cmd.Required {
			expectedPositionalArgs++
		}
	}
	if positionalIndex < expectedPositionalArgs {
		return errMissingPositional(parentCmd.PositionalArguments[positionalIndex].Name)
	}

	return nil
}

func validateFlag(part string, parts []string, i *int, parentCmd *Command, globalFlags []*Flag) error {
	argName := part
	argValue := ""

	if strings.Contains(part, "=") {
		argParts := strings.SplitN(part, "=", 2)
		argName = argParts[0]
		argValue = argParts[1]
	}

	if parentCmd == nil {
		return errInvalidFlag(part)
	}

	allFlags := slices.Concat(parentCmd.Flags, globalFlags)
	arg, err := findFlag(allFlags, argName)
	if err != nil {
		return errFlagNotFound(argName)
	}

	if arg.getType() != BoolArgument && argValue == "" {
		if *i == len(parts)-1 || !looksLikeFlagValue(arg, parts[*i+1]) {
			return errMissingFlagValue(argName)
		}
		argValue = parts[*i+1]
		*i++
	}

	return validateArgumentValue(arg, argValue)
}

// looksLikeFlagValue reports whether next can serve as a value for arg. Tokens
// that don't start with "-" are always accepted. Tokens that do start with "-"
// are accepted only when they parse as a number for IntArgument / FloatArgument
// flags (e.g., --depth -1, --threshold -0.5). String values with a leading
// dash must be quoted or supplied via --flag=value.
func looksLikeFlagValue(arg Argument, next string) bool {
	if !strings.HasPrefix(next, "-") {
		return true
	}
	switch arg.getType() {
	case IntArgument:
		_, err := strconv.Atoi(next)
		return err == nil
	case FloatArgument:
		_, err := strconv.ParseFloat(next, 64)
		return err == nil
	}
	return false
}

func validateShortFlags(part string, parts []string, i *int, parentCmd *Command, globalFlags []*Flag) error {
	combinedFlags := part[1:]

	if len(combinedFlags) == 0 {
		return errInvalidArgumentToken(part)
	}

	for j := 0; j < len(combinedFlags); j++ {
		argName := "-" + string(combinedFlags[j])
		argValue := ""

		if parentCmd == nil {
			return errInvalidArgumentToken(part)
		}

		allFlags := slices.Concat(parentCmd.Flags, globalFlags)
		arg, err := findFlag(allFlags, argName)
		if err != nil {
			return errFlagNotFound(argName)
		}

		if arg.getType() != BoolArgument {
			if j == len(combinedFlags)-1 {
				if *i == len(parts)-1 || !looksLikeFlagValue(arg, parts[*i+1]) {
					return errMissingFlagValue(argName)
				}
				argValue = parts[*i+1]
				*i++
			} else {
				return errCombinedFlagNotLast(argName)
			}
		}

		err = validateArgumentValue(arg, argValue)
		if err != nil {
			return err
		}
	}
	return nil
}

func validatePositionalArgument(part string, positionalIndex *int, parentCmd *Command) error {
	positionalArg := parentCmd.PositionalArguments[*positionalIndex]
	if positionalArg == nil {
		return errUnexpectedArgument(part)
	}
	if !positionalArg.Required && part == "" {
		return nil
	}
	err := validateArgumentValue(positionalArg, part)
	if err != nil {
		return err
	}
	*positionalIndex++
	return nil
}

func findCommand(commands []*Command, name string) (*Command, error) {
	for _, cmd := range commands {
		if cmd.Command == name {
			return cmd, nil
		}
	}
	return nil, errors.New("command not found")
}

func findFlag(arguments []*Flag, name string) (*Flag, error) {
	for _, arg := range arguments {
		if arg.ShortFlag == name || arg.LongFlag == name || arg.PsFlag == name {
			return arg, nil
		}
	}
	return nil, errors.New("argument not found")
}

func validateArgumentValue(arg Argument, value string) error {
	switch arg.getType() {
	case StringArgument:
		return validateStringArgument(arg, value)
	case IntArgument:
		return validateIntArgument(arg, value)
	case FloatArgument:
		return validateFloatArgument(arg, value)
	case BoolArgument:
		// No validation needed for boolean, presence is enough
		return nil
	case FileArgument:
		return validateFileArgument(arg, value)
	case DirArgument:
		return validateDirArgument(arg, value)
	case FileDirArgument:
		return validateFileDirArgument(arg, value)
	default:
		return errUnknownType(string(arg.getType()))
	}
}

func validateStringArgument(arg Argument, value string) error {
	if err := checkEmptyString(arg, value); err != nil {
		return err
	}
	if err := checkUnclosedQuote(arg, value, "\""); err != nil {
		return err
	}
	if err := checkUnclosedQuote(arg, value, "'"); err != nil {
		return err
	}
	return nil
}

func checkEmptyString(arg Argument, value string) error {
	if value != "" {
		return nil
	}
	// The same check fires for both positional arguments and flag values, so
	// route the error to the right kind based on the caller's argument type.
	if _, ok := arg.(*Flag); ok {
		return errMissingFlagValue(arg.getName())
	}
	return errMissingPositionalValue(arg.getName())
}

func checkUnclosedQuote(arg Argument, value, quote string) error {
	if len(value) == 1 && value == quote {
		return errUnclosedQuote()
	}
	if len(value) > 1 && strings.HasPrefix(value, quote) && !strings.HasSuffix(value, quote) {
		return errUnclosedQuoteForArg(arg.getName())
	}
	return nil
}

func validateIntArgument(arg Argument, value string) error {
	if _, err := strconv.Atoi(value); err != nil {
		return errInvalidInt(arg.getName(), err)
	}
	return nil
}

func validateFloatArgument(arg Argument, value string) error {
	if _, err := strconv.ParseFloat(value, 64); err != nil {
		return errInvalidFloat(arg.getName(), err)
	}
	return nil
}

func validateFileArgument(arg Argument, value string) error {
	return validatePath(arg, value, true, false)
}

func validateDirArgument(arg Argument, value string) error {
	return validatePath(arg, value, false, true)
}

func validateFileDirArgument(arg Argument, value string) error {
	return validatePath(arg, value, true, true)
}

func validatePath(arg Argument, value string, wantFile, wantDir bool) error {
	value = removeQuotes(value)

	var pathType string
	switch {
	case wantFile && wantDir:
		pathType = "file or directory"
	case wantFile:
		pathType = "file"
	default:
		pathType = "directory"
	}

	info, err := os.Stat(value)
	if err != nil {
		if os.IsNotExist(err) {
			return errPathNotExist(pathType, arg.getName())
		}
		return errPathAccess(pathType, arg.getName(), err)
	}
	if !wantDir && info.IsDir() {
		return errPathIsDir(arg.getName())
	}
	if !wantFile && !info.IsDir() {
		return errPathIsFile(arg.getName())
	}
	return nil
}

func removeQuotes(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			s = s[1 : len(s)-1]
		}
	}
	return s
}
