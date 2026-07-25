package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/xhrobj/gopherkeeper/internal/buildinfo"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
)

func TestModel_ViewRendersFoxProDesktop(t *testing.T) {
	m := newTestModel(t, config.Config{Address: "localhost:8080"}, buildinfo.Info{})
	m.width = 80
	m.height = 25

	got := m.View().Content
	assertViewContains(t, got,
		"System",
		"Account",
		"Record",
		"Cache",
		"Window",
		"Help",
		"Login",
		"Password",
	)
	assertViewExcludes(t, got,
		"(^-^)/ GophKeeper",
		"Access your Secrets Securely",
	)
}
func TestModel_ViewRendersSmallTerminalMessage(t *testing.T) {
	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.width = 48
	m.height = 14

	got := m.View().Content
	assertViewContains(t, got,
		"Terminal window is too small.",
		"Required: 62×23",
		"Current:  48×14",
	)
}
func TestModel_AboutFitsMinimumTerminal(t *testing.T) {
	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.width = minimumWidth
	m.height = minimumHeight
	m.dialog = dialogAbout

	window, ok := m.dialogPlacement()
	if !ok {
		t.Fatal("About placement is unavailable")
	}
	if window.x+2+window.width > m.width {
		t.Fatalf("About shadow right edge = %d, terminal width = %d", window.x+2+window.width, m.width)
	}
	if window.y+1+window.height > m.height {
		t.Fatalf("About shadow bottom edge = %d, terminal height = %d", window.y+1+window.height, m.height)
	}

	view := m.View().Content
	if width := lipgloss.Width(view); width > minimumWidth {
		t.Fatalf("view width = %d, terminal width = %d", width, minimumWidth)
	}
	if height := lipgloss.Height(view); height > minimumHeight {
		t.Fatalf("view height = %d, terminal height = %d", height, minimumHeight)
	}
	assertViewContains(t, view, "(^-^)/", "GophKeeper", "Access your Secrets Securely")
}
func TestModel_UpdateResizesTerminal(t *testing.T) {
	m := newTestModel(t, config.Config{}, buildinfo.Info{})

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 32})
	got := updated.(model)
	if got.width != 100 || got.height != 32 {
		t.Fatalf("size = %dx%d, want 100x32", got.width, got.height)
	}
}
func TestModel_ViewUsesAlternateScreen(t *testing.T) {
	view := newTestModel(t, config.Config{}, buildinfo.Info{}).View()
	if !view.AltScreen {
		t.Fatal("AltScreen = false, want true")
	}
	if view.WindowTitle != "(^-^)/" {
		t.Fatalf("WindowTitle = %q, want (^-^)/", view.WindowTitle)
	}
	assertViewContains(t, view.Content, "F10 = Menu")
	lines := strings.Split(ansi.Strip(view.Content), "\n")
	if lineIndexContaining(lines, "F10 = Menu") != 0 {
		t.Fatal("F10 menu hint is not rendered on the top menu-bar row")
	}
}
