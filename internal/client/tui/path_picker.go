package tui

import (
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
)

type pathPickerTarget int

const (
	pathPickerCACert pathPickerTarget = iota
	pathPickerSessionDirectory
	pathPickerCacheDirectory
	pathPickerBinarySaveDirectory
	pathPickerBinaryCreateFile
	pathPickerBinaryEditFile

	pathPickerDoubleClickInterval = 350 * time.Millisecond
)

type pathPickerMode struct {
	title      string
	filePicker bool
}

var pathPickerModes = map[pathPickerTarget]pathPickerMode{
	pathPickerCACert:              {title: "Select CA certificate", filePicker: true},
	pathPickerSessionDirectory:    {title: "Select session directory"},
	pathPickerCacheDirectory:      {title: "Select cache directory"},
	pathPickerBinarySaveDirectory: {title: "Select save directory"},
	pathPickerBinaryCreateFile:    {title: "Select binary file", filePicker: true},
	pathPickerBinaryEditFile:      {title: "Select binary file", filePicker: true},
}

func (target pathPickerTarget) mode() pathPickerMode {
	if mode, ok := pathPickerModes[target]; ok {
		return mode
	}
	return pathPickerMode{title: "Select path"}
}

type pathPickerEntry struct {
	name      string
	directory bool
	parent    bool
}

type pathPickerFocus int

const (
	pathPickerTree pathPickerFocus = iota
	pathPickerSelect
	pathPickerCancel
)

type pathPickerReadMsg struct {
	rootDirectory    string
	currentDirectory string
	entries          []pathPickerEntry
	err              error
}

type pathPickerMouseOpenMsg struct {
	rootDirectory    string
	currentDirectory string
	entryIndex       int
	generation       uint64
}

type pathPicker struct {
	target           pathPickerTarget
	rootDirectory    string
	currentDirectory string
	entries          []pathPickerEntry
	selected         int
	offset           int
	height           int
	selectName       string
	loadError        string
	focus            pathPickerFocus
	lastClickIndex   int
	lastClickAt      time.Time
	lastClickValid   bool
	clickGeneration  uint64
	fileName         string
}

func newPathPicker(
	target pathPickerTarget,
	currentValue string,
	screenHeight int,
) (pathPicker, tea.Cmd) {
	rootDirectory := pathPickerRootDirectory()
	startDirectory := pathPickerStartDirectory(currentValue, rootDirectory)

	state := pathPicker{
		target:           target,
		rootDirectory:    rootDirectory,
		currentDirectory: startDirectory,
		height:           pathPickerListHeight(screenHeight),
		focus:            pathPickerTree,
	}

	if target == pathPickerBinarySaveDirectory {
		state.fileName = filepath.Base(strings.TrimSpace(currentValue))
	}

	state.selectName = pathPickerInitialSelection(target, currentValue, rootDirectory, startDirectory)

	return state, state.readDirectory()
}

func (picker pathPicker) isDirectoryPicker() bool {
	return !picker.isFilePicker()
}

func (picker pathPicker) isFilePicker() bool {
	return picker.target.mode().filePicker
}

func (picker *pathPicker) resize(screenHeight int) {
	picker.height = pathPickerListHeight(screenHeight)
	picker.ensureVisible()
}

func (picker *pathPicker) move(step int) {
	if len(picker.entries) == 0 {
		picker.selected = 0
		picker.offset = 0
		return
	}

	picker.selected = clamp(picker.selected+step, 0, len(picker.entries)-1)
	picker.ensureVisible()
}

func (picker *pathPicker) ensureVisible() {
	if len(picker.entries) == 0 {
		picker.offset = 0
		return
	}

	visibleHeight := max(1, picker.height)
	if picker.selected < picker.offset {
		picker.offset = picker.selected
	}

	if picker.selected >= picker.offset+visibleHeight {
		picker.offset = picker.selected - visibleHeight + 1
	}

	maxOffset := max(0, len(picker.entries)-visibleHeight)
	picker.offset = clamp(picker.offset, 0, maxOffset)
}

func (picker *pathPicker) clampSelection() {
	if len(picker.entries) == 0 {
		picker.selected = 0
		picker.offset = 0
		return
	}

	picker.selected = clamp(picker.selected, 0, len(picker.entries)-1)
	picker.ensureVisible()
}

func (picker pathPicker) highlighted() (pathPickerEntry, bool) {
	if picker.selected < 0 || picker.selected >= len(picker.entries) {
		return pathPickerEntry{}, false
	}
	return picker.entries[picker.selected], true
}

func (picker pathPicker) highlightedPath() string {
	entry, ok := picker.highlighted()
	if !ok {
		return ""
	}

	if entry.parent {
		return filepath.Dir(picker.currentDirectory)
	}

	return filepath.Join(picker.currentDirectory, entry.name)
}

func (picker pathPicker) canSelect(entry pathPickerEntry) bool {
	if picker.isDirectoryPicker() {
		return entry.directory
	}
	return !entry.parent && !entry.directory
}

func (picker pathPicker) selectEnabled() bool {
	entry, ok := picker.highlighted()
	return ok && picker.canSelect(entry)
}

func (picker *pathPicker) moveFocus(step int) {
	order := [...]pathPickerFocus{
		pathPickerTree,
		pathPickerSelect,
		pathPickerCancel,
	}

	current := 0
	for index, focus := range order {
		if focus == picker.focus {
			current = index
			break
		}
	}

	for range len(order) {
		current = (current + step + len(order)) % len(order)
		candidate := order[current]

		if candidate == pathPickerSelect && !picker.selectEnabled() {
			continue
		}

		picker.focus = candidate

		return
	}
}

func (picker *pathPicker) focusTree() {
	picker.focus = pathPickerTree
}

func (picker pathPicker) selectedPath() (string, bool) {
	entry, ok := picker.highlighted()
	if !ok || !picker.canSelect(entry) {
		return "", false
	}

	if picker.isDirectoryPicker() && entry.parent {
		return picker.currentDirectory, true
	}

	return picker.highlightedPath(), true
}

func (picker pathPicker) selectedValue() string {
	path, ok := picker.selectedPath()
	if !ok {
		return ""
	}

	value := selectedPathValue(picker.rootDirectory, path)
	if picker.target == pathPickerBinarySaveDirectory && picker.fileName != "" {
		value = filepath.Join(value, picker.fileName)
		if value == filepath.Join(".", picker.fileName) {
			value = picker.fileName
		}
	}

	return value
}

func (picker *pathPicker) openHighlightedDirectory() tea.Cmd {
	entry, ok := picker.highlighted()
	if !ok || !entry.directory {
		return nil
	}

	if entry.parent {
		return picker.up()
	}

	picker.clearMouseClick()
	picker.currentDirectory = filepath.Join(picker.currentDirectory, entry.name)
	picker.entries = nil
	picker.selected = 0
	picker.offset = 0
	picker.loadError = ""

	return picker.readDirectory()
}

func (picker *pathPicker) up() tea.Cmd {
	currentDirectory := filepath.Clean(picker.currentDirectory)
	rootDirectory := filepath.Clean(picker.rootDirectory)

	if currentDirectory == rootDirectory {
		return nil
	}

	parentDirectory := filepath.Dir(currentDirectory)

	if !pathWithinRoot(rootDirectory, parentDirectory) {
		parentDirectory = rootDirectory
	}

	picker.clearMouseClick()
	picker.selectName = filepath.Base(currentDirectory)
	picker.currentDirectory = parentDirectory
	picker.entries = nil
	picker.selected = 0
	picker.offset = 0
	picker.loadError = ""

	return picker.readDirectory()
}

func (picker pathPicker) title() string {
	return picker.target.mode().title
}
