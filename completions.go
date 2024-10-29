package bubblecomplete

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode"
)

func (m Model) getCompletions() []Completion {
	if m.input.Value() == "" && !m.showAll {
		return []Completion{}
	}
	var allCompletions []Completion

	if strings.TrimSpace(m.input.Value()) == "" && m.showAll {
		for _, c := range m.Commands {
			allCompletions = append(allCompletions, c)
		}
	} else {
		allCompletions = getCompletions(m.input.Value(), m.Commands)
	}

	sortCompletions(&allCompletions)
	uniqueCompletions(&allCompletions)
	return allCompletions
}

func sortCompletions(completions *[]Completion) {
	allCompletions := *completions
	for i := 0; i < len(allCompletions); i++ {
		for j := i + 1; j < len(allCompletions); j++ {
			nameI := allCompletions[i].getName()
			nameJ := allCompletions[j].getName()

			// Check if the names start with punctuation
			isPunctI := unicode.IsPunct(rune(nameI[0]))
			isPunctJ := unicode.IsPunct(rune(nameJ[0]))

			// If the first name starts with punctuation and the second doesn't, swap them
			if isPunctI && !isPunctJ {
				allCompletions[i], allCompletions[j] = allCompletions[j], allCompletions[i]
			} else if !isPunctI && isPunctJ {
				// Keep the order as is
				continue
			} else if strings.ToLower(nameI) > strings.ToLower(nameJ) {
				allCompletions[i], allCompletions[j] = allCompletions[j], allCompletions[i]
			}
		}
	}
}

func uniqueCompletions(completions *[]Completion) {
	keys := make(map[string]bool)
	list := []Completion{}
	for _, entry := range *completions {
		if _, value := keys[entry.getName()]; !value {
			keys[entry.getName()] = true
			list = append(list, entry)
		}
	}
	*completions = list
}

func getCompletions(input string, commands []*Command) []Completion {
	var completions []Completion
	var globalFlags []*Flag

	// If the input is empty, return nothing
	if strings.TrimSpace(input) == "" {
		return []Completion{}
	}

	// Split the input into parts so we can handle each part separately
	parts := splitInput(input)
	if len(parts) == 0 {
		return []Completion{}
	}

	// If there is only one part and the input doesn't end with a space, we're still typing the first command
	if len(parts) == 1 && !strings.HasSuffix(input, " ") {
		// Show all commands that start with the input
		for _, c := range commands {
			if strings.HasPrefix(c.Command, parts[0]) {
				completions = append(completions, c)
			}
		}
		return completions
	}

	// Otherwise we have at least one command entered so find the final valid command entered
	var finalCommand *Command
	commandDepth := 0
	for _, enteredInput := range parts {
		for _, c := range commands {
			// If the command is found in the available commands and we've finished typing then use it
			if c.Command == enteredInput && strings.Contains(input, fmt.Sprintf("%s ", c.Command)) {
				finalCommand = c
				commands = c.SubCommands
				commandDepth++
				// If the command has global flags, add them to the global completions
				for _, flag := range c.Flags {
					if flag.Persistent && !containsFlag(input, flag) && strings.Contains(input, fmt.Sprintf("%s ", c.Command)) {
						globalFlags = append(globalFlags, flag)
					}
				}
				break
			}
		}
	}

	// If we haven't found any command, it must be invalid input so return nothing
	if finalCommand == nil {
		return []Completion{}
	}

	// From here it's if - return statements

	argParts := parts[commandDepth:]
	posArgs, flagArgs := splitPositionArgsAndFlags(argParts, finalCommand)

	// If the final command has subcommands
	if len(finalCommand.SubCommands) > 0 {
		completions = handleSubCommandCompletions(finalCommand, parts, commandDepth, input, flagArgs, globalFlags)
		return completions
	}

	// If the final command has positional arguments
	if len(finalCommand.PositionalArguments) > 0 {
		completions = handlePositionalArgumentCompletions(finalCommand, posArgs, flagArgs, input, argParts, globalFlags)
		return completions
	}

	// Otherwise show only the flags
	flagCompletions, _ := getFlagCompletions(input, finalCommand, flagArgs, globalFlags)
	completions = append(completions, flagCompletions...)
	return completions
}

func handleSubCommandCompletions(
	cmd *Command,
	parts []string,
	depth int,
	input string,
	flagArgs []string,
	globalFlags []*Flag,
) []Completion {
	var completions []Completion

	// Show subcommands unless there's more parts than expected (i.e. invalid input or flags for parent command)
	if len(parts) <= depth || (len(parts) == depth+1 && !strings.HasSuffix(input, " ")) {
		completions = append(completions, getSubCommandCompletions(input, cmd, parts)...)
	}

	// Append any flag completions
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
) []Completion {
	var completions []Completion

	// Show the flag arguments if there are no positional arguments entered
	if len(posArgs) == 0 {
		flagCompletions, solo := getFlagCompletions(input, cmd, flagArgs, globalFlags)
		if solo {
			return flagCompletions
		}
		completions = append(completions, flagCompletions...)
	}

	// Handle positional argument completions
	enteringPosArg := (len(posArgs) > 0 || strings.HasSuffix(input, " ")) && len(posArgs) < len(cmd.PositionalArguments)
	enteringLastPosArg := len(posArgs) == len(cmd.PositionalArguments) && !strings.HasPrefix(argParts[len(argParts)-1], "-")
	if enteringPosArg || enteringLastPosArg {
		completions = append(completions, getPositionalArgumentCompletions(input, cmd, posArgs)...)
		return completions
	}

	return completions
}

func getSubCommandCompletions(input string, finalCommand *Command, parts []string) []Completion {
	completions := []Completion{}

	// If we've started typing, show only subcommands that start with the input
	if !strings.HasSuffix(input, " ") {
		for _, command := range finalCommand.SubCommands {
			if strings.HasPrefix(command.Command, parts[len(parts)-1]) {
				// Filter out commands that have already been entered
				if !strings.Contains(input, fmt.Sprintf(" %s ", command.Command)) {
					completions = append(completions, command)
				}
			}
		}
		return completions
	}

	// Otherwise show all subcommands
	for _, command := range finalCommand.SubCommands {
		// Filter out commands that have already been entered
		if !strings.Contains(input, fmt.Sprintf(" %s ", command.Command)) {
			completions = append(completions, command)
		}
	}

	return completions
}

func getPositionalArgumentCompletions(input string, finalCommand *Command, posArgParts []string) []Completion {
	completions := []Completion{}

	// If we haven't entered any positional arguments yet, show the first one
	if len(posArgParts) == 0 {
		return []Completion{finalCommand.PositionalArguments[0]}
	}

	// If we're entering a positional argument value, show only the positional argument for that value
	if yes, arg := isEnteringPosArgValue(input, finalCommand, posArgParts); yes {
		return []Completion{arg}
	}

	// Otherwise show the next positional argument if there is one
	if len(posArgParts) < len(finalCommand.PositionalArguments) {
		return []Completion{finalCommand.PositionalArguments[len(posArgParts)]}
	}

	return completions
}

func isEnteringPosArgValue(input string, finalCommand *Command, posArgParts []string) (bool, *PositionalArgument) {
	if len(posArgParts) == 0 {
		return false, nil
	}

	lastArg := posArgParts[len(posArgParts)-1]
	positionalArgument := finalCommand.PositionalArguments[len(posArgParts)-1]

	// Does the last arg start with a quote?
	if strings.HasPrefix(lastArg, "\"") || strings.HasPrefix(lastArg, "'") {
		quote := lastArg[0:1]
		// If the last arg doesn't end with a quote, we're entering a value
		if !strings.HasSuffix(lastArg, quote) {
			return true, positionalArgument
		}
	}

	if !strings.HasSuffix(input, " ") {
		return true, positionalArgument
	}

	return false, nil
}

// getFlagCompletions gets completions for flags based on the input
//
// Returns a list of completions and a boolean indicating if this should be the only completion shown or not
func getFlagCompletions(input string, finalCommand *Command, flagArgParts []string, globalFlags []*Flag) ([]Completion, bool) {
	completions := []Completion{}

	allFlags := append(finalCommand.Flags, globalFlags...)

	// If we haven't entered any flags yet, show all flags
	if len(flagArgParts) == 0 {
		for _, a := range allFlags {
			completions = append(completions, a)
		}
		return completions, false
	}

	// If we're entering a flag value, show only the flag for that value
	if yes, flag := isEnteringFlagValue(input, finalCommand, flagArgParts); yes {
		return []Completion{flag}, true
	}

	// If we need to enter a flag value, show only the flag for that value
	if yes, flag := needToEnterFlagValue(finalCommand, flagArgParts); yes {
		return []Completion{flag}, true
	}

	// Otherwise if we end with a space, show all flags not yet entered
	if strings.HasSuffix(input, " ") {
		for _, flag := range allFlags {
			if !containsFlag(input, flag) {
				completions = append(completions, flag)
			}
		}
		return completions, false
	}

	// Otherwise finally, show completions based on the argument being entered
	finalPart := flagArgParts[len(flagArgParts)-1]
	// TODO: Refactor to its own function for better readability and testability
	for _, flag := range allFlags {
		if flag.PsFlag != "" && strings.HasPrefix(flag.PsFlag, finalPart) { // If the flag is a powershell flag handle it
			// Filter out arguments that have already been entered except for the one we're entering
			if !containsFlag(input, flag) || finalPart == flag.PsFlag {
				completions = append(completions, flag)
			}
		} else if strings.HasPrefix(flag.ShortFlag, finalPart) || strings.HasPrefix(flag.LongFlag, finalPart) { // Otherwise handle traditional flags
			// If the last argument is a combined short flag, only check for the last character flag
			if !strings.HasPrefix(finalPart, "--") && len(finalPart) > 2 {
				finalPart = "-" + finalPart[len(finalPart)-1:]
			}

			// Filter out arguments that have already been entered except for the one we're entering
			if !containsFlag(input, flag) || (finalPart == flag.ShortFlag || finalPart == flag.LongFlag) {
				completions = append(completions, flag)
			}
		}
	}

	return completions, false
}

func isEnteringFlagValue(input string, finalCommand *Command, flagArgParts []string) (bool, *Flag) {
	if len(flagArgParts) == 0 {
		return false, nil
	}

	lastArg := flagArgParts[len(flagArgParts)-1]

	// Check if we're entering a flag value with a space between the flag and value
	if len(flagArgParts) >= 2 {
		lastFlag := flagArgParts[len(flagArgParts)-2]
		lastValue := lastArg

		if strings.HasPrefix(lastFlag, "-") && !strings.HasPrefix(lastValue, "-") && !strings.Contains(input, fmt.Sprintf(" %s ", lastValue)) {
			flagValueToCompare := lastFlag
			for _, flag := range finalCommand.Flags {
				// If the last flag is a short flag, only compare the last character
				if flag.PsFlag == "" && !strings.HasPrefix(lastFlag, "--") && len(lastFlag) > 2 {
					flagValueToCompare = "-" + lastFlag[len(lastFlag)-1:]
				}
				if containsFlag(flagValueToCompare, flag) && flag.Type != BoolArgument {
					return true, flag
				}
			}
		}
	}

	// If we're entering a flag value with an equals sign between the flag and value
	if strings.Contains(lastArg, "=") {
		if (!stringEndsInQuoteWithoutEquals(lastArg)) || (stringEndsInQuoteWithoutEquals(lastArg) && !strings.HasSuffix(input, " ")) {
			for _, flag := range finalCommand.Flags {
				// If the flag isn't a PowerShell flag, ensure it's a long flag
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

func needToEnterFlagValue(finalCommand *Command, flagArgParts []string) (bool, *Flag) {
	lastArgument := flagArgParts[len(flagArgParts)-1]

	for _, flag := range finalCommand.Flags {
		// If the last argument is a combined short flag, only check for the last character flag
		if flag.PsFlag == "" && !strings.HasPrefix(lastArgument, "--") && len(lastArgument) > 2 {
			lastArgument = "-" + lastArgument[len(lastArgument)-1:]
		}

		// If the last argument contains a flag and isn't a long flag / psflag with an equals sign pattern
		if containsFlag(lastArgument, flag) && !strings.Contains(lastArgument, fmt.Sprintf("%s=", flag.LongFlag)) && !strings.Contains(lastArgument, fmt.Sprintf("%s=", flag.PsFlag)) {
			// Bool arguments don't need a value
			if flag.Type != BoolArgument {
				return true, flag
			}
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

func splitPositionArgsAndFlags(argParts []string, command *Command) ([]string, []string) {
	// For a given input, split the input into flags and their values and positional arguments

	// If the input is empty, return nothing
	if len(argParts) == 0 {
		return []string{}, []string{}
	}

	// If there are no flags, return all positional arguments
	if len(command.Flags) == 0 {
		return argParts, []string{}
	}

	// If there are no positional arguments, return all flags
	if len(command.PositionalArguments) == 0 {
		return []string{}, argParts
	}

	// If there are both positional and flags
	var positionalArgs []string
	var flags []string
	for i := 0; i < len(argParts); i++ {
		// If the argument is a flag, add it and its value to the flags
		if strings.HasPrefix(argParts[i], "-") {
			for _, a := range command.Flags {
				if containsFlag(argParts[i], a) {
					// Add the flag
					flags = append(flags, argParts[i])
					// If the argument is a boolean, don't check for a value
					if a.Type == BoolArgument {
						break
					}
					// If we have enough parts left, add the value
					if i+1 < len(argParts) {
						flags = append(flags, argParts[i+1])
						i++
					}
				} else {
					// It's an invalid but still entered flag
					if !slices.Contains(flags, argParts[i]) {
						flags = append(flags, argParts[i])
					}
				}
			}
		} else {
			// If the argument is not a flag, add it to the positional arguments
			positionalArgs = append(positionalArgs, argParts[i])
		}
	}

	return positionalArgs, flags
}

func containsFlag(command string, flag *Flag) bool {
	command = removeQuotedStrings(command)
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

// removeQuotedStrings removes the contents between quotes from a string
//
// For example, the input `-m "Hello, world!"` would return `-m ""`
func removeQuotedStrings(input string) string {
	var result []rune
	var hold []rune
	var quoteChar rune
	inQuotes := false

	for _, char := range input {
		if inQuotes {
			if char == quoteChar {
				inQuotes = false
				result = append(result, char)
			} else {
				hold = append(hold, char)
			}
		} else {
			if char == '\'' || char == '"' {
				inQuotes = true
				quoteChar = char
				result = append(result, char)
				hold = []rune{}
			} else {
				result = append(result, char)
			}
		}
	}

	// If the quotes were not closed properly, return the original input
	if inQuotes {
		result = append(result, hold...)
	}

	return string(result)
}

func containsShortFlag(command string, flag string) bool {
	if len(flag) > 0 && flag[0] == '-' {
		flag = flag[1:]
	}
	pattern := fmt.Sprintf(`(^|\s)-[a-zA-Z]*%s[a-zA-Z]*($|\s)`, regexp.QuoteMeta(flag))
	match, _ := regexp.MatchString(pattern, command)
	return match
}

func containsLongFlag(command string, flag string) bool {
	if command == flag {
		return true
	}

	if strings.Contains(command, fmt.Sprintf(" %s ", flag)) {
		return true
	}

	if strings.Contains(command, fmt.Sprintf("%s=", flag)) {
		return true
	}

	return false
}

func containsPowerShellFlag(command string, flag string) bool {
	// If the powershell flag is a short flag, check for the short flag pattern
	if len(flag) == 2 && flag[0] == '-' {
		return containsShortFlag(command, flag)
	}

	// Otherwise check for the long flag pattern
	return containsLongFlag(command, flag)
}
