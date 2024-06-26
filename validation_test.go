package bubblecomplete

import "testing"

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
