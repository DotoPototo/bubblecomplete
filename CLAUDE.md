# CLAUDE.md

## Repo Layout

- Root is the `bubblecomplete` library.
- `example/` is a subpackage of the root module (`package main`). Run with
  `go run ./example` from the repo root. API changes still need a matching
  pass through `example/main.go` / `example/commands.go`.

## Gotchas

- `textinput.Model` mutations via `SetStyles`/`SetWidth` need a pointer to the
  field. With a value receiver on `Model`, calling `m.input.SetStyles(...)`
  mutates a discarded copy. Use a pointer-to-input helper.
- `tea.KeyPressMsg.String()` returns `"space"` for the space key. Don't gate
  printable-text logic on `len(msg.String()) > 1` — use `msg.Text` instead.
