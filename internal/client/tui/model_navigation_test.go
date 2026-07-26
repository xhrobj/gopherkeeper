package tui

import (
	"errors"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/buildinfo"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
)

func TestModel_OpenCourseFailureShowsAlert(t *testing.T) {
	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.dialog = dialogAbout
	m.activeButton = 0
	m.openURL = func(string) error { return errors.New("browser unavailable") }

	updated, cmd := m.activateDialogButton()
	m = updated.(model)
	if cmd == nil {
		t.Fatal("Course action command = nil")
	}
	updated, _ = m.Update(cmd())
	m = updated.(model)
	if m.alert != alertError || m.alertReturnDialog != dialogAbout {
		t.Fatalf("alert = %d return dialog = %d", m.alert, m.alertReturnDialog)
	}
	assertViewContains(t, m.View().Content, "Unable to open link", "Browser unavailable")

	updated, _ = m.Update(keyPress("enter"))
	got := updated.(model)
	if got.dialog != dialogAbout || got.alert != alertNone {
		t.Fatalf("state after dismiss = dialog %d alert %d, want About", got.dialog, got.alert)
	}
}
func TestModel_AccountMenuOpensRegisterDialog(t *testing.T) {
	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.activeMenu = int(menuAccount)
	m.menuFocused = true
	m.dropdownOpen = true

	updated, _ := m.Update(keyPress("r"))
	m = updated.(model)

	if m.dialog != dialogRegister {
		t.Fatalf("dialog = %d, want register", m.dialog)
	}
	if m.dropdownOpen || m.menuFocused {
		t.Fatal("menu remains open after action")
	}
}
func TestModel_HelpMenuOpensControlsDialog(t *testing.T) {
	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.activeMenu = int(menuHelp)
	m.menuFocused = true

	updated, _ := m.Update(keyPress("enter"))
	m = updated.(model)
	updated, _ = m.Update(keyPress("enter"))
	m = updated.(model)

	if m.dialog != dialogControls {
		t.Fatalf("dialog = %d, want controls", m.dialog)
	}
	if m.dropdownOpen || m.menuFocused {
		t.Fatal("menu remains open after Controls action")
	}

	assertViewContains(t, m.View().Content,
		"Controls",
		"Menu",
		"F10",
		"Arrow keys",
		"Mouse",
		"Click",
		"Ctrl+Q",
	)
}
func TestModel_HelpItemMnemonicsOpenDialogs(t *testing.T) {
	tests := []struct {
		key        string
		wantDialog dialogID
	}{
		{key: "c", wantDialog: dialogControls},
		{key: "g", wantDialog: dialogAbout},
	}

	for _, test := range tests {
		m := newTestModel(t, config.Config{}, buildinfo.Info{})
		m.menuFocused = true

		updated, _ := m.Update(keyPress("h"))
		m = updated.(model)
		if !m.dropdownOpen || m.activeMenu != int(menuHelp) {
			t.Fatalf("Help menu was not opened before mnemonic %q", test.key)
		}

		updated, _ = m.Update(keyPress(test.key))
		got := updated.(model)
		if got.dialog != test.wantDialog {
			t.Fatalf("mnemonic %q opened dialog %d, want %d", test.key, got.dialog, test.wantDialog)
		}
		if got.menuFocused || got.dropdownOpen {
			t.Fatalf("menu remains open after mnemonic %q", test.key)
		}
	}
}
func TestModel_AboutOKClosesDialog(t *testing.T) {
	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.dialog = dialogAbout
	m.activeButton = 1

	updated, _ := m.Update(keyPress("enter"))
	got := updated.(model)

	if got.dialog != dialogNone {
		t.Fatalf("dialog = %d, want none", got.dialog)
	}
}
func TestModel_AboutLinkOpensCoursePage(t *testing.T) {
	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.dialog = dialogAbout
	m.activeButton = 0

	var opened string
	m.openURL = func(value string) error {
		opened = value
		return nil
	}

	updated, cmd := m.Update(keyPress("enter"))
	got := updated.(model)
	if cmd == nil {
		t.Fatal("Course did not return browser command")
	}
	_ = cmd()

	if opened != aboutURL {
		t.Fatalf("opened URL = %q, want %q", opened, aboutURL)
	}
	if got.dialog != dialogAbout {
		t.Fatalf("dialog = %d, want about", got.dialog)
	}
}
func TestModel_AboutTabSwitchesButtons(t *testing.T) {
	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.dialog = dialogAbout
	m.activeButton = 1

	updated, _ := m.Update(keyPress("tab"))
	got := updated.(model)
	if got.activeButton != 0 {
		t.Fatalf("active button = %d, want Course", got.activeButton)
	}

	updated, _ = got.Update(keyPress("shift+tab"))
	got = updated.(model)
	if got.activeButton != 1 {
		t.Fatalf("active button = %d, want OK", got.activeButton)
	}
}
func TestModel_MenuNavigationIncludesCacheForSecondaryDialog(t *testing.T) {
	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.dialog = dialogAbout
	m.activeMenu = int(menuAccount)
	m.menuFocused = true

	updated, _ := m.Update(keyPress("right"))
	got := updated.(model)

	if got.activeMenu != int(menuCache) {
		t.Fatalf("active menu = %d, want Cache (%d)", got.activeMenu, menuCache)
	}
}
func TestModel_WindowMenuClosesSecondaryDialog(t *testing.T) {
	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.dialog = dialogAbout
	m.activeMenu = int(menuWindow)
	m.menuFocused = true

	updated, _ := m.Update(keyPress("enter"))
	m = updated.(model)
	updated, _ = m.Update(keyPress("enter"))
	m = updated.(model)

	if m.dialog != dialogNone {
		t.Fatalf("dialog = %d, want none", m.dialog)
	}
	if m.dropdownOpen || m.menuFocused {
		t.Fatal("menu remains open after Close Active Window")
	}
}
func TestModel_ClosingSecondaryDialogDoesNotRestorePreviousDialog(t *testing.T) {
	m := newTestModel(t, config.Config{Address: "localhost:8080"}, buildinfo.Info{})
	m.authentication.session = authSession{state: authAuthenticated, login: "alice"}
	m.dialog = dialogCurrentUser

	updated, _ := m.activate(actionConfig)
	m = updated.(model)
	m.configForm.focus = configCancel
	updated, _ = m.Update(keyPress("enter"))
	got := updated.(model)

	if got.dialog != dialogNone {
		t.Fatalf("dialog = %d, want none after closing Config", got.dialog)
	}
}
func TestModel_ClosedLoginDoesNotReappearAfterOtherDialog(t *testing.T) {
	m := newTestModel(t, config.Config{Address: "localhost:8080"}, buildinfo.Info{})
	m.dialog = dialogLogin

	updated, _ := m.Update(keyPress("esc"))
	m = updated.(model)
	updated, _ = m.activate(actionConfig)
	m = updated.(model)
	m.configForm.focus = configCancel
	updated, _ = m.Update(keyPress("enter"))
	got := updated.(model)

	if got.dialog != dialogNone {
		t.Fatalf("dialog = %d, want none", got.dialog)
	}
}
