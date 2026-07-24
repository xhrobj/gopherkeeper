package tui

import (
	"time"

	recordmodel "github.com/xhrobj/gopherkeeper/internal/model"
)

type recordListState int

const (
	recordListIdle recordListState = iota
	recordListLoading
	recordListReady
	recordListFailed

	recordDoubleClickInterval = 350 * time.Millisecond
)

type recordWorkspace struct {
	open           bool
	state          recordListState
	records        []recordmodel.RecordMetadata
	selected       int
	offset         int
	failure        string
	lastClickIndex int
	lastClickAt    time.Time
	lastClickValid bool
}

func (workspace *recordWorkspace) beginServer() {
	workspace.open = true
	workspace.clearClick()
	if workspace.state == recordListReady {
		return
	}

	workspace.state = recordListLoading
	workspace.records = nil
	workspace.selected = 0
	workspace.offset = 0
	workspace.failure = ""
}

func (workspace *recordWorkspace) apply(records []recordmodel.RecordMetadata, pageSize int) {
	selectedIndex := workspace.selected
	selectedID := ""
	if workspace.hasSelection() {
		selectedID = workspace.records[workspace.selected].ID
	}

	workspace.records = append([]recordmodel.RecordMetadata(nil), records...)
	workspace.state = recordListReady
	workspace.failure = ""

	if len(workspace.records) == 0 {
		workspace.selected = 0
		workspace.offset = 0
		workspace.clearClick()
		return
	}

	workspace.selected = clamp(selectedIndex, 0, len(workspace.records)-1)
	if selectedID != "" {
		for index, metadata := range workspace.records {
			if metadata.ID == selectedID {
				workspace.selected = index
				break
			}
		}
	}
	workspace.ensureVisible(pageSize)
	workspace.clearClick()
}

func (workspace *recordWorkspace) prepend(metadata recordmodel.RecordMetadata, pageSize int) {
	workspace.open = true
	if workspace.state != recordListReady {
		workspace.records = []recordmodel.RecordMetadata{metadata}
		workspace.state = recordListReady
		workspace.failure = ""
		workspace.selected = 0
		workspace.offset = 0
		workspace.clearClick()

		return
	}

	filtered := make([]recordmodel.RecordMetadata, 0, len(workspace.records)+1)
	filtered = append(filtered, metadata)

	for _, current := range workspace.records {
		if current.ID != metadata.ID {
			filtered = append(filtered, current)
		}
	}

	workspace.records = filtered
	workspace.selected = 0
	workspace.offset = 0
	workspace.ensureVisible(pageSize)
	workspace.clearClick()
}

func (workspace *recordWorkspace) remove(recordID string, pageSize int) bool {
	index := -1
	for currentIndex, metadata := range workspace.records {
		if metadata.ID == recordID {
			index = currentIndex
			break
		}
	}

	if index < 0 {
		return false
	}

	workspace.records = append(workspace.records[:index], workspace.records[index+1:]...)
	if len(workspace.records) == 0 {
		workspace.selected = 0
		workspace.offset = 0
	} else {
		workspace.selected = clamp(workspace.selected, 0, len(workspace.records)-1)
		workspace.ensureVisible(pageSize)
	}
	workspace.clearClick()

	return true
}

func (workspace *recordWorkspace) fail(message string) {
	workspace.state = recordListFailed
	workspace.records = nil
	workspace.selected = 0
	workspace.offset = 0
	workspace.failure = message
	workspace.clearClick()
}

func (workspace *recordWorkspace) clear() {
	workspace.open = false
	workspace.state = recordListIdle
	workspace.records = nil
	workspace.selected = 0
	workspace.offset = 0
	workspace.failure = ""
	workspace.clearClick()
}

func (workspace *recordWorkspace) clearClick() {
	workspace.lastClickIndex = 0
	workspace.lastClickAt = time.Time{}
	workspace.lastClickValid = false
}

func (workspace *recordWorkspace) registerClick(index int, now time.Time) bool {
	if workspace.lastClickValid && workspace.lastClickIndex == index && !workspace.lastClickAt.IsZero() && now.Sub(workspace.lastClickAt) <= recordDoubleClickInterval {
		workspace.clearClick()
		return true
	}

	workspace.lastClickIndex = index
	workspace.lastClickAt = now
	workspace.lastClickValid = true

	return false
}

func (workspace recordWorkspace) hasSelection() bool {
	return workspace.state == recordListReady &&
		workspace.selected >= 0 && workspace.selected < len(workspace.records)
}

func (workspace recordWorkspace) selectedRecord() (recordmodel.RecordMetadata, bool) {
	if !workspace.hasSelection() {
		return recordmodel.RecordMetadata{}, false
	}
	return workspace.records[workspace.selected], true
}

func (workspace *recordWorkspace) move(step, pageSize int) {
	if len(workspace.records) == 0 {
		workspace.selected = 0
		workspace.offset = 0
		workspace.clearClick()
		return
	}

	workspace.selected = clamp(workspace.selected+step, 0, len(workspace.records)-1)
	workspace.ensureVisible(pageSize)
	workspace.clearClick()
}

func (workspace *recordWorkspace) moveTo(index, pageSize int) {
	if len(workspace.records) == 0 {
		workspace.selected = 0
		workspace.offset = 0
		return
	}

	workspace.selected = clamp(index, 0, len(workspace.records)-1)
	workspace.ensureVisible(pageSize)
}

func (workspace *recordWorkspace) ensureVisible(pageSize int) {
	pageSize = max(1, pageSize)
	if workspace.selected < workspace.offset {
		workspace.offset = workspace.selected
	}
	if workspace.selected >= workspace.offset+pageSize {
		workspace.offset = workspace.selected - pageSize + 1
	}
	workspace.offset = clamp(workspace.offset, 0, max(0, len(workspace.records)-pageSize))
}
