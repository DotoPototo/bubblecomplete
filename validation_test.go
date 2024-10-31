package bubblecomplete

import (
	"errors"
	"testing"
)

func TestRemoveQuotes(t *testing.T) {
	input := "\"This is a test\""
	expected := "This is a test"
	result := removeQuotes(input)
	if result != expected {
		t.Errorf("removeQuotes(%q) == %q, expected %q", input, result, expected)
	}

	input = "'This is a test'"
	expected = "This is a test"
	result = removeQuotes(input)
	if result != expected {
		t.Errorf("removeQuotes(%q) == %q, expected %q", input, result, expected)
	}

	input = "\"This is a test"
	expected = "\"This is a test"
	result = removeQuotes(input)
	if result != expected {
		t.Errorf("removeQuotes(%q) == %q, expected %q", input, result, expected)
	}

	input = "'This is a test"
	expected = "'This is a test"
	result = removeQuotes(input)
	if result != expected {
		t.Errorf("removeQuotes(%q) == %q, expected %q", input, result, expected)
	}

	input = "'This is a test='"
	expected = "This is a test="
	result = removeQuotes(input)
	if result != expected {
		t.Errorf("removeQuotes(%q) == %q, expected %q", input, result, expected)
	}
}

func TestValidateCommandInput(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected error
	}{
		// VALID CAT COMMAND TESTING
		{
			name:     "valid input",
			input:    "cat ./README.md",
			expected: nil,
		},
		{
			name:     "valid input with long flag",
			input:    "cat --show-ends ./README.md",
			expected: nil,
		},
		{
			name:     "valid input with long flags and flag value",
			input:    "cat --show-ends --file-name=README.md ./README.md",
			expected: nil,
		},
		{
			name:     "valid input with short flag",
			input:    "cat -n ./README.md",
			expected: nil,
		},
		{
			name:     "valid input short flags and flag value",
			input:    "cat -n -f TEST ./README.md",
			expected: nil,
		},
		{
			name:     "valid input with combined short flags",
			input:    "cat -np ./README.md",
			expected: nil,
		},
		{
			name:     "valid input short flag and long flag",
			input:    "cat -f TEST --plain ./README.md",
			expected: nil,
		},
		{
			name:     "valid input with flag arg after space",
			input:    "cat -f TEST ./README.md",
			expected: nil,
		},
		{
			name:     "valind input with flag arg after equals",
			input:    "cat -f=TEST ./README.md",
			expected: nil,
		},
		{
			name:     "valid input with flag arg after space in single quotes",
			input:    "cat -f 'TEST FILE' ./README.md",
			expected: nil,
		},
		{
			name:     "valind input with flag arg after equals in single quotes",
			input:    "cat -f='TEST FILE' ./README.md",
			expected: nil,
		},
		{
			name:     "valid input with flag arg after space in double quotes",
			input:    "cat -f \"TEST FILE\" ./README.md",
			expected: nil,
		},
		{
			name:     "valid input with flag arg after equals in double quotes",
			input:    "cat -f=\"TEST FILE\" ./README.md",
			expected: nil,
		},
		{
			name:     "valid PowerShell style flag",
			input:    "ps -boolarg ./README.md",
			expected: nil,
		},
		// INVALID CAT COMMAND TESTING
		{
			name:     "invalid input with non existent long flag",
			input:    "cat --invalid ./README.md",
			expected: errors.New("flag '--invalid' not found"),
		},
		{
			name:     "invalid input with non existent short flag",
			input:    "cat -i ./README.md",
			expected: errors.New("flag '-i' not found"),
		},
		{
			name:     "invalid input with non existent combined short flag",
			input:    "cat -ni ./README.md",
			expected: errors.New("flag '-ni' not found"),
		},
		// NESTED COMMAND TESTING
		{
			name:     "valid nested command",
			input:    "git commit -m \"Initial commit\"",
			expected: nil,
		},
		{
			name:     "valid nested command with short and long flags",
			input:    "git commit -m \"Initial commit\" --amend -a",
			expected: nil,
		},
		// VALID FLAG TYPE TESTING
		{
			name:     "valid bool flag",
			input:    "ps -boolarg ./README.md",
			expected: nil,
		},
		{
			name:     "valid string flag",
			input:    "ps -stringarg \"test\" ./README.md",
			expected: nil,
		},
		{
			name:     "valid int flag",
			input:    "ps -intarg 123 ./README.md",
			expected: nil,
		},
		{
			name:     "valid float flag",
			input:    "ps -floatarg 123.456 ./README.md",
			expected: nil,
		},
		{
			name:     "valid file flag",
			input:    "ps -filearg ./README.md ./README.md",
			expected: nil,
		},
		{
			name:     "valid dir flag",
			input:    "ps -dirarg ./ ./README.md",
			expected: nil,
		},
		{
			name:     "valid file or dir flag (file)",
			input:    "ps -FileDirArg ./README.md ./README.md",
			expected: nil,
		},
		{
			name:     "valid file or dir flag (dir)",
			input:    "ps -FileDirArg ./ ./README.md",
			expected: nil,
		},
		// INVALID FLAG TYPE TESTING
		{
			name:     "invalid bool flag",
			input:    "ps -boolarg \"test\" ./README.md",
			expected: errors.New("file does not exist for argument: Input"),
		},
		{
			name:     "invalid string flag",
			input:    "ps -stringarg ./README.md",
			expected: errors.New("missing positional argument: Input"),
		},
		{
			name:     "invalid int flag",
			input:    "ps -intarg 123.456 ./README.md",
			expected: errors.New("invalid integer value for argument: -intarg"),
		},
		{
			name:     "invalid float flag",
			input:    "ps -floatarg \"123\" ./README.md",
			expected: errors.New("invalid float value for argument: -floatarg"),
		},
		{
			name:     "invalid file flag",
			input:    "ps -filearg 123 ./README.md",
			expected: errors.New("file does not exist for argument: -filearg"),
		},
		{
			name:     "invalid dir flag",
			input:    "ps -dirarg 123 ./README.md",
			expected: errors.New("directory does not exist for argument: -dirarg"),
		},
		{
			name:     "invalid file or dir flag",
			input:    "ps -FileDirArg 123 ./README.md",
			expected: errors.New("file or directory does not exist for argument: -FileDirArg"),
		},
		// EDGE CASE TESTING
		{
			name:     "empty input",
			input:    "",
			expected: errors.New("empty command"),
		},
		{
			name:     "command with only spaces",
			input:    "		",
			expected: errors.New("invalid command: \t\t"),
		},
		{
			name:     "unknown command",
			input:    "unknown",
			expected: errors.New("invalid command: unknown"),
		},
		{
			name:     "flag without command",
			input:    "cat",
			expected: errors.New("missing positional argument: File"),
		},
		{
			name:     "double spaces between command and args",
			input:    "cat         ./README.md",
			expected: nil,
		},
		{
			name:     "extra positional argument",
			input:    "cat ./README.md ./README.md",
			expected: errors.New("unexpected argument: ./README.md"),
		},
		{
			name:     "single quotes inside double quoted string",
			input:    "cat -f \"TEST 'FILE'\" ./README.md",
			expected: nil,
		},
		{
			name:     "single quotes inside single quoted string",
			input:    "cat -f 'TEST 'FILE'' ./README.md",
			expected: errors.New("file does not exist for argument: File"),
		},
		{
			name:     "double quotes inside double quoted string",
			input:    "cat -f \"TEST \"FILE\"\" ./README.md",
			expected: errors.New("file does not exist for argument: File"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := validateCommandInput(tc.input, TestCommands)

			// Check if both are nil or both are not nil
			if (result == nil) != (tc.expected == nil) {
				t.Errorf("Expected error: %v, got: %v", tc.expected, result)
				return
			}

			// If we expect an error, check the message
			if tc.expected != nil {
				if result.Error() != tc.expected.Error() {
					t.Errorf("Expected error message: %q, got: %q", tc.expected.Error(), result.Error())
				}
			}
		})
	}
}
