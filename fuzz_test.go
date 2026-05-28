package bubblecomplete

import (
	"reflect"
	"testing"
)

// seedInputs covers the parse/validate/complete cases worth running under
// every fuzz target: empty, whitespace, quote variants (closed, unclosed,
// mixed), flag forms, multi-byte and ANSI-ish content, and long-ish strings.
var seedInputs = []string{
	"",
	" ",
	"  ",
	"git",
	"git ",
	"git status",
	"git commit -m \"hello world\"",
	"git commit --message=value",
	"cat \"file with spaces.txt\"",
	"cat 'single quoted'",
	"cat \"unterminated",
	"cat 'mix\"quote'",
	"--",
	"-",
	"--=",
	"--flag=",
	"--flag=\"\"",
	"-xyz",
	"-1",
	// Uses a deliberately non-existent file so the seed doesn't depend on
	// repo state; validation reaches the int-value path before the positional
	// path either way.
	"ps -intarg -1 ./fuzz-nonexistent.txt",
	"日本 file",
	"🐛",
	"a\tb",
	"\x00",
	"\"\"",
	"''",
}

func FuzzTokenize(f *testing.F) {
	for _, s := range seedInputs {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, input string) {
		// Both APIs must survive any input without panicking. tokenize is the
		// foundation; splitInput is the legacy wrapper that calls it.
		tokens := tokenize(input)
		raws := splitInput(input)

		// Invariant 1: splitInput equals the Raw fields of tokenize. Catches
		// drift between the wrapper and the underlying tokenizer.
		want := make([]string, len(tokens))
		for i, tk := range tokens {
			want[i] = tk.Raw
		}
		if len(want) == 0 {
			want = nil
		}
		if !reflect.DeepEqual(raws, want) {
			t.Errorf("splitInput vs tokenize Raw drift for %q:\n  splitInput: %#v\n  tokenize:   %#v", input, raws, want)
		}

		// Invariant 2: every token's recorded byte offsets must slice the
		// original input back to its Raw value. Catches rune-size miscounts
		// (e.g., invalid UTF-8 where len(string(RuneError)) is 3 but the
		// range loop advanced by 1).
		for i, tk := range tokens {
			if tk.Start < 0 || tk.End > len(input) || tk.Start > tk.End {
				t.Errorf("token %d has out-of-range offsets [%d:%d] for input of len %d", i, tk.Start, tk.End, len(input))
				continue
			}
			if got := input[tk.Start:tk.End]; got != tk.Raw {
				t.Errorf("token %d offsets [%d:%d] slice to %q, want Raw %q (input %q)", i, tk.Start, tk.End, got, tk.Raw, input)
			}
		}
	})
}

func FuzzValidateCommandInput(f *testing.F) {
	for _, s := range seedInputs {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, input string) {
		_ = validateCommandInput(input, TestCommands)
	})
}

func FuzzGetCompletions(f *testing.F) {
	for _, s := range seedInputs {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, input string) {
		_, _ = getCompletions(input, TestCommands)
	})
}
