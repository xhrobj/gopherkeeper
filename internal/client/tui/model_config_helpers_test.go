package tui

import (
	"os"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/client/config"
)

func changeWorkingDirectory(t *testing.T, directory string) {
	t.Helper()
	original, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if err := os.Chdir(directory); err != nil {
		t.Fatalf("change working directory: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(original); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})
}

func withConfigAddress(cfg config.Config, value string) config.Config {
	cfg.Address = value
	return cfg
}

func withConfigGRPCAddress(cfg config.Config, value string) config.Config {
	cfg.GRPCAddress = value
	return cfg
}

func withConfigCACertFile(cfg config.Config, value string) config.Config {
	cfg.CACertFile = value
	return cfg
}

func withConfigSessionDir(cfg config.Config, value string) config.Config {
	cfg.SessionDir = value
	return cfg
}

func withConfigCacheDir(cfg config.Config, value string) config.Config {
	cfg.CacheDir = value
	return cfg
}

func assertConfigTransportGeometry(t *testing.T, lines []string, bounds []layoutBounds) {
	t.Helper()

	labels := []string{"[X] HTTPS", "[ ] gRPC"}
	if len(bounds) != len(labels) {
		t.Fatalf("transport bounds = %d, want %d", len(bounds), len(labels))
	}
	for index, label := range labels {
		assertRenderedLabelWithinBounds(t, lines, "transport", label, bounds[index])
	}
}

func assertConfigFieldGeometry(t *testing.T, lines []string, bounds []layoutBounds, cfg config.Config) {
	t.Helper()

	values := []string{cfg.Address, cfg.GRPCAddress, cfg.CACertFile, cfg.SessionDir, cfg.CacheDir}
	if len(bounds) != len(values) {
		t.Fatalf("field bounds = %d, want %d", len(bounds), len(values))
	}
	for index, value := range values {
		assertRenderedLabelWithinBounds(t, lines, "field", value, bounds[index])
	}
}

func assertConfigBrowseGeometry(
	t *testing.T,
	lines []string,
	browseBounds []layoutBounds,
	fieldBounds []layoutBounds,
) {
	t.Helper()

	if len(browseBounds) != 3 {
		t.Fatalf("browse bounds = %d, want 3", len(browseBounds))
	}
	for index, bounds := range browseBounds {
		assertRenderedLabelWithinBounds(t, lines, "browse button", configBrowseLabel, bounds)
		if bounds.y != fieldBounds[index+2].y {
			t.Fatalf("browse button %d y = %d, field y = %d", index, bounds.y, fieldBounds[index+2].y)
		}
	}
}

func assertConfigButtonGeometry(t *testing.T, lines []string, bounds []layoutBounds) {
	t.Helper()

	labels := []string{"< Save >", "< Cancel >"}
	if len(bounds) != len(labels) {
		t.Fatalf("button bounds = %d, want %d", len(bounds), len(labels))
	}
	for index, label := range labels {
		assertRenderedLabelWithinBounds(t, lines, "dialog button", label, bounds[index])
	}
}
