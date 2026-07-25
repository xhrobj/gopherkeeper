package tui

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/xhrobj/gopherkeeper/internal/buildinfo"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
	recordmodel "github.com/xhrobj/gopherkeeper/internal/model"
)

func TestModel_ViewEnablesMouseCellMotion(t *testing.T) {
	view := newTestModel(t, config.Config{}, buildinfo.Info{}).View()
	if view.MouseMode != tea.MouseModeCellMotion {
		t.Fatalf("MouseMode = %d, want MouseModeCellMotion", view.MouseMode)
	}
}

func TestModel_MouseClickOpensMenuAndActivatesItem(t *testing.T) {
	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.width = 80
	m.height = 25

	updated, _ := m.Update(mouseClick(menuOffset(int(menuAccount))+1, menuBarY))
	m = updated.(model)
	if !m.dropdownOpen || m.activeMenu != int(menuAccount) {
		t.Fatalf("Account dropdown state = open:%t menu:%d", m.dropdownOpen, m.activeMenu)
	}

	dropdownX := buildDropdownMenuLayout(m.width, m.activeMenu, testMenuDefinitions(m.dialog)[m.activeMenu]).bounds.x
	updated, _ = m.Update(mouseClick(dropdownX+2, dropdownY+1))
	m = updated.(model)
	if m.dialog != dialogLogin {
		t.Fatalf("dialog = %d, want login", m.dialog)
	}
	if m.dropdownOpen || m.menuFocused {
		t.Fatal("menu remains open after mouse activation")
	}
}

func TestModel_MouseClickIgnoresDisabledMenu(t *testing.T) {
	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.width = 80
	m.height = 25

	updated, _ := m.Update(mouseClick(menuOffset(int(menuRecord))+1, menuBarY))
	got := updated.(model)
	if got.dropdownOpen || got.menuFocused {
		t.Fatal("disabled Records menu was opened by mouse")
	}
}

func TestModel_MouseClickOutsideClosesDropdown(t *testing.T) {
	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.width = 80
	m.height = 25
	m.activeMenu = int(menuSystem)
	m.menuFocused = true
	m.dropdownOpen = true

	updated, _ := m.Update(mouseClick(40, 10))
	got := updated.(model)
	if got.dropdownOpen || got.menuFocused {
		t.Fatal("dropdown remains open after outside click")
	}
}

func TestModel_MouseClickPressesCloseButton(t *testing.T) {
	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.width = 80
	m.height = 25
	buttons := m.dialogButtonBounds()
	if len(buttons) != 2 {
		t.Fatalf("button count = %d, want 2", len(buttons))
	}

	updated, _ := m.Update(mouseClick(buttons[1].x+buttons[1].width/2, buttons[1].y))
	got := updated.(model)
	if got.dialog != dialogNone {
		t.Fatalf("dialog = %d, want none", got.dialog)
	}
}

func TestModel_MouseClickPressesAboutButtons(t *testing.T) {
	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.width = 80
	m.height = 25
	m.dialog = dialogAbout

	var opened string
	m.openURL = func(value string) error {
		opened = value
		return nil
	}

	buttons := m.dialogButtonBounds()
	if len(buttons) != 2 {
		t.Fatalf("button count = %d, want 2", len(buttons))
	}

	updated, cmd := m.Update(mouseClick(buttons[0].x+buttons[0].width/2, buttons[0].y))
	m = updated.(model)
	if cmd == nil {
		t.Fatal("Course click did not return browser command")
	}
	_ = cmd()
	if opened != aboutURL {
		t.Fatalf("opened URL = %q, want %q", opened, aboutURL)
	}
	if m.dialog != dialogAbout {
		t.Fatalf("dialog = %d, want about after Course", m.dialog)
	}

	buttons = m.dialogButtonBounds()
	updated, _ = m.Update(mouseClick(buttons[1].x+buttons[1].width/2, buttons[1].y))
	got := updated.(model)
	if got.dialog != dialogNone {
		t.Fatalf("dialog = %d, want none after OK", got.dialog)
	}
}

func TestModel_AlertButtonBoundsMatchRenderedButtonRow(t *testing.T) {
	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.width = 80
	m.height = 25
	m.showAlertWithHighlight(
		alertNotice,
		"Registration successful",
		"Registered as alice. Please log in.",
		"alice",
		dialogNone,
	)

	overlay := m.renderAlert()
	lines := strings.Split(ansi.Strip(overlay), "\n")
	buttonRow := -1
	for index, line := range lines {
		if strings.Contains(line, "< OK >") {
			buttonRow = index
			break
		}
	}
	if buttonRow < 0 {
		t.Fatal("rendered alert does not contain an OK button")
	}

	windowY := max(2, (m.height-lipgloss.Height(overlay))/2)
	buttons := m.dialogButtonBounds()
	if len(buttons) != 1 {
		t.Fatalf("alert button count = %d, want 1", len(buttons))
	}
	if buttons[0].y != windowY+buttonRow {
		t.Fatalf(
			"alert click row = %d, rendered button row = %d",
			buttons[0].y,
			windowY+buttonRow,
		)
	}

	updated, _ := m.Update(mouseClick(buttons[0].x+buttons[0].width/2, buttons[0].y-1))
	m = updated.(model)
	if m.alert == alertNone {
		t.Fatal("click one row above the alert button dismissed the alert")
	}

	updated, _ = m.Update(mouseClick(buttons[0].x+buttons[0].width/2, buttons[0].y))
	got := updated.(model)
	if got.alert != alertNone {
		t.Fatal("click on the rendered alert button did not dismiss the alert")
	}
}

func mouseClick(x, y int) tea.MouseClickMsg {
	return tea.MouseClickMsg(tea.Mouse{X: x, Y: y, Button: tea.MouseLeft})
}

func TestModel_MouseClickEditsAndCancelsConfig(t *testing.T) {
	initial := config.Config{Address: "localhost:8080"}
	m := newTestModel(t, initial, buildinfo.Info{})
	m.width = 100
	m.height = 32
	m.dialog = dialogConfig
	m.configForm = newConfigForm(m.config)

	fields := m.configFieldBounds()
	if len(fields) != 4 {
		t.Fatalf("field count = %d, want 4", len(fields))
	}
	updated, _ := m.Update(mouseClick(fields[2].x+1, fields[2].y))
	m = updated.(model)
	if m.configForm.focus != configSessionDir {
		t.Fatalf("focus = %d, want session dir", m.configForm.focus)
	}

	buttons := m.dialogButtonBounds()
	if len(buttons) != 2 {
		t.Fatalf("button count = %d, want 2", len(buttons))
	}
	updated, _ = m.Update(mouseClick(buttons[1].x+buttons[1].width/2, buttons[1].y))
	got := updated.(model)
	if got.dialog != dialogNone {
		t.Fatalf("dialog = %d, want none after Cancel", got.dialog)
	}
	if got.config != initial {
		t.Fatalf("config = %#v, want %#v", got.config, initial)
	}
}

func TestModel_MouseClickServerStatusCheckStartsRequest(t *testing.T) {
	m := newTestModel(t, config.Config{Address: "localhost:8888"}, buildinfo.Info{})
	m.width = 100
	m.height = 32
	m.dialog = dialogServerStatus
	m.statusState = serverStatusReady
	m.statusValue = "ok"
	m.activeButton = 1
	m.backend = backendStub{health: func(context.Context) (string, error) {
		return "ok", nil
	}}

	buttons := m.dialogButtonBounds()
	if len(buttons) != 2 {
		t.Fatalf("button count = %d, want 2", len(buttons))
	}

	button := buttons[0]
	updated, command := m.Update(mouseClick(
		button.x+button.width/2,
		button.y,
	))
	got := updated.(model)
	if command == nil || got.statusState != serverStatusReady || !got.operations.request(operationServerStatus).pending {
		t.Fatalf(
			"Check click state = command %t state %d pending %t",
			command != nil,
			got.statusState,
			got.operations.request(operationServerStatus).pending,
		)
	}
	if got.activeButton != 0 {
		t.Fatalf("active button = %d, want Check (0)", got.activeButton)
	}
}

func TestModel_MouseClickServerStatusOKRespectsNetworkBlock(t *testing.T) {
	tests := []struct {
		name       string
		state      serverStatusState
		pending    bool
		wantDialog dialogID
	}{
		{name: "checking", state: serverStatusReady, pending: true, wantDialog: dialogServerStatus},
		{name: "success", state: serverStatusReady, wantDialog: dialogNone},
		{name: "error", state: serverStatusFailed, wantDialog: dialogNone},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			m := newTestModel(t, config.Config{Address: "localhost:8888"}, buildinfo.Info{})
			m.width = 100
			m.height = 32
			m.dialog = dialogServerStatus
			m.statusState = test.state
			m.statusValue = "ok"
			m.statusFailure = serverStatusFailure{status: "Unreachable", reason: "Connection refused"}
			m.operations.request(operationServerStatus).pending = test.pending

			buttons := m.dialogButtonBounds()
			if len(buttons) != 2 {
				t.Fatalf("button count = %d, want 2", len(buttons))
			}

			button := buttons[1]
			updated, _ := m.Update(mouseClick(
				button.x+button.width/2,
				button.y,
			))
			got := updated.(model)
			if got.dialog != test.wantDialog {
				t.Fatalf("dialog = %d, want %d", got.dialog, test.wantDialog)
			}
		})
	}
}

func TestModel_ControlsButtonBoundsMatchRenderedButtonRow(t *testing.T) {
	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.width = 100
	m.height = 36
	m.dialog = dialogControls

	window := m.renderDialog()
	windowHeight := lipgloss.Height(window)
	windowY := max(2, (m.height-windowHeight)/2)
	buttonRow := lineIndexContaining(strings.Split(ansi.Strip(window), "\n"), "< OK >")
	if buttonRow < 0 {
		t.Fatal("rendered Controls button was not found")
	}
	if buttonRow != controlsButtonRow {
		t.Fatalf("Controls button row = %d, want constant %d", buttonRow, controlsButtonRow)
	}

	buttons := m.dialogButtonBounds()
	if len(buttons) != 1 {
		t.Fatalf("button count = %d, want 1", len(buttons))
	}
	if buttons[0].y != windowY+buttonRow {
		t.Fatalf("button y = %d, want rendered row %d", buttons[0].y, windowY+buttonRow)
	}
}

func TestModel_ConfigButtonBoundsMatchRenderedButtonRow(t *testing.T) {
	m := newTestModel(t, config.Config{Address: "localhost:8080"}, buildinfo.Info{})
	m.width = 100
	m.height = 32
	m.dialog = dialogConfig
	m.configForm = newConfigForm(m.config)

	window := m.renderDialog()
	windowHeight := lipgloss.Height(window)
	windowY := max(2, (m.height-windowHeight)/2)
	buttonRow := lineIndexContaining(strings.Split(ansi.Strip(window), "\n"), "< Save >")
	if buttonRow < 0 {
		t.Fatal("rendered Config buttons were not found")
	}

	buttons := m.dialogButtonBounds()
	if len(buttons) != 2 {
		t.Fatalf("button count = %d, want 2", len(buttons))
	}
	wantY := windowY + buttonRow
	for index, bounds := range buttons {
		if bounds.y != wantY {
			t.Fatalf("button %d y = %d, want rendered row %d", index, bounds.y, wantY)
		}
	}
}

func TestModel_RecordViewButtonBoundsIgnoreButtonLabelsInPayload(t *testing.T) {
	tests := []struct {
		name   string
		record recordmodel.Record
		label  string
	}{
		{
			name: "close label in text payload",
			record: recordmodel.Record{
				Metadata: recordmodel.RecordMetadata{Type: recordmodel.RecordTypeText, Title: "< Close >"},
				Payload:  &recordmodel.TextPayload{Text: "Text contains < Close > before the real button"},
			},
			label: "< Close >",
		},
		{
			name: "save label in binary notes",
			record: recordmodel.Record{
				Metadata: recordmodel.RecordMetadata{Type: recordmodel.RecordTypeBinary, Title: "Binary"},
				Payload: &recordmodel.BinaryPayload{
					Filename: "sample.bin",
					Metadata: "Notes contain < Save As... > before the real button",
				},
			},
			label: "< Save As... >",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			m := newTestModel(t, config.Config{}, buildinfo.Info{})
			m.width = 100
			m.height = 36
			m.dialog = dialogRecordView
			m.recordFeature.view = recordViewState{status: recordViewReady, record: test.record}

			window := m.renderDialog()
			lines := strings.Split(ansi.Strip(window), "\n")
			buttonRow := -1
			for index := len(lines) - 1; index >= 0; index-- {
				if strings.Contains(lines[index], test.label) {
					buttonRow = index
					break
				}
			}
			if buttonRow < 0 {
				t.Fatalf("rendered button %q was not found", test.label)
			}

			windowY := max(2, (m.height-lipgloss.Height(window))/2)
			buttons := m.dialogButtonBounds()
			if len(buttons) == 0 {
				t.Fatal("record view button bounds are empty")
			}
			if buttons[0].y != windowY+buttonRow {
				t.Fatalf("button y = %d, want rendered row %d", buttons[0].y, windowY+buttonRow)
			}
		})
	}
}

func TestModel_MouseClickIgnoresDisabledConfigSave(t *testing.T) {
	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.width = 100
	m.height = 32
	m.dialog = dialogConfig
	m.configForm = newConfigForm(m.config)
	m.configForm.focus = configCancel

	buttons := m.dialogButtonBounds()
	if len(buttons) != 2 {
		t.Fatalf("button count = %d, want 2", len(buttons))
	}

	updated, _ := m.Update(mouseClick(buttons[0].x+buttons[0].width/2, buttons[0].y))
	got := updated.(model)
	if got.dialog != dialogConfig {
		t.Fatalf("dialog = %d, want Config after disabled Save click", got.dialog)
	}
	if got.configForm.focus != configCancel {
		t.Fatalf("focus = %d, want unchanged Cancel", got.configForm.focus)
	}
	if got.configForm.errorMessage != "" {
		t.Fatalf("error message = %q, want empty", got.configForm.errorMessage)
	}
}

func TestModel_MouseClickOpensConfigPathPicker(t *testing.T) {
	m := newTestModel(t, config.Config{Address: "localhost:8080", CACertFile: "/tmp/ca.pem"}, buildinfo.Info{})
	m.width = 100
	m.height = 32
	m.dialog = dialogConfig
	m.configForm = newConfigForm(m.config)

	buttons := m.configBrowseButtonBounds()
	if len(buttons) != 3 {
		t.Fatalf("browse button count = %d, want 3", len(buttons))
	}

	updated, command := m.Update(mouseClick(buttons[2].x+buttons[2].width/2, buttons[2].y))
	got := updated.(model)
	if got.dialog != dialogPathPicker {
		t.Fatalf("dialog = %d, want config path picker", got.dialog)
	}
	if got.pathPicker.target != pathPickerCacheDirectory {
		t.Fatalf("picker target = %d, want cache directory", got.pathPicker.target)
	}
	if command == nil {
		t.Fatal("picker init command = nil")
	}
}

func TestModel_ConfigPathPickerMouseSelectButtonAppliesHighlightedPath(t *testing.T) {
	rootDirectory := t.TempDir()
	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.width = 100
	m.height = 32
	m.dialog = dialogPathPicker
	m.pathPicker = pathPicker{
		target:           pathPickerCACert,
		rootDirectory:    rootDirectory,
		currentDirectory: filepath.Join(rootDirectory, "certs"),
		entries:          []pathPickerEntry{{name: "ca.pem"}},
		height:           pathPickerListHeight(m.height),
		focus:            pathPickerTree,
	}

	buttons := m.pathPickerButtonBounds()
	if len(buttons) != 2 {
		t.Fatalf("picker button count = %d, want 2", len(buttons))
	}

	updated, command := m.Update(mouseClick(buttons[0].x+buttons[0].width/2, buttons[0].y))
	got := updated.(model)
	if command != nil {
		t.Fatal("Select button returned an unexpected command")
	}
	if got.dialog != dialogConfig {
		t.Fatalf("dialog = %d, want Config", got.dialog)
	}
	want := filepath.Join("certs", "ca.pem")
	if got.configForm.fields[configCACertFile].value != want {
		t.Fatalf("selected certificate = %q, want %q", got.configForm.fields[configCACertFile].value, want)
	}
}

func TestModel_ConfigPathPickerMouseSelectButtonIgnoresDisabledSelection(t *testing.T) {
	rootDirectory := t.TempDir()
	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.width = 100
	m.height = 32
	m.dialog = dialogPathPicker
	m.pathPicker = pathPicker{
		target:           pathPickerCACert,
		rootDirectory:    rootDirectory,
		currentDirectory: rootDirectory,
		entries:          []pathPickerEntry{{name: "certs", directory: true}},
		height:           pathPickerListHeight(m.height),
		focus:            pathPickerTree,
	}

	button := m.pathPickerButtonBounds()[0]
	updated, command := m.Update(mouseClick(button.x+button.width/2, button.y))
	got := updated.(model)
	if command != nil {
		t.Fatal("disabled Select returned an unexpected command")
	}
	if got.dialog != dialogPathPicker {
		t.Fatalf("dialog = %d, want picker to stay open", got.dialog)
	}
	if got.configForm.fields[configCACertFile].value != "" {
		t.Fatalf("disabled Select changed certificate to %q", got.configForm.fields[configCACertFile].value)
	}
}

func TestModel_ConfigPathPickerMouseCancelButtonReturnsToConfig(t *testing.T) {
	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.width = 100
	m.height = 32
	m.dialog = dialogPathPicker
	m.pathPicker = pathPicker{
		target:  pathPickerCacheDirectory,
		entries: []pathPickerEntry{{name: "cache", directory: true}},
		height:  pathPickerListHeight(m.height),
	}

	button := m.pathPickerButtonBounds()[1]
	updated, command := m.Update(mouseClick(button.x+button.width/2, button.y))
	got := updated.(model)
	if command != nil {
		t.Fatal("Cancel button returned an unexpected command")
	}
	if got.dialog != dialogConfig {
		t.Fatalf("dialog = %d, want Config", got.dialog)
	}
}

func TestModel_ConfigPathPickerMouseDoubleClickSelectsFile(t *testing.T) {
	rootDirectory := t.TempDir()
	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.width = 100
	m.height = 32
	m.dialog = dialogPathPicker
	m.pathPicker = pathPicker{
		target:           pathPickerCACert,
		rootDirectory:    rootDirectory,
		currentDirectory: filepath.Join(rootDirectory, "certs"),
		entries:          []pathPickerEntry{{name: "ca.pem"}},
		height:           pathPickerListHeight(m.height),
		focus:            pathPickerTree,
	}

	window, ok := m.dialogPlacement()
	if !ok {
		t.Fatal("picker placement is unavailable")
	}
	x := window.x + 3
	y := window.y + 4

	updated, firstCommand := m.Update(mouseClick(x, y))
	m = updated.(model)
	if firstCommand == nil {
		t.Fatal("first click did not schedule single-click handling")
	}
	updated, secondCommand := m.Update(mouseClick(x, y))
	got := updated.(model)
	if secondCommand != nil {
		t.Fatal("double click returned an unexpected command")
	}
	if got.dialog != dialogConfig {
		t.Fatalf("dialog = %d, want Config after double click", got.dialog)
	}
	want := filepath.Join("certs", "ca.pem")
	if got.configForm.fields[configCACertFile].value != want {
		t.Fatalf("double-click selection = %q, want %q", got.configForm.fields[configCACertFile].value, want)
	}
}
func TestModel_ConfigPathPickerMouseDoubleClickOnParentSelectsCurrentDirectory(t *testing.T) {
	rootDirectory := canonicalPath(t.TempDir())
	childDirectory := filepath.Join(rootDirectory, "cache")
	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.width = 100
	m.height = 32
	m.dialog = dialogPathPicker
	m.pathPicker = pathPicker{
		target:           pathPickerCacheDirectory,
		rootDirectory:    rootDirectory,
		currentDirectory: childDirectory,
		entries:          []pathPickerEntry{{name: "..", directory: true, parent: true}},
		height:           pathPickerListHeight(m.height),
		focus:            pathPickerTree,
	}

	window, ok := m.dialogPlacement()
	if !ok {
		t.Fatal("picker placement is unavailable")
	}
	x := window.x + 3
	y := window.y + 4

	updated, firstCommand := m.Update(mouseClick(x, y))
	m = updated.(model)
	if firstCommand == nil {
		t.Fatal("first click did not schedule single-click handling")
	}
	updated, secondCommand := m.Update(mouseClick(x, y))
	got := updated.(model)
	if secondCommand != nil {
		t.Fatal("double click returned an unexpected command")
	}
	if got.dialog != dialogConfig {
		t.Fatalf("dialog = %d, want Config after double click", got.dialog)
	}
	want := filepath.Base(childDirectory)
	if got.configForm.fields[configCacheDir].value != want {
		t.Fatalf("double-click current-directory selection = %q, want %q", got.configForm.fields[configCacheDir].value, want)
	}
}
