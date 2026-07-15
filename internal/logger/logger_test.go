package logger

import (
	"strings"
	"testing"

	"go.uber.org/zap"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name      string
		level     string
		wantDebug bool
		wantInfo  bool
		wantError string
	}{
		{name: "default level", wantInfo: true},
		{name: "debug level", level: "debug", wantDebug: true, wantInfo: true},
		{name: "error level", level: "error"},
		{name: "invalid level", level: "verbose", wantError: `LOG_LEVEL "verbose"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(levelEnv, tt.level)

			logger, err := New()
			if tt.wantError != "" {
				assertNewError(t, err, tt.wantError)
				return
			}

			if err != nil {
				t.Fatalf("New() error = %v", err)
			}

			t.Cleanup(func() {
				_ = logger.Sync()
			})

			if got := logger.Core().Enabled(zap.DebugLevel); got != tt.wantDebug {
				t.Errorf("debug enabled = %t, want %t", got, tt.wantDebug)
			}

			if got := logger.Core().Enabled(zap.InfoLevel); got != tt.wantInfo {
				t.Errorf("info enabled = %t, want %t", got, tt.wantInfo)
			}
		})
	}
}

func assertNewError(t *testing.T, err error, wantError string) {
	t.Helper()

	if err == nil {
		t.Fatal("New() error = nil, want error")
	}

	if !strings.Contains(err.Error(), wantError) {
		t.Fatalf("New() error = %q, want substring %q", err, wantError)
	}
}
