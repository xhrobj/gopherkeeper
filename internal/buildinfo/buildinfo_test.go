package buildinfo

import (
	"bytes"
	"testing"
)

func TestPrintBuildInfo(t *testing.T) {
	var buf bytes.Buffer

	err := Print(&buf, "v1.1.2", "2026-06-29", "deadbeaf")
	if err != nil {
		t.Fatalf("Print() error = %v", err)
	}

	got := buf.String()
	want := "Build version: v1.1.2\n" +
		"Build date: 2026-06-29\n" +
		"Build commit: deadbeaf\n"

	if got != want {
		t.Fatalf("Print() = %q, want %q", got, want)
	}
}

func TestPrintBuildInfoWithEmptyValues(t *testing.T) {
	var buf bytes.Buffer

	err := Print(&buf, "", "", "")
	if err != nil {
		t.Fatalf("Print() error = %v", err)
	}

	got := buf.String()
	want := "Build version: " + notAvailable + "\n" +
		"Build date: " + notAvailable + "\n" +
		"Build commit: " + notAvailable + "\n"

	if got != want {
		t.Fatalf("Print() = %q, want %q", got, want)
	}
}
