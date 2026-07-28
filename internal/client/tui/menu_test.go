package tui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/xhrobj/gopherkeeper/internal/buildinfo"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
)

func TestRenderMenuBar_HighlightsRequestedTopLevelLetters(t *testing.T) {
	theme := newTheme()
	definitions := menuDefinitions(dialogLogin, false)
	definitions[menuRecord].disabled = false
	definitions[menuRecord].items[0].disabled = false
	definitions[menuCache].disabled = false
	definitions[menuCache].items[0].disabled = false

	view := renderMenuBar(theme, definitions, 80, int(menuHelp), false, false)
	for _, mnemonic := range []string{"S", "R", "C", "W"} {
		if !strings.Contains(view, theme.menuMnemonic.Render(mnemonic)) {
			t.Fatalf("top-level mnemonic %q is not highlighted", mnemonic)
		}
	}
}

func TestRenderMenuBar_BlocksEnabledMenusDuringNetworkRequest(t *testing.T) {
	theme := newTheme()
	definitions := testMenuDefinitions(dialogLogin)

	busy := renderMenuBar(theme, definitions, 80, int(menuSystem), false, true)
	if !strings.Contains(busy, theme.menuBlocked.Render(" System ")) {
		t.Fatal("enabled System menu is not visually blocked")
	}
	if !strings.Contains(busy, theme.menuDisabled.Render(" Record ")) {
		t.Fatal("already disabled Record menu changed its disabled style")
	}
	if strings.Contains(busy, theme.menuMnemonic.Render("S")) {
		t.Fatal("blocked menu still highlights its mnemonic")
	}
	if renderMenuHint(theme, true) != theme.menuHintBlocked.Render(menuHintText) {
		t.Fatal("F10 hint is not visually blocked")
	}

	idle := renderMenuBar(theme, definitions, 80, int(menuSystem), false, false)
	if !strings.Contains(idle, theme.menuMnemonic.Render("S")) {
		t.Fatal("enabled menu did not restore its normal mnemonic after the request")
	}
	if renderMenuHint(theme, false) != theme.menuHint.Render(menuHintText) {
		t.Fatal("F10 hint did not restore its normal style after the request")
	}
}

func TestAccountMenu_ListsLoginBeforeRegister(t *testing.T) {
	items := menus[menuAccount].items
	if len(items) < 2 || items[0].action != actionLogin || items[1].action != actionRegister {
		t.Fatalf("Account menu order = %#v, want Login then Register", items)
	}
}

func TestRenderDropdown_HighlightsBrowseMnemonic(t *testing.T) {
	theme := newTheme()
	definition := menuDefinitions(dialogNone, false)[menuRecord]
	definition.items[0].disabled = false
	view := testRenderDropdown(theme, definition, 0)
	if !strings.Contains(view, theme.dropdownPickMnemonic.Render("B")) {
		t.Fatal("Browse mnemonic B is not highlighted")
	}
	if strings.Contains(ansi.Strip(view), "Browse Server") {
		t.Fatal("Record menu still contains the redundant Server suffix")
	}
}

func TestRenderDropdown_HighlightsWindowCloseMnemonic(t *testing.T) {
	theme := newTheme()
	view := testRenderDropdown(theme, menus[menuWindow], 0)
	if !strings.Contains(view, theme.dropdownPickMnemonic.Render("W")) {
		t.Fatal("Close Active Window mnemonic W is not highlighted")
	}
}

func TestRenderDropdown_HighlightsSecondLogoutO(t *testing.T) {
	theme := newTheme()
	item := menus[menuAccount].items[4]
	item.disabled = false
	contentWidth := dropdownWidth(menus[menuAccount])
	got := renderMenuItemLine(theme, item, contentWidth, true, false)
	gap := contentWidth - lipgloss.Width(" "+item.label)
	want := theme.dropdownPick.Render(" Log") +
		theme.dropdownPickMnemonic.Render("o") +
		theme.dropdownPick.Render("ut"+strings.Repeat(" ", gap))

	if got != want {
		t.Fatal("Logout does not highlight the second o")
	}
}

func TestRenderDropdown_KeepsLabelsOnOneLine(t *testing.T) {
	got := ansi.Strip(testRenderDropdown(newTheme(), menus[menuSystem], 0))

	if !strings.Contains(got, "│ Server Status │") {
		t.Fatalf("dropdown does not contain one-line Server Status row:\n%s", got)
	}
	if strings.Contains(got, "Server\n") || strings.Contains(got, "Status\n") {
		t.Fatalf("Server Status was wrapped:\n%s", got)
	}
}

func TestRenderDropdown_RendersConnectedSeparator(t *testing.T) {
	got := ansi.Strip(testRenderDropdown(newTheme(), menus[menuSystem], 0))
	contentWidth := dropdownWidth(menus[menuSystem])
	want := "├" + strings.Repeat("─", contentWidth) + "┤"

	if !strings.Contains(got, want) {
		t.Fatalf("dropdown separator not found: want %q in\n%s", want, got)
	}
}

func TestRenderDropdown_RendersQuitShortcut(t *testing.T) {
	got := ansi.Strip(testRenderDropdown(newTheme(), menus[menuSystem], 2))

	if !strings.Contains(got, "Quit") || !strings.Contains(got, "^Q") {
		t.Fatalf("dropdown does not contain Quit shortcut:\n%s", got)
	}

	for _, line := range strings.Split(got, "\n") {
		if strings.Contains(line, "Quit") && !strings.Contains(line, "^Q") {
			t.Fatalf("Quit and ^Q are not on the same row: %q", line)
		}
	}
}

func TestHelpMenu_ListsOnlyControlsAndAbout(t *testing.T) {
	items := menus[menuHelp].items
	if len(items) != 2 {
		t.Fatalf("Help menu item count = %d, want 2", len(items))
	}
	if items[0].action != actionControls || items[1].action != actionAbout {
		t.Fatalf("Help menu actions = %#v, want Controls then About", items)
	}
}

func TestRenderDropdown_HighlightsHelpItemMnemonics(t *testing.T) {
	theme := newTheme()
	controls := testRenderDropdown(theme, menus[menuHelp], 0)
	about := testRenderDropdown(theme, menus[menuHelp], 1)

	if !strings.Contains(controls, theme.dropdownPickMnemonic.Render("C")) {
		t.Fatal("Controls mnemonic C is not highlighted in selected row")
	}
	if !strings.Contains(about, theme.dropdownPickMnemonic.Render("G")) {
		t.Fatal("About mnemonic G is not highlighted in selected row")
	}
}

func TestRenderDropdown_HighlightsSystemItemMnemonics(t *testing.T) {
	theme := newTheme()
	serverStatus := testRenderDropdown(theme, menus[menuSystem], 0)
	config := testRenderDropdown(theme, menus[menuSystem], 1)
	quit := testRenderDropdown(theme, menus[menuSystem], 2)

	if !strings.Contains(serverStatus, theme.dropdownPickMnemonic.Render("u")) {
		t.Fatal("Server Status mnemonic u is not highlighted in selected row")
	}
	if !strings.Contains(config, theme.dropdownPickMnemonic.Render("C")) {
		t.Fatal("Config mnemonic C is not highlighted in selected row")
	}
	if !strings.Contains(quit, theme.dropdownPickMnemonic.Render("Q")) {
		t.Fatal("Quit mnemonic Q is not highlighted in selected row")
	}
}

func TestMenuItemByMnemonic_SelectsSystemActions(t *testing.T) {
	tests := []struct {
		key        rune
		wantIndex  int
		wantAction actionID
	}{
		{key: 'u', wantIndex: 0, wantAction: actionServerStatus},
		{key: 'U', wantIndex: 0, wantAction: actionServerStatus},
		{key: 'c', wantIndex: 1, wantAction: actionConfig},
		{key: 'C', wantIndex: 1, wantAction: actionConfig},
		{key: 'q', wantIndex: 2, wantAction: actionQuit},
		{key: 'Q', wantIndex: 2, wantAction: actionQuit},
	}

	for _, test := range tests {
		index, item, ok := menuItemByMnemonic(menus[menuSystem], test.key)
		if !ok {
			t.Fatalf("mnemonic %q was not found", test.key)
		}
		if index != test.wantIndex || item.action != test.wantAction {
			t.Fatalf("mnemonic %q = (index %d, action %d), want (%d, %d)",
				test.key, index, item.action, test.wantIndex, test.wantAction)
		}
	}
}

func TestMenuItemByMnemonic_SelectsHelpActions(t *testing.T) {
	tests := []struct {
		key        rune
		wantIndex  int
		wantAction actionID
	}{
		{key: 'c', wantIndex: 0, wantAction: actionControls},
		{key: 'C', wantIndex: 0, wantAction: actionControls},
		{key: 'g', wantIndex: 1, wantAction: actionAbout},
		{key: 'G', wantIndex: 1, wantAction: actionAbout},
	}

	for _, test := range tests {
		index, item, ok := menuItemByMnemonic(menus[menuHelp], test.key)
		if !ok {
			t.Fatalf("mnemonic %q was not found", test.key)
		}
		if index != test.wantIndex || item.action != test.wantAction {
			t.Fatalf("mnemonic %q = (index %d, action %d), want (%d, %d)",
				test.key, index, item.action, test.wantIndex, test.wantAction)
		}
	}
}

func TestMenuDefinitions_UseDialogEllipsesConsistently(t *testing.T) {
	if menus[menuRecord].name != "Record" {
		t.Fatalf("record menu name = %q, want Record", menus[menuRecord].name)
	}

	assertMenuLabel(t, menuSystem, "Server Status", false)
	assertMenuLabel(t, menuSystem, "Config...", false)
	assertMenuLabel(t, menuAccount, "Register...", false)
	assertMenuLabel(t, menuAccount, "Login...", false)
	assertMenuLabel(t, menuAccount, "Current User", true)
	assertMenuLabel(t, menuRecord, "Edit...", true)
	assertMenuLabel(t, menuCache, "Browse...", false)
	assertMenuLabel(t, menuCache, "View", true)
	assertMenuLabel(t, menuCache, "Sync...", true)
	assertMenuLabel(t, menuHelp, "Controls", false)
	assertMenuLabel(t, menuHelp, "About GophKeeper", false)

	if menus[menuWindow].disabled {
		t.Fatal("Window menu is disabled")
	}

	loginMenus := testMenuDefinitions(dialogLogin)
	if loginMenus[menuWindow].items[0].disabled {
		t.Fatal("Close Active Window is disabled for Login")
	}
	if loginMenus[menuWindow].disabled {
		t.Fatal("Window menu is disabled with Login open")
	}

	emptyMenus := testMenuDefinitions(dialogNone)
	if !emptyMenus[menuWindow].items[0].disabled || !emptyMenus[menuWindow].disabled {
		t.Fatal("Window menu is enabled without an active window")
	}

	aboutMenus := testMenuDefinitions(dialogAbout)
	if aboutMenus[menuWindow].items[0].disabled {
		t.Fatal("Close Active Window is disabled for About")
	}
	if aboutMenus[menuWindow].disabled {
		t.Fatal("Window menu is disabled with an active command")
	}
}

func TestMenuItemByMnemonic_SelectsAccountActions(t *testing.T) {
	tests := []struct {
		key        rune
		wantIndex  int
		wantAction actionID
		wantOK     bool
	}{
		{key: 'r', wantIndex: 1, wantAction: actionRegister, wantOK: true},
		{key: 'l', wantIndex: 0, wantAction: actionLogin, wantOK: true},
		{key: 'u', wantOK: false},
		{key: 'o', wantOK: false},
	}

	for _, test := range tests {
		index, item, ok := menuItemByMnemonic(menus[menuAccount], test.key)
		if ok != test.wantOK {
			t.Fatalf("mnemonic %q available = %t, want %t", test.key, ok, test.wantOK)
		}
		if ok && (index != test.wantIndex || item.action != test.wantAction) {
			t.Fatalf("mnemonic %q = (index %d, action %d), want (%d, %d)",
				test.key, index, item.action, test.wantIndex, test.wantAction)
		}
	}
}

func TestRenderDropdown_MarksCurrentItemWhite(t *testing.T) {
	theme := newTheme()
	item := menus[menuAccount].items[0]
	contentWidth := dropdownWidth(menus[menuAccount])
	got := renderMenuItemLine(theme, item, contentWidth, false, true)
	want := theme.dropdownCurrent.Render(" Login..." + strings.Repeat(" ", contentWidth-len(" Login...")))
	if got != want {
		t.Fatal("current Login item is not rendered as one solid current-item row")
	}
}

func TestRenderDropdown_DoesNotHighlightDisabledAccountMnemonics(t *testing.T) {
	theme := newTheme()
	contentWidth := dropdownWidth(menus[menuAccount])

	currentUser := renderMenuItemLine(theme, menus[menuAccount].items[2], contentWidth, false, false)
	if strings.Contains(currentUser, theme.dropdownDisabledMnemonic.Render("U")) {
		t.Fatal("disabled Current User mnemonic U is highlighted")
	}

	logout := renderMenuItemLine(theme, menus[menuAccount].items[4], contentWidth, false, false)
	if strings.Contains(logout, theme.dropdownDisabledMnemonic.Render("o")) {
		t.Fatal("disabled Logout mnemonic o is highlighted")
	}
}

func TestMenuDefinitions_SwitchAccountActionsAfterLogin(t *testing.T) {
	definitions := menuDefinitions(dialogCurrentUser, true)
	items := definitions[menuAccount].items
	if !items[0].disabled || !items[1].disabled {
		t.Fatal("Register or Login remains enabled after login")
	}
	if items[2].disabled || items[4].disabled {
		t.Fatal("Current User or Logout remains disabled after login")
	}
}

func TestRenderDropdown_CurrentSelectedKeepsSelectionBackground(t *testing.T) {
	theme := newTheme()
	item := menus[menuAccount].items[0]
	contentWidth := dropdownWidth(menus[menuAccount])
	got := renderMenuItemLine(theme, item, contentWidth, true, true)
	want := theme.dropdownCurrentSelected.Render(" Login..." + strings.Repeat(" ", contentWidth-len(" Login...")))
	if got != want {
		t.Fatal("selected current item does not keep the selected-row background")
	}
}

func TestRenderMenuBar_FillsWidth(t *testing.T) {
	got := renderMenuBar(newTheme(), testMenuDefinitions(dialogLogin), 80, int(menuSystem), false, false)
	if width := lipgloss.Width(got); width != 80 {
		t.Fatalf("menu bar width = %d, want 80", width)
	}
	if strings.Contains(got, "SIGNED OUT") {
		t.Fatal("menu bar contains legacy status text")
	}
}

func TestMenuMnemonics(t *testing.T) {
	definitions := testMenuDefinitions(dialogLogin)
	tests := []struct {
		key  rune
		want menuID
	}{
		{key: 's', want: menuSystem},
		{key: 'a', want: menuAccount},
		{key: 'c', want: menuCache},
		{key: 'h', want: menuHelp},
	}

	for _, test := range tests {
		index, ok := menuIndexByMnemonic(definitions, test.key)
		if !ok {
			t.Fatalf("mnemonic %q was not found", test.key)
		}
		if menuID(index) != test.want {
			t.Fatalf("mnemonic %q selects menu %d, want %d", test.key, index, test.want)
		}
	}

	for _, disabled := range []rune{'r'} {
		if _, ok := menuIndexByMnemonic(definitions, disabled); ok {
			t.Fatalf("disabled mnemonic %q is selectable", disabled)
		}
	}
}

func TestModel_MenuNavigationSkipsDisabledMenus(t *testing.T) {
	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.activeMenu = int(menuAccount)
	m.menuFocused = true

	updated, _ := m.Update(keyPress("right"))
	got := updated.(model)
	if got.activeMenu != int(menuCache) {
		t.Fatalf("active menu = %d, want Cache (%d)", got.activeMenu, menuCache)
	}
	if !got.dropdownOpen {
		t.Fatal("dropdown is closed after menu navigation")
	}
}

func TestModel_MenuMnemonicOpensDropdown(t *testing.T) {
	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.menuFocused = true

	updated, _ := m.Update(keyPress("a"))
	got := updated.(model)
	if got.activeMenu != int(menuAccount) {
		t.Fatalf("active menu = %d, want Account (%d)", got.activeMenu, menuAccount)
	}
	if !got.menuFocused || !got.dropdownOpen {
		t.Fatal("Account menu was not opened by mnemonic")
	}
}

func TestMenuAltShortcut(t *testing.T) {
	loginDefinitions := testMenuDefinitions(dialogLogin)
	index, ok := menuIndexByAltKey(loginDefinitions, "alt+c")
	if !ok || index != int(menuCache) {
		t.Fatalf("alt+c = (%d, %t), want Cache", index, ok)
	}
	index, ok = menuIndexByAltKey(loginDefinitions, "alt+h")
	if !ok || index != int(menuHelp) {
		t.Fatalf("alt+h = (%d, %t), want Help", index, ok)
	}
	if _, ok := menuIndexByAltKey(loginDefinitions, "alt+r"); ok {
		t.Fatal("disabled Records menu is available through Alt shortcut")
	}
	index, ok = menuIndexByAltKey(loginDefinitions, "alt+w")
	if !ok || index != int(menuWindow) {
		t.Fatalf("alt+w = (%d, %t), want Window", index, ok)
	}

	aboutDefinitions := testMenuDefinitions(dialogAbout)
	index, ok = menuIndexByAltKey(aboutDefinitions, "alt+w")
	if !ok || index != int(menuWindow) {
		t.Fatalf("alt+w = (%d, %t), want Window", index, ok)
	}
}

func assertMenuLabel(t *testing.T, menu menuID, label string, disabled bool) {
	t.Helper()
	for _, item := range menus[menu].items {
		if item.label == label {
			if item.disabled != disabled {
				t.Fatalf("menu %q disabled = %t, want %t", label, item.disabled, disabled)
			}
			return
		}
	}
	t.Fatalf("menu item %q was not found", label)
}
