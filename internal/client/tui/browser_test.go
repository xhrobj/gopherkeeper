package tui

import "testing"

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
			wantPath: "open",
			wantArgs: []string{"open", aboutURL},
		},
		{
			name:     "Linux",
			goos:     "linux",
			wantPath: "xdg-open",
			wantArgs: []string{"xdg-open", aboutURL},
		},
		{
			name:     "Windows",
			goos:     "windows",
			wantPath: "rundll32",
			wantArgs: []string{"rundll32", "url.dll,FileProtocolHandler", aboutURL},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			command, err := browserCommand(test.goos, aboutURL)
			if err != nil {
				t.Fatalf("browserCommand() error = %v", err)
			}
			if command.Path == "" || command.Args[0] != test.wantPath {
				t.Fatalf("command path = %q args = %#v, want %q", command.Path, command.Args, test.wantPath)
			}
			if len(command.Args) != len(test.wantArgs) {
				t.Fatalf("command args = %#v, want %#v", command.Args, test.wantArgs)
			}
			for index := range test.wantArgs {
				if command.Args[index] != test.wantArgs[index] {
					t.Fatalf("command args = %#v, want %#v", command.Args, test.wantArgs)
				}
			}
		})
	}
}

func TestBrowserCommand_RejectsUnsupportedPlatform(t *testing.T) {
	if _, err := browserCommand("plan9", aboutURL); err == nil {
		t.Fatal("browserCommand() error = nil, want unsupported platform error")
	}
}
