package buildinfo

import (
	"bytes"
	"testing"
)

func TestPrint(t *testing.T) {
	var buf bytes.Buffer

	if err := Print(&buf, Info{
		Version: "v0.1.2",
		Date:    "2026-06-29",
		Commit:  "deadbeef",
	}); err != nil {
		t.Fatalf("Print() error = %v", err)
	}

	got := buf.String()
	want := "Build version: v0.1.2\n" +
		"Build date: 2026-06-29\n" +
		"Build commit: deadbeef\n"

	if got != want {
		t.Fatalf("Print() = %q, want %q", got, want)
	}
}

func TestPrint_WithEmptyValues(t *testing.T) {
	var buf bytes.Buffer

	if err := Print(&buf, Info{}); err != nil {
		t.Fatalf("Print() error = %v", err)
	}

	got := buf.String()
	want := "Build version: " + NotAvailable + "\n" +
		"Build date: " + NotAvailable + "\n" +
		"Build commit: " + NotAvailable + "\n"

	if got != want {
		t.Fatalf("Print() = %q, want %q", got, want)
	}
}

func TestValue(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "value", value: "v0.9.0", want: "v0.9.0"},
		{name: "empty", value: "", want: NotAvailable},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := Value(test.value); got != test.want {
				t.Fatalf("Value(%q) = %q, want %q", test.value, got, test.want)
			}
		})
	}
}
