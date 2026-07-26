package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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
	title       string
	filePicker  bool
	configFocus configFocus
	browseFocus configFocus
	configIndex int
}

var pathPickerModes = map[pathPickerTarget]pathPickerMode{
	pathPickerCACert:              {title: "Select CA certificate", filePicker: true, configFocus: configCACertFile, browseFocus: configCACertBrowse, configIndex: 1},
	pathPickerSessionDirectory:    {title: "Select session directory", configFocus: configSessionDir, browseFocus: configSessionBrowse, configIndex: 2},
	pathPickerCacheDirectory:      {title: "Select cache directory", configFocus: configCacheDir, browseFocus: configCacheBrowse, configIndex: 3},
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

func (picker pathPicker) readDirectory() tea.Cmd {
	rootDirectory := picker.rootDirectory
	currentDirectory := picker.currentDirectory
	directoriesOnly := picker.isDirectoryPicker()

	return func() tea.Msg {
		directoryEntries, err := os.ReadDir(currentDirectory)
		if err != nil {
			return pathPickerReadMsg{
				rootDirectory:    rootDirectory,
				currentDirectory: currentDirectory,
				err:              err,
			}
		}

		entries := make([]pathPickerEntry, 0, len(directoryEntries)+1)
		for _, entry := range directoryEntries {
			isDirectory := entry.IsDir()
			if directoriesOnly && !isDirectory {
				continue
			}
			entries = append(entries, pathPickerEntry{
				name:      entry.Name(),
				directory: isDirectory,
			})
		}

		sort.SliceStable(entries, func(i, j int) bool {
			if entries[i].directory != entries[j].directory {
				return entries[i].directory
			}
			return entries[i].name < entries[j].name
		})

		if filepath.Clean(currentDirectory) != filepath.Clean(rootDirectory) {
			entries = append([]pathPickerEntry{{
				name:      "..",
				directory: true,
				parent:    true,
			}}, entries...)
		}

		return pathPickerReadMsg{
			rootDirectory:    rootDirectory,
			currentDirectory: currentDirectory,
			entries:          entries,
		}
	}
}

func (picker *pathPicker) applyReadResult(msg pathPickerReadMsg) {
	if filepath.Clean(msg.rootDirectory) != filepath.Clean(picker.rootDirectory) ||
		filepath.Clean(msg.currentDirectory) != filepath.Clean(picker.currentDirectory) {
		return
	}

	picker.entries = msg.entries
	picker.loadError = ""
	if msg.err != nil {
		picker.loadError = fmt.Sprintf("Unable to read directory: %v", msg.err)
		picker.entries = nil
	}

	if picker.selectName != "" {
		for index, entry := range picker.entries {
			if entry.name == picker.selectName {
				picker.selected = index
				break
			}
		}
		picker.selectName = ""
	}

	picker.clampSelection()
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

func pathPickerRootDirectory() string {
	directory, err := os.Getwd()
	if err != nil {
		return filepath.Clean(".")
	}

	return canonicalPath(directory)
}

func canonicalPath(path string) string {
	path = filepath.Clean(path)

	resolved, err := filepath.EvalSymlinks(path)
	if err == nil {
		return filepath.Clean(resolved)
	}

	parent := filepath.Dir(path)
	if parent == path {
		return path
	}

	return filepath.Join(canonicalPath(parent), filepath.Base(path))
}

func pathPickerStartDirectory(currentValue, rootDirectory string) string {
	rootDirectory = filepath.Clean(rootDirectory)

	selectionPath, ok := pathPickerCurrentSelectionPath(currentValue, rootDirectory)
	if !ok {
		return rootDirectory
	}

	candidate := filepath.Clean(filepath.Dir(selectionPath))
	for pathWithinRoot(rootDirectory, candidate) {
		info, err := os.Stat(candidate)
		if err == nil && info.IsDir() {
			return candidate
		}

		parent := filepath.Dir(candidate)
		if parent == candidate {
			break
		}
		candidate = parent
	}

	return rootDirectory
}

func pathPickerInitialSelection(
	target pathPickerTarget,
	currentValue string,
	rootDirectory string,
	startDirectory string,
) string {
	if target == pathPickerBinarySaveDirectory {
		if filepath.Clean(startDirectory) != filepath.Clean(rootDirectory) {
			return ".."
		}
		return ""
	}

	selectionPath, ok := pathPickerCurrentSelectionPath(currentValue, rootDirectory)
	if !ok || !pathWithinRoot(rootDirectory, selectionPath) {
		return ""
	}
	if filepath.Clean(filepath.Dir(selectionPath)) != filepath.Clean(startDirectory) {
		return ""
	}
	return filepath.Base(selectionPath)
}

func pathPickerCurrentSelectionPath(currentValue, rootDirectory string) (string, bool) {
	value := strings.TrimSpace(currentValue)
	if value == "" {
		return "", false
	}

	if !filepath.IsAbs(value) {
		value = filepath.Join(rootDirectory, value)
	}

	value = canonicalPath(value)

	return value, true
}

func pathWithinRoot(rootDirectory, path string) bool {
	relative, err := filepath.Rel(canonicalPath(rootDirectory), canonicalPath(path))
	if err != nil {
		return false
	}

	return relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func pathPickerListHeight(screenHeight int) int {
	return clamp(screenHeight-12, 6, 14)
}

func pathPickerWindowWidth(screenWidth int) int {
	return clamp(screenWidth-16, 54, 82)
}

type pathPickerWindowLayout struct {
	contentWidth int
	listBounds   layoutBounds
}

func newPathPickerWindowLayout(windowWidth, listHeight int) pathPickerWindowLayout {
	contentWidth := max(1, windowWidth-4)

	return pathPickerWindowLayout{
		contentWidth: contentWidth,
		listBounds: layoutBounds{
			x:      2,
			y:      4,
			width:  contentWidth,
			height: max(1, listHeight),
		},
	}
}

func (picker pathPicker) title() string {
	return picker.target.mode().title
}

const pathPickerButtonGap = 3

func pathPickerButtonRow(listHeight int) int {
	return max(1, listHeight) + 5
}

func pathPickerButtonsLayout(t theme, width int, state pathPicker) buttonRowLayout {
	selectStyle := t.button
	cancelStyle := t.button

	if !state.selectEnabled() {
		selectStyle = t.buttonDisabled
	} else if state.focus == pathPickerSelect {
		selectStyle = t.buttonActive
	}

	if state.focus == pathPickerCancel {
		cancelStyle = t.buttonActive
	}

	return centeredButtonRowLayout(t.windowBody, width, pathPickerButtonGap, []styledButton{
		{label: "< Select >", style: selectStyle},
		{label: "< Cancel >", style: cancelStyle},
	})
}

func renderPathPickerButtons(t theme, width int, state pathPicker) string {
	return pathPickerButtonsLayout(t, width, state).content
}

func renderPathPickerWindow(t theme, width int, state pathPicker) string {
	layout := newPathPickerWindowLayout(width, state.height)
	contentWidth := layout.contentWidth
	selectedPath := renderConfigReadOnlyInput(
		t,
		t.readOnly,
		state.selectedValue(),
		max(1, contentWidth-lipgloss.Width("Path  ")),
		true,
	)
	pathRow := t.label.Render("Path") + t.windowBody.Width(2).Render("") + selectedPath

	list := renderPathPickerList(t, contentWidth, state)
	buttons := renderPathPickerButtons(t, contentWidth, state)

	title := t.windowTitle.Width(width).Render(state.title())
	body := t.windowBody.
		Width(width).
		Padding(1, 2).
		Render(strings.Join([]string{pathRow, "", list, "", buttons}, "\n"))

	return lipgloss.JoinVertical(lipgloss.Left, title, body)
}

func renderPathPickerList(t theme, width int, state pathPicker) string {
	width = max(1, width)
	height := max(1, state.height)
	lines := make([]string, height)

	for index := range lines {
		lines[index] = t.configPickerArea.Width(width).Render("")
	}

	if state.loadError != "" {
		lines[0] = renderPathPickerMessage(t, width, state.loadError)
		return strings.Join(lines, "\n")
	}

	if len(state.entries) == 0 {
		message := "No directories found"
		if state.isFilePicker() {
			message = "No files found"
		}
		lines[0] = renderPathPickerMessage(t, width, message)
		return strings.Join(lines, "\n")
	}

	end := min(len(state.entries), state.offset+height)

	for entryIndex := state.offset; entryIndex < end; entryIndex++ {
		row := entryIndex - state.offset
		entry := state.entries[entryIndex]
		selected := entryIndex == state.selected
		lines[row] = renderPathPickerEntry(
			t,
			width,
			entry,
			entryIndex,
			len(state.entries),
			selected,
			state,
		)
	}

	return strings.Join(lines, "\n")
}

func renderPathPickerMessage(t theme, width int, message string) string {
	runes := []rune(message)

	message = string(runes[:visiblePrefixEnd(runes, width)])
	padding := max(0, width-lipgloss.Width(message))

	return t.configPickerFile.Render(message) + t.configPickerArea.Width(padding).Render("")
}

func renderPathPickerEntry(
	t theme,
	width int,
	entry pathPickerEntry,
	entryIndex int,
	entryCount int,
	selected bool,
	state pathPicker,
) string {
	nameStyle := t.configPickerFile

	if entry.directory && state.isFilePicker() {
		nameStyle = t.configPickerDirectory
	}

	if selected {
		nameStyle = t.configPickerSelectedFile
		if entry.directory && state.isFilePicker() {
			nameStyle = t.configPickerSelectedDirectory
		}
	}

	branch := "├─"
	if entryIndex == entryCount-1 {
		branch = "└─"
	}

	branchPart := t.configPickerFile.Render(branch)
	nameWidth := max(0, width-lipgloss.Width(branchPart)-2)
	nameRunes := []rune(entry.name)
	name := string(nameRunes[:visiblePrefixEnd(nameRunes, nameWidth)])
	namePart := nameStyle.Render(" " + name + " ")
	padding := max(0, width-lipgloss.Width(branchPart)-lipgloss.Width(namePart))

	return branchPart + namePart + t.configPickerArea.Width(padding).Render("")
}

func configBrowseTarget(focus configFocus) (pathPickerTarget, bool) {
	for target, mode := range pathPickerModes {
		if mode.browseFocus != 0 && focus == mode.browseFocus {
			return target, true
		}
	}

	return 0, false
}

func configTargetFieldFocus(target pathPickerTarget) configFocus {
	return target.mode().configFocus
}

func configTargetFieldIndex(target pathPickerTarget) int {
	return target.mode().configIndex
}

func selectedPathValue(rootDirectory, path string) string {
	rootDirectory = filepath.Clean(rootDirectory)
	path = filepath.Clean(path)

	if pathWithinRoot(rootDirectory, path) {
		if relative, err := filepath.Rel(rootDirectory, path); err == nil {
			path = relative
		}
	}

	return path
}
