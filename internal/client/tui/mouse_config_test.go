package tui

import (
	"path/filepath"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/buildinfo"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
)

func TestModel_MouseClickEditsAndCancelsConfig(t *testing.T) {
	initial := config.Config{Address: "localhost:8080"}
	m := newTestModel(t, initial, buildinfo.Info{})
	m.width = 100
	m.height = 32
	m.dialog = dialogConfig
	m.configForm = newConfigForm(m.config)

	fields := m.configFieldBounds()
	if len(fields) != 5 {
		t.Fatalf("field count = %d, want 5", len(fields))
	}
	updated, _ := m.Update(mouseClick(fields[3].x+1, fields[3].y))
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

func TestModel_MouseClickSelectsConfigTransport(t *testing.T) {
	m := newTestModel(t, config.Config{
		Transport:   config.TransportHTTPS,
		Address:     "localhost:8888",
		GRPCAddress: "localhost:9090",
	}, buildinfo.Info{})
	m.width = 100
	m.height = 32
	m.dialog = dialogConfig
	m.configForm = newConfigForm(m.config)

	bounds := m.configTransportBounds()
	if len(bounds) != 2 {
		t.Fatalf("transport bounds = %d, want 2", len(bounds))
	}

	updated, _ := m.Update(mouseClick(bounds[1].x+1, bounds[1].y))
	got := updated.(model)
	if got.configForm.transport != config.TransportGRPC || got.configForm.focus != configTransport {
		t.Fatalf("transport = %q focus = %d, want gRPC transport focus", got.configForm.transport, got.configForm.focus)
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
