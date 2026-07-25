package tui

import (
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	recordmodel "github.com/xhrobj/gopherkeeper/internal/model"
)

func TestRecordWorkspaceDimensions(t *testing.T) {
	if got := recordWorkspaceWidth(40); got != 56 {
		t.Fatalf("small width = %d, want 56", got)
	}
	if got := recordWorkspaceWidth(200); got != 112 {
		t.Fatalf("large width = %d, want 112", got)
	}
	if got := recordWorkspaceHeight(10); got != 14 {
		t.Fatalf("small height = %d, want 14", got)
	}
	if got := recordWorkspacePageSize(25); got < 1 {
		t.Fatalf("page size = %d", got)
	}
}

func TestRenderRecordWorkspace_RendersFoxProTable(t *testing.T) {
	workspace := recordWorkspace{
		open:     true,
		state:    recordListReady,
		selected: 1,
		records: []recordmodel.RecordMetadata{
			{ID: "first-id", Type: recordmodel.RecordTypeText, Title: "Recovery codes", Revision: 1},
			{
				ID:        "second-id",
				Type:      recordmodel.RecordTypeCredentials,
				Title:     "GitHub",
				Revision:  3,
				UpdatedAt: time.Date(2026, time.July, 19, 12, 0, 0, 0, time.Local),
			},
		},
	}

	view := ansi.Strip(renderRecordWorkspace(newTheme(), 90, 21, workspace, true, false, ""))
	for _, want := range []string{
		"Records Online",
		"TYPE",
		"TITLE",
		"REV",
		"UPDATED",
		"│",
		"─┼─",
		"credentials",
		"GitHub",
		"2026-07-19 12:00",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("view does not contain %q:\n%s", want, view)
		}
	}
	for _, unwanted := range []string{"ID: second-id", "PgUp/PgDn", "Server", "alice"} {
		if strings.Contains(view, unwanted) {
			t.Fatalf("record workspace contains redundant text %q:\n%s", unwanted, view)
		}
	}
}

func TestRenderRecordWorkspace_ShowsCacheLoginInTitle(t *testing.T) {
	workspace := recordWorkspace{
		source: recordSourceCache,
		login:  "alice",
		open:   true,
		state:  recordListReady,
	}

	view := ansi.Strip(renderRecordWorkspace(newTheme(), 90, 21, workspace, true, false, ""))
	if !strings.Contains(view, "Records from Cache: alice") {
		t.Fatalf("cache title does not contain login:\n%s", view)
	}
}

func TestRenderRecordWorkspace_UsesInactiveTitleForCoveredWindow(t *testing.T) {
	theme := newTheme()
	active := renderRecordWorkspace(theme, 70, 18, recordWorkspace{}, true, false, "")
	inactive := renderRecordWorkspace(theme, 70, 18, recordWorkspace{}, false, false, "")

	if !strings.Contains(active, renderWindowTitle(theme.windowTitle, 70, "Records Online", "", false)) {
		t.Fatal("active Records title does not use the active window-title style")
	}
	if !strings.Contains(inactive, renderWindowTitle(theme.windowTitleInactive, 70, "Records Online", "", false)) {
		t.Fatal("covered Records title does not use the inactive window-title style")
	}
}

func TestRenderRecordRows_ShowsTransientStates(t *testing.T) {
	tests := []struct {
		name      string
		workspace recordWorkspace
		want      string
	}{
		{name: "loading", workspace: recordWorkspace{state: recordListLoading}, want: ""},
		{name: "failed", workspace: recordWorkspace{state: recordListFailed, failure: "Connection refused"}, want: "Connection refused"},
		{name: "failed fallback", workspace: recordWorkspace{state: recordListFailed}, want: "Unable to load records"},
		{name: "empty", workspace: recordWorkspace{state: recordListReady}, want: `¯\_(ツ)_/¯ no records found`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rows := renderRecordRows(newTheme(), 70, 3, test.workspace)
			if len(rows) != 3 {
				t.Fatalf("row count = %d, want 3", len(rows))
			}
			view := ansi.Strip(strings.Join(rows, "\n"))
			if test.want == "" {
				if strings.TrimSpace(view) != "" {
					t.Fatalf("loading rows are not empty:\n%s", view)
				}
			} else if !strings.Contains(view, test.want) {
				t.Fatalf("rows do not contain %q:\n%s", test.want, view)
			}
		})
	}
}

func TestRecordColumns_PreserveMinimumTitleWidth(t *testing.T) {
	columns := recordColumns(20)
	if columns.title != 8 {
		t.Fatalf("title width = %d, want minimum 8", columns.title)
	}
	line := ansi.Strip(renderRecordMetadataLine(newTheme(), columns, recordmodel.RecordMetadata{
		Type:     recordmodel.RecordTypeCredentials,
		Title:    "very long title",
		Revision: 42,
	}, false))
	if !strings.Contains(line, "very lo…") {
		t.Fatalf("long title was not truncated predictably: %q", line)
	}
}

func TestRenderRecordWorkspace_ShowsSpinnerInStableTitleAndKeepsPreviousTable(t *testing.T) {
	theme := newTheme()
	workspace := recordWorkspace{
		state: recordListReady,
		records: []recordmodel.RecordMetadata{{
			ID: "42", Type: recordmodel.RecordTypeText, Title: "Previous record", Revision: 1,
		}},
	}
	idle := ansi.Strip(renderRecordWorkspace(theme, 70, 18, workspace, true, false, ""))
	loading := ansi.Strip(renderRecordWorkspace(theme, 70, 18, workspace, true, true, "⠋"))

	if strings.Index(idle, "Records") != strings.Index(loading, "Records") {
		t.Fatalf("Records title moved while loading:\nidle:\n%s\nloading:\n%s", idle, loading)
	}
	if !strings.Contains(loading, "Records Online ⠋") {
		t.Fatalf("loading title has no spinner:\n%s", loading)
	}
	if !strings.Contains(loading, "Previous record") {
		t.Fatalf("previous table disappeared during refresh:\n%s", loading)
	}
	if strings.Contains(loading, "Loading records") || strings.Contains(loading, "Please wait") {
		t.Fatalf("loading workspace still contains a loading message or overlay:\n%s", loading)
	}
}

func TestRenderRecordMetadataLine_UsesWhiteTypeText(t *testing.T) {
	theme := newTheme()
	columns := recordColumns(80)
	metadata := recordmodel.RecordMetadata{Type: recordmodel.RecordTypeCredentials, Title: "Account", Revision: 1}
	line := renderRecordMetadataLine(theme, columns, metadata, false)
	want := renderRecordCell(theme.recordRow, string(metadata.Type), columns.recordType, lipgloss.Left)
	if !strings.Contains(line, want) {
		t.Fatalf("record type does not use regular white row style:\n%s", line)
	}
}
