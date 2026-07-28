package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/xhrobj/gopherkeeper/internal/buildinfo"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
)

func TestModel_ConfigOpensCACertificateFilePicker(t *testing.T) {
	rootDirectory := canonicalPath(t.TempDir())
	changeWorkingDirectory(t, rootDirectory)
	directory := filepath.Join(rootDirectory, "certs")
	if err := os.Mkdir(directory, 0o700); err != nil {
		t.Fatalf("create certificate directory: %v", err)
	}
	certificate := filepath.Join(directory, "ca.pem")
	if err := os.WriteFile(certificate, []byte("certificate"), 0o600); err != nil {
		t.Fatalf("write certificate: %v", err)
	}

	m := newTestModel(t, config.Config{Address: "localhost:8888", CACertFile: certificate}, buildinfo.Info{})
	m.dialog = dialogConfig
	m.configForm = newConfigForm(m.config)
	m.configForm.focus = configCACertBrowse

	updated, command := m.Update(keyPress("enter"))
	m = updated.(model)
	if m.dialog != dialogPathPicker {
		t.Fatalf("dialog = %d, want config path picker", m.dialog)
	}
	if command == nil {
		t.Fatal("file picker init command = nil")
	}
	if m.pathPicker.currentDirectory != directory {
		t.Fatalf("picker directory = %q, want %q", m.pathPicker.currentDirectory, directory)
	}
	if m.pathPicker.rootDirectory != rootDirectory {
		t.Fatalf("picker root = %q, want %q", m.pathPicker.rootDirectory, rootDirectory)
	}

	updated, _ = m.Update(command())
	m = updated.(model)
	entry, ok := m.pathPicker.highlighted()
	if !ok || entry.name != "ca.pem" || entry.directory {
		t.Fatalf("highlighted entry = %+v, %t, want ca.pem file", entry, ok)
	}

	updated, command = m.Update(keyPress("enter"))
	m = updated.(model)
	if command != nil || m.dialog != dialogPathPicker {
		t.Fatal("Enter on a file selected it without using the Select button")
	}

	updated, _ = m.Update(keyPress("tab"))
	m = updated.(model)
	if m.pathPicker.focus != pathPickerSelect {
		t.Fatalf("picker focus = %d, want Select", m.pathPicker.focus)
	}
	updated, _ = m.Update(keyPress("enter"))
	got := updated.(model)
	if got.dialog != dialogConfig {
		t.Fatalf("dialog after file selection = %d, want Config", got.dialog)
	}
	wantCertificate := filepath.Join("certs", "ca.pem")
	if got.configForm.fields[configCACertFile].value != wantCertificate {
		t.Fatalf("selected certificate = %q, want %q", got.configForm.fields[configCACertFile].value, wantCertificate)
	}
	if got.configForm.focus != configCACertFile {
		t.Fatalf("focus after file selection = %d, want CA cert field", got.configForm.focus)
	}
}

func TestModel_ConfigDirectoryPickerSelectsSessionDirectory(t *testing.T) {
	parent := canonicalPath(t.TempDir())
	changeWorkingDirectory(t, parent)
	directory := filepath.Join(parent, "sessions")
	if err := os.Mkdir(directory, 0o700); err != nil {
		t.Fatalf("create session directory: %v", err)
	}

	m := newTestModel(t, config.Config{Address: "localhost:8888", CACertFile: "/tmp/ca.pem"}, buildinfo.Info{})
	m.dialog = dialogConfig
	m.configForm = newConfigForm(m.config)
	m.configForm.fields[configSessionDir].setValue("sessions")
	m.configForm.focus = configSessionBrowse

	updated, command := m.Update(keyPress("enter"))
	m = updated.(model)
	if command == nil {
		t.Fatal("directory picker init command = nil")
	}
	updated, _ = m.Update(command())
	m = updated.(model)
	updated, _ = m.Update(keyPress("tab"))
	m = updated.(model)
	if m.pathPicker.focus != pathPickerSelect {
		t.Fatalf("picker focus = %d, want Select", m.pathPicker.focus)
	}
	updated, _ = m.Update(keyPress("enter"))
	got := updated.(model)

	if got.configForm.fields[configSessionDir].value != "sessions" {
		t.Fatalf("selected session directory = %q, want %q", got.configForm.fields[configSessionDir].value, "sessions")
	}
	if got.configForm.focus != configSessionDir {
		t.Fatalf("focus after directory selection = %d, want Session dir", got.configForm.focus)
	}
}

func TestNewConfigPathPicker_PreselectsConfiguredDirectory(t *testing.T) {
	rootDirectory := canonicalPath(t.TempDir())
	changeWorkingDirectory(t, rootDirectory)
	localDirectory := filepath.Join(rootDirectory, ".local")
	cacheDirectory := filepath.Join(localDirectory, "cache")
	if err := os.MkdirAll(cacheDirectory, 0o700); err != nil {
		t.Fatalf("create cache directory: %v", err)
	}

	picker, command := newPathPicker(
		pathPickerCacheDirectory,
		filepath.Join(".local", "cache"),
		32,
	)
	if picker.currentDirectory != localDirectory {
		t.Fatalf("picker directory = %q, want parent %q", picker.currentDirectory, localDirectory)
	}
	picker.applyReadResult(command().(pathPickerReadMsg))
	entry, ok := picker.highlighted()
	if !ok || entry.name != "cache" || !entry.directory {
		t.Fatalf("highlighted entry = %+v, %t, want configured cache directory", entry, ok)
	}
	want := filepath.Join(".local", "cache")
	if got := picker.selectedValue(); got != want {
		t.Fatalf("selected value = %q, want %q", got, want)
	}
}

func TestConfigPathPicker_UpStopsAtLaunchDirectory(t *testing.T) {
	rootDirectory := canonicalPath(t.TempDir())
	changeWorkingDirectory(t, rootDirectory)
	childDirectory := filepath.Join(rootDirectory, "certs")
	if err := os.Mkdir(childDirectory, 0o700); err != nil {
		t.Fatalf("create child directory: %v", err)
	}

	picker, _ := newPathPicker(
		pathPickerCACert,
		filepath.Join(childDirectory, "ca.pem"),
		32,
	)
	command := picker.up()
	if command == nil {
		t.Fatal("up command below picker root is nil")
	}
	if picker.currentDirectory != rootDirectory {
		t.Fatalf("current directory after up = %q, want root %q", picker.currentDirectory, rootDirectory)
	}

	picker.applyReadResult(command().(pathPickerReadMsg))
	entry, ok := picker.highlighted()
	if !ok || entry.name != "certs" {
		t.Fatalf("highlighted entry after up = %+v, %t, want certs", entry, ok)
	}

	if command = picker.up(); command != nil {
		t.Fatal("up command at picker root is not nil")
	}
	if picker.currentDirectory != rootDirectory {
		t.Fatalf("current directory escaped root: %q", picker.currentDirectory)
	}
}

func TestModel_ConfigPathPickerParentEntryMovesToParent(t *testing.T) {
	rootDirectory := canonicalPath(t.TempDir())
	changeWorkingDirectory(t, rootDirectory)
	childDirectory := filepath.Join(rootDirectory, "certs")
	if err := os.Mkdir(childDirectory, 0o700); err != nil {
		t.Fatalf("create child directory: %v", err)
	}

	picker, readCommand := newPathPicker(
		pathPickerCACert,
		filepath.Join(childDirectory, "ca.pem"),
		32,
	)
	picker.applyReadResult(readCommand().(pathPickerReadMsg))
	for index, entry := range picker.entries {
		if entry.parent {
			picker.selected = index
			break
		}
	}
	entry, ok := picker.highlighted()
	if !ok || !entry.parent || entry.name != ".." {
		t.Fatalf("highlighted entry = %+v, %t, want parent entry", entry, ok)
	}

	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.dialog = dialogPathPicker
	m.pathPicker = picker

	updated, command := m.Update(keyPress("enter"))
	got := updated.(model)
	if command == nil {
		t.Fatal("parent entry did not start loading the parent directory")
	}
	if got.pathPicker.currentDirectory != rootDirectory {
		t.Fatalf("current directory after parent entry = %q, want %q", got.pathPicker.currentDirectory, rootDirectory)
	}
}

func TestModel_ConfigPathPickerLeftRightAndSDoNotNavigate(t *testing.T) {
	rootDirectory := canonicalPath(t.TempDir())
	changeWorkingDirectory(t, rootDirectory)
	childDirectory := filepath.Join(rootDirectory, "certs")
	if err := os.Mkdir(childDirectory, 0o700); err != nil {
		t.Fatalf("create child directory: %v", err)
	}

	for _, key := range []string{"left", "right", "s"} {
		picker, _ := newPathPicker(
			pathPickerCACert,
			filepath.Join(childDirectory, "ca.pem"),
			32,
		)
		m := newTestModel(t, config.Config{}, buildinfo.Info{})
		m.dialog = dialogPathPicker
		m.pathPicker = picker

		updated, command := m.Update(keyPress(key))
		got := updated.(model)
		if command != nil {
			t.Fatalf("%s unexpectedly started navigation", key)
		}
		if got.pathPicker.currentDirectory != childDirectory {
			t.Fatalf("current directory after %s = %q, want %q", key, got.pathPicker.currentDirectory, childDirectory)
		}
	}
}

func TestConfigPathPicker_DirectoryModeHidesFiles(t *testing.T) {
	directory := canonicalPath(t.TempDir())
	changeWorkingDirectory(t, directory)
	childDirectory := filepath.Join(directory, "sessions")
	if err := os.Mkdir(childDirectory, 0o700); err != nil {
		t.Fatalf("create directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(directory, "session.json"), []byte("{}"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	picker, command := newPathPicker(pathPickerCacheDirectory, directory, 32)
	rawMessage := command()
	message, ok := rawMessage.(pathPickerReadMsg)
	if !ok {
		t.Fatalf("read command message type = %T, want pathPickerReadMsg", rawMessage)
	}
	picker.applyReadResult(message)

	if len(picker.entries) != 1 {
		t.Fatalf("directory picker entries = %d, want 1", len(picker.entries))
	}
	if picker.entries[0].name != "sessions" || !picker.entries[0].directory {
		t.Fatalf("directory picker entry = %+v, want sessions directory", picker.entries[0])
	}
}

func TestRenderConfigPathPickerList_UsesWhiteTreeAndPaddedSelection(t *testing.T) {
	theme := newTheme()
	state := pathPicker{
		target: pathPickerCACert,
		entries: []pathPickerEntry{
			{name: "certs", directory: true},
			{name: "ca.pem"},
		},
		selected: 1,
		height:   4,
	}

	view := renderPathPickerList(theme, 30, state)
	plain := ansi.Strip(view)
	if !strings.Contains(plain, "├─ certs ") || !strings.Contains(plain, "└─ ca.pem ") {
		t.Fatalf("file picker does not render compact tree with one horizontal stroke: %q", plain)
	}
	if strings.Contains(plain, ">") {
		t.Fatalf("file picker still renders a selection arrow: %q", plain)
	}
	if !strings.Contains(view, theme.configPickerFile.Render("├─")+theme.configPickerDirectory.Render(" certs ")) {
		t.Fatal("tree branch is not white or directory name is not gray")
	}
	if !strings.Contains(view, theme.configPickerSelectedFile.Render(" ca.pem ")) {
		t.Fatal("selected file does not include inverted spaces on both sides")
	}
	lines := strings.Split(view, "\n")
	wantSelectedLine := theme.configPickerFile.Render("└─") +
		theme.configPickerSelectedFile.Render(" ca.pem ") +
		theme.configPickerArea.Width(30-lipgloss.Width("└─ ca.pem ")).Render("")
	if lines[1] != wantSelectedLine {
		t.Fatalf("selected row = %q, want branch outside padded selection", lines[1])
	}
	if !strings.Contains(view, theme.configPickerArea.Width(30).Render("")) {
		t.Fatal("file picker does not render the blue list background")
	}
}

func TestRenderConfigPathPickerList_RendersDirectoryModeDirectoriesWhite(t *testing.T) {
	theme := newTheme()
	state := pathPicker{
		target:  pathPickerCacheDirectory,
		entries: []pathPickerEntry{{name: "cache", directory: true}},
		height:  2,
	}

	view := renderPathPickerList(theme, 24, state)

	if !strings.Contains(view, theme.configPickerFile.Render("└─")+theme.configPickerSelectedFile.Render(" cache ")) {
		t.Fatal("directory picker does not render a white branch and padded inverted directory")
	}
}

func TestRenderConfigPathPickerButtons_ReflectFocusAndAvailability(t *testing.T) {
	theme := newTheme()
	state := pathPicker{
		target:  pathPickerCACert,
		entries: []pathPickerEntry{{name: "certs", directory: true}},
		focus:   pathPickerTree,
	}

	view := renderPathPickerButtons(theme, 48, state)
	if !strings.Contains(view, theme.buttonDisabled.Render("< Select >")) {
		t.Fatal("Select is not disabled for a directory in the file picker")
	}
	if !strings.Contains(view, theme.button.Render("< Cancel >")) {
		t.Fatal("Cancel button is missing")
	}

	state.entries = []pathPickerEntry{{name: "ca.pem"}}
	state.focus = pathPickerSelect
	view = renderPathPickerButtons(theme, 48, state)
	if !strings.Contains(view, theme.buttonActive.Render("< Select >")) {
		t.Fatal("focused Select button is not active")
	}

	state.focus = pathPickerCancel
	view = renderPathPickerButtons(theme, 48, state)
	if !strings.Contains(view, theme.buttonActive.Render("< Cancel >")) {
		t.Fatal("focused Cancel button is not active")
	}
}

func TestConfigPathPicker_TabSkipsDisabledSelect(t *testing.T) {
	picker := pathPicker{
		target:  pathPickerCACert,
		entries: []pathPickerEntry{{name: "certs", directory: true}},
		focus:   pathPickerTree,
	}

	picker.moveFocus(1)
	if picker.focus != pathPickerCancel {
		t.Fatalf("focus = %d, want Cancel after disabled Select", picker.focus)
	}
	picker.moveFocus(1)
	if picker.focus != pathPickerTree {
		t.Fatalf("focus = %d, want Tree", picker.focus)
	}

	picker.entries = []pathPickerEntry{{name: "ca.pem"}}
	picker.moveFocus(1)
	if picker.focus != pathPickerSelect {
		t.Fatalf("focus = %d, want Select", picker.focus)
	}
}

func TestConfigPathPicker_SelectedValueMatchesConfigFieldValue(t *testing.T) {
	rootDirectory := canonicalPath(t.TempDir())
	state := pathPicker{
		target:           pathPickerSessionDirectory,
		rootDirectory:    rootDirectory,
		currentDirectory: rootDirectory,
		entries:          []pathPickerEntry{{name: "sessions", directory: true}},
	}

	want := "sessions"
	if got := state.selectedValue(); got != want {
		t.Fatalf("selected value = %q, want %q", got, want)
	}

	state.target = pathPickerCACert
	state.entries = []pathPickerEntry{{name: "certs", directory: true}}
	if got := state.selectedValue(); got != "" {
		t.Fatalf("non-selectable selected value = %q, want empty", got)
	}
}

func TestModel_ConfigPathPickerSelectsCurrentDirectoryFromParentEntry(t *testing.T) {
	rootDirectory := canonicalPath(t.TempDir())
	childDirectory := filepath.Join(rootDirectory, "cache")
	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.dialog = dialogPathPicker
	m.pathPicker = pathPicker{
		target:           pathPickerCacheDirectory,
		rootDirectory:    rootDirectory,
		currentDirectory: childDirectory,
		entries:          []pathPickerEntry{{name: "..", directory: true, parent: true}},
		focus:            pathPickerTree,
	}

	updated, _ := m.Update(keyPress("tab"))
	m = updated.(model)
	if m.pathPicker.focus != pathPickerSelect {
		t.Fatalf("focus = %d, want Select", m.pathPicker.focus)
	}
	updated, _ = m.Update(keyPress("enter"))
	got := updated.(model)
	if got.dialog != dialogConfig {
		t.Fatalf("dialog = %d, want Config", got.dialog)
	}
	want := filepath.Base(childDirectory)
	if got.configForm.fields[configCacheDir].value != want {
		t.Fatalf("selected current directory = %q, want %q", got.configForm.fields[configCacheDir].value, want)
	}
}

func TestConfigPathPicker_EnteringDirectoryKeepsSelectedValueOnParentEntry(t *testing.T) {
	rootDirectory := canonicalPath(t.TempDir())
	childDirectory := filepath.Join(rootDirectory, "cache")
	if err := os.Mkdir(childDirectory, 0o700); err != nil {
		t.Fatalf("create child directory: %v", err)
	}

	picker := pathPicker{
		target:           pathPickerCacheDirectory,
		rootDirectory:    rootDirectory,
		currentDirectory: rootDirectory,
		entries:          []pathPickerEntry{{name: "cache", directory: true}},
	}
	before := picker.selectedValue()
	command := picker.openHighlightedDirectory()
	if command == nil {
		t.Fatal("opening selected directory returned nil command")
	}
	picker.applyReadResult(command().(pathPickerReadMsg))

	entry, ok := picker.highlighted()
	if !ok || !entry.parent {
		t.Fatalf("highlighted entry after entering directory = %+v, %t, want ..", entry, ok)
	}
	if after := picker.selectedValue(); after != before {
		t.Fatalf("selected value after entering directory = %q, want unchanged %q", after, before)
	}
}

func TestRenderConfigPathPickerWindow_ShowsValueInsertedBySelect(t *testing.T) {
	rootDirectory := canonicalPath(t.TempDir())
	state := pathPicker{
		target:           pathPickerCACert,
		rootDirectory:    rootDirectory,
		currentDirectory: filepath.Join(rootDirectory, "certs"),
		entries:          []pathPickerEntry{{name: "ca.pem"}},
		height:           1,
	}

	plain := ansi.Strip(renderPathPickerWindow(newTheme(), 76, state))
	wantPath := filepath.Join("certs", "ca.pem")
	if !strings.Contains(plain, "Path  "+wantPath) {
		t.Fatalf("picker does not show selected config value: %q", plain)
	}
	if strings.Contains(plain, rootDirectory) {
		t.Fatalf("picker exposes absolute launch path: %q", plain)
	}
	for _, removed := range []string{"Directory", "Enter Select", "Right Open", "Current"} {
		if strings.Contains(plain, removed) {
			t.Fatalf("picker still renders removed text %q: %q", removed, plain)
		}
	}
}

func TestRenderConfigPathPickerWindow_SeparatesButtonsWithBlankLine(t *testing.T) {
	state := pathPicker{
		target:  pathPickerCACert,
		entries: []pathPickerEntry{{name: "ca.pem"}},
		height:  1,
	}

	plain := ansi.Strip(renderPathPickerWindow(newTheme(), 76, state))
	lines := strings.Split(plain, "\n")
	buttonRow := -1
	for index, line := range lines {
		if strings.Contains(line, "< Select >") {
			buttonRow = index
			break
		}
	}
	if buttonRow <= 0 || strings.TrimSpace(lines[buttonRow-1]) != "" {
		t.Fatalf("buttons are not separated from picker by a blank line: %q", plain)
	}
}

func TestConfigSelectedPath(t *testing.T) {
	rootDirectory := filepath.Join(string(filepath.Separator), "tmp", "project")
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "inside root",
			path: filepath.Join(rootDirectory, ".local", "cache"),
			want: filepath.Join(".local", "cache"),
		},
		{
			name: "root directory",
			path: rootDirectory,
			want: ".",
		},
		{
			name: "outside root",
			path: filepath.Join(filepath.Dir(rootDirectory), "other", "cache"),
			want: filepath.Join(filepath.Dir(rootDirectory), "other", "cache"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := selectedPathValue(rootDirectory, test.path); got != test.want {
				t.Fatalf("selected path = %q, want %q", got, test.want)
			}
		})
	}
}
