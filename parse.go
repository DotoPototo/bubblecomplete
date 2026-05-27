package bubblecomplete

import "strings"

// token is the unit produced by tokenize. It carries enough metadata for
// completion, validation, and quote-error reporting to share a single parse.
type token struct {
	// Raw is the token text as typed, including surrounding quote characters.
	Raw string
	// Unquoted is the token text with surrounding quote characters removed.
	Unquoted string
	// Start is the byte offset in the input where the token begins.
	Start int
	// End is the byte offset in the input immediately after the token ends.
	End int
	// Quoted is true if the token opened with a quote character.
	Quoted bool
	// Quote is the quote character (' or ") if Quoted is true.
	Quote rune
	// Closed is true if a quoted token's closing quote was seen.
	// Always true for unquoted tokens.
	Closed bool
}

// tokenize splits input into tokens. Behavior matches splitInput's historical
// rules: spaces split tokens outside of quotes; a closing quote ends its
// token; an unclosed quote runs to the end of the input.
func tokenize(input string) []token {
	var tokens []token
	var raw strings.Builder
	var unquoted strings.Builder
	inQuotes := false
	var quoteChar rune
	quoted := false
	closed := false
	tokenStart := -1

	flush := func(end int) {
		if raw.Len() == 0 {
			return
		}
		tok := token{
			Raw:      raw.String(),
			Unquoted: unquoted.String(),
			Start:    tokenStart,
			End:      end,
			Closed:   closed || !quoted,
		}
		if quoted {
			tok.Quoted = true
			tok.Quote = quoteChar
		}
		tokens = append(tokens, tok)
		raw.Reset()
		unquoted.Reset()
		inQuotes = false
		quoted = false
		closed = false
		quoteChar = 0
		tokenStart = -1
	}

	pos := 0
	for _, char := range input {
		size := len(string(char))
		switch {
		case char == ' ' && !inQuotes:
			flush(pos)
		case char == '"' || char == '\'':
			if inQuotes && char == quoteChar {
				raw.WriteRune(char)
				inQuotes = false
				closed = true
				flush(pos + size)
			} else if !inQuotes {
				// Only mark the token as Quoted (and record Quote) if the
				// opening quote is the first character of the token.
				// Mid-token quotes like in --flag="value" still drive the
				// lexer state but don't change the token's metadata.
				openedToken := raw.Len() == 0
				if tokenStart == -1 {
					tokenStart = pos
				}
				raw.WriteRune(char)
				inQuotes = true
				quoteChar = char
				if openedToken {
					quoted = true
				}
			} else {
				// in-quotes, different quote char: literal content
				if tokenStart == -1 {
					tokenStart = pos
				}
				raw.WriteRune(char)
				unquoted.WriteRune(char)
			}
		default:
			if tokenStart == -1 {
				tokenStart = pos
			}
			raw.WriteRune(char)
			unquoted.WriteRune(char)
		}
		pos += size
	}
	flush(pos)
	return tokens
}
