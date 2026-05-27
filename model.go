package bubblecomplete

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
)

// MARK: Types and Vars

// Model is the Bubble Tea component that drives input, completion, validation,
// and history. Construct with [New]; embed in a host model and forward
// messages to [Model.Update]. Compose into a host view with [Model.View] or
// [Model.Render].
type Model struct {
	// ---- Input ----

	input     textinput.Model
	lastInput string

	// ---- Commands ----

	// The commands available
	Commands     []*Command
	validationErr error

	// ---- Completions ----

	completions      []Completion
	completionIndex  int
	completionHolder string
	showAll          bool
	matchPrefix      string

	// ---- History ----

	// Slice that holds the history of commands
	History         []string
	filteredHistory []string
	historyIndex    int
	historyFilePath string

	// ---- Other ----

	Err    error
	loaded bool
	width  int

	// ---- Options ----

	// The maximum number of history items to store
	HistoryLimit int
	// Whether to indent completions to match the input
	IndentCompletions bool
	// Whether to trim the input on enter
	Autotrim bool
	// The base offset from the left, applied to the completions
	CompletionsOffset int
	// Whether to show different border styles to indicate scrolling
	ShowBorderScroll bool
	// Whether to show the horizontal scrollbar to indicate scrolling
	ShowScrollbar bool
	// The position of the completions relative to the input
	CompletionsPosition Position
	// The number of rows to show in the completions
	CompletionRows int
	// Whether to show type indicator icons
	ShowIcons bool
	// Whether to render the description column next to each completion name
	ShowDescriptions bool
	// The icon for command completions
	CommandIcon string
	// The icon for argument completions
	ArgumentIcon string
	// The icon for flag completions
	FlagIcon string

	styles Styles
	keymap KeyMap
}

// Completion is the internal interface every completable entity implements.
// All methods are unexported; the interface exists to unify rendering across
// [Command], [PositionalArgument], and [Flag] and is not an extension point
// for external packages.
type Completion interface {
	getName() string
	getDescription() string
	getAutocomplete() string
}

// Position selects whether the completion list renders above or below the input.
type Position int

const (
	// PositionAbove renders the completion list above the input.
	PositionAbove Position = iota
	// PositionBelow renders the completion list below the input.
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

// Command defines a top-level command or subcommand. A command must have
// either SubCommands or PositionalArguments, never both. Validation rules are
// enforced by [Command.Validate], which [New] runs on every command.
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

// Argument is the internal interface satisfied by [PositionalArgument] and
// [Flag] so the validator can reason about value types uniformly. Like
// [Completion], the methods are unexported and external implementations are
// not supported.
type Argument interface {
	getName() string
	getDescription() string
	getType() ArgumentType
}

// ArgumentType classifies the kind of value a positional argument or flag accepts.
type ArgumentType string

const (
	StringArgument  ArgumentType = "string"
	IntArgument     ArgumentType = "int"
	FloatArgument   ArgumentType = "float"
	BoolArgument    ArgumentType = "bool"
	FileArgument    ArgumentType = "file"
	DirArgument     ArgumentType = "dir"
	FileDirArgument ArgumentType = "filedir"
)

// PositionalArgument is an ordered, non-flag value a command accepts after its
// command word. Required arguments must be supplied; optional arguments may be
// omitted. Order matters at parse time.
type PositionalArgument struct {
	Name        string
	Description string
	Type        ArgumentType
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

func (a PositionalArgument) getType() ArgumentType {
	return a.Type
}

// Flag describes a single command-line flag. At least one of ShortFlag,
// LongFlag, or PsFlag must be set. PsFlag is the PowerShell-style single-dash
// flag form (e.g. "-Verbose") and is mutually exclusive with ShortFlag and
// LongFlag on the same Flag. Persistent flags propagate to every subcommand
// of the command they're declared on.
type Flag struct {
	ShortFlag   string
	LongFlag    string
	PsFlag      string
	Description string
	Type        ArgumentType
	Persistent  bool
}

func (a Flag) getName() string {
	if a.PsFlag != "" {
		return a.PsFlag
	}
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
	if a.PsFlag != "" {
		return a.PsFlag
	}
	if a.ShortFlag != "" {
		return a.ShortFlag
	}
	return a.LongFlag
}

func (a Flag) getType() ArgumentType {
	return a.Type
}

// MARK: Public Functions

// New creates a new model with the given commands.
//
// Apply Option values to configure construction-time settings such as history,
// styles, key bindings, and layout. Returns an error if any command is invalid.
func New(commands []*Command, width int, opts ...Option) (Model, error) {
	for _, cmd := range commands {
		if err := cmd.Validate(); err != nil {
			return Model{}, err
		}
	}

	if width < 1 {
		width = 1
	}

	input := textinput.New()
	input.Focus()
	input.CharLimit = 1000
	input.ShowSuggestions = true
	input.SetWidth(width)
	input.Placeholder = "Enter command..."

	inputKeyMap := textinput.DefaultKeyMap()
	inputKeyMap.AcceptSuggestion = key.NewBinding()
	inputKeyMap.NextSuggestion = key.NewBinding()
	inputKeyMap.PrevSuggestion = key.NewBinding()
	input.KeyMap = inputKeyMap

	m := Model{
		input:               input,
		Commands:            commands,
		width:               width,
		completionIndex:     -1,
		historyIndex:        -1,
		HistoryLimit:        100,
		Autotrim:            true,
		IndentCompletions:   true,
		CompletionsOffset:   0,
		ShowBorderScroll:    false,
		ShowScrollbar:       false,
		CompletionsPosition: PositionBelow,
		CompletionRows:      5,
		ShowIcons:           false,
		ShowDescriptions:    true,
		CommandIcon:         "\u203A",
		ArgumentIcon:        "\u25C6",
		FlagIcon:            "\u25C7",
		styles:              DefaultStyles(),
		keymap:              DefaultKeyMap(),
	}
	for _, opt := range opts {
		opt(&m)
	}
	m.applyInputValidationStyle()
	return m, nil
}

// Styles returns the current style set. Modify the returned value and pass it to SetStyles to apply changes.
func (m Model) Styles() Styles {
	return m.styles
}

// SetStyles replaces the component's styles.
func (m *Model) SetStyles(s Styles) {
	m.styles = s
	m.applyInputValidationStyle()
}

// applyInputValidationStyle pushes the current valid/invalid input style into
// the embedded textinput so Render stays free of side effects. Call after any
// change to validationErr or styles.
func (m *Model) applyInputValidationStyle() {
	style := m.styles.Input.Valid
	if m.validationErr != nil {
		style = m.styles.Input.Invalid
	}
	s := m.input.Styles()
	s.Focused.Text = style
	s.Blurred.Text = style
	m.input.SetStyles(s)
}

// Value returns the current input value.
func (m Model) Value() string {
	return m.input.Value()
}

// ValidationError returns the validation error for the current input, or nil if the input is valid.
func (m Model) ValidationError() error {
	return m.validationErr
}

// KeyMap returns the current key bindings. Modify the returned value and pass it to SetKeyMap to apply changes.
func (m Model) KeyMap() KeyMap {
	return m.keymap
}

// SetKeyMap replaces the component's key bindings.
func (m *Model) SetKeyMap(k KeyMap) {
	m.keymap = k
}

// SetWidth sets the width of the model.
//
// This is used to calculate the offset for the completions and when text
// should be wrapped. Values less than 1 are clamped to 1.
func (m *Model) SetWidth(width int) {
	if width < 1 {
		width = 1
	}
	m.width = width
	m.input.SetWidth(width)
}

// SetPlaceholder sets the input placeholder text shown when the input is empty.
func (m *Model) SetPlaceholder(placeholder string) {
	m.input.Placeholder = placeholder
}

// Validate checks structural invariants of the command: non-empty name, no
// internal whitespace, no SubCommands+PositionalArguments together, no
// duplicate sibling names or flag aliases, and recursively validates every
// flag, positional argument, and subcommand. New calls this on every command
// it's given.
func (c *Command) Validate() error {
	if c == nil {
		return fmt.Errorf("commands cannot be nil")
	}
	if c.Command == "" {
		return fmt.Errorf("commands must have a command name")
	}
	if strings.TrimSpace(c.Command) != c.Command {
		return fmt.Errorf("command names cannot have leading or trailing whitespace: %q", c.Command)
	}
	if containsWhitespace(c.Command) {
		return fmt.Errorf("command names cannot contain whitespace: %q", c.Command)
	}

	if len(c.SubCommands) > 0 && len(c.PositionalArguments) > 0 {
		return fmt.Errorf("command %q cannot define both SubCommands and PositionalArguments", c.Command)
	}

	seenFlagAlias := make(map[string]struct{})
	for _, flag := range c.Flags {
		if flag == nil {
			return fmt.Errorf("command %q has a nil flag", c.Command)
		}
		if err := flag.Validate(); err != nil {
			return err
		}
		for _, alias := range flagAliases(flag) {
			if _, exists := seenFlagAlias[alias]; exists {
				return fmt.Errorf("command %q has duplicate flag alias: %q", c.Command, alias)
			}
			seenFlagAlias[alias] = struct{}{}
		}
	}

	seenArg := make(map[string]struct{})
	for _, arg := range c.PositionalArguments {
		if arg == nil {
			return fmt.Errorf("command %q has a nil positional argument", c.Command)
		}
		if err := arg.Validate(); err != nil {
			return err
		}
		if _, exists := seenArg[arg.Name]; exists {
			return fmt.Errorf("command %q has duplicate positional argument: %q", c.Command, arg.Name)
		}
		seenArg[arg.Name] = struct{}{}
	}

	seenSub := make(map[string]struct{})
	for _, subCmd := range c.SubCommands {
		if subCmd == nil {
			return fmt.Errorf("command %q has a nil subcommand", c.Command)
		}
		if err := subCmd.Validate(); err != nil {
			return err
		}
		if _, exists := seenSub[subCmd.Command]; exists {
			return fmt.Errorf("command %q has duplicate subcommand: %q", c.Command, subCmd.Command)
		}
		seenSub[subCmd.Command] = struct{}{}
	}

	return nil
}

// Validate checks the positional argument has a name and a known ArgumentType.
func (p *PositionalArgument) Validate() error {
	if p == nil {
		return fmt.Errorf("positional arguments cannot be nil")
	}
	if p.Name == "" {
		return fmt.Errorf("positional arguments must have a name")
	}
	if p.Type == "" {
		return fmt.Errorf("positional arguments must have a type")
	}
	if !isValidArgumentType(p.Type) {
		return fmt.Errorf("positional argument %q has an invalid type: %q", p.Name, p.Type)
	}
	return nil
}

// Validate checks the flag has at least one form set, that each form is
// well-shaped (short flags are a single ASCII character, long flags start with
// "--", PsFlag bodies are two-or-more runes and exclusive with the other
// forms), and that Type is a known ArgumentType.
func (f *Flag) Validate() error {
	if f == nil {
		return fmt.Errorf("flags cannot be nil")
	}
	if f.ShortFlag == "" && f.LongFlag == "" && f.PsFlag == "" {
		return fmt.Errorf("flags must have at least one flag defined")
	}

	// Short flag validation. Short flags are ASCII-only because the runtime
	// parser (validateShortFlags) walks the token byte-by-byte to support
	// combined forms like "-xyz", which is incompatible with multi-byte runes.
	if f.ShortFlag != "" {
		if !strings.HasPrefix(f.ShortFlag, "-") {
			return fmt.Errorf("short flags must start with a dash")
		}
		body := f.ShortFlag[1:]
		if body == "" {
			return fmt.Errorf("flags must have a flag name")
		}
		if body == "-" {
			return fmt.Errorf("short flag body cannot be a dash: %q", f.ShortFlag)
		}
		if len(body) != 1 {
			return fmt.Errorf("short flags must be a single ASCII character: %q", f.ShortFlag)
		}
	}

	// Long flag validation
	if f.LongFlag != "" {
		if !strings.HasPrefix(f.LongFlag, "--") {
			return fmt.Errorf("long flags must start with two dashes")
		}
		body := f.LongFlag[2:]
		if body == "" {
			return fmt.Errorf("flags must have a flag name")
		}
		if containsWhitespace(body) {
			return fmt.Errorf("long flag names cannot contain whitespace: %q", f.LongFlag)
		}
	}

	// PowerShell flag validation
	if f.PsFlag != "" {
		if !strings.HasPrefix(f.PsFlag, "-") {
			return fmt.Errorf("powershell flags must start with a dash")
		}
		body := f.PsFlag[1:]
		if body == "" {
			return fmt.Errorf("flags must have a flag name")
		}
		// PsFlag bodies are matched by string prefix at runtime, so multi-byte
		// runes are fine. Use rune count rather than byte length here.
		if utf8.RuneCountInString(body) < 2 {
			return fmt.Errorf("powershell flags must be more than one character")
		}
		if containsWhitespace(body) {
			return fmt.Errorf("powershell flag names cannot contain whitespace: %q", f.PsFlag)
		}
		if f.ShortFlag != "" || f.LongFlag != "" {
			return fmt.Errorf("powershell flags cannot have short or long flags defined")
		}
	}

	if f.Type == "" {
		return fmt.Errorf("flags must have a type")
	}
	if !isValidArgumentType(f.Type) {
		return fmt.Errorf("flag has an invalid type: %q", f.Type)
	}
	return nil
}

func flagAliases(f *Flag) []string {
	var out []string
	if f.ShortFlag != "" {
		out = append(out, f.ShortFlag)
	}
	if f.LongFlag != "" {
		out = append(out, f.LongFlag)
	}
	if f.PsFlag != "" {
		out = append(out, f.PsFlag)
	}
	return out
}

func containsWhitespace(s string) bool {
	for _, r := range s {
		if unicode.IsSpace(r) {
			return true
		}
	}
	return false
}

func isValidArgumentType(t ArgumentType) bool {
	switch t {
	case StringArgument, IntArgument, FloatArgument, BoolArgument, FileArgument, DirArgument, FileDirArgument:
		return true
	}
	return false
}
