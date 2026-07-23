# Changelog

## v2.0.0 - 2026-07-23

### Breaking changes

- The module path is now `github.com/dotopototo/bubblecomplete/v2`. Update
  imports and run `go get github.com/dotopototo/bubblecomplete/v2`.
- Built on Bubble Tea v2 (`charm.land/bubbletea/v2`, `bubbles/v2`,
  `lipgloss/v2`). Host programs must migrate to Bubble Tea v2.
- `Model.View()` returns a `tea.View`. Use the new `Model.Render()` where a
  string is needed.
- `Model.Err` is removed. Use `Model.Error()`.
- The `Completion` and `Argument` interfaces are no longer exported. They
  could not be implemented outside the package.
- Style fields on `Model` (`ValidCommandStyle`, `InvalidCommandStyle`, and
  friends) moved into the `Styles` struct. Configure with `WithStyles` or
  `SetStyles`, starting from `DefaultStyles()`.
- Requires Go 1.26 or newer.

### Added

- Opt-in filesystem path completion (`WithFilesystemCompletions`): live
  candidates for file and dir arguments, Tab accept with shell-style
  drill-down into directories, path validity colouring, tilde and quote
  handling, auto-quoting of names with spaces, symlink-aware descriptions,
  and a bounded directory cache. Tune with `WithFilesystemCompletionLimit`
  and `WithHiddenFiles`.
- Functional options on `New`: `WithHistoryLimit`, `WithKeyMap`,
  `WithStyles`, `WithPlaceholder`, `WithIcons`, `WithScrollbar`,
  `WithCompletionsPosition`, and more.
- Structured validation errors: `ValidationError` carries a
  `ValidationErrorKind` and unwraps to the underlying error. Read via
  `Model.ValidationError()`.
- Configurable key bindings via `KeyMap` and `DefaultKeyMap()`.
- Hardened history persistence: atomic writes, a 10 MiB read cap, and
  `ClearHistory`.

### Fixed

- Token-aware command and flag detection. Quoted content no longer produces
  false flag matches.
- The tokenizer handles invalid UTF-8 input safely.
- Completion rows clamp to the terminal width. Long names truncate with an
  ellipsis instead of overflowing on narrow terminals.
