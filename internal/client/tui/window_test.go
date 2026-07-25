package tui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestRenderControls_RendersGroupedAlignedRows(t *testing.T) {
	const width = 46
	if controlsColumnGap != 1 {
		t.Fatalf("controls column gap = %d, want 1", controlsColumnGap)
	}

	theme := newTheme()
	styled := renderControls(theme, width)
	if !strings.Contains(styled, theme.controlsSeparator.Render(":")) {
		t.Fatal("Controls colon is not rendered with the pale-gray separator style")
	}
	got := ansi.Strip(styled)
	lines := strings.Split(got, "\n")
	menuLine := lines[0]
	if index := strings.Index(menuLine, "Menu"); index != 16 {
		t.Fatalf("Controls table starts at column %d, want 16 after one-cell right shift", index)
	}
	if !strings.Contains(menuLine, "Menu: F10") {
		t.Fatalf("Controls row does not contain colon layout: %q", menuLine)
	}

	assertViewContains(t, got,
		"Menu",
		"F10",
		"Choose menu item",
		"Highlighted letter",
		"Navigate menu",
		"Arrow keys",
		"Move in list",
		"Up Arrow / Down Arrow",
		"Delete record",
		"Del",
		"Next control",
		"Tab / Right Arrow",
		"Previous control",
		"Shift+Tab / Left Arrow",
		"Quit",
		"Ctrl+Q",
		"< OK >",
	)

	separatorColumn := strings.Index(menuLine, ":")
	if separatorColumn < 0 {
		t.Fatalf("Controls colon was not found: %q", menuLine)
	}
	assertControlsSpacerRows(t, lines)
	assertControlsSeparatorAlignment(t, lines, separatorColumn)
	assertControlsRowWidths(t, lines, width)
}

func assertControlsSpacerRows(t *testing.T, lines []string) {
	t.Helper()
	for _, index := range []int{1, 4, 8, 12, 16, 18} {
		if strings.TrimSpace(lines[index]) != "" {
			t.Fatalf("controls spacer row %d is not empty: %q", index, lines[index])
		}
	}
}

func assertControlsSeparatorAlignment(t *testing.T, lines []string, separatorColumn int) {
	t.Helper()
	for _, index := range []int{0, 2, 3, 5, 6, 7, 9, 10, 11, 13, 14, 15, 17} {
		if strings.Index(lines[index], ":") != separatorColumn {
			t.Fatalf("controls colon row %d is not aligned: %q", index, lines[index])
		}
	}
}

func assertControlsRowWidths(t *testing.T, lines []string, width int) {
	t.Helper()
	for _, line := range lines {
		if line != "" && lipgloss.Width(line) != width {
			t.Fatalf("controls row width = %d, want %d: %q", lipgloss.Width(line), width, line)
		}
	}
}

func TestRenderAbout_RendersFoxProStyleWindow(t *testing.T) {
	const width = 72
	got := renderAboutWindow(newTheme(), width, "v0.9.0", "2026-07-17", "9a36fb0", 1)
	plain := ansi.Strip(got)

	assertViewContains(t, plain,
		"(^-^)/",
		"GophKeeper",
		"Access your Secrets Securely",
		"Version:",
		"v0.9.0",
		"Build date:",
		"2026-07-17",
		"Commit:",
		"9a36fb0",
		"Mikhail Eliseev",
		"This project was completed as part of",
		"Yandex Practicum’s “Advanced Go Developer” course",
		aboutURL,
		"Inspired by FoxPro 2.x for DOS",
		"< Course >",
		"< OK >",
	)
	assertViewExcludes(t, plain,
		"About GophKeeper",
		"< Done >",
		"Второй итоговый проект",
		"Продвинутый Go-разработчик",
		"Яндекс.Практикум",
		"┌", "┐", "└", "┘",
		"╔", "╗", "╚", "╝",
	)

	if gotWidth := lipgloss.Width(got); gotWidth != width {
		t.Fatalf("about width = %d, want %d", gotWidth, width)
	}
	const wantHeight = 20
	if gotHeight := lipgloss.Height(got); gotHeight != wantHeight {
		t.Fatalf("about height = %d, want %d", gotHeight, wantHeight)
	}

	lines := strings.Split(plain, "\n")
	if strings.TrimSpace(lines[0]) != "" || strings.TrimSpace(lines[len(lines)-1]) != "" {
		t.Fatal("about does not have empty top and bottom frame rows")
	}
	mascotLine := lineIndexContaining(lines, "(^-^)/")
	titleLine := lineIndexContaining(lines, "GophKeeper")
	sloganLine := lineIndexContaining(lines, "Access your Secrets Securely")
	if titleLine != mascotLine {
		t.Fatalf("mascot/title lines = %d/%d, want the same row", mascotLine, titleLine)
	}
	if sloganLine != titleLine+2 || strings.TrimSpace(lines[titleLine+1]) != "" {
		t.Fatalf(
			"title/slogan lines = %d/%d, want one empty row between them",
			titleLine,
			sloganLine,
		)
	}
	for _, label := range []string{"Version:", "Build date:", "Commit:"} {
		lineIndex := lineIndexContaining(lines, label)
		if lineIndex < 0 {
			t.Fatalf("build info line %q not found", label)
		}
		colonIndex := strings.Index(lines[lineIndex], ":")
		if colonIndex != width/2-1 {
			t.Fatalf("%s colon index = %d, want %d", label, colonIndex, width/2-1)
		}
	}

	buttonLine := lineIndexContaining(lines, "< OK >")
	if buttonLine != len(lines)-2 {
		t.Fatalf("button line = %d, want one empty row before bottom (%d)", buttonLine, len(lines)-2)
	}
}
func TestRenderAlertWindow_HasNoBorderAndAlignedTopSpacing(t *testing.T) {
	theme := newTheme()
	view := renderAlertWindow(
		theme,
		alertNotice,
		"Registration successful",
		"Registered as alice. Please log in.",
		"alice",
	)
	plain := ansi.Strip(view)
	lines := strings.Split(plain, "\n")

	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "" {
		t.Fatalf("alert does not have an empty row above the title: %q", plain)
	}
	for _, border := range []string{"┌", "┐", "└", "┘", "│", "─"} {
		if strings.Contains(plain, border) {
			t.Fatalf("alert contains border character %q: %q", border, plain)
		}
	}
	if !strings.Contains(view, theme.aboutTitle.Render("alice")) {
		t.Fatal("registered login is not rendered with the yellow highlight style")
	}
}

func TestRenderErrorAlert_WrapsCompleteMessage(t *testing.T) {
	theme := newTheme()
	message := strings.Repeat("binary save failed ", 12)
	view := renderAlertWindow(theme, alertError, "Unable to save binary", message, "")
	plain := ansi.Strip(view)
	for _, word := range strings.Fields(message) {
		if !strings.Contains(plain, word) {
			t.Fatalf("alert lost message fragment %q:\n%s", word, plain)
		}
	}
	if got, want := alertButtonRow(message, ""), 4+len(wrapAlertText(message, 50)); got != want {
		t.Fatalf("alert button row = %d, want %d", got, want)
	}
}

func TestRenderErrorAlert_DoesNotSplitRevisionConflictWords(t *testing.T) {
	theme := newTheme()
	message := "The record was changed on another device. Reopen it and try again."
	plain := ansi.Strip(renderAlertWindow(theme, alertError, "Unable to update record", message, ""))

	for _, word := range strings.Fields(message) {
		if !strings.Contains(plain, word) {
			t.Fatalf("alert split word %q:\n%s", word, plain)
		}
	}
}
