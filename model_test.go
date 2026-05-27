package bubblecomplete

import (
	"strings"
	"testing"
)

// assertErrorContains shrinks the wantErr + wantMsg dance to one call site.
// wantMsg of "" skips the substring check (handy for "any error will do" cases).
func assertErrorContains(t *testing.T, err error, wantErr bool, wantMsg string) {
	t.Helper()
	if wantErr && err == nil {
		t.Fatalf("expected error containing %q, got nil", wantMsg)
	}
	if !wantErr && err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if wantErr && wantMsg != "" && !strings.Contains(err.Error(), wantMsg) {
		t.Errorf("error %q does not contain %q", err.Error(), wantMsg)
	}
}

func TestCommandValidate(t *testing.T) {
	cases := []struct {
		name      string
		cmd       *Command
		wantErr   bool
		wantMsg   string // substring to match in error; empty = any error
	}{
		{
			name:    "nil command",
			cmd:     nil,
			wantErr: true, wantMsg: "cannot be nil",
		},
		{
			name:    "empty name",
			cmd:     &Command{},
			wantErr: true, wantMsg: "must have a command name",
		},
		{
			name:    "leading whitespace",
			cmd:     &Command{Command: " git"},
			wantErr: true, wantMsg: "leading or trailing whitespace",
		},
		{
			name:    "trailing whitespace",
			cmd:     &Command{Command: "git "},
			wantErr: true, wantMsg: "leading or trailing whitespace",
		},
		{
			name:    "internal whitespace",
			cmd:     &Command{Command: "git push"},
			wantErr: true, wantMsg: "cannot contain whitespace",
		},
		{
			name: "both SubCommands and PositionalArguments",
			cmd: &Command{
				Command:             "x",
				SubCommands:         []*Command{{Command: "a"}},
				PositionalArguments: []*PositionalArgument{{Name: "p", Type: StringArgument}},
			},
			wantErr: true, wantMsg: "cannot define both",
		},
		{
			name: "duplicate sibling subcommands",
			cmd: &Command{
				Command: "x",
				SubCommands: []*Command{
					{Command: "a"},
					{Command: "a"},
				},
			},
			wantErr: true, wantMsg: "duplicate subcommand",
		},
		{
			name: "duplicate positional argument names",
			cmd: &Command{
				Command: "x",
				PositionalArguments: []*PositionalArgument{
					{Name: "p", Type: StringArgument},
					{Name: "p", Type: IntArgument},
				},
			},
			wantErr: true, wantMsg: "duplicate positional argument",
		},
		{
			name: "duplicate flag short alias across flags",
			cmd: &Command{
				Command: "x",
				Flags: []*Flag{
					{ShortFlag: "-r", Type: BoolArgument},
					{ShortFlag: "-r", Type: StringArgument},
				},
			},
			wantErr: true, wantMsg: "duplicate flag alias",
		},
		{
			name: "duplicate long alias across short and long flags",
			cmd: &Command{
				Command: "x",
				Flags: []*Flag{
					{LongFlag: "--shared", Type: BoolArgument},
					{ShortFlag: "-s", LongFlag: "--shared", Type: StringArgument},
				},
			},
			wantErr: true, wantMsg: "duplicate flag alias",
		},
		{
			name: "nil flag",
			cmd: &Command{
				Command: "x",
				Flags:   []*Flag{nil},
			},
			wantErr: true, wantMsg: "nil flag",
		},
		{
			name: "nil subcommand",
			cmd: &Command{
				Command:     "x",
				SubCommands: []*Command{nil},
			},
			wantErr: true, wantMsg: "nil subcommand",
		},
		{
			name: "nil positional argument",
			cmd: &Command{
				Command:             "x",
				PositionalArguments: []*PositionalArgument{nil},
			},
			wantErr: true, wantMsg: "nil positional argument",
		},
		{
			name: "valid command with all features",
			cmd: &Command{
				Command: "git",
				SubCommands: []*Command{
					{Command: "commit", Flags: []*Flag{{ShortFlag: "-m", Type: StringArgument}}},
				},
				Flags: []*Flag{
					{LongFlag: "--help", Type: BoolArgument, Persistent: true},
				},
			},
			wantErr: false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.cmd.Validate()
			assertErrorContains(t, err, c.wantErr, c.wantMsg)
		})
	}
}

func TestFlagValidate(t *testing.T) {
	cases := []struct {
		name    string
		flag    *Flag
		wantErr bool
		wantMsg string
	}{
		{
			name:    "nil flag",
			flag:    nil,
			wantErr: true, wantMsg: "cannot be nil",
		},
		{
			name:    "no flag name",
			flag:    &Flag{Type: BoolArgument},
			wantErr: true, wantMsg: "at least one flag",
		},
		{
			name:    "short flag without dash",
			flag:    &Flag{ShortFlag: "r", Type: BoolArgument},
			wantErr: true, wantMsg: "must start with a dash",
		},
		{
			name:    "short flag double dash body",
			flag:    &Flag{ShortFlag: "--", Type: BoolArgument},
			wantErr: true, wantMsg: "cannot be a dash",
		},
		{
			name:    "long flag without two dashes",
			flag:    &Flag{LongFlag: "-message", Type: StringArgument},
			wantErr: true, wantMsg: "must start with two dashes",
		},
		{
			name:    "long flag with whitespace",
			flag:    &Flag{LongFlag: "--my flag", Type: StringArgument},
			wantErr: true, wantMsg: "cannot contain whitespace",
		},
		{
			name:    "powershell flag too short",
			flag:    &Flag{PsFlag: "-v", Type: BoolArgument},
			wantErr: true, wantMsg: "more than one character",
		},
		{
			name:    "powershell flag with whitespace",
			flag:    &Flag{PsFlag: "-my flag", Type: StringArgument},
			wantErr: true, wantMsg: "cannot contain whitespace",
		},
		{
			name:    "powershell with short flag combined",
			flag:    &Flag{PsFlag: "-Verbose", ShortFlag: "-v", Type: BoolArgument},
			wantErr: true, wantMsg: "powershell flags cannot have short or long",
		},
		{
			name:    "missing type",
			flag:    &Flag{ShortFlag: "-r"},
			wantErr: true, wantMsg: "must have a type",
		},
		{
			name:    "invalid type",
			flag:    &Flag{ShortFlag: "-r", Type: "weird"},
			wantErr: true, wantMsg: "invalid type",
		},
		{
			name:    "valid bool short flag",
			flag:    &Flag{ShortFlag: "-r", Type: BoolArgument},
			wantErr: false,
		},
		{
			name:    "valid short + long",
			flag:    &Flag{ShortFlag: "-m", LongFlag: "--message", Type: StringArgument},
			wantErr: false,
		},
		{
			name:    "valid PsFlag",
			flag:    &Flag{PsFlag: "-Verbose", Type: BoolArgument},
			wantErr: false,
		},
		{
			name:    "short flag non-ASCII rune (multi-byte)",
			flag:    &Flag{ShortFlag: "-é", Type: BoolArgument},
			wantErr: true, wantMsg: "single ASCII letter",
		},
		{
			name:    "short flag CJK rune",
			flag:    &Flag{ShortFlag: "-中", Type: BoolArgument},
			wantErr: true, wantMsg: "single ASCII letter",
		},
		{
			name:    "PsFlag one-rune non-ASCII body still rejected",
			flag:    &Flag{PsFlag: "-é", Type: BoolArgument},
			wantErr: true, wantMsg: "more than one character",
		},
		{
			name:    "PsFlag two-rune non-ASCII body accepted",
			flag:    &Flag{PsFlag: "-éé", Type: BoolArgument},
			wantErr: false,
		},
		{
			name:    "short flag digit body rejected",
			flag:    &Flag{ShortFlag: "-1", Type: IntArgument},
			wantErr: true, wantMsg: "ASCII letter",
		},
		{
			name:    "short flag punctuation body rejected",
			flag:    &Flag{ShortFlag: "-?", Type: BoolArgument},
			wantErr: true, wantMsg: "ASCII letter",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.flag.Validate()
			assertErrorContains(t, err, c.wantErr, c.wantMsg)
		})
	}
}

func TestPositionalArgumentValidate(t *testing.T) {
	cases := []struct {
		name    string
		arg     *PositionalArgument
		wantErr bool
		wantMsg string
	}{
		{
			name:    "nil",
			arg:     nil,
			wantErr: true, wantMsg: "cannot be nil",
		},
		{
			name:    "empty name",
			arg:     &PositionalArgument{Type: StringArgument},
			wantErr: true, wantMsg: "must have a name",
		},
		{
			name:    "missing type",
			arg:     &PositionalArgument{Name: "file"},
			wantErr: true, wantMsg: "must have a type",
		},
		{
			name:    "invalid type",
			arg:     &PositionalArgument{Name: "file", Type: "weird"},
			wantErr: true, wantMsg: "invalid type",
		},
		{
			name:    "valid",
			arg:     &PositionalArgument{Name: "file", Type: FileArgument, Required: true},
			wantErr: false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.arg.Validate()
			assertErrorContains(t, err, c.wantErr, c.wantMsg)
		})
	}
}
