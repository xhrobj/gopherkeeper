package tui

import (
	"context"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/xhrobj/gopherkeeper/internal/buildinfo"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
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
