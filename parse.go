package bubblecomplete

import (
	"strings"
	"unicode/utf8"
)

// token is the unit produced by tokenize. It carries enough metadata for
// completion, validation, and quote-error reporting to share a single parse.
type token struct {
	// Sliced from input verbatim so invalid UTF-8 bytes survive.
	Raw string
	// Quote characters stripped. Invalid UTF-8 in input becomes U+FFFD here
	// (unlike Raw); use Raw when byte-fidelity matters.
	Unquoted   string
	Start, End int
	Quoted     bool
	// Quote is the opener (' or "), valid only when Quoted is true.
	Quote rune
	// Closed is true if the opening quote was matched, or always for unquoted tokens.
	Closed bool
}

// tokenize splits input into tokens. Behavior matches splitInput's historical
// rules: spaces split tokens outside of quotes; a closing quote ends its
// token; an unclosed quote runs to the end of the input.
func tokenize(input string) []token {
	var tokens []token
	var unquoted strings.Builder
	inQuotes := false
	var quoteChar rune
	quoted := false
	closed := false
	tokenStart := -1

	flush := func(end int) {
		if tokenStart == -1 {
			return
		}
		tok := token{
			// Raw is sliced from the original input so it preserves the
			// exact bytes — including invalid UTF-8 sequences that would
			// otherwise be normalized to U+FFFD by WriteRune.
			Raw:      input[tokenStart:end],
			Unquoted: unquoted.String(),
			Start:    tokenStart,
			End:      end,
			// Closed semantics: a token whose opener-quote was matched
			// (closed), OR a token that did NOT open with a quote AND
			// has no still-open inner quote at flush time. The
			// inQuotes guard catches mid-token inner-quote forms like
			// `--message="test ` where the token starts unquoted but
			// opens an inner quote that never closes.
			Closed: closed || (!quoted && !inQuotes),
		}
		if quoted {
			tok.Quoted = true
			tok.Quote = quoteChar
		}
		tokens = append(tokens, tok)
		unquoted.Reset()
		inQuotes = false
		quoted = false
		closed = false
		quoteChar = 0
		tokenStart = -1
	}

	// Iterate by byte index using DecodeRuneInString so size matches what
	// range would advance by — including size=1 for invalid UTF-8 sequences
	// (where len(string(utf8.RuneError)) would have returned 3 and skewed
	// the byte offsets we record on each token).
	pos := 0
	for pos < len(input) {
		char, size := utf8.DecodeRuneInString(input[pos:])
		switch {
		case char == ' ' && !inQuotes:
			flush(pos)
		case char == '"' || char == '\'':
			switch {
			case inQuotes && char == quoteChar:
				inQuotes = false
				closed = true
				flush(pos + size)
			case !inQuotes:
				// Only mark the token as Quoted (and record Quote) if the
				// opening quote is the first character of the token.
				// Mid-token quotes like in --flag="value" still drive the
				// lexer state but don't change the token's metadata.
				openedToken := tokenStart == -1
				if openedToken {
					tokenStart = pos
				}
				inQuotes = true
				quoteChar = char
				if openedToken {
					quoted = true
				}
			default:
				// in-quotes, different quote char: literal content
				if tokenStart == -1 {
					tokenStart = pos
				}
				unquoted.WriteRune(char)
			}
		default:
			if tokenStart == -1 {
				tokenStart = pos
			}
			unquoted.WriteRune(char)
		}
		pos += size
	}
	flush(pos)
	return tokens
}
