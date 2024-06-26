package bubblecomplete

import "testing"

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

func TestRemoveQuotedStrings(t *testing.T) {
	input := "git stash pop"
	expected := "git stash pop"
	result := removeQuotedStrings(input)
	if result != expected {
		t.Errorf("removeQuotedStrings(%q) == %q, expected %q", input, result, expected)
	}

	input = "git commit -m \"hello\" --amend"
	expected = "git commit -m \"\" --amend"
	result = removeQuotedStrings(input)
	if result != expected {
		t.Errorf("removeQuotedStrings(%q) == %q, expected %q", input, result, expected)
	}

	input = "cp \"/home/me/file.txt\" \"/home/me\""
	expected = "cp \"\" \"\""
	result = removeQuotedStrings(input)
	if result != expected {
		t.Errorf("removeQuotedStrings(%q) == %q, expected %q", input, result, expected)
	}

	input = "cat 'my/file'"
	expected = "cat ''"
	result = removeQuotedStrings(input)
	if result != expected {
		t.Errorf("removeQuotedStrings(%q) == %q, expected %q", input, result, expected)
	}

	input = "git commit -m \"some quote' --amend"
	expected = "git commit -m \"some quote' --amend"
	result = removeQuotedStrings(input)
	if result != expected {
		t.Errorf("removeQuotedStrings(%q) == %q, expected %q", input, result, expected)
	}

	input = "cat '\"'\" --help"
	expected = "cat ''\" --help"
	result = removeQuotedStrings(input)
	if result != expected {
		t.Errorf("removeQuotedStrings(%q) == %q, expected %q", input, result, expected)
	}
}
