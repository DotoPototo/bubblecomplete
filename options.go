package bubblecomplete

// Option configures a Model at construction. Pass to New.
type Option func(*Model)

// WithHistoryLimit sets the maximum number of history entries to retain.
func WithHistoryLimit(n int) Option {
	return func(m *Model) { m.HistoryLimit = n }
}

// WithHistoryFilePath enables history persistence to the given JSON file.
// Loads any existing history. Errors are surfaced via Model.Err.
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
