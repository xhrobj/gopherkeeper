package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

const pathPickerButtonGap = 3

type pathPickerWindowLayout struct {
	content      string
	contentWidth int
	listBounds   layoutBounds
	buttonBounds []layoutBounds
}

func pathPickerListHeight(screenHeight int) int {
	return clamp(screenHeight-12, 6, 14)
}

func pathPickerWindowWidth(screenWidth int) int {
	return clamp(screenWidth-16, 54, 82)
}

func newPathPickerWindowLayout(t theme, width int, state pathPicker) pathPickerWindowLayout {
	const (
		horizontalPadding = 2
		verticalPadding   = 1
		listRow           = 2
	)

	contentWidth := max(1, width-2*horizontalPadding)
	selectedPath := renderConfigReadOnlyInput(
		t,
		t.readOnly,
		state.selectedValue(),
		max(1, contentWidth-lipgloss.Width("Path  ")),
		true,
	)
	pathRow := t.label.Render("Path") + t.windowBody.Width(2).Render("") + selectedPath
	list := renderPathPickerList(t, contentWidth, state)
	listHeight := max(1, state.height)
	buttonRow := listRow + listHeight + 1
	buttonLayout := pathPickerButtonsLayout(t, contentWidth, state).positioned(0, buttonRow)

	title := t.windowTitle.Width(width).Render(state.title())
	body := t.windowBody.
		Width(width).
		Padding(verticalPadding, horizontalPadding).
		Render(strings.Join([]string{pathRow, "", list, "", buttonLayout.content}, "\n"))
	contentOffsetY := lipgloss.Height(title) + verticalPadding

	return pathPickerWindowLayout{
		content:      lipgloss.JoinVertical(lipgloss.Left, title, body),
		contentWidth: contentWidth,
		listBounds: layoutBounds{
			x:      horizontalPadding,
			y:      contentOffsetY + listRow,
			width:  contentWidth,
			height: listHeight,
		},
		buttonBounds: translateLayoutBounds(
			buttonLayout.bounds,
			horizontalPadding,
			contentOffsetY,
		),
	}
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
	return newPathPickerWindowLayout(t, width, state).content
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
