package tui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/xhrobj/gopherkeeper/internal/buildinfo"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
	recordmodel "github.com/xhrobj/gopherkeeper/internal/model"
)

func TestModel_AuthWindowBoundsMatchRenderedControls(t *testing.T) {
	tests := []struct {
		name        string
		dialog      dialogID
		buttonLabel string
		fieldCount  int
		setup       func(*model)
		fields      func(model) []layoutBounds
	}{
		{
			name:        "login",
			dialog:      dialogLogin,
			buttonLabel: "< Login >",
			fieldCount:  2,
			setup: func(m *model) {
				m.authentication.loginForm = newLoginForm()
				m.authentication.loginForm.login.setValue("alice")
				m.authentication.loginForm.password.setValue("secret")
			},
			fields: model.loginFieldBounds,
		},
		{
			name:        "register",
			dialog:      dialogRegister,
			buttonLabel: "< Register >",
			fieldCount:  3,
			setup: func(m *model) {
				m.authentication.registerForm = newRegisterForm()
				m.authentication.registerForm.login.setValue("alice")
				m.authentication.registerForm.password.setValue("secret")
				m.authentication.registerForm.repeatPassword.setValue("secret")
			},
			fields: model.registerFieldBounds,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertAuthWindowBoundsMatchRenderedControls(
				t,
				test.dialog,
				test.buttonLabel,
				test.fieldCount,
				test.setup,
				test.fields,
			)
		})
	}
}

func TestModel_AboutButtonBoundsMatchRenderedButtonRow(t *testing.T) {
	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.width = 80
	m.height = 25
	m.dialog = dialogAbout

	window := m.renderDialog()
	windowHeight := lipgloss.Height(window)
	windowY := max(2, (m.height-windowHeight)/2)
	buttonRow := lineIndexContaining(strings.Split(ansi.Strip(window), "\n"), "< Course >")
	if buttonRow < 0 {
		t.Fatal("rendered About buttons were not found")
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

func TestModel_CurrentUserButtonBoundsMatchRenderedButtonRow(t *testing.T) {
	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.width = 80
	m.height = 25
	m.dialog = dialogCurrentUser
	m.authentication.session = authSession{state: authAuthenticated, login: "alice"}

	window := m.renderDialog()
	lines := strings.Split(ansi.Strip(window), "\n")
	buttonRow := lineIndexContaining(lines, okButtonLabel)
	if buttonRow < 0 {
		t.Fatal("rendered Current User window does not contain an OK button")
	}

	windowY := max(2, (m.height-lipgloss.Height(window))/2)
	buttons := m.dialogButtonBounds()
	if len(buttons) != 1 {
		t.Fatalf("Current User button count = %d, want 1", len(buttons))
	}
	if buttons[0].y != windowY+buttonRow {
		t.Fatalf(
			"Current User click row = %d, rendered button row = %d",
			buttons[0].y,
			windowY+buttonRow,
		)
	}

	updated, _ := m.Update(mouseClick(buttons[0].x+buttons[0].width/2, buttons[0].y))
	if got := updated.(model); got.dialog != dialogNone {
		t.Fatalf("dialog = %d, want none after Current User OK", got.dialog)
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
	buttons := m.dialogButtonBounds()
	if len(buttons) != 1 {
		t.Fatalf("button count = %d, want 1", len(buttons))
	}
	if buttons[0].y != windowY+buttonRow {
		t.Fatalf("button y = %d, want rendered row %d", buttons[0].y, windowY+buttonRow)
	}
}

func TestModel_ServerStatusButtonBoundsMatchRenderedButtonRow(t *testing.T) {
	m := newTestModel(t, config.Config{Address: "localhost:8888"}, buildinfo.Info{})
	m.width = 100
	m.height = 32
	m.dialog = dialogServerStatus
	m.statusState = serverStatusReady
	m.statusValue = "ok"

	window := m.renderDialog()
	windowHeight := lipgloss.Height(window)
	windowY := max(2, (m.height-windowHeight)/2)
	buttonRow := lineIndexContaining(strings.Split(ansi.Strip(window), "\n"), "< Check >")
	if buttonRow < 0 {
		t.Fatal("rendered Server Status buttons were not found")
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

func TestModel_CacheBrowseButtonBoundsMatchRenderedButtonRow(t *testing.T) {
	m := newCacheTestModel(t, backendStub{}, false)
	m.width = 100
	m.height = 32
	m.dialog = dialogCacheBrowse
	m.cacheFeature.form = newCacheBrowseForm("alice")
	m.cacheFeature.form.password.setValue("correct-horse-battery-staple")
	m.cacheFeature.form.focus = cacheBrowseCancel

	window := m.renderDialog()
	buttonRow := lineIndexContaining(strings.Split(ansi.Strip(window), "\n"), "< Browse >")
	if buttonRow < 0 {
		t.Fatal("rendered Open Local Cache buttons were not found")
	}

	buttons := m.dialogButtonBounds()
	if len(buttons) != 2 {
		t.Fatalf("button count = %d, want 2", len(buttons))
	}

	wantY := max(2, (m.height-lipgloss.Height(window))/2) + buttonRow
	for index, bounds := range buttons {
		if bounds.y != wantY {
			t.Fatalf("button %d y = %d, want rendered row %d", index, bounds.y, wantY)
		}
	}

	updated, _ := m.Update(mouseClick(buttons[1].x+buttons[1].width/2, buttons[1].y))
	got := updated.(model)
	if got.dialog != dialogNone {
		t.Fatalf("dialog = %d, want none after Cancel", got.dialog)
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
			assertRecordViewButtonBounds(t, test.record, test.label)
		})
	}
}

func TestModel_SyncButtonBoundsMatchRenderedButtonRow(t *testing.T) {
	m := newSyncTestModel(t, backendStub{})
	m.width = 100
	m.height = 32
	m.dialog = dialogSync
	m.syncFeature.form = newSyncForm()
	m.syncFeature.form.password.setValue(syncFormTestPassword)
	m.syncFeature.form.focus = syncSubmit

	window := m.renderDialog()
	buttonRow := lineIndexContaining(strings.Split(ansi.Strip(window), "\n"), "< Sync >")
	if buttonRow < 0 {
		t.Fatal("rendered Sync buttons were not found")
	}

	buttons := m.dialogButtonBounds()
	if len(buttons) != 2 {
		t.Fatalf("button count = %d, want 2", len(buttons))
	}

	wantY := max(2, (m.height-lipgloss.Height(window))/2) + buttonRow
	for index, bounds := range buttons {
		if bounds.y != wantY {
			t.Fatalf("button %d y = %d, want rendered row %d", index, bounds.y, wantY)
		}
	}
}

func TestModel_SyncResultButtonBoundsMatchRenderedButtonRow(t *testing.T) {
	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.width = 80
	m.height = 25
	m.dialog = dialogSyncResult
	m.syncFeature.result = SyncSummary{Added: 1, Updated: 2, Removed: 3, Unchanged: 4}

	window := m.renderDialog()
	buttonRow := lineIndexContaining(strings.Split(ansi.Strip(window), "\n"), okButtonLabel)
	if buttonRow < 0 {
		t.Fatal("rendered Sync result button was not found")
	}

	buttons := m.dialogButtonBounds()
	if len(buttons) != 1 {
		t.Fatalf("button count = %d, want 1", len(buttons))
	}

	wantY := max(2, (m.height-lipgloss.Height(window))/2) + buttonRow
	if buttons[0].y != wantY {
		t.Fatalf("button y = %d, want rendered row %d", buttons[0].y, wantY)
	}

	updated, _ := m.Update(mouseClick(buttons[0].x+buttons[0].width/2, buttons[0].y))
	got := updated.(model)
	if got.dialog != dialogNone {
		t.Fatalf("dialog = %d, want none after OK", got.dialog)
	}
}

func TestModel_PathPickerButtonBoundsMatchRenderedButtonRow(t *testing.T) {
	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.width = 100
	m.height = 32
	m.dialog = dialogPathPicker
	m.pathPicker = pathPicker{
		target:  pathPickerCACert,
		entries: []pathPickerEntry{{name: "ca.pem"}},
		height:  pathPickerListHeight(m.height),
		focus:   pathPickerSelect,
	}

	window := m.renderDialog()
	buttonRow := lineIndexContaining(strings.Split(ansi.Strip(window), "\n"), "< Select >")
	if buttonRow < 0 {
		t.Fatal("rendered Path Picker buttons were not found")
	}

	buttons := m.pathPickerButtonBounds()
	if len(buttons) != 2 {
		t.Fatalf("button count = %d, want 2", len(buttons))
	}

	wantY := max(2, (m.height-lipgloss.Height(window))/2) + buttonRow
	for index, bounds := range buttons {
		if bounds.y != wantY {
			t.Fatalf("button %d y = %d, want rendered row %d", index, bounds.y, wantY)
		}
	}
}
