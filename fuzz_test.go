package bubblecomplete

import (
	"reflect"
	"testing"

	tea "charm.land/bubbletea/v2"
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
		first := validateCommandInput(input, TestCommands)
		second := validateCommandInput(input, TestCommands)
		// Determinism: same input must produce equivalent error state.
		// Catches any hidden state mutation in validation or its helpers.
		if (first == nil) != (second == nil) {
			t.Errorf("nondeterministic validation for %q: first=%v second=%v", input, first, second)
		}
		if first != nil && second != nil && first.Error() != second.Error() {
			t.Errorf("validation message drifted between calls for %q:\n  first:  %q\n  second: %q", input, first.Error(), second.Error())
		}
	})
}

func FuzzGetCompletions(f *testing.F) {
	for _, s := range seedInputs {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, input string) {
		first, prefix1 := getCompletions(input, TestCommands)
		second, prefix2 := getCompletions(input, TestCommands)
		// Determinism: same input, same TestCommands → same result both
		// times. Catches any hidden state in completion generation.
		if prefix1 != prefix2 {
			t.Errorf("match prefix drift for %q: %q vs %q", input, prefix1, prefix2)
		}
		if len(first) != len(second) {
			t.Errorf("completion count drift for %q: %d vs %d", input, len(first), len(second))
			return
		}
		for i := range first {
			if first[i].getName() != second[i].getName() {
				t.Errorf("completion[%d] drift for %q: %q vs %q", i, input, first[i].getName(), second[i].getName())
			}
		}
	})
}

// quoteSeedInputs covers the quote/path inputs worth running under the
// stripQuotes and resolvePath fuzz targets: bare values, matched and
// unbalanced quotes, mixed quote chars, tilde forms, separators, and the
// pathological "two quotes" cases. Standalone from seedInputs because the
// path-resolver targets exercise narrower-but-still-tricky inputs.
var quoteSeedInputs = []string{
	"",
	"a",
	`"`,
	`'`,
	`""`,
	`''`,
	`"a"`,
	`'a'`,
	`"a`,
	`'a`,
	`"a'`,
	`'a"`,
	`"foo"bar"`,
	"~",
	"~/",
	"~/foo",
	"~/foo/bar",
	"~user/foo",
	"./foo",
	"../foo",
	"/abs/path",
	"/",
	"foo",
	"日本",
	"\x00",
}

func FuzzStripQuotes(f *testing.F) {
	for _, s := range quoteSeedInputs {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		unquoted, opener := stripQuotes(s)

		// Invariant 1: opener is one of {0, '"', '\''}; nothing else can leak.
		if opener != 0 && opener != '"' && opener != '\'' {
			t.Errorf("stripQuotes returned non-quote opener %q for %q", opener, s)
		}

		// Invariant 2: when opener is 0, the result is the input verbatim.
		// stripQuotes never strips anything from a non-quote-led input.
		if opener == 0 && unquoted != s {
			t.Errorf("opener=0 but input changed: %q → %q", s, unquoted)
		}

		// Invariant 3: when opener is non-zero, the input started with that
		// quote and the unquoted text does not start with another opener
		// of the same kind. Catches accidental double-strip.
		if opener != 0 {
			if len(s) == 0 || rune(s[0]) != opener {
				t.Errorf("opener %q but input %q didn't start with it", opener, s)
			}
			// stripQuotes removes at most two bytes (opener + matching
			// closer); the result length must be within [len(s)-2, len(s)-1].
			if len(unquoted) < len(s)-2 || len(unquoted) > len(s)-1 {
				t.Errorf("stripQuotes removed unexpected byte count: %q (len %d) → %q (len %d)",
					s, len(s), unquoted, len(unquoted))
			}
		}
	})
}

func FuzzResolvePath(f *testing.F) {
	for _, s := range quoteSeedInputs {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, typed string) {
		const (
			cwd  = "/work"
			home = "/Users/jane"
		)

		// Run both expandTilde modes — neither must panic, hang, or return
		// nonsensical offsets for any input.
		for _, expand := range []bool{true, false} {
			fullClean, parent, base, userPrefix := resolvePath(typed, cwd, home, expand)

			// Invariant 1: empty input → all empty returns.
			if typed == "" {
				if fullClean != "" || parent != "" || base != "" || userPrefix != "" {
					t.Errorf("empty input should return empty: full=%q parent=%q base=%q prefix=%q",
						fullClean, parent, base, userPrefix)
				}
				continue
			}

			// Invariant 2: userPrefix is a prefix of the typed string after
			// the bare-"~" normalisation (which converts "~" to "~/").
			normalised := typed
			if normalised == "~" {
				normalised = "~/"
			}
			if userPrefix != "" && !startsWith(normalised, userPrefix) {
				t.Errorf("userPrefix %q is not a prefix of normalised typed %q (raw %q)",
					userPrefix, normalised, typed)
			}

			// Invariant 3: parent ends with a separator when base is empty.
			// This is the contract candidate generation relies on.
			if base == "" && parent != "" && !endsInSeparator(parent) {
				t.Errorf("base is empty but parent %q doesn't end in a separator", parent)
			}
		}
	})
}

// startsWith is a fuzz-helper alternative to strings.HasPrefix so the fuzz
// target stays free of the strings package import within this file.
func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func FuzzRender(f *testing.F) {
	for _, s := range seedInputs {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, input string) {
		m, err := New(TestCommands, 100)
		if err != nil {
			t.Fatal(err)
		}
		// Drive the input through Update so the model reaches a realistic
		// state, not just a SetValue-ed one.
		for _, r := range input {
			m, _ = m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
		}
		// Render must be idempotent — a second call on the same model state
		// must produce byte-identical output. Catches any future regression
		// that puts side effects back into the render path.
		first := m.Render()
		second := m.Render()
		if first != second {
			t.Errorf("Render not idempotent for input %q", input)
		}
	})
}
