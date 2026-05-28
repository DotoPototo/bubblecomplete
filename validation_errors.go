package bubblecomplete

// ValidationErrorKind classifies a ValidationError so hosts can react without
// parsing error strings.
type ValidationErrorKind int

const (
	// InvalidCommand reports a token that did not match any command at its position.
	InvalidCommand ValidationErrorKind = iota
	// UnknownFlag reports a flag-shaped token that does not exist on the active command.
	UnknownFlag
	// MalformedFlag reports a flag token whose syntax or usage is invalid
	// (empty short-flag body, non-bool flag not last in a combined group, etc.).
	MalformedFlag
	// MissingFlagValue reports a non-bool flag that was not given a value.
	MissingFlagValue
	// InvalidArgumentValue reports a value that failed type or shape validation.
	InvalidArgumentValue
	// MissingPositionalArgument reports a required positional argument that was not provided.
	MissingPositionalArgument
	// UnexpectedArgument reports an extra token past the command's accepted arguments.
	UnexpectedArgument
	// PathNotFound reports a file/dir argument whose path could not be
	// resolved on disk — either it does not exist or os.Stat returned some
	// other access error (e.g., permission denied). The underlying os error
	// is available via ValidationError.Err.
	PathNotFound
	// UnclosedQuote reports an argument value with an unbalanced quote.
	UnclosedQuote
)

// ValidationError describes a single validation failure. Use errors.As to
// inspect Kind from a returned error.
type ValidationError struct {
	// Kind classifies the failure.
	Kind ValidationErrorKind
	// Token is the offending input token, when relevant.
	Token string
	// Argument is the name of the command argument or flag, when relevant.
	Argument string
	// Err is an optional underlying error (e.g., from strconv parsing).
	Err error

	msg string
}

// Error returns the human-readable message. Messages match the strings the
// library has always emitted to avoid breaking callers that string-match.
func (e *ValidationError) Error() string {
	return e.msg
}

// Unwrap returns the underlying error, if any.
func (e *ValidationError) Unwrap() error {
	return e.Err
}

func errInvalidCommand(token string) *ValidationError {
	return &ValidationError{
		Kind:  InvalidCommand,
		Token: token,
		msg:   "invalid command: " + token,
	}
}

func errEmptyCommand() *ValidationError {
	return &ValidationError{
		Kind: InvalidCommand,
		msg:  "empty command",
	}
}

func errUnexpectedArgument(token string) *ValidationError {
	return &ValidationError{
		Kind:  UnexpectedArgument,
		Token: token,
		msg:   "unexpected argument: " + token,
	}
}

func errMissingPositional(name string) *ValidationError {
	return &ValidationError{
		Kind:     MissingPositionalArgument,
		Argument: name,
		msg:      "missing positional argument: " + name,
	}
}

func errMissingPositionalValue(name string) *ValidationError {
	return &ValidationError{
		Kind:     MissingPositionalArgument,
		Argument: name,
		msg:      "missing value for argument: " + name,
	}
}

func errInvalidFlag(token string) *ValidationError {
	return &ValidationError{
		Kind:  UnknownFlag,
		Token: token,
		msg:   "invalid flag: " + token,
	}
}

func errInvalidArgumentToken(token string) *ValidationError {
	return &ValidationError{
		Kind:  MalformedFlag,
		Token: token,
		msg:   "invalid argument: " + token,
	}
}

func errFlagNotFound(name string) *ValidationError {
	return &ValidationError{
		Kind:     UnknownFlag,
		Argument: name,
		msg:      "flag '" + name + "' not found",
	}
}

func errMissingFlagValue(name string) *ValidationError {
	return &ValidationError{
		Kind:     MissingFlagValue,
		Argument: name,
		msg:      "missing value for flag '" + name + "'",
	}
}

func errCombinedFlagNotLast(name string) *ValidationError {
	return &ValidationError{
		Kind:     MalformedFlag,
		Argument: name,
		msg:      "flag '" + name + "' must be the last in a combined group",
	}
}

func errInvalidInt(name string, err error) *ValidationError {
	return &ValidationError{
		Kind:     InvalidArgumentValue,
		Argument: name,
		Err:      err,
		msg:      "invalid integer value for argument: " + name,
	}
}

func errInvalidFloat(name string, err error) *ValidationError {
	return &ValidationError{
		Kind:     InvalidArgumentValue,
		Argument: name,
		Err:      err,
		msg:      "invalid float value for argument: " + name,
	}
}

func errPathNotExist(pathType, name string) *ValidationError {
	return &ValidationError{
		Kind:     PathNotFound,
		Argument: name,
		msg:      pathType + " does not exist for argument: " + name,
	}
}

func errPathAccess(pathType, name string, err error) *ValidationError {
	return &ValidationError{
		Kind:     PathNotFound,
		Argument: name,
		Err:      err,
		msg:      "error accessing " + pathType + " for argument: " + name,
	}
}

func errPathIsDir(name string) *ValidationError {
	return &ValidationError{
		Kind:     InvalidArgumentValue,
		Argument: name,
		msg:      "file path is a directory: " + name,
	}
}

func errPathIsFile(name string) *ValidationError {
	return &ValidationError{
		Kind:     InvalidArgumentValue,
		Argument: name,
		msg:      "directory path is a file: " + name,
	}
}

func errUnclosedQuote() *ValidationError {
	return &ValidationError{
		Kind: UnclosedQuote,
		msg:  "missing closing quote",
	}
}

func errUnclosedQuoteForArg(name string) *ValidationError {
	return &ValidationError{
		Kind:     UnclosedQuote,
		Argument: name,
		msg:      "missing closing quote for argument: " + name,
	}
}

func errUnknownType(t string) *ValidationError {
	return &ValidationError{
		Kind: InvalidArgumentValue,
		msg:  "unknown argument type: " + t,
	}
}
