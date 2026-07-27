package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
)

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
