# Bubblecomplete

<p>
  <a href="https://pkg.go.dev/github.com/dotopototo/bubblecomplete"><img src="https://pkg.go.dev/badge/github.com/dotopototo/bubblecomplete.svg" alt="Go Reference"></a>
  <a href="https://github.com/DotoPototo/bubblecomplete/releases"><img src="https://img.shields.io/github/v/release/DotoPototo/bubblecomplete?include_prereleases&sort=semver" alt="Latest Release"></a>
  <a href="https://github.com/DotoPototo/bubblecomplete/actions/workflows/ci.yml"><img src="https://github.com/DotoPototo/bubblecomplete/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://goreportcard.com/report/github.com/dotopototo/bubblecomplete"><img src="https://goreportcard.com/badge/github.com/dotopototo/bubblecomplete" alt="Go Report"></a>
</p>

A polished command-prompt component for [Bubble Tea](https://github.com/charmbracelet/bubbletea). Type, Tab through suggestions, browse the filesystem inline, and hand the validated input back to the host. No parser required.

![demo](./.github/example.gif)

## Features

- **Tab cycling** through commands, subcommands, flags, and positional arguments
- **Live filesystem completion** for path arguments, with green / underline / red validity overlay on the typed token
- **Shell ergonomics** — trailing slash on dir accepts, trailing space on file accepts, auto-quote on basenames with spaces, single-match auto-accept, drill-down on the next Tab
- **Quote-aware parsing** — `'…'`, `"…"`, unclosed, and equals-form `--flag=…` all do what you'd expect
- **Persistent history** — JSON-backed across sessions, with inline suggestions for last-typed commands
- **Structured validation errors** — route on `ValidationErrorKind`, no string-matching
- **Themable** — styles, key bindings, icons, and layout are all configurable

## Install

```shell
go get github.com/dotopototo/bubblecomplete
```

## Defining commands

Commands are plain struct literals — no DSL, no builder. A command has **either** subcommands **or** positional arguments (never both), and any number of flags:

```go
var commands = []*bubblecomplete.Command{
    {
        Command:     "cp",
        Description: "Copy files and directories",
        PositionalArguments: []*bubblecomplete.PositionalArgument{
            {Name: "source", Type: bubblecomplete.FileDirArgument, Required: true},
            {Name: "dest",   Type: bubblecomplete.DirArgument,     Required: true},
        },
        Flags: []*bubblecomplete.Flag{
            {ShortFlag: "-r", Description: "Recursive", Type: bubblecomplete.BoolArgument},
        },
    },
}
```

Argument types: `StringArgument`, `IntArgument`, `FloatArgument`, `BoolArgument`, `FileArgument`, `DirArgument`, `FileDirArgument`. Flags can be short (`-v`), long (`--verbose`), or PowerShell-style (`-Verbose`), and may be marked `Persistent: true` to propagate to subcommands. See the [godoc](https://pkg.go.dev/github.com/dotopototo/bubblecomplete#Command) for the full struct surface.

## Usage

Embed the component in your Bubble Tea model and forward `Update` / `View`:

```go
import (
    "log"

    tea "charm.land/bubbletea/v2"
    "github.com/dotopototo/bubblecomplete"
)

type model struct {
    bc bubblecomplete.Model
}

func initialModel() tea.Model {
    bc, err := bubblecomplete.New(commands, 100,
        bubblecomplete.WithFilesystemCompletions(true),
        bubblecomplete.WithHistoryLimit(50),
    )
    if err != nil {
        log.Fatal(err)
    }
    return model{bc: bc}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmd tea.Cmd
    m.bc, cmd = m.bc.Update(msg)
    if sel, ok := msg.(bubblecomplete.SelectedCommandMsg); ok {
        // sel.Command is the validated input; sel.Err is non-nil on validation failure.
        _ = sel
    }
    return m, cmd
}

func (m model) View() tea.View { return m.bc.View() }
```

A complete runnable program lives in [`./example`](./example/main.go) — `go run ./example` to try it.

## Filesystem completions

Turn it on with `WithFilesystemCompletions(true)`. As the user types a value for a `FileArgument`, `DirArgument`, or `FileDirArgument`, the completion list shows real filesystem entries from the relevant parent directory — directories first (with trailing `/`), then files, sorted case-fold. Tab cycles, Shift+Tab reverses, Right (or any keystroke) accepts. Single matches auto-accept, and a second Tab drills into the just-accepted directory — matching standard shell tab-completion behaviour.

The typed value gets a validity overlay:

| State | Meaning |
| --- | --- |
| **green** | resolves to an entry of the expected kind |
| **underlined** | strict prefix of one or more existing entries (mid-navigation) |
| **red** | doesn't resolve, or resolves to the wrong kind |

> **Invariant:** when the overlay is shown, green means Enter accepts. The classifier and submit-time validator share a resolver, so the colour can't lie about whether submitting would succeed. (The overlay is suppressed in a few documented cases — wide inputs that overflow the terminal, file cycling — where Enter still behaves correctly but the colour is absent.)

Tilde expansion (`~`, `~/`), quote handling (`"…"`, `'…'`, equals-form `--path="…"`), auto-quote on basenames with spaces, and a subtle `+ N more` footer when the directory exceeds the candidate limit are all built in. Full behaviour, limits, and caveats live in the [godoc](https://pkg.go.dev/github.com/dotopototo/bubblecomplete#WithFilesystemCompletions).

## Configuration

Common options:

| Option                  | Default | Description |
| ----------------------- | ------- | ----------- |
| `FilesystemCompletions` | `false` | Opt into live filesystem completion + validity overlay |
| `HistoryLimit`          | `100`   | Max history entries; `≤ 0` disables history |
| `HistoryFilePath`       | —       | JSON file for history persistence (via `WithHistoryFilePath` / `SetHistoryFilePath`) |
| `ShowIcons`             | `false` | Type-indicator icons on each row |
| `ShowDescriptions`      | `true`  | Render the description column |
| `CompletionRows`        | `5`     | Visible rows before scrolling |
| _…and more_             | —       | _Layout, scrollbar, icon glyphs, hidden files, candidate cap — see [`Model` godoc](https://pkg.go.dev/github.com/dotopototo/bubblecomplete#Model)_ |

Every option has a matching `With*` construction function, and most are also mutable public fields on `Model` for runtime tweaks. Styles and key bindings live on their respective accessors: [`Styles()` / `SetStyles`](https://pkg.go.dev/github.com/dotopototo/bubblecomplete#Model.Styles) and [`KeyMap()` / `SetKeyMap`](https://pkg.go.dev/github.com/dotopototo/bubblecomplete#Model.KeyMap).

## Roadmap

- [ ] PowerShell flag aliases (e.g. `-v` for `-Verbose`)
- [ ] Mutually-exclusive flags
- [ ] Pluggable completion provider (for dynamic / remote sources)

## Non-Goals

Bubblecomplete is a prompt component, not a shell. Don't expect:

- Executing commands — the component hands the submitted string back via `SelectedCommandMsg` and the host runs it
- Full POSIX shell parsing (substitution, here-docs, redirections)
- Shell alias resolution
- Environment variable expansion (`$HOME`, `${FOO}`)
- Glob expansion (`*.go`)

## FAQ

**Why do colours look off?** Bubble Tea components render best in 24-bit terminals. Make sure your terminal (and tmux config, if applicable) advertises truecolor — `$TERM=xterm-256color` and `$COLORTERM=truecolor`.

**Is `Model` goroutine-safe?** No, and intentionally so. Bubble Tea runs `Update` on a single goroutine and all state changes happen there. Background work should return `tea.Msg` values for the next `Update` to consume — never mutate `Model` from another goroutine.

## License

[MIT](./LICENSE)
