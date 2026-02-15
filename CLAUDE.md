# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Bubblecomplete is a Go library providing command suggestion and autocompletion for [Bubble Tea](https://github.com/charmbracelet/bubbletea) TUI applications. It supports nested subcommands, positional arguments with type validation, flags (short/long/persistent/PowerShell-style), and command history.

## Build & Test Commands

```bash
go test -v ./...          # Run all tests
go test -run TestName     # Run a single test
go run ./example/main.go  # Run the example application
```

## Architecture

The library is a single Go package (`bubblecomplete`) with clear separation of concerns:

- **model.go** — Core types: `Model`, `Command`, `PositionalArgument`, `Flag`, and the `Completion` interface. The `New()` constructor lives here. Commands, Flags, and PositionalArguments all implement the `Completion` interface (`getName()`, `getDescription()`, `getAutocomplete()`).
- **bubblecomplete.go** — Bubble Tea integration: `Update()` handles keyboard input as a state machine (tab, enter, arrows, backspace). Also contains history management and `splitInput()` which parses input respecting quoted strings.
- **completions.go** — Completion generation logic. `getCompletions()` recursively traverses command hierarchies to determine context-appropriate suggestions. Handles subcommand navigation, positional argument tracking, flag detection (including combined short flags like `-xyz`, PowerShell flags like `-Verbose`), and long flag value syntax (`--flag=value`).
- **validation.go** — Real-time input validation. Recursively validates command input including type checking for arguments (`StringArgument`, `IntArgument`, `FloatArgument`, `BoolArgument`, `FileArgument`, `DirArgument`, `FileDirArgument`). Includes PowerShell flag validation.
- **render.go** — UI rendering with Lipgloss. Handles completion list display, scrolling, and responsive width calculation.
- **styles.go** — Lipgloss style definitions with adaptive colors for light/dark terminal support.
- **test_data.go** — Shared `TestCommands` used across all test files.

## Flag Types

Three flag styles are supported, but PowerShell flags (`PsFlag`) are mutually exclusive with short/long flags on the same Flag struct:
- **ShortFlag** — Unix short form (e.g., `-r`), can be combined (`-xyz`)
- **LongFlag** — Unix long form (e.g., `--recursive`), supports `--flag=value`
- **PsFlag** — PowerShell style (e.g., `-Verbose`), must be 2+ characters

## Command Structure Rules

Commands follow the hierarchy: `command [subcommands] [flags] [positionalArguments]`. A command has EITHER subcommands OR positional arguments, never both. Subcommands can be nested arbitrarily deep (e.g., `git stash pop`). Persistent flags propagate to all subcommands.

## Dependencies

Built on the Charmbracelet stack: `bubbletea` (TUI framework), `bubbles` (UI components — uses `textinput`), `lipgloss` (styling), `termenv` (terminal environment).
