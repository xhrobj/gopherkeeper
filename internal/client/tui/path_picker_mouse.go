package tui

import (
	"path/filepath"
	"time"

	tea "charm.land/bubbletea/v2"
)

func (picker *pathPicker) registerMouseClick(index int, now time.Time) (bool, tea.Cmd) {
	if index < 0 || index >= len(picker.entries) {
		picker.clearMouseClick()
		return false, nil
	}

	picker.selected = index
	picker.ensureVisible()

	if picker.lastClickValid &&
		picker.lastClickIndex == index &&
		!picker.lastClickAt.IsZero() &&
		now.Sub(picker.lastClickAt) <= pathPickerDoubleClickInterval {
		picker.clearMouseClick()
		return true, nil
	}

	picker.clearMouseClick()
	picker.lastClickIndex = index
	picker.lastClickAt = now
	picker.lastClickValid = true

	generation := picker.clickGeneration
	rootDirectory := picker.rootDirectory
	currentDirectory := picker.currentDirectory

	return false, tea.Tick(pathPickerDoubleClickInterval, func(time.Time) tea.Msg {
		return pathPickerMouseOpenMsg{
			rootDirectory:    rootDirectory,
			currentDirectory: currentDirectory,
			entryIndex:       index,
			generation:       generation,
		}
	})
}

func (picker pathPicker) acceptsMouseOpen(msg pathPickerMouseOpenMsg) bool {
	return picker.lastClickValid &&
		picker.clickGeneration == msg.generation &&
		picker.lastClickIndex == msg.entryIndex &&
		filepath.Clean(picker.rootDirectory) == filepath.Clean(msg.rootDirectory) &&
		filepath.Clean(picker.currentDirectory) == filepath.Clean(msg.currentDirectory)
}

func (picker *pathPicker) clearMouseClick() {
	picker.lastClickIndex = 0
	picker.lastClickAt = time.Time{}
	picker.lastClickValid = false
	picker.clickGeneration++
}
