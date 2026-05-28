package bubblecomplete

// Option configures a Model at construction. Pass to New.
type Option func(*Model)

// WithHistoryLimit sets the maximum number of history entries to retain.
// Values of zero or less disable history entirely. Safe to call in any order
// relative to WithHistoryFilePath; already-loaded history is re-capped to the
// new limit.
func WithHistoryLimit(n int) Option {
	return func(m *Model) {
		m.HistoryLimit = n
		if n <= 0 {
			m.History = nil
			m.input.SetSuggestions(nil)
			return
		}
		if len(m.History) > n {
			m.History = m.History[:n]
			m.input.SetSuggestions(m.History)
		}
	}
}

// WithHistoryFilePath enables history persistence to the given JSON file.
// Loads any existing history. Errors are surfaced via Model.Error.
func WithHistoryFilePath(path string) Option {
	return func(m *Model) { m.SetHistoryFilePath(path) }
}

// WithPlaceholder sets the input placeholder text.
func WithPlaceholder(s string) Option {
	return func(m *Model) { m.SetPlaceholder(s) }
}

// WithWidth overrides the width passed to New.
func WithWidth(w int) Option {
	return func(m *Model) { m.SetWidth(w) }
}

// WithCompletionRows sets the number of completion rows shown before scrolling.
// Values less than 1 are clamped to 1.
func WithCompletionRows(n int) Option {
	if n < 1 {
		n = 1
	}
	return func(m *Model) { m.CompletionRows = n }
}

// WithCompletionsPosition places the completion list above or below the input.
func WithCompletionsPosition(p Position) Option {
	return func(m *Model) { m.CompletionsPosition = p }
}

// WithCompletionsOffset sets a fixed left offset for the completion list.
func WithCompletionsOffset(n int) Option {
	return func(m *Model) { m.CompletionsOffset = n }
}

// WithIndentCompletions controls whether the completion list indents to the active token.
func WithIndentCompletions(b bool) Option {
	return func(m *Model) { m.IndentCompletions = b }
}

// WithAutotrim controls whether input is trimmed on submit.
func WithAutotrim(b bool) Option {
	return func(m *Model) { m.Autotrim = b }
}

// WithScrollbar toggles the vertical scrollbar on the completion list.
func WithScrollbar(b bool) Option {
	return func(m *Model) { m.ShowScrollbar = b }
}

// WithBorderScroll toggles colored border indicators for scroll position.
func WithBorderScroll(b bool) Option {
	return func(m *Model) { m.ShowBorderScroll = b }
}

// WithIcons toggles type indicator icons in completion rows.
func WithIcons(b bool) Option {
	return func(m *Model) { m.ShowIcons = b }
}

// WithDescriptions toggles the description column next to each completion name.
func WithDescriptions(b bool) Option {
	return func(m *Model) { m.ShowDescriptions = b }
}

// WithKeyMap replaces the default key bindings.
func WithKeyMap(k KeyMap) Option {
	return func(m *Model) { m.SetKeyMap(k) }
}

// WithStyles replaces the default styles.
func WithStyles(s Styles) Option {
	return func(m *Model) { m.SetStyles(s) }
}

// WithFilesystemCompletions enables live filesystem completion for values
// bound to FileArgument, DirArgument, or FileDirArgument. Default false.
//
// When enabled, the completion list for an active file/dir value is replaced
// wholesale with matching entries from the relevant parent directory. Tab
// cycles through candidates the same way it does for command/flag rows.
// See also [WithFilesystemCompletionLimit] and [WithHiddenFiles].
func WithFilesystemCompletions(b bool) Option {
	return func(m *Model) { m.FilesystemCompletions = b }
}

// WithFilesystemCompletionLimit caps the number of filesystem candidates
// shown per keystroke. Default 200. Values ≤ 0 are clamped to 1 at the
// use site (so direct mutation of the public field is safe).
func WithFilesystemCompletionLimit(n int) Option {
	return func(m *Model) { m.FilesystemCompletionLimit = n }
}

// WithHiddenFiles surfaces dotfiles in filesystem completion candidates
// even when the typed basename prefix does not start with ".". Default
// false (dotfiles only surface when the user explicitly types a leading
// "."). Has no effect when WithFilesystemCompletions is false.
func WithHiddenFiles(b bool) Option {
	return func(m *Model) { m.HiddenFiles = b }
}
