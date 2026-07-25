package tui

import (
	"context"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
	recordmodel "github.com/xhrobj/gopherkeeper/internal/model"
)

func TestModel_MouseClickSelectsRecordAndDoubleClickOpensIt(t *testing.T) {
	requestedID := ""
	backend := recordsBackendStub{
		getRecord: func(_ context.Context, id string) (recordmodel.Record, error) {
			requestedID = id
			return recordmodel.Record{}, nil
		},
	}
	m := newRecordsTestModel(t, config.Config{}, backend)
	m.width = 100
	m.height = 30
	m.recordFeature.workspace = recordWorkspace{
		open:    true,
		state:   recordListReady,
		records: []recordmodel.RecordMetadata{{ID: "first"}, {ID: "second"}, {ID: "third"}},
	}
	window, ok := m.workspacePlacement()
	if !ok {
		t.Fatal("workspace placement is unavailable")
	}

	updated, cmd := m.Update(mouseClick(window.x+3, window.y+recordWorkspaceFirstRowOffset+1))
	got := updated.(model)
	if got.recordFeature.workspace.selected != 1 {
		t.Fatalf("selected after single click = %d, want 1", got.recordFeature.workspace.selected)
	}
	if got.dialog != dialogNone {
		t.Fatalf("dialog after single click = %v, want none", got.dialog)
	}
	if cmd != nil {
		t.Fatal("single click unexpectedly started GetRecord")
	}

	updated, cmd = got.Update(mouseClick(window.x+3, window.y+recordWorkspaceFirstRowOffset+1))
	got = updated.(model)
	if got.dialog != dialogRecordView || got.recordFeature.view.status != recordViewLoading {
		t.Fatalf("record view state = dialog %v, status %v", got.dialog, got.recordFeature.view.status)
	}
	if cmd == nil {
		t.Fatal("double click did not start GetRecord")
	}
	_ = commandResult[recordViewResultMsg](t, cmd)
	if requestedID != "second" {
		t.Fatalf("requested ID = %q, want %q", requestedID, "second")
	}
}

func TestModel_ResizeKeepsRecordMouseHitboxInSyncWithVisiblePage(t *testing.T) {
	const recordsCount = 40

	records := make([]recordmodel.RecordMetadata, recordsCount)
	for index := range records {
		records[index].ID = string(rune('a' + index%26))
	}

	m := newRecordsTestModel(t, config.Config{}, recordsBackendStub{
		getRecord: func(context.Context, string) (recordmodel.Record, error) {
			return recordmodel.Record{}, nil
		},
	})
	m.width = 100
	m.height = 40
	m.recordFeature.workspace = recordWorkspace{
		open:     true,
		state:    recordListReady,
		records:  records,
		selected: recordsCount - 1,
		offset:   recordsCount - recordWorkspacePageSize(m.height),
	}

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 25})
	got := updated.(model)
	wantOffset := recordsCount - recordWorkspacePageSize(got.height)
	if got.recordFeature.workspace.offset != wantOffset {
		t.Fatalf("offset after resize = %d, want %d", got.recordFeature.workspace.offset, wantOffset)
	}

	window, ok := got.workspacePlacement()
	if !ok {
		t.Fatal("workspace placement is unavailable after resize")
	}

	updated, cmd := got.Update(mouseClick(
		window.x+recordWorkspaceBodyHorizontalPadding,
		window.y+recordWorkspaceFirstRowOffset,
	))
	got = updated.(model)
	if got.recordFeature.workspace.selected != wantOffset {
		t.Fatalf("selected after clicking first visible row = %d, want %d", got.recordFeature.workspace.selected, wantOffset)
	}
	if cmd != nil {
		t.Fatal("single click after resize unexpectedly started GetRecord")
	}
}
