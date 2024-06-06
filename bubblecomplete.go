package bubblecomplete

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// MARK: Types and Vars

var terminalSize tea.WindowSizeMsg

// Colors
var (
	mainBacgkround = lipgloss.AdaptiveColor{Light: "#F9F9F9", Dark: "#282828"}
	altBackground  = lipgloss.AdaptiveColor{Light: "#F0F0F0", Dark: "#181818"}
	green          = lipgloss.AdaptiveColor{Light: "#02BA84", Dark: "#02BF87"}
	red            = lipgloss.AdaptiveColor{Light: "#FE5F86", Dark: "#FE5F86"}
	pink           = lipgloss.AdaptiveColor{Light: "#FF2C70", Dark: "#FF2C70"}
	bluegray       = lipgloss.AdaptiveColor{Light: "#5C6773", Dark: "#1f262d"}
)

// Styles
var (
	lg                         = lipgloss.NewStyle()
	validCommandStyle          = lg.Foreground(green)
	highlightedCompletionStyle = lg.Foreground(pink).Background(bluegray).Bold(true)
	completionRowStyle         = lg.Background(mainBacgkround)
	altCompletionRowStyle      = lg.Background(altBackground)
	completionsBoxStyle        = lg.Border(lipgloss.RoundedBorder())
)

type Model struct {
	// Input
	input     textinput.Model
	lastInput string

	// Command Related
	Commands     []*Command
	validCommand error

	// Completions
	completions      []Completion
	completionIndex  int
	completionHolder string

	// History
	History         []string
	filteredHistory []string
	historyIndex    int

	// Other
	loaded bool

	// Options
	HistoryLimit        int
	IndentCompletions   bool
	Autotrim            bool
	CompletionsOffset   int
	WrapCompletions     bool
	ValidCommandStyling bool
}

type Completion interface {
	getName() string
	getDescription() string
}

type Command struct {
	Command     string
	SubCommands []*Command
	Arguments   []*Argument
	Description string
}

func (c Command) getName() string {
	return c.Command
}

func (c Command) getDescription() string {
	return c.Description
}

type Argument struct {
	Argument    string
	Description string
	Type        argumentType
}

type argumentType string

const (
	StringArgument argumentType = "string"
	IntArgument    argumentType = "int"
	FloatArgument  argumentType = "float"
	BoolArgument   argumentType = "bool"
	FileArgument   argumentType = "file"
)

func (a Argument) getName() string {
	return a.Argument
}

func (a Argument) getDescription() string {
	// TODO: Improve styling of argument type
	return fmt.Sprintf("%s [%s]", a.Description, a.Type)
}

// TODO: Public or Private?
type SelectedCommandMsg struct {
	command string
	err     error
}

// MARK: Public Functions

func New(commands []*Command) Model {
	input := textinput.New()
	input.Focus()
	input.CharLimit = 1000
	input.Width = 100
	input.ShowSuggestions = true

	inputKeyMap := textinput.DefaultKeyMap
	inputKeyMap.AcceptSuggestion = key.NewBinding()
	inputKeyMap.NextSuggestion = key.NewBinding()
	inputKeyMap.PrevSuggestion = key.NewBinding()

	input.KeyMap = inputKeyMap

	// TODO: Make getters and setters for certain options
	return Model{
		input:               input,
		Commands:            commands,
		completionIndex:     -1,
		historyIndex:        -1,
		HistoryLimit:        100,
		Autotrim:            true,
		IndentCompletions:   true,
		CompletionsOffset:   2,
		ValidCommandStyling: true,
		WrapCompletions:     true,
	}
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	// Handle key presses
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			m, cmd = m.keyTab()
		case "backspace":
			m, cmd = m.keyBackspace()
		case "enter":
			m, cmd = m.keyEnter()
		case "up":
			m, cmd = m.keyUp()
		case "down":
			m, cmd = m.keyDown()
		case "right":
			m, cmd = m.keyRight()
		default:
			m, cmd = m.keyDefault()
		}
	case tea.WindowSizeMsg:
		terminalSize = msg
	}
	cmds = append(cmds, cmd)

	// Update the text input
	m.input, cmd = m.input.Update(msg)
	cmds = append(cmds, cmd)

	// If the input has changed, update the completions and validate the input
	if m.input.Value() != "" && m.input.Value() != m.lastInput && m.completionHolder == "" {
		// TODO: Debounce this
		m.lastInput = m.input.Value()
		m.completions = m.getCompletions()
		m.validCommand = m.validateInput()
	}

	// If not loaded, start the blinking cursor
	if !m.loaded {
		m.loaded = true
		cmds = append(cmds, textinput.Blink)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if m.historyIndex != -1 {
		return m.input.View()
	}

	return m.showCompletionsRender()
}

// MARK: Private Functions

func (m Model) showCompletionsRender() string {
	// TODO: Show the error somewhere or only return to user?
	if m.validCommand == nil {
		m.input.TextStyle = validCommandStyle
	}

	completionsText := []string{}
	completionsDescriptions := []string{}
	maxCommandLength := 0
	maxTotalLength := 0
	if len(m.completions) > 0 && len(m.input.Value()) > 0 {
		for _, comp := range m.completions {
			name := comp.getName()
			description := comp.getDescription()

			completionsText = append(completionsText, name)
			completionsDescriptions = append(completionsDescriptions, description)

			if len(name) > maxCommandLength {
				maxCommandLength = len(name)
			}
			if len(name)+len(description) > maxTotalLength {
				maxTotalLength = len(name) + len(description)
			}
		}
	}
	maxCommandLength += 3

	completionsRow := make([]string, 0, len(completionsText))
	for i := 0; i < len(completionsText); i++ {
		textWidth := lipgloss.Width(completionsText[i])
		text := lipgloss.JoinHorizontal(
			lipgloss.Left,
			completionsText[i],
			lg.
				Width(maxTotalLength-textWidth).
				PaddingLeft(maxCommandLength-textWidth).
				Render(completionsDescriptions[i]),
		)
		if i == m.completionIndex {
			completionsRow = append(
				completionsRow,
				highlightedCompletionStyle.Render(text),
			)
		} else if i%2 == 0 {
			completionsRow = append(
				completionsRow,
				altCompletionRowStyle.Render(text),
			)
		} else {
			completionsRow = append(
				completionsRow,
				completionRowStyle.Render(text),
			)
		}
	}

	if len(completionsRow) != 0 {
		startCompletionsIndex := max(0, m.completionIndex-4)
		endCompletionsIndex := min(len(completionsRow), startCompletionsIndex+5)
		completions := lipgloss.JoinVertical(
			lipgloss.Left,
			completionsRow[startCompletionsIndex:endCompletionsIndex]...,
		)

		offset := m.calculateCompletionsOffset(completions)

		return m.input.View() + "\n" + completionsBoxStyle.
			Margin(0, 0, 0, offset).
			Render(
				completions,
			)
	}

	return m.input.View()
}

func (m Model) calculateCompletionsOffset(completions string) int {
	if !m.IndentCompletions {
		return m.CompletionsOffset
	}

	input := m.input.Value()
	parts := splitInput(input)
	finalPart := parts[len(parts)-1]
	var trimIndexForOffset int

	// If we're entering a string argument then set offset to the end of the last part
	start := finalPart[0]
	end := finalPart[len(finalPart)-1]
	if (string(start) == "\"" || string(start) == "'") && start != end {
		if len(parts) == 1 {
			trimIndexForOffset = 0
		}
		trimIndexForOffset = strings.LastIndex(input, parts[len(parts)-2])
	} else if (string(start) == "\"" || string(start) == "'") && start == end {
		// If we finished entering a string argument and the last character is a space set the offset to the last space
		if strings.HasSuffix(input, " ") {
			trimIndexForOffset = strings.LastIndex(input, " ")
		} else {
			// Otherwise set the offset to the argument start
			trimIndexForOffset = strings.LastIndex(input, parts[len(parts)-2])
		}
	} else {
		// Otherwise set the offset to the last space
		trimIndexForOffset = strings.LastIndex(input, " ")
	}
	trimmedInput := input[:trimIndexForOffset+1]

	offset := lipgloss.Width(trimmedInput) + m.CompletionsOffset
	if offset+lipgloss.Width(completions) > terminalSize.Width {
		offset = terminalSize.Width - lipgloss.Width(completions) - 2
	}

	return offset
}

func (m Model) keyUp() (Model, tea.Cmd) {
	if len(m.History) == 0 {
		return m, nil
	}

	if len(m.filteredHistory) == 0 {
		for _, h := range m.History {
			if strings.HasPrefix(h, m.input.Value()) {
				m.filteredHistory = append(m.filteredHistory, h)
			}
		}
		if len(m.filteredHistory) == 0 {
			return m, nil
		}
		m.filteredHistory = append(m.filteredHistory, m.input.Value())
	}

	if m.historyIndex == -1 {
		m.historyIndex = len(m.filteredHistory) - 2
		m.input.SetValue(m.filteredHistory[m.historyIndex])
		m.input.SetCursor(len(m.input.Value()))
		return m, nil
	}

	if m.historyIndex > 0 {
		m.historyIndex--
		m.input.SetValue(m.filteredHistory[m.historyIndex])
		m.input.SetCursor(len(m.input.Value()))
		return m, nil
	}

	return m, nil
}

func (m Model) keyDown() (Model, tea.Cmd) {
	if len(m.History) == 0 || len(m.filteredHistory) == 0 {
		return m, nil
	}

	if m.historyIndex < len(m.filteredHistory)-1 {
		m.historyIndex++
		m.input.SetValue(m.filteredHistory[m.historyIndex])
		m.input.SetCursor(len(m.input.Value()))
		return m, nil
	}

	return m, nil
}

func (m Model) keyRight() (Model, tea.Cmd) {
	if len(m.input.AvailableSuggestions()) == 0 {
		return m, nil
	}
	m.input.SetValue(m.input.CurrentSuggestion())
	m.input.SetCursor(len(m.input.Value()))
	return m, nil
}

func (m Model) keyTab() (Model, tea.Cmd) {
	if len(m.completions) == 0 {
		return m, nil
	}

	if len(m.completions) == 1 {
		if strings.HasSuffix(strings.TrimSpace(m.input.Value()), m.completions[0].getName()) {
			return m, nil
		}
	}

	if m.completionIndex < len(m.completions)-1 {
		m.completionIndex++
	} else {
		m.completionIndex = -1
	}

	if m.completionHolder == "" {
		m.completionHolder = m.input.Value()
	}

	if m.completionIndex == -1 {
		m.input.SetValue(m.completionHolder)
		m.completionHolder = ""
		return m, nil
	}

	pretext := m.completionHolder
	parts := splitInput(m.completionHolder)
	if !strings.HasSuffix(pretext, " ") && pretext != "" {
		if len(parts) == 0 {
			return m, nil
		}
		pretext = strings.Join(parts[:len(parts)-1], " ")
		if len(parts) > 1 {
			pretext += " "
		}
	}

	m.input.SetValue(pretext + m.completions[m.completionIndex].getName())
	m.input.SetCursor(len(m.input.Value()))
	return m, nil
}

func (m Model) keyEnter() (Model, tea.Cmd) {
	command := m.input.Value()
	if m.Autotrim {
		command = strings.TrimSpace(m.input.Value())
	}

	if len(m.History) > 0 {
		if m.History[len(m.History)-1] != command {
			m.History = append(m.History, command)
		}
	} else {
		m.History = append(m.History, command)
	}

	if len(m.History) > m.HistoryLimit {
		m.History = m.History[1:]
	}
	m = m.resetModel()
	m.input.SetSuggestions(m.History)
	return m, func() tea.Msg {
		return SelectedCommandMsg{command: command, err: m.validCommand}
	}
}

func (m Model) keyBackspace() (Model, tea.Cmd) {
	m.completionIndex = -1
	return m, nil
}

func (m Model) keyDefault() (Model, tea.Cmd) {
	m.completionHolder = ""
	m.completionIndex = -1
	m.historyIndex = -1
	m.filteredHistory = []string{}
	return m, nil
}

func (m Model) resetModel() Model {
	m.input.SetValue("")
	m.completions = []Completion{}
	m.completionIndex = -1
	m.historyIndex = -1
	return m
}

func splitInput(input string) []string {
	var result []string
	var buffer strings.Builder
	inQuotes := false
	var quoteChar rune

	flushBuffer := func() {
		if buffer.Len() > 0 {
			result = append(result, buffer.String())
			buffer.Reset()
		}
	}

	for _, char := range input {
		switch {
		case char == ' ' && !inQuotes:
			flushBuffer()
		case char == '"' || char == '\'':
			if inQuotes && char == quoteChar {
				inQuotes = false
				buffer.WriteRune(char)
				flushBuffer()
			} else if !inQuotes {
				inQuotes = true
				quoteChar = char
				buffer.WriteRune(char)
			} else {
				buffer.WriteRune(char)
			}
		default:
			buffer.WriteRune(char)
		}
	}

	flushBuffer()
	return result
}

func (m Model) getCompletions() []Completion {
	if m.input.Value() == "" {
		return []Completion{}
	}
	allCompletions := getCompletions(m.input.Value(), m.Commands)

	// Sort completions alphabetically by command
	for i := 0; i < len(allCompletions); i++ {
		for j := i + 1; j < len(allCompletions); j++ {
			if allCompletions[i].getName() > allCompletions[j].getName() {
				allCompletions[i], allCompletions[j] = allCompletions[j], allCompletions[i]
			}
		}
	}

	return allCompletions
}

func getCompletions(
	fullCommand string,
	availableCommands []*Command,
) []Completion {
	var completions []Completion

	// Extract the command and arguments from the full command
	commands, arguments, success := strings.Cut(fullCommand, " -")
	if len(commands) == 0 {
		return completions
	}
	commandParts := strings.Fields(commands)
	if len(commandParts) == 0 {
		return completions
	}
	if success {
		arguments = "-" + arguments
	}
	argumentParts := splitInput(arguments)

	// Find the final entered command in the available commands
	var finalCommand *Command
	depth := 0
	for _, enteredCommand := range commandParts {
		for _, c := range availableCommands {
			// If the command is found in the available commands and we've finished typing then use it
			if c.Command == enteredCommand &&
				strings.Contains(fullCommand, fmt.Sprintf("%s ", c.Command)) {
				finalCommand = c
				availableCommands = c.SubCommands
				depth++
				break
			}
		}
	}

	// If the final command is nil, then we haven't finished typing the first command yet
	if finalCommand == nil {
		for _, c := range availableCommands {
			if strings.HasPrefix(c.Command, fullCommand) {
				completions = append(completions, c)
			}
		}
		return completions
	}

	// If the depth is less than the number of full commands entered, we have an invalid command
	fullCommandCount := len(commandParts) - 1
	if strings.HasSuffix(fullCommand, " ") {
		fullCommandCount++
	}
	if depth < fullCommandCount {
		return completions
	}

	// If the final command has subcommands, return them as completions
	if len(finalCommand.SubCommands) > 0 {
		for _, command := range finalCommand.SubCommands {
			// If we haven't started typing the command yet, show all subcommands
			if strings.Contains(
				fullCommand,
				fmt.Sprintf("%s ", commandParts[len(commandParts)-1]),
			) {
				completions = append(completions, command)
			} else if strings.HasPrefix(command.Command, commandParts[len(commandParts)-1]) {
				completions = append(completions, command)
			}
		}
		return completions
	}

	// ----- If we're got here, we're dealing with arguments -----

	// If we haven't entered any arguments yet, show all arguments
	if len(argumentParts) == 0 {
		for _, a := range finalCommand.Arguments {
			completions = append(completions, a)
		}
		return completions
	}

	// If we're already entering an arg value, show only the argument for that value
	if len(argumentParts) >= 2 {
		lastArgument := argumentParts[len(argumentParts)-2]
		lastValue := argumentParts[len(argumentParts)-1]

		if strings.HasPrefix(lastArgument, "-") && !strings.HasPrefix(lastValue, "-") &&
			!strings.Contains(fullCommand, fmt.Sprintf(" %s ", lastValue)) {
			for _, a := range finalCommand.Arguments {
				if a.Argument == lastArgument {
					// Bool arguments don't need a value
					if a.Type != BoolArgument {
						return []Completion{a}
					}
				}
			}
		}
	}

	// If we end with a space aka we're about to enter another argument or value
	if strings.HasSuffix(fullCommand, " ") {
		// If an argument value is expected next, show that corresponding argument
		lastArgument := argumentParts[len(argumentParts)-1]
		for _, a := range finalCommand.Arguments {
			if a.Argument == lastArgument {
				// Bool arguments don't need a value
				if a.Type != BoolArgument {
					return []Completion{a}
				}
			}
		}

		// Otherwise, show all arguments not yet entered
		for _, a := range finalCommand.Arguments {
			if !strings.Contains(fullCommand, fmt.Sprintf(" %s ", a.Argument)) {
				completions = append(completions, a)
			}
		}
		return completions
	}

	// Finally, if here then show completions based on the argument being entered
	for _, a := range finalCommand.Arguments {
		if strings.HasPrefix(a.Argument, argumentParts[len(argumentParts)-1]) {
			// Filter out arguments that have already been entered
			if !strings.Contains(fullCommand, fmt.Sprintf(" %s ", a.Argument)) {
				completions = append(completions, a)
			}
		}
	}

	return completions
}

func (m Model) validateInput() error {
	if m.input.Value() == "" {
		return nil
	}

	err := validateCommand(m.input.Value(), m.Commands)
	if err != nil {
		return err
	}
	return nil
}

// Helper function to find a command by name
func findCommand(commands []*Command, name string) (*Command, error) {
	for _, cmd := range commands {
		if cmd.Command == name {
			return cmd, nil
		}
	}
	return nil, errors.New("command not found")
}

// Helper function to find an argument by name
func findArgument(arguments []*Argument, name string) (*Argument, error) {
	for _, arg := range arguments {
		if arg.Argument == name {
			return arg, nil
		}
	}
	return nil, errors.New("argument not found")
}

// Helper function to validate argument value based on type
func validateArgumentValue(arg *Argument, value string) error {
	switch arg.Type {
	case StringArgument:
		if value == "" {
			return errors.New("missing value for argument: " + arg.Argument)
		}
	case IntArgument:
		if _, err := strconv.Atoi(value); err != nil {
			return errors.New("invalid integer value for argument: " + arg.Argument)
		}
	case FloatArgument:
		if _, err := strconv.ParseFloat(value, 64); err != nil {
			return errors.New("invalid float value for argument: " + arg.Argument)
		}
	case BoolArgument:
		return nil // No validation needed for boolean, presence is enough
	case FileArgument:
		if file, err := os.Stat(value); err != nil {
			return errors.New("invalid file path for argument: " + arg.Argument)
		} else if file.IsDir() {
			return errors.New("file path is a directory: " + arg.Argument)
		}
	default:
		return errors.New("unknown argument type: " + string(arg.Type))
	}
	return nil
}

func validateCommand(input string, commands []*Command) error {
	parts := splitInput(input)
	if len(parts) == 0 {
		return errors.New("empty command")
	}

	currentCommands := commands
	var parentCmd *Command

	for i := 0; i < len(parts); i++ {
		part := parts[i]

		if i == 0 || (i > 0 && !strings.HasPrefix(part, "-")) {
			cmd, err := findCommand(currentCommands, part)
			if err != nil {
				return fmt.Errorf("command '%s' not found", part)
			}
			parentCmd = cmd
			currentCommands = cmd.SubCommands
			continue
		}

		if strings.HasPrefix(part, "-") {
			argName := part
			argValue := ""

			if strings.Contains(part, "=") {
				parts := strings.SplitN(part, "=", 2)
				argName = parts[0]
				argValue = parts[1]
			}

			if parentCmd == nil {
				return errors.New("invalid argument: " + part)
			}

			arg, err := findArgument(parentCmd.Arguments, argName)
			if err != nil {
				return fmt.Errorf("argument '%s' not found", argName)
			}

			if arg.Type != BoolArgument && argValue == "" {
				if i == len(parts)-1 || strings.HasPrefix(parts[i+1], "-") {
					return fmt.Errorf("missing value for argument '%s'", argName)
				}
				argValue = parts[i+1]
				i++
			}

			err = validateArgumentValue(arg, argValue)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
