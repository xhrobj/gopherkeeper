package tui

import (
	"context"
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/xhrobj/gopherkeeper/internal/buildinfo"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
	"github.com/xhrobj/gopherkeeper/internal/client/failure"
)

func commandResult[T any](t *testing.T, command tea.Cmd) T {
	t.Helper()
	var zero T
	if command == nil {
		t.Fatal("command = nil")
		return zero
	}
	message := command()
	if result, ok := message.(T); ok {
		return result
	}
	value := reflect.ValueOf(message)
	if value.IsValid() && value.Kind() == reflect.Slice {
		for index := 0; index < value.Len(); index++ {
			candidate, ok := value.Index(index).Interface().(tea.Cmd)
			if !ok || candidate == nil {
				continue
			}
			if result, ok := candidate().(T); ok {
				return result
			}
		}
	}
	t.Fatalf("command result = %T, want %T", message, zero)
	return zero
}

func mustNewModel(
	t *testing.T,
	ctx context.Context,
	cfg config.Config,
	configFile string,
	info buildinfo.Info,
	backendFactory BackendFactory,
) model {
	t.Helper()

	m, err := newModel(ctx, cfg, configFile, info, backendFactory)
	if err != nil {
		t.Fatalf("newModel() error = %v", err)
	}
	return m
}

func newTestModel(t *testing.T, cfg config.Config, info buildinfo.Info) model {
	t.Helper()

	m, err := newModel(
		context.Background(),
		cfg,
		"",
		info,
		staticBackendFactory(backendStub{}),
	)
	if err != nil {
		t.Fatalf("newModel() error = %v", err)
	}
	m.operations.cancel(operationCurrentUser)
	m.startupCmd = nil
	m.authentication.session = authSession{state: authGuest}
	m.dialog = dialogLogin
	m.statusMinDuration = 0

	return m
}

func testMenuDefinitions(dialog dialogID) []menuDefinition {
	return menuDefinitions(dialog, false)
}

func testRenderDropdown(t theme, definition menuDefinition, selected int) string {
	return renderDropdown(t, definition, selected, actionNone)
}

func lineIndexContaining(lines []string, value string) int {
	for index, line := range lines {
		if strings.Contains(line, value) {
			return index
		}
	}
	return -1
}
func assertViewExcludes(t *testing.T, value string, parts ...string) {
	t.Helper()

	plain := ansi.Strip(value)
	for _, part := range parts {
		if strings.Contains(plain, part) {
			t.Errorf("view unexpectedly contains %q: %q", part, plain)
		}
	}
}
func keyPress(key string) tea.KeyPressMsg {
	if key == "enter" {
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter})
	}
	return tea.KeyPressMsg(tea.Key{Text: key, Code: []rune(key)[0]})
}
func assertViewContains(t *testing.T, value string, parts ...string) {
	t.Helper()

	plain := ansi.Strip(value)
	for _, part := range parts {
		if !strings.Contains(plain, part) {
			t.Errorf("view does not contain %q: %q", part, plain)
		}
	}
}

func unavailableTestError() error {
	return failure.Wrap(
		failure.Unavailable,
		"test server unavailable",
		failure.Reason(failure.Unavailable),
		nil,
	)
}

func tlsCertificateTestError() error {
	return failure.Wrap(
		failure.TLSCertificate,
		"test TLS certificate failure",
		failure.Reason(failure.TLSCertificate),
		nil,
	)
}

func assertRenderedLabelsWithinBounds(
	t *testing.T,
	content string,
	bounds []layoutBounds,
	labels []string,
	group string,
) {
	t.Helper()

	if len(bounds) != len(labels) {
		t.Fatalf("%s bounds = %d, want %d", group, len(bounds), len(labels))
	}
	lines := strings.Split(ansi.Strip(content), "\n")
	row := lineIndexContaining(lines, labels[0])
	if row < 0 {
		t.Fatalf("rendered %s controls were not found", group)
	}

	for index, label := range labels {
		if bounds[index].y != row {
			t.Fatalf("%s %d y = %d, want rendered row %d", group, index, bounds[index].y, row)
		}
		assertRenderedLabelWithinBounds(t, lines, group, label, bounds[index])
	}
}

func assertRenderedLabelWithinBounds(
	t *testing.T,
	lines []string,
	name string,
	label string,
	bounds layoutBounds,
) {
	t.Helper()

	if bounds.y < 0 || bounds.y >= len(lines) {
		t.Fatalf("%s y = %d, rendered height = %d", name, bounds.y, len(lines))
	}
	labelX := strings.Index(lines[bounds.y], label)
	if labelX < 0 {
		t.Fatalf("%s label %q was not found on rendered row %d: %q", name, label, bounds.y, lines[bounds.y])
	}
	labelWidth := lipgloss.Width(label)
	if labelX < bounds.x || labelX+labelWidth > bounds.x+bounds.width {
		t.Fatalf(
			"%s label range %d..%d is outside bounds %d..%d",
			name,
			labelX,
			labelX+labelWidth,
			bounds.x,
			bounds.x+bounds.width,
		)
	}
}
