package bubblecomplete

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
)

// MARK: Types and Vars

// defaultInputCharLimit caps how many runes the embedded textinput will
// accept. High enough to never hit in normal CLI input; low enough to bound
// memory if something pastes a megabyte.
const defaultInputCharLimit = 1000

// Model is the Bubble Tea component that drives input, completion, validation,
// and history. Construct with [New]; embed in a host model and forward
// messages to [Model.Update]. Compose into a host view with [Model.View] or
// [Model.Render].
type Model struct {
	// ---- Input ----

	input     textinput.Model
	lastInput string

	// ---- Commands ----

	// Commands is the command tree the user can complete against and submit.
	Commands      []*Command
	validationErr error

	// ---- Completions ----

	completions      []completion
	completionIndex  int
	completionHolder string
	showAll          bool
	matchPrefix      string

	// ---- History ----

	// History is the in-memory command history, most recent first.
	History         []string
	filteredHistory []string
	historyIndex    int
	historyFilePath string

	// ---- Other ----

	err    error
	loaded bool
	width  int

	// ---- Options ----

	// HistoryLimit caps how many commands are retained. ≤ 0 disables history.
	HistoryLimit int
	// IndentCompletions aligns the completion list with the active token.
	IndentCompletions bool
	// Autotrim strips surrounding whitespace from the submitted command.
	Autotrim bool
	// CompletionsOffset is an additional left offset added to the completion list.
	CompletionsOffset int
	// ShowBorderScroll tints the top/bottom border when more rows exist off-screen.
	ShowBorderScroll bool
	// ShowScrollbar renders a vertical scrollbar on the completion list.
	ShowScrollbar bool
	// CompletionsPosition places the list above or below the input.
	CompletionsPosition Position
	// CompletionRows is the visible row count before scrolling.
	CompletionRows int
	// ShowIcons renders a type-indicator glyph on each row.
	ShowIcons bool
	// ShowDescriptions renders the description column next to each name.
	ShowDescriptions bool
	// CommandIcon is the glyph used for command rows when ShowIcons is true.
	CommandIcon string
	// ArgumentIcon is the glyph used for positional / filesystem rows when ShowIcons is true.
	ArgumentIcon string
	// FlagIcon is the glyph used for flag rows when ShowIcons is true.
	FlagIcon string

	// ---- Filesystem completion (opt-in) ----
	//
	// The fields below are public for parity with other Model knobs, but
	// changes made after construction are picked up on the NEXT input
	// change — pathState and m.completions are not refreshed in place.
	// Hosts that need an immediate refresh after toggling at runtime can
	// force one by re-typing or by clearing and re-setting input.

	// FilesystemCompletions enables live filesystem completion and validity
	// colouring for FileArgument, DirArgument, and FileDirArgument values.
	// Default false. Hosts that don't enable this see the pre-feature
	// behaviour unchanged.
	FilesystemCompletions bool
	// FilesystemCompletionLimit caps the per-keystroke candidate count.
	// Default 200. Values ≤ 0 are clamped to 1 at the use site.
	FilesystemCompletionLimit int
	// HiddenFiles surfaces dotfiles in completions even when the typed
	// basename prefix does not start with ".". Default false.
	HiddenFiles bool

	// pathCache is lazily initialised on first use by recomputePathState.
	pathCache *dirCache
	// pathState is the per-keystroke result of activeFileArgument plus
	// classification and candidate generation. Read by Render and the
	// validation-style suppression.
	pathState pathState

	styles Styles
	keymap KeyMap
}

// completion is the internal interface every completable entity implements.
// It exists to unify rendering across [Command], [PositionalArgument], and
// [Flag]. Kept unexported because external packages cannot implement it
// (the methods are unexported); a public extension point may be added in a
// future release.
type completion interface {
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

// argument is the internal interface satisfied by [PositionalArgument] and
// [Flag] so the validator can reason about value types uniformly. The methods
// are unexported and external implementations are not supported.
type argument interface {
	getName() string
	getDescription() string
	getType() ArgumentType
}

// ArgumentType classifies the kind of value a positional argument or flag accepts.
type ArgumentType string

const (
	// StringArgument accepts any text. Values containing whitespace must be
	// quoted by the user.
	StringArgument ArgumentType = "string"
	// IntArgument accepts any value parseable by strconv.Atoi.
	IntArgument ArgumentType = "int"
	// FloatArgument accepts any value parseable by strconv.ParseFloat.
	FloatArgument ArgumentType = "float"
	// BoolArgument is presence-only for flags: the flag's presence in the
	// input is true, its absence is false. Positional bool arguments have no
	// runtime use today but the type is reserved.
	BoolArgument ArgumentType = "bool"
	// FileArgument requires the value to resolve to an existing file on disk.
	FileArgument ArgumentType = "file"
	// DirArgument requires the value to resolve to an existing directory.
	DirArgument ArgumentType = "dir"
	// FileDirArgument accepts an existing file or directory.
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

func (p PositionalArgument) getName() string {
	return p.Name
}

func (p PositionalArgument) getDescription() string {
	isRequired := "required"
	if !p.Required {
		isRequired = "optional"
	}
	if p.Type == BoolArgument {
		return fmt.Sprintf("%s [%s]", p.Description, isRequired)
	}
	return fmt.Sprintf("%s [%s] [%s]", p.Description, p.Type, isRequired)
}

func (p PositionalArgument) getAutocomplete() string {
	return ""
}

func (p PositionalArgument) getType() ArgumentType {
	return p.Type
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

func (f Flag) getName() string {
	if f.PsFlag != "" {
		return f.PsFlag
	}
	if f.ShortFlag != "" && f.LongFlag != "" {
		return fmt.Sprintf("%s %s", f.ShortFlag, f.LongFlag)
	}
	if f.ShortFlag != "" {
		return f.ShortFlag
	}
	return f.LongFlag
}

func (f Flag) getDescription() string {
	if f.Type == BoolArgument {
		return f.Description
	}
	return fmt.Sprintf("%s [%s]", f.Description, f.Type)
}

func (f Flag) getAutocomplete() string {
	if f.PsFlag != "" {
		return f.PsFlag
	}
	if f.ShortFlag != "" {
		return f.ShortFlag
	}
	return f.LongFlag
}

func (f Flag) getType() ArgumentType {
	return f.Type
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
	input.CharLimit = defaultInputCharLimit
	input.ShowSuggestions = true
	input.SetWidth(width)
	input.Placeholder = "Enter command..."

	inputKeyMap := textinput.DefaultKeyMap()
	inputKeyMap.AcceptSuggestion = key.NewBinding()
	inputKeyMap.NextSuggestion = key.NewBinding()
	inputKeyMap.PrevSuggestion = key.NewBinding()
	input.KeyMap = inputKeyMap

	m := Model{
		input:                     input,
		Commands:                  commands,
		width:                     width,
		completionIndex:           -1,
		historyIndex:              -1,
		HistoryLimit:              100,
		Autotrim:                  true,
		IndentCompletions:         true,
		CompletionsOffset:         0,
		ShowBorderScroll:          false,
		ShowScrollbar:             false,
		CompletionsPosition:       PositionBelow,
		CompletionRows:            5,
		ShowIcons:                 false,
		ShowDescriptions:          true,
		CommandIcon:               "\u203A",
		ArgumentIcon:              "\u25C6",
		FlagIcon:                  "\u25C7",
		FilesystemCompletions:     false,
		FilesystemCompletionLimit: 200,
		HiddenFiles:               false,
		styles:                    DefaultStyles(),
		keymap:                    DefaultKeyMap(),
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
//
// When the user is mid-typing a partial filesystem path that resolves to
// PathNotFound, the whole-input invalid style is suppressed — the path-range
// overlay (PathPartial) provides the in-progress signal instead. Other
// validation errors still drive whole-input red.
func (m *Model) applyInputValidationStyle() {
	style := m.styles.Input.Valid
	if m.validationErr != nil && !m.isPartialPathMidType() {
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

// Error returns the current component-level error (e.g., from history file
// I/O), or nil if the last operation succeeded. This is distinct from
// ValidationError, which reports errors about the typed input.
func (m Model) Error() error {
	return m.err
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
		return errors.New("commands cannot be nil")
	}
	if c.Command == "" {
		return errors.New("commands must have a command name")
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
		return errors.New("positional arguments cannot be nil")
	}
	if p.Name == "" {
		return errors.New("positional arguments must have a name")
	}
	if p.Type == "" {
		return errors.New("positional arguments must have a type")
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
		return errors.New("flags cannot be nil")
	}
	if f.ShortFlag == "" && f.LongFlag == "" && f.PsFlag == "" {
		return errors.New("flags must have at least one flag defined")
	}

	// Short flag validation. Short flags must be a single ASCII letter:
	// the runtime parser walks the token byte-by-byte for combined forms
	// like "-xyz" (incompatible with multi-byte runes), and detection
	// requires letter bodies so non-flag tokens like "-1" are not
	// misclassified as flags. Use a long or PowerShell flag for non-letter
	// names like "-1" or "-?".
	if f.ShortFlag != "" {
		if !strings.HasPrefix(f.ShortFlag, "-") {
			return errors.New("short flags must start with a dash")
		}
		body := f.ShortFlag[1:]
		if body == "" {
			return errors.New("flags must have a flag name")
		}
		if body == "-" {
			return fmt.Errorf("short flag body cannot be a dash: %q", f.ShortFlag)
		}
		if len(body) != 1 || !isASCIILetter(body[0]) {
			return fmt.Errorf("short flags must be a single ASCII letter: %q", f.ShortFlag)
		}
	}

	if f.LongFlag != "" {
		if !strings.HasPrefix(f.LongFlag, "--") {
			return errors.New("long flags must start with two dashes")
		}
		body := f.LongFlag[2:]
		if body == "" {
			return errors.New("flags must have a flag name")
		}
		if containsWhitespace(body) {
			return fmt.Errorf("long flag names cannot contain whitespace: %q", f.LongFlag)
		}
	}

	if f.PsFlag != "" {
		if !strings.HasPrefix(f.PsFlag, "-") {
			return errors.New("powershell flags must start with a dash")
		}
		body := f.PsFlag[1:]
		if body == "" {
			return errors.New("flags must have a flag name")
		}
		// PsFlag bodies are matched by string prefix at runtime, so multi-byte
		// runes are fine. Use rune count rather than byte length here.
		if utf8.RuneCountInString(body) < 2 {
			return errors.New("powershell flags must be more than one character")
		}
		if containsWhitespace(body) {
			return fmt.Errorf("powershell flag names cannot contain whitespace: %q", f.PsFlag)
		}
		if f.ShortFlag != "" || f.LongFlag != "" {
			return errors.New("powershell flags cannot have short or long flags defined")
		}
	}

	if f.Type == "" {
		return errors.New("flags must have a type")
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

func isASCIILetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

func isValidArgumentType(t ArgumentType) bool {
	switch t {
	case StringArgument, IntArgument, FloatArgument, BoolArgument, FileArgument, DirArgument, FileDirArgument:
		return true
	}
	return false
}
