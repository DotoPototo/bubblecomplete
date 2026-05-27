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

func TestValidationError_KindInspection(t *testing.T) {
	cases := []struct {
		name      string
		input     string
		wantKind  ValidationErrorKind
		wantToken string
		wantArg   string
	}{
		{"invalid command", "unknown", InvalidCommand, "unknown", ""},
		{"unexpected argument", "cat ./README.md extra", UnexpectedArgument, "extra", ""},
		{"missing positional", "cat", MissingPositionalArgument, "", "File"},
		{"unknown flag", "git commit --invalid", UnknownFlag, "", "--invalid"},
		{"missing flag value", "git commit -m", MissingFlagValue, "", "-m"},
		{"invalid int value", "ps -intarg notanumber ./README.md", InvalidArgumentValue, "", "-intarg"},
		{"path not found", "ps no-such-file.txt", PathNotFound, "", "Input"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := validateCommandInput(c.input, TestCommands)
			if err == nil {
				t.Fatalf("expected error for %q", c.input)
			}
			var ve *ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("expected *ValidationError, got %T", err)
			}
			if ve.Kind != c.wantKind {
				t.Errorf("Kind = %d, want %d", ve.Kind, c.wantKind)
			}
			if ve.Token != c.wantToken {
				t.Errorf("Token = %q, want %q", ve.Token, c.wantToken)
			}
			if ve.Argument != c.wantArg {
				t.Errorf("Argument = %q, want %q", ve.Argument, c.wantArg)
			}
		})
	}
}

func TestValidationError_MalformedFlagSurfaces(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantMsg string
	}{
		{
			name:    "bare dash without flag chars",
			input:   "git -",
			wantMsg: "invalid argument: -",
		},
		{
			name:    "non-bool short flag not last in combined group",
			input:   "cat -fn ./README.md",
			wantMsg: "flag '-f' must be the last in a combined group",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := validateCommandInput(c.input, TestCommands)
			if err == nil {
				t.Fatalf("expected error for %q", c.input)
			}
			var ve *ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("expected *ValidationError, got %T", err)
			}
			if ve.Kind != MalformedFlag {
				t.Errorf("Kind = %d, want MalformedFlag", ve.Kind)
			}
			if ve.Error() != c.wantMsg {
				t.Errorf("message = %q, want %q", ve.Error(), c.wantMsg)
			}
		})
	}
}

func TestCheckEmptyString_RoutesByArgumentType(t *testing.T) {
	// checkEmptyString is the defensive guard for an empty value reaching
	// validateArgumentValue. The Kind must reflect the calling context
	// (flag vs positional) so hosts can route on Kind even if upstream
	// guards regress.
	flag := &Flag{ShortFlag: "-m", Type: StringArgument}
	pos := &PositionalArgument{Name: "File", Type: StringArgument, Required: true}

	flagErr := checkEmptyString(flag, "")
	var ve *ValidationError
	if !errors.As(flagErr, &ve) {
		t.Fatalf("flag empty: expected *ValidationError, got %T", flagErr)
	}
	if ve.Kind != MissingFlagValue {
		t.Errorf("flag empty: Kind = %d, want MissingFlagValue", ve.Kind)
	}
	if ve.Error() != "missing value for flag '-m'" {
		t.Errorf("flag empty message = %q", ve.Error())
	}

	posErr := checkEmptyString(pos, "")
	if !errors.As(posErr, &ve) {
		t.Fatalf("positional empty: expected *ValidationError, got %T", posErr)
	}
	if ve.Kind != MissingPositionalArgument {
		t.Errorf("positional empty: Kind = %d, want MissingPositionalArgument", ve.Kind)
	}
	if ve.Error() != "missing value for argument: File" {
		t.Errorf("positional empty message = %q", ve.Error())
	}

	if got := checkEmptyString(flag, "non-empty"); got != nil {
		t.Errorf("non-empty value should return nil, got %v", got)
	}
}

func TestValidationError_UnwrapPreservesUnderlying(t *testing.T) {
	err := validateCommandInput("ps -intarg abc ./README.md", TestCommands)
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *ValidationError, got %T", err)
	}
	if ve.Err == nil {
		t.Fatal("expected non-nil underlying error for strconv failure")
	}
}
