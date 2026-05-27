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

See the [example](./example/main.go) for a full working implementation.

The command structure expects the following:

```
command [subcommands] [flags] [positionalArguments]
```

- A command can have multiple subcommands or positional arguments, and flags
  > Git can have multiple sub commands such as `git commit`, `git stash` and `git status`
- A command can have subcommands _or_ positional arguments, but not both.
  > i.e. the subcommand `stash` in `git stash` has more subcommands like `git stash pop`, `git stash apply`, etc. but the subcommand `push` in `git push` has positional arguments like `git push origin main`
- Any command or subcommand can have flags.
  > Git can have a flag such as `--version` or a subcommand such as `git commit -m "message"`
- Any command or subcommand can have both positional arguments and flags.
  > Cat expects the positional argument for the file and then flags `cat file.txt -n`
- The order of positional arguments is important
  > `cp file.txt destination` is different from `cp destination file.txt`

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
| Subcommands         | A slice of `bubblecomplete.Command` structs representing subcommands                   | `[]*bubblecomplete.Command`            |
| PositionalArguments | A slice of `bubblecomplete.PositionalArgument` structs representing required arguments | `[]*bubblecomplete.PositionalArgument` |
| Flags               | A slice of `bubblecomplete.Flag` structs representing flags                            | `[]*bubblecomplete.Flag`               |

#### Positional Arguments

| Field       | Description                      | Type                          |
| ----------- | -------------------------------- | ----------------------------- |
| Name        | The argument name                | `string`                      |
| Description | A description of the argument    | `string`                      |
| Type        | The type of the argument         | `bubblecomplete.argumentType` |
| Required    | Whether the argument is required | `bool`                        |

#### Flags

| Field       | Description                                                                        | Type                          |
| ----------- | ---------------------------------------------------------------------------------- | ----------------------------- |
| ShortFlag   | The short flag identifier i.e. `-v`                                                | `string`                      |
| LongFlag    | The long flag identifier i.e. `--verbose`                                          | `string`                      |
| PsFlag      | PowerShell style flag i.e. `-verbose` - not compatible with ShortFlag and LongFlag | `string`                      |
| Description | A description of the flag                                                          | `string`                      |
| Type        | The type of argument the flag expects                                              | `bubblecomplete.argumentType` |
| Persistent  | A persistent flag is available to all subcommands of the command                   | `bool`                        |

#### Argument Types

| Type            | Description                                                                        |
| --------------- | ---------------------------------------------------------------------------------- |
| StringArgument  | A string argument that can be set to any value                                     |
| IntArgument     | An integer argument that can be set to any integer value                           |
| FloatArgument   | A float argument that can be set to any float value                                |
| BoolArgument    | A boolean argument that can be set to `true` or `false` (or left empty for `true`) |
| FileArgument    | A file argument that can be set to a valid file path                               |
| DirArgument     | A directory argument that can be set to a valid directory path                     |
| FileDirArgument | A file or directory argument that can be set to a valid file or directory path     |

Create a `bubblecomplete.Model` struct with the commands and set any options you want to set, and assign it to your bubbletea program model.

```go
type model struct {
	bubblecomplete bubblecomplete.Model
}

func createModel() tea.Model {
	bc, err := bubblecomplete.New(commands, 100)
	if err != nil {
		panic(err)
	}
	bc.HistoryLimit = 50

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

#### General

| Option              | Description                                                                              | Default         |
| ------------------- | ---------------------------------------------------------------------------------------- | --------------- |
| Autotrim            | Trim extra whitespace from the ends of the input                                         | `true`          |
| CompletionsOffset   | The left margin offset of the completion list                                            | `0`             |
| CompletionsPosition | The position of the completion list relative to the input                                | `PositionBelow` |
| CompletionRows      | The number of rows to show in the completion list before scrolling                       | `5`             |
| HistoryFilePath     | The path to a `.json` file to store the command history for persistence between sessions | -               |
| HistoryLimit        | The maximum number of history entries to store and save                                  | `100`           |
| IndentCompletions   | Indent the completion list to match the current input length                             | `true`          |
| ShowBorderScroll    | Show different border colors around the completion list to indicate scrolling            | `false`         |
| ShowScrollbar       | Show a vertical scrollbar to indicate scrolling                                          | `false`         |

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

| Field                    | Description                                          | Default           |
| ------------------------ | ---------------------------------------------------- | ----------------- |
| Input.Valid              | Style for valid user input                           | (green)           |
| Input.Invalid            | Style for invalid user input                         | (muted text)      |
| Completion.Match         | Style for the matched prefix in completions          | (bold pink)       |
| Completion.SelectedRow   | Style for the selected completion row                | (bold, highlight) |
| Completion.Row           | Style for odd completion rows                        | (subtle bg)       |
| Completion.AltRow        | Style for even completion rows                       | (alt subtle bg)   |
| Completion.Border        | Style for the completions border                     | (rounded border)  |
| Completion.Description   | Style for completion descriptions                    | (muted)           |
| Completion.Icon          | Style for type indicator icons                       | (pink)            |
| Scrollbar.Thumb          | Style for the scrollbar thumb                        | (pink)            |
| Scrollbar.Track          | Style for the scrollbar track                        | (border color)    |

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

## Roadmap

- [x] Update to bubbletea v2
- [x] Support PowerShell style flags
- [ ] Support PowerShell aliases for flags i.e. `-v` for `-verbose`
- [ ] Autocomplete for filepaths
  - [ ] Underlined white if part of a valid path
  - [ ] Green if full valid path
  - [ ] Red if invalid path
- [ ] Option to have flags disable other flags if they're mutually exclusive
- [ ] Improved documentation comments for public functions and structs
- [x] Wider range of tests for more critical functions, for improved maintainability
- [ ] Option to not show the descriptions of the commands, flags etc
- [x] More exposed color options for the completion list, scrolling etc

## FAQ

### Colors

Bubblecomplete (and Bubble Tea applications in general) work best with a terminal that supports 24-bit color. If you're using a terminal that doesn't support 24-bit color, you may see some odd colors. If you're using a terminal that supports 24-bit color, ensure that it's enabled in your terminal emulator, tmux config (if you're using tmux) etc and that your `$TERM` environment variable is set to a value that supports 24-bit color (such as `xterm-256color`) and your `$COLORTERM` environment variable is set to `truecolor`.
