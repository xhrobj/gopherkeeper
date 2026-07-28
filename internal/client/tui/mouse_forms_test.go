package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/xhrobj/gopherkeeper/internal/buildinfo"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
)

const (
	mouseTestLogin           = "alice"
	mouseTestPassword        = "secret42"
	wantPasswordFocusMessage = "focus = %d, want password"
)

func TestModel_MouseClickLoginFieldsAndButtons(t *testing.T) {
	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.width = 100
	m.height = 32

	fields := m.loginFieldBounds()
	updated, _ := m.updateMouse(clickInside(fields[1]))
	m = updated.(model)
	if m.authentication.loginForm.focus != loginPassword {
		t.Fatalf(wantPasswordFocusMessage, m.authentication.loginForm.focus)
	}

	buttons := m.dialogButtonBounds()
	updated, command := m.updateMouse(clickInside(buttons[0]))
	m = updated.(model)
	if command != nil || m.operations.pending(operationLogin) {
		t.Fatal("disabled login submit started a request")
	}

	m.authentication.loginForm.login.setValue(mouseTestLogin)
	m.authentication.loginForm.password.setValue(mouseTestPassword)
	buttons = m.dialogButtonBounds()
	updated, command = m.updateMouse(clickInside(buttons[0]))
	m = updated.(model)
	if command == nil || !m.operations.pending(operationLogin) {
		t.Fatal("enabled login submit did not start a request")
	}

	m = newTestModel(t, config.Config{}, buildinfo.Info{})
	m.width = 100
	m.height = 32
	buttons = m.dialogButtonBounds()
	updated, command = m.updateMouse(clickInside(buttons[1]))
	got := updated.(model)
	if command != nil || got.dialog != dialogNone {
		t.Fatalf("close state = dialog %d command %t", got.dialog, command != nil)
	}
}

func TestModel_MouseClickRegisterFieldsAndButtons(t *testing.T) {
	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.width = 100
	m.height = 34
	m.dialog = dialogRegister
	m.authentication.registerForm = newRegisterForm()

	fields := m.registerFieldBounds()
	updated, _ := m.updateMouse(clickInside(fields[2]))
	m = updated.(model)
	if m.authentication.registerForm.focus != registerRepeatPassword {
		t.Fatalf("focus = %d, want repeat password", m.authentication.registerForm.focus)
	}

	buttons := m.dialogButtonBounds()
	updated, command := m.updateMouse(clickInside(buttons[0]))
	m = updated.(model)
	if command != nil || m.operations.pending(operationRegister) {
		t.Fatal("disabled register submit started a request")
	}

	m.authentication.registerForm.login.setValue(mouseTestLogin)
	m.authentication.registerForm.password.setValue(mouseTestPassword)
	m.authentication.registerForm.repeatPassword.setValue(mouseTestPassword)
	buttons = m.dialogButtonBounds()
	updated, command = m.updateMouse(clickInside(buttons[0]))
	m = updated.(model)
	if command == nil || !m.operations.pending(operationRegister) {
		t.Fatal("enabled register submit did not start a request")
	}

	m = newTestModel(t, config.Config{}, buildinfo.Info{})
	m.width = 100
	m.height = 34
	m.dialog = dialogRegister
	m.authentication.registerForm = newRegisterForm()
	buttons = m.dialogButtonBounds()
	updated, command = m.updateMouse(clickInside(buttons[1]))
	got := updated.(model)
	if command != nil || got.dialog != dialogNone {
		t.Fatalf("close state = dialog %d command %t", got.dialog, command != nil)
	}
}

func TestModel_MouseClickCacheBrowseFieldsAndButtons(t *testing.T) {
	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.width = 100
	m.height = 34
	m.dialog = dialogCacheBrowse
	m.cacheFeature.form = newCacheBrowseForm("")

	fields := m.cacheBrowseFieldBounds()
	updated, _ := m.updateMouse(clickInside(fields[1]))
	m = updated.(model)
	if m.cacheFeature.form.focus != cacheBrowsePassword {
		t.Fatalf(wantPasswordFocusMessage, m.cacheFeature.form.focus)
	}

	buttons := m.dialogButtonBounds()
	updated, command := m.updateMouse(clickInside(buttons[0]))
	m = updated.(model)
	if command != nil || m.operations.pending(operationOpenCache) {
		t.Fatal("disabled cache submit started a request")
	}

	m.cacheFeature.form.login.setValue(mouseTestLogin)
	m.cacheFeature.form.password.setValue(mouseTestPassword)
	buttons = m.dialogButtonBounds()
	updated, command = m.updateMouse(clickInside(buttons[0]))
	m = updated.(model)
	if command == nil || !m.operations.pending(operationOpenCache) {
		t.Fatal("enabled cache submit did not start a request")
	}

	m = newTestModel(t, config.Config{}, buildinfo.Info{})
	m.width = 100
	m.height = 34
	m.dialog = dialogCacheBrowse
	m.cacheFeature.form = newCacheBrowseForm(mouseTestLogin)
	buttons = m.dialogButtonBounds()
	updated, command = m.updateMouse(clickInside(buttons[1]))
	got := updated.(model)
	if command != nil || got.dialog != dialogNone {
		t.Fatalf("cancel state = dialog %d command %t", got.dialog, command != nil)
	}
}

func TestModel_MouseClickSyncFieldsAndButtons(t *testing.T) {
	m := newSyncTestModel(t, backendStub{})
	m.width = 100
	m.height = 32
	m.dialog = dialogSync
	m.syncFeature.form = newSyncForm()

	fields := m.syncFieldBounds()
	updated, _ := m.updateMouse(clickInside(fields[0]))
	m = updated.(model)
	if m.syncFeature.form.focus != syncPassword {
		t.Fatalf(wantPasswordFocusMessage, m.syncFeature.form.focus)
	}

	buttons := m.dialogButtonBounds()
	updated, command := m.updateMouse(clickInside(buttons[0]))
	m = updated.(model)
	if command != nil || m.operations.pending(operationSync) {
		t.Fatal("disabled sync submit started a request")
	}

	m.syncFeature.form.password.setValue(mouseTestPassword)
	buttons = m.dialogButtonBounds()
	updated, command = m.updateMouse(clickInside(buttons[0]))
	m = updated.(model)
	if command == nil || !m.operations.pending(operationSync) {
		t.Fatal("enabled sync submit did not start a request")
	}

	m = newSyncTestModel(t, backendStub{})
	m.width = 100
	m.height = 32
	m.dialog = dialogSync
	m.syncFeature.form = newSyncForm()
	buttons = m.dialogButtonBounds()
	updated, command = m.updateMouse(clickInside(buttons[1]))
	got := updated.(model)
	if command != nil || got.dialog != dialogNone {
		t.Fatalf("cancel state = dialog %d command %t", got.dialog, command != nil)
	}
}

func TestModel_MouseClickGuardsAndMenuToggle(t *testing.T) {
	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.width = 100
	m.height = 32

	rightClick := tea.MouseClickMsg(tea.Mouse{X: 1, Y: 1, Button: tea.MouseRight})
	updated, command := m.updateMouse(rightClick)
	got := updated.(model)
	if command != nil || got.dialog != dialogLogin {
		t.Fatal("right mouse click changed the model")
	}

	m.width = minimumWidth - 1
	updated, command = m.updateMouse(mouseClick(1, 1))
	got = updated.(model)
	if command != nil || got.width != minimumWidth-1 {
		t.Fatal("small-screen click changed the model")
	}

	m.width = 100
	m.operations.request(operationLogin).pending = true
	updated, command = m.updateMouse(mouseClick(1, 1))
	got = updated.(model)
	if command != nil || !got.operations.pending(operationLogin) {
		t.Fatal("blocked click changed operation state")
	}

	m.operations.request(operationLogin).pending = false
	m.activeMenu = int(menuSystem)
	m.menuFocused = true
	m.dropdownOpen = true
	updated, command = m.updateMouse(mouseClick(menuOffset(int(menuSystem))+1, menuBarY))
	got = updated.(model)
	if command != nil || got.dropdownOpen || got.menuFocused {
		t.Fatal("click on active menu did not close it")
	}
}

func clickInside(bounds layoutBounds) tea.MouseClickMsg {
	return mouseClick(bounds.x+max(0, bounds.width/2), bounds.y)
}
