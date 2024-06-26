package bubblecomplete

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// MARK: Types and Vars

type Model struct {
	// ---- Input ----

	input     textinput.Model
	lastInput string

	// ---- Commands ----

	// The commands available
	Commands     []*Command
	validCommand error

	// ---- Completions ----

	completions      []Completion
	completionIndex  int
	completionHolder string
	showAll          bool

	// ---- History ----

	// Slice that holds the history of commands
	History         []string
	filteredHistory []string
	historyIndex    int
	historyFilePath string

	// ---- Other ----

	Err               error
	loaded            bool
	scrollbarProgress progress.Model
	width             int

	// ---- Options ----

	// The maximum number of history items to store
	HistoryLimit int
	// Whether to indent completions to match the input
	IndentCompletions bool
	// Whether to trim the input on enter
	Autotrim bool
	// The base offset from the left, applied to the completions
	CompletionsOffset int
	// The text style for valid commands
	ValidCommandStyle lipgloss.Style
	// The text style for invalid commands
	InvalidCommandStyle lipgloss.Style
	// Whether to show different border styles to indicate scrolling
	ShowBorderScroll bool
	// Whether to show the horizontal scrollbar to indicate scrolling
	ShowScrollbar bool
	// The position of the completions relative to the input
	CompletionsPosition Position
	// The number of rows to show in the completions
	CompletionRows int
}

type Completion interface {
	getName() string
	getDescription() string
	getAutocomplete() string
}

type Position int

const (
	PositionAbove Position = iota
	PositionBelow
)

func (p Position) String() string {
	switch p {
	case PositionAbove:
		return "above"
	case PositionBelow:
		return "below"
	default:
		return "unknown"
	}
}

type Command struct {
	Command             string
	Description         string
	SubCommands         []*Command
	PositionalArguments []*PositionalArgument
	Flags               []*Flag
}

func (c Command) getName() string {
	return c.Command
}

func (c Command) getDescription() string {
	return c.Description
}

func (c Command) getAutocomplete() string {
	return c.Command
}

type Argument interface {
	getName() string
	getDescription() string
	getType() argumentType
}

type argumentType string

const (
	StringArgument  argumentType = "string"
	IntArgument     argumentType = "int"
	FloatArgument   argumentType = "float"
	BoolArgument    argumentType = "bool"
	FileArgument    argumentType = "file"
	DirArgument     argumentType = "dir"
	FileDirArgument argumentType = "filedir"
)

type PositionalArgument struct {
	Name        string
	Description string
	Type        argumentType
	Required    bool
}

func (a PositionalArgument) getName() string {
	return a.Name
}

func (a PositionalArgument) getDescription() string {
	isRequired := "required"
	if !a.Required {
		isRequired = "optional"
	}
	if a.Type == BoolArgument {
		return fmt.Sprintf("%s [%s]", a.Description, isRequired)
	}
	return fmt.Sprintf("%s [%s] [%s]", a.Description, a.Type, isRequired)
}

func (a PositionalArgument) getAutocomplete() string {
	return ""
}

func (a PositionalArgument) getType() argumentType {
	return a.Type
}

type Flag struct {
	ShortFlag   string
	LongFlag    string
	Description string
	Type        argumentType
	Persistent  bool
}

func (a Flag) getName() string {
	if a.ShortFlag != "" && a.LongFlag != "" {
		return fmt.Sprintf("%s %s", a.ShortFlag, a.LongFlag)
	}
	if a.ShortFlag != "" {
		return a.ShortFlag
	}
	return a.LongFlag
}

func (a Flag) getDescription() string {
	if a.Type == BoolArgument {
		return a.Description
	}
	return fmt.Sprintf("%s [%s]", a.Description, a.Type)
}

func (a Flag) getAutocomplete() string {
	if a.ShortFlag != "" {
		return a.ShortFlag
	}
	return a.LongFlag
}

func (a Flag) getType() argumentType {
	return a.Type
}

// MARK: Public Functions

// New creates a new model with the given commands
//
// Returns an error if any of the commands are invalid
func New(commands []*Command, width int) (Model, error) {
	for _, cmd := range commands {
		if err := cmd.Validate(); err != nil {
			return Model{}, err
		}
	}

	input := textinput.New()
	input.Focus()
	input.CharLimit = 1000
	input.ShowSuggestions = true

	inputKeyMap := textinput.DefaultKeyMap
	inputKeyMap.AcceptSuggestion = key.NewBinding()
	inputKeyMap.NextSuggestion = key.NewBinding()
	inputKeyMap.PrevSuggestion = key.NewBinding()
	input.KeyMap = inputKeyMap

	progress := progress.New(progress.WithDefaultGradient())
	progress.ShowPercentage = false

	return Model{
		input:               input,
		Commands:            commands,
		width:               width,
		completionIndex:     -1,
		historyIndex:        -1,
		HistoryLimit:        100,
		Autotrim:            true,
		IndentCompletions:   true,
		CompletionsOffset:   0,
		ValidCommandStyle:   lg.Foreground(green),
		InvalidCommandStyle: lg.Foreground(textColor),
		scrollbarProgress:   progress,
		ShowBorderScroll:    false,
		ShowScrollbar:       false,
		CompletionsPosition: PositionBelow,
		CompletionRows:      5,
	}, nil
}

// SetWidth sets the width of the model
//
// This is used to calculate the offset for the completions and when text should be wrapped
func (m *Model) SetWidth(width int) {
	m.width = width
}

// Set the input placeholder text
func (m *Model) SetPlaceholder(placeholder string) {
	m.input.Placeholder = placeholder
}

func (c *Command) Validate() error {
	if c.Command == "" {
		return fmt.Errorf("commands must have a command name")
	}

	for _, flag := range c.Flags {
		if err := flag.Validate(); err != nil {
			return err
		}
	}

	for _, arg := range c.PositionalArguments {
		if err := arg.Validate(); err != nil {
			return err
		}
	}

	for _, subCmd := range c.SubCommands {
		if err := subCmd.Validate(); err != nil {
			return err
		}
	}

	return nil
}

func (p *PositionalArgument) Validate() error {
	if p.Name == "" {
		return fmt.Errorf("positional arguments must have a name")
	}
	if p.Type == "" {
		return fmt.Errorf("positional arguments must have a type")
	}
	return nil
}

func (f *Flag) Validate() error {
	if f.ShortFlag == "" && f.LongFlag == "" {
		return fmt.Errorf("flags must have at least one short or long flag defined")
	}

	// Short flag validation
	if f.ShortFlag != "" {
		if !strings.HasPrefix(f.ShortFlag, "-") {
			return fmt.Errorf("short flags must start with a dash")
		}
		if len(f.ShortFlag) > 2 {
			return fmt.Errorf("short flags must be one character")
		}
		if f.ShortFlag[1:] == "" {
			return fmt.Errorf("flags must have a flag name")
		}
	}

	// Long flag validation
	if f.LongFlag != "" {
		if !strings.HasPrefix(f.LongFlag, "--") {
			return fmt.Errorf("long flags must start with two dashes")
		}
		if f.LongFlag[2:] == "" {
			return fmt.Errorf("flags must have a flag name")
		}
	}

	if f.Type == "" {
		return fmt.Errorf("flags must have a type")
	}
	return nil
}
