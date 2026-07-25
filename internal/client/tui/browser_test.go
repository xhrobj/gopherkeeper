package tui

import (
	"slices"
	"testing"
)

func TestBrowserCommand(t *testing.T) {
	tests := []struct {
		name     string
		goos     string
		wantPath string
		wantArgs []string
	}{
		{
			name:     "macOS",
			goos:     "darwin",
			wantPath: darwinOpenCommand,
			wantArgs: []string{darwinOpenCommand, aboutURL},
		},
		{
			name:     "Linux",
			goos:     "linux",
			wantPath: linuxOpenCommand,
			wantArgs: []string{linuxOpenCommand, aboutURL},
		},
		{
			name:     "Windows",
			goos:     "windows",
			wantPath: windowsOpenCommand,
			wantArgs: []string{windowsOpenCommand, "url.dll,FileProtocolHandler", aboutURL},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertBrowserCommand(t, test.goos, test.wantPath, test.wantArgs)
		})
	}
}

func assertBrowserCommand(t *testing.T, goos, wantPath string, wantArgs []string) {
	t.Helper()

	command, err := browserCommand(goos, aboutURL)
	if err != nil {
		t.Fatalf("browserCommand() error = %v", err)
	}
	if command.Path != wantPath {
		t.Fatalf("command path = %q, want %q", command.Path, wantPath)
	}
	if !slices.Equal(command.Args, wantArgs) {
		t.Fatalf("command args = %#v, want %#v", command.Args, wantArgs)
	}
}

func TestBrowserCommand_RejectsUnsupportedPlatform(t *testing.T) {
	if _, err := browserCommand("plan9", aboutURL); err == nil {
		t.Fatal("browserCommand() error = nil, want unsupported platform error")
	}
}
