package bubblecomplete

import "testing"

func TestStringEndsInQuote(t *testing.T) {
	cases := []struct {
		input    string
		expected bool
	}{
		{"Hello World", false},
		{"Hello World\"", true},
		{"Hello World'", true},
		{"Hello \"World\"", true},
		{"Hello 'World'", true},
		{"Hello", false},
	}

	for _, c := range cases {
		result := stringEndsInQuote(c.input)
		if result != c.expected {
			t.Errorf("stringEndsInQuote(%q) == %t, expected %t", c.input, result, c.expected)
		}
	}
}
