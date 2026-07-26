package tui

import (
	"testing"
	"time"

	recordmodel "github.com/xhrobj/gopherkeeper/internal/model"
)

func TestRecordWorkspace_BeginServerPreservesReadyTable(t *testing.T) {
	workspace := recordWorkspace{
		open:     true,
		state:    recordListReady,
		records:  []recordmodel.RecordMetadata{{ID: "old"}},
		selected: 0,
		offset:   0,
	}

	workspace.beginServer()
	if !workspace.open || workspace.state != recordListReady {
		t.Fatalf("refresh workspace state = %#v", workspace)
	}
	if len(workspace.records) != 1 || workspace.records[0].ID != "old" {
		t.Fatalf("ready table was cleared during refresh: %#v", workspace)
	}

	workspace.clear()
	workspace.beginServer()
	if !workspace.open || workspace.state != recordListLoading || len(workspace.records) != 0 {
		t.Fatalf("initial server workspace = %#v", workspace)
	}
}

func TestRecordWorkspace_ApplyAndFail(t *testing.T) {
	workspace := recordWorkspace{open: true, state: recordListLoading}
	records := []recordmodel.RecordMetadata{{ID: "42"}}

	workspace.apply(records, 10)
	if workspace.state != recordListReady || len(workspace.records) != 1 {
		t.Fatalf("ready workspace = %#v", workspace)
	}
	records[0].ID = "changed"
	if workspace.records[0].ID != "42" {
		t.Fatal("workspace retained caller-owned record slice")
	}

	workspace.fail("Connection refused")
	if workspace.state != recordListFailed || workspace.failure != "Connection refused" {
		t.Fatalf("failed workspace = %#v", workspace)
	}
	if len(workspace.records) != 0 || workspace.selected != 0 || workspace.offset != 0 {
		t.Fatalf("failed workspace list state = %#v", workspace)
	}
	if _, ok := workspace.selectedRecord(); ok {
		t.Fatal("failed workspace unexpectedly has selected record")
	}
}

func TestRecordWorkspace_ApplyPreservesSelectionByID(t *testing.T) {
	workspace := recordWorkspace{
		state:    recordListReady,
		records:  []recordmodel.RecordMetadata{{ID: "a"}, {ID: "b"}, {ID: "c"}},
		selected: 1,
	}

	workspace.apply([]recordmodel.RecordMetadata{{ID: "c"}, {ID: "a"}, {ID: "b"}}, 2)

	selected, ok := workspace.selectedRecord()
	if !ok || selected.ID != "b" {
		t.Fatalf("selected record = %#v, %t; want b", selected, ok)
	}
	if workspace.selected != 2 || workspace.offset != 1 {
		t.Fatalf("selection after reorder = %d, offset = %d; want 2, 1", workspace.selected, workspace.offset)
	}
}

func TestRecordWorkspace_ApplyUsesNearestIndexWhenSelectionDisappears(t *testing.T) {
	workspace := recordWorkspace{
		state:    recordListReady,
		records:  []recordmodel.RecordMetadata{{ID: "a"}, {ID: "b"}, {ID: "c"}},
		selected: 1,
	}

	workspace.apply([]recordmodel.RecordMetadata{{ID: "a"}, {ID: "c"}}, 2)

	selected, ok := workspace.selectedRecord()
	if !ok || selected.ID != "c" {
		t.Fatalf("selected record = %#v, %t; want c", selected, ok)
	}
	if workspace.selected != 1 || workspace.offset != 0 {
		t.Fatalf("selection after removal = %d, offset = %d; want 1, 0", workspace.selected, workspace.offset)
	}
}

func TestRecordWorkspace_ApplyClampsSelectionAndOffsetAfterListShrinks(t *testing.T) {
	records := make([]recordmodel.RecordMetadata, 8)
	for index := range records {
		records[index].ID = string(rune('a' + index))
	}
	workspace := recordWorkspace{
		state:    recordListReady,
		records:  records,
		selected: 7,
		offset:   5,
	}

	workspace.apply(records[:3], 2)

	if workspace.selected != 2 || workspace.offset != 1 {
		t.Fatalf("selection after shrink = %d, offset = %d; want 2, 1", workspace.selected, workspace.offset)
	}
}

func TestRecordWorkspace_MoveKeepsSelectionVisible(t *testing.T) {
	records := make([]recordmodel.RecordMetadata, 8)
	for index := range records {
		records[index] = recordmodel.RecordMetadata{ID: string(rune('a' + index))}
	}
	workspace := recordWorkspace{state: recordListReady, records: records}

	workspace.move(5, 3)
	if workspace.selected != 5 || workspace.offset != 3 {
		t.Fatalf("after move: selected = %d, offset = %d, want 5, 3", workspace.selected, workspace.offset)
	}

	workspace.move(-4, 3)
	if workspace.selected != 1 || workspace.offset != 1 {
		t.Fatalf("after reverse move: selected = %d, offset = %d, want 1, 1", workspace.selected, workspace.offset)
	}

	workspace.moveTo(99, 3)
	if workspace.selected != 7 || workspace.offset != 5 {
		t.Fatalf("after moveTo end: selected = %d, offset = %d, want 7, 5", workspace.selected, workspace.offset)
	}

	workspace.moveTo(-1, 0)
	if workspace.selected != 0 || workspace.offset != 0 {
		t.Fatalf("after moveTo start: selected = %d, offset = %d, want 0, 0", workspace.selected, workspace.offset)
	}
}

func TestRecordWorkspace_MoveEmptyListResetsSelection(t *testing.T) {
	workspace := recordWorkspace{selected: 4, offset: 3}
	workspace.move(1, 5)
	if workspace.selected != 0 || workspace.offset != 0 {
		t.Fatalf("empty move: selected = %d, offset = %d", workspace.selected, workspace.offset)
	}
	workspace.selected = 4
	workspace.offset = 3
	workspace.moveTo(2, 5)
	if workspace.selected != 0 || workspace.offset != 0 {
		t.Fatalf("empty moveTo: selected = %d, offset = %d", workspace.selected, workspace.offset)
	}
}

func TestFormatRecordTime(t *testing.T) {
	if got := formatRecordTime(time.Time{}); got != "" {
		t.Fatalf("zero time = %q", got)
	}

	value := time.Date(2026, time.July, 19, 12, 34, 0, 0, time.Local)
	if got := formatRecordTime(value); got != "2026-07-19 12:34" {
		t.Fatalf("formatted time = %q", got)
	}
}

func TestRecordWorkspacePrepend_SwitchesToServerSource(t *testing.T) {
	workspace := recordWorkspace{source: recordSourceNone}
	workspace.prepend(recordmodel.RecordMetadata{
		ID: "server-id", Type: recordmodel.RecordTypeText, Title: "Server note", Revision: 1,
	}, 5)

	if workspace.source != recordSourceServer || !workspace.open || !workspace.hasSelection() {
		t.Fatalf("workspace after server prepend = %#v", workspace)
	}
}
