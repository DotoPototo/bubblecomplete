# CLAUDE.md

## Repo Layout

- Root is the `bubblecomplete` library.
- `example/` is a separate Go module with `replace github.com/dotopototo/bubblecomplete => ..`.
  API changes in the root need a matching pass through `example/main.go` and
  `example/go.mod`.

## Gotchas

- `textinput.Model` mutations via `SetStyles`/`SetWidth` need a pointer to the
  field. With a value receiver on `Model`, calling `m.input.SetStyles(...)`
  mutates a discarded copy. Use a pointer-to-input helper.
- `tea.KeyPressMsg.String()` returns `"space"` for the space key. Don't gate
  printable-text logic on `len(msg.String()) > 1` — use `msg.Text` instead.
