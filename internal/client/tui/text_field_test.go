package tui

import (
	"strings"
	"testing"
)

func TestNormalizeTextFieldValue_UsesFirstASCIILine(t *testing.T) {
	got := normalizeTextFieldValue("abcéDEF\r\nignored")
	if got != "abcDEF" {
		t.Fatalf("normalized value = %q, want abcDEF", got)
	}
}

func TestNormalizeTextFieldValue_LimitsLength(t *testing.T) {
	got := normalizeTextFieldValue(strings.Repeat("x", maxTextFieldLength+20))
	if len(got) != maxTextFieldLength {
		t.Fatalf("length = %d, want %d", len(got), maxTextFieldLength)
	}
}

func TestTextFieldInsert_KeepsMaximumLength(t *testing.T) {
	field := newASCIITextField(strings.Repeat("a", maxTextFieldLength-2), false)
	field.insert("bcdef")
	if len(field.value) != maxTextFieldLength || !strings.HasSuffix(field.value, "bc") {
		t.Fatalf("inserted value length/suffix = %d/%q", len(field.value), field.value[len(field.value)-2:])
	}
	if field.cursor != maxTextFieldLength {
		t.Fatalf("cursor = %d, want %d", field.cursor, maxTextFieldLength)
	}
}

func TestPrintableTextKey_RejectsNonASCII(t *testing.T) {
	field := newASCIITextField("", false)
	if field.insertKey("я") {
		t.Fatal("non-ASCII key was accepted")
	}
	if !field.insertKey("A") || field.value != "A" {
		t.Fatalf("ASCII field = %q", field.value)
	}
}

func TestNormalizeUnicodeTextFieldValue_PreservesPath(t *testing.T) {
	got := normalizeUnicodeTextFieldValue("/Users/m1/Документы/gopherkeeper/ca.pem\nignored")
	want := "/Users/m1/Документы/gopherkeeper/ca.pem"
	if got != want {
		t.Fatalf("normalized Unicode value = %q, want %q", got, want)
	}
}

func TestUnicodeTextField_UsesRunePosition(t *testing.T) {
	field := newUnicodeTextField("/tmp/файл.pem")
	field.cursor = 5
	field.insert("новый-")
	want := "/tmp/новый-файл.pem"
	if field.value != want {
		t.Fatalf("inserted config value = %q, want %q", field.value, want)
	}
	if field.cursor != 11 {
		t.Fatalf("cursor = %d, want 11 rune positions", field.cursor)
	}
}

func TestUnicodeTextField_AcceptsUnicode(t *testing.T) {
	field := newUnicodeTextField("")
	if !field.insertKey("я") || field.value != "я" {
		t.Fatalf("Unicode field = %q", field.value)
	}
}

func TestMaskedTextField_PreservesCursorLength(t *testing.T) {
	field := newASCIITextField("secret", true)
	field.cursor = 3
	if got := field.displayValue(); got != "******" {
		t.Fatalf("masked value = %q, want ******", got)
	}
	if field.cursor != 3 {
		t.Fatalf("cursor = %d, want 3", field.cursor)
	}
}

func TestUnicodeTextField_LimitsLengthInRunes(t *testing.T) {
	field := newUnicodeTextField(strings.Repeat("я", maxTextFieldLength+20))
	if got := len([]rune(field.value)); got != maxTextFieldLength {
		t.Fatalf("rune length = %d, want %d", got, maxTextFieldLength)
	}
}

func TestDigitsTextField_FiltersAndLimitsInput(t *testing.T) {
	field := newDigitsTextFieldWithLimit("12 3A٤", 4)
	if field.value != "123" {
		t.Fatalf("value = %q, want 123", field.value)
	}
	field.insert("45x")
	if field.value != "1234" {
		t.Fatalf("value = %q, want 1234", field.value)
	}
}
