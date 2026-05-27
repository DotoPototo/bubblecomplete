package bubblecomplete

import (
	"reflect"
	"strings"
	"testing"
)

func TestContainsLongFlag(t *testing.T) {
	command := "This is a test command --flag1 --flag2=value --anotherFlag='--flag3' --testing '--flag4'"

	flag := "--flag1"
	expected := true
	result := containsLongFlag(command, flag)
	if result != expected {
		t.Errorf("containsLongFlag(%q, %q) == %t, expected %t", command, flag, result, expected)
	}

	flag = "--flag2"
	expected = true
	result = containsLongFlag(command, flag)
	if result != expected {
		t.Errorf("containsLongFlag(%q, %q) == %t, expected %t", command, flag, result, expected)
	}

	flag = "--flag3"
	expected = false
	result = containsLongFlag(command, flag)
	if result != expected {
		t.Errorf("containsLongFlag(%q, %q) == %t, expected %t", command, flag, result, expected)
	}

	flag = "--flag4"
	expected = false
	result = containsLongFlag(command, flag)
	if result != expected {
		t.Errorf("containsLongFlag(%q, %q) == %t, expected %t", command, flag, result, expected)
	}

	command = "--flag2"

	flag = "--flag1"
	expected = false
	result = containsLongFlag(command, flag)
	if result != expected {
		t.Errorf("containsLongFlag(%q, %q) == %t, expected %t", command, flag, result, expected)
	}

	flag = "--flag2"
	expected = true
	result = containsLongFlag(command, flag)
	if result != expected {
		t.Errorf("containsLongFlag(%q, %q) == %t, expected %t", command, flag, result, expected)
	}
}

func TestContainsLongFlag_EndOfInput(t *testing.T) {
	// Token-based detection must recognize a long flag that ends the input
	// with no trailing space. The previous string-substring approach missed
	// this because it required " --flag " with spaces on both sides.
	cases := []struct {
		command string
		flag    string
		want    bool
	}{
		{"git commit --amend", "--amend", true},
		{"git commit --amend ", "--amend", true},
		{"git commit --amend=true", "--amend", true},
		{"git commit '--amend'", "--amend", false}, // quoted, must not match
		{"git status", "--amend", false},
	}
	for _, c := range cases {
		t.Run(c.command, func(t *testing.T) {
			got := containsLongFlag(c.command, c.flag)
			if got != c.want {
				t.Errorf("containsLongFlag(%q, %q) = %t, want %t", c.command, c.flag, got, c.want)
			}
		})
	}
}

func TestContainsShortFlag(t *testing.T) {
	command := "This is a test command -f -m \"Added flag -x\" --fsomething '-p' -yz"

	flag := "-f"
	expected := true
	result := containsShortFlag(command, flag)
	if result != expected {
		t.Errorf("containsShortFlag(%q, %q) == %t, expected %t", command, flag, result, expected)
	}

	flag = "-m"
	expected = true
	result = containsShortFlag(command, flag)
	if result != expected {
		t.Errorf("containsShortFlag(%q, %q) == %t, expected %t", command, flag, result, expected)
	}

	flag = "-x"
	expected = false
	result = containsShortFlag(command, flag)
	if result != expected {
		t.Errorf("containsShortFlag(%q, %q) == %t, expected %t", command, flag, result, expected)
	}

	flag = "-p"
	expected = false
	result = containsShortFlag(command, flag)
	if result != expected {
		t.Errorf("containsShortFlag(%q, %q) == %t, expected %t", command, flag, result, expected)
	}

	flag = "-y"
	expected = true
	result = containsShortFlag(command, flag)
	if result != expected {
		t.Errorf("containsShortFlag(%q, %q) == %t, expected %t", command, flag, result, expected)
	}

	flag = "-z"
	expected = true
	result = containsShortFlag(command, flag)
	if result != expected {
		t.Errorf("containsShortFlag(%q, %q) == %t, expected %t", command, flag, result, expected)
	}

	command = "-f"

	flag = "-f"
	expected = true
	result = containsShortFlag(command, flag)
	if result != expected {
		t.Errorf("containsShortFlag(%q, %q) == %t, expected %t", command, flag, result, expected)
	}

	flag = "-l"
	expected = false
	result = containsShortFlag(command, flag)
	if result != expected {
		t.Errorf("containsShortFlag(%q, %q) == %t, expected %t", command, flag, result, expected)
	}
}

func TestStringEndsInQuoteWithoutEquals(t *testing.T) {
	input := "\"This is a test\""
	expected := true
	result := stringEndsInQuoteWithoutEquals(input)
	if result != expected {
		t.Errorf("stringEndsInQuoteWithoutEquals(%q) == %t, expected %t", input, result, expected)
	}

	input = "'This is a test'"
	expected = true
	result = stringEndsInQuoteWithoutEquals(input)
	if result != expected {
		t.Errorf("stringEndsInQuoteWithoutEquals(%q) == %t, expected %t", input, result, expected)
	}

	input = "\"This is a test\"="
	expected = false
	result = stringEndsInQuoteWithoutEquals(input)
	if result != expected {
		t.Errorf("stringEndsInQuoteWithoutEquals(%q) == %t, expected %t", input, result, expected)
	}

	input = "'This is a test'="
	expected = false
	result = stringEndsInQuoteWithoutEquals(input)
	if result != expected {
		t.Errorf("stringEndsInQuoteWithoutEquals(%q) == %t, expected %t", input, result, expected)
	}

	input = "'This is a test='"
	expected = false
	result = stringEndsInQuoteWithoutEquals(input)
	if result != expected {
		t.Errorf("stringEndsInQuoteWithoutEquals(%q) == %t, expected %t", input, result, expected)
	}
}

func TestSortedGetCompletions(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "c",
			input:    "c",
			expected: []string{"cat", "cp"},
		},
		{
			name:     "ca",
			input:    "ca",
			expected: []string{"cat"},
		},
		{
			name:     "cat",
			input:    "cat",
			expected: []string{"cat"},
		},
		{
			name:     "cat ",
			input:    "cat ",
			expected: []string{"File", "--show-ends", "-f --file-name", "-n --number", "-p --plain"},
		},
		{
			name:     "cat m",
			input:    "cat m",
			expected: []string{"File"},
		},
		{
			name:     "cat -",
			input:    "cat -",
			expected: []string{"--show-ends", "-f --file-name", "-n --number", "-p --plain"},
		},
		{
			name:     "ps -f",
			input:    "ps -f",
			expected: []string{"-filearg", "-floatarg"},
		},
		{
			name:     "ps -F",
			input:    "ps -F",
			expected: []string{"-FileDirArg"},
		},
		{
			name:     "ps --",
			input:    "ps --",
			expected: []string{},
		},
		{
			name:     "git ",
			input:    "git ",
			expected: []string{"branch", "checkout", "clone", "commit", "merge", "pull", "push", "rebase", "stash", "status", "tag", "--help", "--version"},
		},
		{
			name:     "git c",
			input:    "git c",
			expected: []string{"checkout", "clone", "commit"},
		},
		{
			name:     "git -",
			input:    "git -",
			expected: []string{"--help", "--version"},
		},
		{
			name:     "git co",
			input:    "git co",
			expected: []string{"commit"},
		},
		{
			name:     "git commit",
			input:    "git commit",
			expected: []string{"commit"},
		},
		{
			name:     "git commit ",
			input:    "git commit ",
			expected: []string{"--amend", "--help", "-a --all", "-m --message"},
		},
		{
			name:     "git commit -m",
			input:    "git commit -m",
			expected: []string{"-m --message"},
		},
		{
			name:     "git commit -m ",
			input:    "git commit -m ",
			expected: []string{"-m --message"},
		},
		{
			name:     "git commit --message",
			input:    "git commit --message",
			expected: []string{"-m --message"},
		},
		{
			name:     "git commit --message ",
			input:    "git commit --message ",
			expected: []string{"-m --message"},
		},
		{
			name:     "git commit --message=",
			input:    "git commit --message=",
			expected: []string{"-m --message"},
		},
		{
			name:     "git commit --message=\"",
			input:    "git commit --message=\"",
			expected: []string{"-m --message"},
		},
		{
			name:     "git commit --message=\"test",
			input:    "git commit --message=\"test",
			expected: []string{"-m --message"},
		},
		{
			name:     "git commit --message=\"test\"",
			input:    "git commit --message=\"test\"",
			expected: []string{"-m --message"},
		},
		{
			name:     "git commit --message=\"test\" ",
			input:    "git commit --message=\"test\" ",
			expected: []string{"--amend", "--help", "-a --all"},
		},
		{
			name:     "git commit --message=\"test\" -a",
			input:    "git commit --message=\"test\" -a",
			expected: []string{"-a --all"},
		},
		{
			name:     "git commit --message=\"test\" --a",
			input:    "git commit --message=\"test\" --a",
			expected: []string{"--amend", "-a --all"},
		},
		{
			name:     "git commit --message=\"test ",
			input:    "git commit --message=\"test ",
			expected: []string{"-m --message"},
		},
		{
			name:     "git commit --message=\"test message ",
			input:    "git commit --message=\"test message ",
			expected: []string{"-m --message"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, _ := getCompletions(tc.input, TestCommands)
			sortCompletions(&result)
			uniqueCompletions(&result)

			var resultStrings []string
			for _, r := range result {
				resultStrings = append(resultStrings, r.getName())
			}

			if (len(result) == 0) && (len(tc.expected) == 0) {
				return
			}

			if !reflect.DeepEqual(resultStrings, tc.expected) {
				t.Errorf("Expected completions: %v, got: %v", tc.expected, resultStrings)
			}
		})
	}
}

func TestGetCompletionsMatchPrefix(t *testing.T) {
	testCases := []struct {
		name           string
		input          string
		expectedPrefix string
	}{
		{
			name:           "partial command prefix",
			input:          "c",
			expectedPrefix: "c",
		},
		{
			name:           "longer partial command prefix",
			input:          "ca",
			expectedPrefix: "ca",
		},
		{
			name:           "full command no trailing space",
			input:          "cat",
			expectedPrefix: "cat",
		},
		{
			name:           "full command with trailing space shows all - no prefix",
			input:          "cat ",
			expectedPrefix: "",
		},
		{
			name:           "subcommand partial prefix",
			input:          "git co",
			expectedPrefix: "co",
		},
		{
			name:           "subcommand full with space - no prefix",
			input:          "git commit ",
			expectedPrefix: "",
		},
		{
			name:           "flag partial prefix",
			input:          "git commit -m",
			expectedPrefix: "-m",
		},
		{
			name:           "long flag partial prefix",
			input:          "git commit --me",
			expectedPrefix: "--me",
		},
		{
			name:           "powershell flag partial prefix",
			input:          "ps -f",
			expectedPrefix: "-f",
		},
		{
			name:           "empty input",
			input:          " ",
			expectedPrefix: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, prefix := getCompletions(tc.input, TestCommands)
			if prefix != tc.expectedPrefix {
				t.Errorf("Expected prefix %q, got %q", tc.expectedPrefix, prefix)
			}
		})
	}
}

func TestMatchingPolicy_CaseSensitivity(t *testing.T) {
	cases := []struct {
		name      string
		input     string
		wantNames []string
	}{
		{"commands lowercase matches", "gi", []string{"git"}},
		{"commands uppercase does not match", "GI", nil},
		{"long flags case-sensitive: lowercase matches", "cat --sh", []string{"--show-ends"}},
		{"long flags case-sensitive: uppercase does not match", "cat --SH", nil},
		{"PowerShell flags case-sensitive: matching case matches", "ps -FileD", []string{"-FileDirArg"}},
		{"PowerShell flags case-sensitive: lowercase does not match", "ps -filed", nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			completions, _ := getCompletions(c.input, TestCommands)
			got := completionNames(completions)
			if !reflect.DeepEqual(got, c.wantNames) {
				t.Errorf("got %v, want %v", got, c.wantNames)
			}
		})
	}
}

func TestMatchingPolicy_DescriptionsExcluded(t *testing.T) {
	// "Concatenate" appears only in cat's description, never in a name.
	// Typing it must not surface any completion.
	completions, _ := getCompletions("Concatenate", TestCommands)
	if len(completions) != 0 {
		t.Errorf("Expected description-only text to return no completions, got %d", len(completions))
	}
}

func TestMatchingPolicy_FlagsSortAfterNonFlags(t *testing.T) {
	// "cat " yields one positional arg ("File") and four flags; after the
	// punctuation-last sort the positional must come first.
	completions, _ := getCompletions("cat ", TestCommands)
	sortCompletions(&completions)
	names := completionNames(completions)
	if len(names) < 2 {
		t.Fatalf("expected at least 2 completions, got %v", names)
	}
	if names[0] != "File" {
		t.Errorf("expected first completion to be the positional 'File', got %v", names)
	}
	for i, n := range names[1:] {
		if !strings.HasPrefix(n, "-") {
			t.Errorf("expected flags after the positional, got %q at index %d", n, i+1)
		}
	}
}

func completionNames(comps []Completion) []string {
	if len(comps) == 0 {
		return nil
	}
	out := make([]string, len(comps))
	for i, c := range comps {
		out[i] = c.getName()
	}
	return out
}
