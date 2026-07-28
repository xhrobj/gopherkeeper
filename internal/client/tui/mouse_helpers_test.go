package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/xhrobj/gopherkeeper/internal/buildinfo"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
	recordmodel "github.com/xhrobj/gopherkeeper/internal/model"
)

func mouseClick(x, y int) tea.MouseClickMsg {
	return tea.MouseClickMsg(tea.Mouse{X: x, Y: y, Button: tea.MouseLeft})
}

func assertRecordViewButtonBounds(t *testing.T, record recordmodel.Record, label string) {
	t.Helper()
	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.width = 100
	m.height = 36
	m.dialog = dialogRecordView
	m.recordFeature.view = recordViewState{status: recordViewReady, record: record}

	window := m.renderDialog()
	buttonRow := lastLineIndexContaining(strings.Split(ansi.Strip(window), "\n"), label)
	if buttonRow < 0 {
		t.Fatalf("rendered button %q was not found", label)
	}

	buttons := m.dialogButtonBounds()
	if len(buttons) == 0 {
		t.Fatal("record view button bounds are empty")
	}
	wantY := max(2, (m.height-lipgloss.Height(window))/2) + buttonRow
	if buttons[0].y != wantY {
		t.Fatalf("button y = %d, want rendered row %d", buttons[0].y, wantY)
	}
}

func lastLineIndexContaining(lines []string, value string) int {
	for index := len(lines) - 1; index >= 0; index-- {
		if strings.Contains(lines[index], value) {
			return index
		}
	}
	return -1
}

func assertAuthWindowBoundsMatchRenderedControls(
	t *testing.T,
	dialog dialogID,
	buttonLabel string,
	fieldCount int,
	setup func(*model),
	fields func(model) []layoutBounds,
) {
	t.Helper()

	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.width = 100
	m.height = 34
	m.dialog = dialog
	setup(&m)

	window := m.renderDialog()
	lines := strings.Split(ansi.Strip(window), "\n")
	fieldRow := lineIndexContaining(lines, "alice")
	buttonRow := lineIndexContaining(lines, buttonLabel)
	if fieldRow < 0 || buttonRow < 0 {
		t.Fatalf("rendered controls were not found:\n%s", ansi.Strip(window))
	}

	windowY := max(2, (m.height-lipgloss.Height(window))/2)
	assertAuthFieldBounds(t, fields(m), fieldCount, windowY+fieldRow)
	assertAuthButtonBounds(t, m.dialogButtonBounds(), windowY+buttonRow)
}

func assertAuthFieldBounds(t *testing.T, bounds []layoutBounds, wantCount, firstRow int) {
	t.Helper()

	if len(bounds) != wantCount {
		t.Fatalf("field count = %d, want %d", len(bounds), wantCount)
	}
	for index, fieldBounds := range bounds {
		wantY := firstRow + index*2
		if fieldBounds.y != wantY {
			t.Fatalf("field %d y = %d, want rendered row %d", index, fieldBounds.y, wantY)
		}
	}
}

func assertAuthButtonBounds(t *testing.T, bounds []layoutBounds, wantY int) {
	t.Helper()

	if len(bounds) != 2 {
		t.Fatalf("button count = %d, want 2", len(bounds))
	}
	for index, buttonBounds := range bounds {
		if buttonBounds.y != wantY {
			t.Fatalf("button %d y = %d, want rendered row %d", index, buttonBounds.y, wantY)
		}
	}
}
