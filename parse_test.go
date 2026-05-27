package bubblecomplete

import (
	"reflect"
	"testing"
)

func TestTokenize(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  []token
	}{
		{
			name:  "empty input",
			input: "",
			want:  nil,
		},
		{
			name:  "single word",
			input: "git",
			want: []token{
				{Raw: "git", Unquoted: "git", Start: 0, End: 3, Closed: true},
			},
		},
		{
			name:  "two words",
			input: "git commit",
			want: []token{
				{Raw: "git", Unquoted: "git", Start: 0, End: 3, Closed: true},
				{Raw: "commit", Unquoted: "commit", Start: 4, End: 10, Closed: true},
			},
		},
		{
			name:  "double-quoted token",
			input: `cat "my file"`,
			want: []token{
				{Raw: "cat", Unquoted: "cat", Start: 0, End: 3, Closed: true},
				{Raw: `"my file"`, Unquoted: "my file", Start: 4, End: 13, Quoted: true, Quote: '"', Closed: true},
			},
		},
		{
			name:  "single-quoted token",
			input: `echo 'hi there'`,
			want: []token{
				{Raw: "echo", Unquoted: "echo", Start: 0, End: 4, Closed: true},
				{Raw: `'hi there'`, Unquoted: "hi there", Start: 5, End: 15, Quoted: true, Quote: '\'', Closed: true},
			},
		},
		{
			name:  "unclosed quote runs to end",
			input: `cat "unfinished`,
			want: []token{
				{Raw: "cat", Unquoted: "cat", Start: 0, End: 3, Closed: true},
				{Raw: `"unfinished`, Unquoted: "unfinished", Start: 4, End: 15, Quoted: true, Quote: '"', Closed: false},
			},
		},
		{
			name:  "different quote inside open quote is literal",
			input: `say "it's me"`,
			want: []token{
				{Raw: "say", Unquoted: "say", Start: 0, End: 3, Closed: true},
				{Raw: `"it's me"`, Unquoted: "it's me", Start: 4, End: 13, Quoted: true, Quote: '"', Closed: true},
			},
		},
		{
			name:  "leading and trailing spaces collapse",
			input: "  git  ",
			want: []token{
				{Raw: "git", Unquoted: "git", Start: 2, End: 5, Closed: true},
			},
		},
		{
			// Quoted means "token opened with a quote", not "contains a quote".
			// --flag="value" is a single unquoted token whose content includes a
			// closed quoted segment.
			name:  "mid-token quote does not mark token as Quoted",
			input: `--message="hello world"`,
			want: []token{
				{Raw: `--message="hello world"`, Unquoted: `--message=hello world`, Start: 0, End: 23, Closed: true},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := tokenize(c.input)
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("tokenize(%q) =\n  got  %#v\n  want %#v", c.input, got, c.want)
			}
		})
	}
}

func TestTokenize_SplitInputParity(t *testing.T) {
	// splitInput must produce the same []string it always did, regardless of
	// the new tokenize internals.
	cases := []string{
		"",
		"git",
		"git commit",
		`cat "my file"`,
		`echo 'hi there'`,
		`cat "unfinished`,
		`say "it's me"`,
		"  git  ",
		`git commit --message="test message"`,
	}
	for _, input := range cases {
		t.Run(input, func(t *testing.T) {
			got := splitInput(input)
			tokens := tokenize(input)
			want := make([]string, len(tokens))
			for i, tk := range tokens {
				want[i] = tk.Raw
			}
			if len(want) == 0 {
				want = nil
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("splitInput diverged from tokenize Raw for %q: got %v, tokenize Raw %v", input, got, want)
			}
		})
	}
}
