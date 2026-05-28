# Bubblecomplete

Bubblecomplete is a command suggestion and autocompletion component for [Bubble Tea](https://github.com/charmbracelet/bubbletea) applications.

![example/main.go](./.github/example.gif)

## Installing

First use `go get` to install the latest version of the library.

```shell
go get -u github.com/dotopototo/bubblecomplete
```

Or alternatively, to get a specific branch:

```shell
go get -u github.com/dotopototo/bubblecomplete@beta
```

Next include bubblecomplete in your application.

```go
import "github.com/dotopototo/bubblecomplete"
```

## Usage

See the [example](./example/main.go) for a full working implementation. Run it from the repo root:

```shell
go run ./example
```

The command structure expects the following:

```
command [subcommands] [flags] [positionalArguments]
```

- A command has **either** subcommands **or** positional arguments — never both.
  > `git stash` has subcommands (`pop`, `apply`, `drop`, …); `git push` has positional arguments (`origin main`).
- Subcommands can be nested arbitrarily deep.
  > `git stash pop` is a subcommand of `stash`, which is a subcommand of `git`.
- Any command or subcommand can also have flags, in any combination with positional arguments.
  > `cat file.txt -n` has both a positional argument and a flag.
- The order of positional arguments matters.
  > `cp file.txt destination` is not the same as `cp destination file.txt`.
- Flags marked `Persistent: true` are inherited by every subcommand of the command they're declared on.
  > A `--help` flag on `git` is also available as `git stash --help`.

These rules are enforced by `Command.Validate()`, which `New` calls on every supplied command. Building a `Model` with conflicting subcommands and positional arguments, duplicate flag aliases, or invalid argument types returns an error.

#### Setup

Create a slice of `bubblecomplete.Command` structs to represent the available commands, with `bubblecomplete.PositionalArgument` structs for any required arguments and `bubblecomplete.Flag` structs for any flags that can be passed.

```go
var commands = []*bubblecomplete.Command{
	{
		Command:     "cp",
		Description: "Copy files and directories",
		PositionalArguments: []*bubblecomplete.PositionalArgument{
			{
				Name:        "file",
				Description: "File to copy",
				Type:        bubblecomplete.FileDirArgument,
				Required:    true,
			},
			{
				Name:        "destination",
				Description: "Destination to copy the file to",
				Type:        bubblecomplete.DirArgument,
				Required:    true,
			},
		},
		Flags: []*bubblecomplete.Flag{
			{
				ShortFlag:   "-r",
				Description: "Copy directories recursively",
				Type:        bubblecomplete.BoolArgument,
			},
		},
	},
}
```

#### Commands

| Field               | Description                                                                            | Type                                   |
| ------------------- | -------------------------------------------------------------------------------------- | -------------------------------------- |
| Command             | The command name                                                                       | `string`                               |
| Description         | A description of the command                                                           | `string`                               |
| SubCommands         | A slice of `bubblecomplete.Command` structs representing subcommands                   | `[]*bubblecomplete.Command`            |
| PositionalArguments | A slice of `bubblecomplete.PositionalArgument` structs representing required arguments | `[]*bubblecomplete.PositionalArgument` |
| Flags               | A slice of `bubblecomplete.Flag` structs representing flags                            | `[]*bubblecomplete.Flag`               |

#### Positional Arguments

| Field       | Description                      | Type                          |
| ----------- | -------------------------------- | ----------------------------- |
| Name        | The argument name                | `string`                      |
| Description | A description of the argument    | `string`                      |
| Type        | The type of the argument         | `bubblecomplete.ArgumentType` |
| Required    | Whether the argument is required | `bool`                        |

#### Flags

| Field       | Description                                                                        | Type                          |
| ----------- | ---------------------------------------------------------------------------------- | ----------------------------- |
| ShortFlag   | Single-ASCII-letter short flag, e.g. `-v`. Use a long or PowerShell flag for non-letter names. | `string` |
| LongFlag    | The long flag identifier i.e. `--verbose`                                          | `string`                      |
| PsFlag      | PowerShell-style flag, e.g. `-Verbose` (two or more characters; mutually exclusive with ShortFlag and LongFlag on the same Flag) | `string` |
| Description | A description of the flag                                                          | `string`                      |
| Type        | The type of argument the flag expects                                              | `bubblecomplete.ArgumentType` |
| Persistent  | A persistent flag is available to all subcommands of the command                   | `bool`                        |

#### Argument Types

| Type            | Description                                                                        |
| --------------- | ---------------------------------------------------------------------------------- |
| StringArgument  | A string argument that can be set to any value                                     |
| IntArgument     | An integer argument that can be set to any integer value                           |
| FloatArgument   | A float argument that can be set to any float value                                |
| BoolArgument    | A presence flag — the flag's presence in the input is true, its absence is false. No value follows the flag. |
| FileArgument    | A file argument that can be set to a valid file path                               |
| DirArgument     | A directory argument that can be set to a valid directory path                     |
| FileDirArgument | A file or directory argument that can be set to a valid file or directory path     |

Create a `bubblecomplete.Model` struct with the commands and set any options you want to set, and assign it to your bubbletea program model.

```go
type model struct {
	bubblecomplete bubblecomplete.Model
}

func createModel() tea.Model {
	bc, err := bubblecomplete.New(commands, 100,
		bubblecomplete.WithHistoryLimit(50),
		bubblecomplete.WithIcons(true),
	)
	if err != nil {
		panic(err)
	}

	m := model{
		bubblecomplete: bc,
	}

	return m
}

func (m model) View() tea.View {
  return m.bubblecomplete.View()
}
```

When composing Bubblecomplete with other content, use `Render()` to get the
component's output as a string:

```go
func (m model) View() tea.View {
  return tea.NewView("Header\n" + m.bubblecomplete.Render())
}
```

## Options

Every option below has a matching `With*` construction option (for example
`WithCompletionRows(5)` or `WithDescriptions(false)`). Most are also public
fields on `Model` so they can be tweaked at runtime — exceptions are noted in
their row. `Styles` and `KeyMap` are private; configure them via the dedicated
`Styles()` / `SetStyles()` and `KeyMap()` / `SetKeyMap()` accessors documented
below.

#### General

| Option                    | Description                                                                              | Default         |
| ------------------------- | ---------------------------------------------------------------------------------------- | --------------- |
| Autotrim                  | Trim extra whitespace from the ends of the input                                         | `true`          |
| CompletionsOffset         | The left margin offset of the completion list                                            | `0`             |
| CompletionsPosition       | The position of the completion list relative to the input                                | `PositionBelow` |
| CompletionRows            | The number of rows to show in the completion list before scrolling                       | `5`             |
| FilesystemCompletions     | Enable live filesystem completion and validity colouring for file/dir argument values    | `false`         |
| FilesystemCompletionLimit | Maximum filesystem candidates surfaced per keystroke                                     | `200`           |
| HiddenFiles               | Surface dotfiles even when the typed prefix does not start with `.`                      | `false`         |
| HistoryFilePath           | The path to a `.json` file to store the command history for persistence between sessions (configure via `WithHistoryFilePath` / `SetHistoryFilePath` — no public field) | -               |
| HistoryLimit              | The maximum number of history entries to store and save                                  | `100`           |
| IndentCompletions         | Indent the completion list to match the current input length                             | `true`          |
| ShowBorderScroll          | Show different border colors around the completion list to indicate scrolling            | `false`         |
| ShowScrollbar             | Show a vertical scrollbar to indicate scrolling                                          | `false`         |
| ShowDescriptions          | Render the description column next to each completion name                               | `true`          |

#### Icons

| Option       | Description                        | Default |
| ------------ | ---------------------------------- | ------- |
| ShowIcons    | Show type indicator icons          | `false` |
| CommandIcon  | Icon for command completions       | `›`     |
| ArgumentIcon | Icon for argument completions      | `◆`     |
| FlagIcon     | Icon for flag completions          | `◇`     |

#### Styles

Styles are grouped on a `Styles` struct accessed via `Styles()` / `SetStyles()`:

```go
s := bc.Styles()
s.Input.Valid = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
s.Completion.Match = lipgloss.NewStyle().Bold(true)
bc.SetStyles(s)
```

| Field                    | Description                                                  | Default           |
| ------------------------ | ------------------------------------------------------------ | ----------------- |
| Input.Valid              | Style for valid user input                                   | (green)           |
| Input.Invalid            | Style for invalid user input                                 | (muted text)      |
| Input.PathValid          | Overlay on the typed path value when it resolves correctly   | (green)           |
| Input.PathPartial        | Overlay on the typed path value while mid-navigation         | (underline)       |
| Input.PathInvalid        | Overlay on the typed path value when it cannot resolve       | (red)             |
| Completion.Match         | Style for the matched prefix in completions                  | (bold pink)       |
| Completion.SelectedRow   | Style for the selected completion row                        | (bold, highlight) |
| Completion.Row           | Style for odd completion rows                                | (subtle bg)       |
| Completion.AltRow        | Style for even completion rows                               | (alt subtle bg)   |
| Completion.Border        | Style for the completions border                             | (rounded border)  |
| Completion.Description   | Style for completion descriptions                            | (muted)           |
| Completion.Icon          | Style for type indicator icons                               | (pink)            |
| Scrollbar.Thumb          | Style for the scrollbar thumb                                | (pink)            |
| Scrollbar.Track          | Style for the scrollbar track                                | (border color)    |

#### Key Bindings

Bindings are exposed via `KeyMap()` / `SetKeyMap()`:

```go
import "charm.land/bubbles/v2/key"

k := bc.KeyMap()
k.Submit = key.NewBinding(key.WithKeys("enter", "ctrl+s"))
bc.SetKeyMap(k)
```

| Field            | Description                                | Default              |
| ---------------- | ------------------------------------------ | -------------------- |
| NextCompletion   | Cycle forward through completions          | `tab`, `ctrl+n`      |
| PrevCompletion   | Cycle backward through completions         | `shift+tab`, `ctrl+p`|
| AcceptCompletion | Accept the selected or inline completion   | `right`, `ctrl+e`    |
| Submit           | Submit the current input                   | `enter`              |
| HistoryPrev      | Walk back through command history          | `up`                 |
| HistoryNext      | Walk forward through command history       | `down`               |

### History

- The newest command is at `History[0]`. Trimming keeps the newest `HistoryLimit` entries.
- A `HistoryLimit` of zero or less disables history entirely.
- Adjacent duplicates are suppressed: pressing enter twice on the same input only stores one entry.
- Invalid commands are stored — history is "what the user submitted," not "what validated."
- `WithHistoryFilePath(path)` (or `SetHistoryFilePath`) loads history from a JSON file on startup and saves atomically on each submit. Errors during load or save surface via `Model.Error()`.

### Input Parsing

- Only the ASCII space character (`U+0020`) separates tokens. Tabs, newlines, and other Unicode whitespace are part of the token they appear in, not separators.
- Tokens preserve their quote characters in their raw form. Inside double or single quotes, spaces become part of the token until the matching closing quote (or end of input for unclosed quotes).
- The same tokenizer powers completion, validation, and quote-error reporting — they always see the same parse of the input.
- Out of scope: shell escapes (`\"`), adjacent quoted segments (`"foo""bar"` is two tokens, not one), variable expansion, and globs. See [Non-Goals](#non-goals).

### Filesystem Argument Validation

`FileArgument`, `DirArgument`, and `FileDirArgument` validate the value against the real filesystem using `os.Stat`. The value is run through a shared resolver first: a leading `~` (or `~/`) is expanded against `$HOME`; relative paths are joined with the process's working directory. Unclosed quotes (`"foo` with no closing `"`) surface as `UnclosedQuote`, matching how `StringArgument` already validates. Single-quoted values suppress tilde expansion, mirroring shell semantics. A configurable working-directory / filesystem abstraction is on the roadmap.

### Filesystem Completions

`WithFilesystemCompletions(true)` enables live filesystem completion and per-token validity colouring for any `FileArgument`, `DirArgument`, or `FileDirArgument` value the user is editing. Default is off; the feature is opt-in.

When active, the completion list is replaced wholesale with matching entries from the relevant parent directory — directories first (with a trailing `/` for navigation), then files, sorted case-fold. `Tab` cycles forward, `Shift+Tab` cycles backward, `Right` (or typing) accepts. Tab-accepting a directory adds the trailing slash so the next keystroke drills into it. When there's only **one** candidate, `Tab` auto-accepts immediately and exits cycling — so a second `Tab` lists the children of the just-accepted directory, matching shell tab-completion behaviour.

If the directory contains more matches than `FilesystemCompletionLimit` (default 200), the list is capped and a subtle footer appears below the completion box. The footer wording reflects what we know: `+ N more — type to narrow` when N verified candidates were dropped after the kind filter, or `N symlinks unresolved — narrow to filter` (weaker wording) when only the symlink stat budget was exhausted and we couldn't verify those entries' kinds. Both forms combine when both kinds of drop happen. The footer is informational only — it is NOT a selectable completion row, so `Tab` cycling can't accidentally accept it as input.

The typed value gets a coloured overlay reflecting its filesystem state:

- **green** — resolves to an entry of the expected kind
- **underlined white** — strict prefix of one or more existing entries (mid-navigation)
- **red** — does not resolve, or resolves to the wrong kind

The invariant the feature guarantees, *when the overlay is shown*: green ⇔ Enter accepts. Classifier and `validatePath` resolve and stat by the same rules. The overlay can be suppressed by the limitations listed below (overflow, cycling); in those cases a green-resolving path is still accepted on Enter, but the colour is absent.

**Quoting.** Values are quote-aware: typing `cat "~/My Doc<Tab>"` completes inside the quotes. If the user types `cat ~/M<Tab>` and the matched basename contains a space, the inserted token is auto-quoted with double quotes. Equals-form flag values (`--path=~/foo`) preserve the `--path=` prefix; only the value portion is coloured.

**Tilde policy.** `~` and `~/` expand outside quotes and inside `"..."`; not inside `'...'`. This deviates pragmatically from strict POSIX (which would not expand inside `"..."`) — users who quote because of spaces still expect `~` to work.

**Modal context.** While editing a path value, the completion list shows only filesystem entries. Flag suggestions are intentionally hidden in this mode — to surface them, move the cursor out of the value position (type a space or backspace out).

**Caching.** Directory reads are cached for 2 seconds across up to 64 distinct parent directories. The first keystroke into a new directory pays one `os.ReadDir`; subsequent keystrokes avoid the repeat read but still scan the cached entries and may stat a bounded number of symlinks. Permission errors and missing directories are cached too, so a bad path doesn't trigger a retry storm.

**Documented limitations.**

- **Permission denied** silently falls back to the argument hint row. The whole-input style still signals the error via the validation system, but no specific "permission denied" row appears in the completion list.
- **Devices, sockets, pipes** never appear as completion candidates regardless of `ArgumentType`. They still validate per kind if typed directly (a `FileArgument` accepts `/dev/null` because `!IsDir()` matches).
- **Filenames containing literal quote characters** (`"` or `'`) are inserted verbatim and cannot be re-parsed by the tokenizer as a single token. Vanishingly rare in practice.
- **Multi-token unquoted paths** like `cat ~/My Doc` are not recovered: once a space is typed, the tokenizer has split the path. Open a quote first (`cat "~/My Doc`) or rely on auto-quote (`cat ~/M<Tab>` picks the spaced basename).
- **Cold huge directories** (10k+ entries) produce a one-time stall on the first read. Subsequent keystrokes within the TTL window avoid the repeat read.
- **Case sensitivity** is a heuristic: case-sensitive on Linux, case-insensitive on macOS and Windows. APFS case-sensitive volumes on macOS will surface completions the filesystem won't actually open.
- **Unicode normalisation** on macOS HFS+: the filesystem stores filenames in NFD form, but users typically type in NFC. A typed `~/Documents/résumé.pdf` (NFC) won't match the on-disk `résumé.pdf` (NFD) and the path will silently colour red. Not fixed in v1; would require pulling in `golang.org/x/text/unicode/norm`.
- **Inputs wider than the terminal** lose the validity overlay (the underlying textinput's scroll window can't be reliably indexed into for offset math).
- **During Tab cycling**, the overlay refreshes per-cycle and tracks the cycled candidate's validity. Directory candidates (no trailing space) show their classification per cycle — `pathPartial` (mid-navigation) under `FileArgument`, `pathValid` under `DirArgument`. File candidates carry a trailing space (bash-style "token done"), which makes `activeFileArgument` inactive for that preview — files cycle without an overlay colour. The cycled file values are all real files by construction (they survived the candidate kind filter), so the missing colour signals nothing.

## Roadmap

- [x] Update to bubbletea v2
- [x] Support PowerShell style flags
- [ ] Support PowerShell aliases for flags i.e. `-v` for `-Verbose`
- [x] Autocomplete for filepaths
  - [x] Underlined white if part of a valid path
  - [x] Green if full valid path
  - [x] Red if invalid path
- [ ] Option to have flags disable other flags if they're mutually exclusive
- [x] Improved documentation comments for public functions and structs
- [x] Wider range of tests for more critical functions, for improved maintainability
- [x] Option to not show the descriptions of the commands, flags etc
- [x] More exposed color options for the completion list, scrolling etc

## Non-Goals

Some behaviors are intentionally out of scope. Don't expect:

- Full POSIX shell parsing (e.g., command substitution, here-docs, redirections)
- Executing commands — Bubblecomplete only reports the submitted string back to the host via `SelectedCommandMsg`
- Shell alias resolution
- Environment variable expansion (`$HOME`, `${FOO}`)
- Glob expansion (`*.go`)
- Command-specific dynamic completions — until a provider API lands in Part 4

If you need any of these, do them in the host application after receiving the submitted command.

## FAQ

### Colors

Bubblecomplete (and Bubble Tea applications in general) work best with a terminal that supports 24-bit color. If you're using a terminal that doesn't support 24-bit color, you may see some odd colors. If you're using a terminal that supports 24-bit color, ensure that it's enabled in your terminal emulator, tmux config (if you're using tmux) etc and that your `$TERM` environment variable is set to a value that supports 24-bit color (such as `xterm-256color`) and your `$COLORTERM` environment variable is set to `truecolor`.

### Concurrency

`Model` is not goroutine-safe. This is intentional: Bubble Tea runs `Update` on a single goroutine, and all model state changes should happen there. Background work should return `tea.Msg` values that the next `Update` consumes; never mutate `Model` from another goroutine.
