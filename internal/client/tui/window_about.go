package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/xhrobj/gopherkeeper/internal/buildinfo"
)

const aboutButtonGap = 3

type aboutWindowLayout struct {
	content      string
	buttonBounds []layoutBounds
}

func aboutWindowWidth(screenWidth int) int {
	return clamp(screenWidth-8, 58, 82)
}

func renderAboutWindow(
	t theme,
	width int,
	version string,
	date string,
	commit string,
	activeButton int,
) string {
	return newAboutWindowLayout(t, width, version, date, commit, activeButton).content
}

func newAboutWindowLayout(
	t theme,
	width int,
	version string,
	date string,
	commit string,
	activeButton int,
) aboutWindowLayout {
	rows := []string{
		renderAboutEmptyRow(t, width),
		renderAboutCenteredText(t.aboutTitle, width, "(^-^)/ GophKeeper"),
		renderAboutEmptyRow(t, width),
		renderAboutCenteredText(t.aboutTitle, width, "Access your Secrets Securely"),
		renderAboutEmptyRow(t, width),
		renderAboutBuildRow(t, width, "Version", version),
		renderAboutBuildRow(t, width, "Build date", date),
		renderAboutBuildRow(t, width, "Commit", commit),
		renderAboutEmptyRow(t, width),
		renderAboutCenteredText(t.aboutInspiredYellow, width, "Inspired by FoxPro 2.x for DOS"),
		renderAboutEmptyRow(t, width),
		renderAboutCenteredText(t.aboutValue, width, "Mikhail Eliseev"),
		renderAboutEmptyRow(t, width),
		renderAboutCenteredText(t.aboutText, width, "This project was completed as part of"),
		renderAboutCenteredText(t.aboutText, width, "Yandex Practicum’s “Advanced Go Developer” course"),
		renderAboutEmptyRow(t, width),
		renderAboutCenteredText(t.aboutText, width, aboutURL),
		renderAboutEmptyRow(t, width),
	}

	buttonRow := len(rows)
	buttonLayout := aboutButtonsLayout(t, width, activeButton).positioned(0, buttonRow)
	rows = append(rows, buttonLayout.content, renderAboutEmptyRow(t, width))

	return aboutWindowLayout{
		content:      strings.Join(rows, "\n"),
		buttonBounds: buttonLayout.bounds,
	}
}

func (m model) aboutWindowLayout() aboutWindowLayout {
	return newAboutWindowLayout(
		m.theme,
		aboutWindowWidth(m.width),
		buildinfo.Value(m.info.Version),
		buildinfo.Value(m.info.Date),
		buildinfo.Value(m.info.Commit),
		m.activeButton,
	)
}

func renderAboutEmptyRow(t theme, width int) string {
	return t.aboutBody.Width(width).Render("")
}

func renderAboutCenteredText(style lipgloss.Style, width int, value string) string {
	return style.Width(width).AlignHorizontal(lipgloss.Center).Render(value)
}

func renderAboutBuildRow(t theme, width int, label, value string) string {
	colonIndex := width / 2
	if width%2 == 0 {
		colonIndex--
	}

	leftLabel := t.aboutLabel.
		Width(max(0, colonIndex)).
		AlignHorizontal(lipgloss.Right).
		Render(label)
	colon := t.aboutLabel.Render(":")
	separator := t.aboutBody.Render(" ")
	valuePart := t.aboutValue.Render(value)
	rightPadding := max(0, width-colonIndex-2-lipgloss.Width(valuePart))

	return leftLabel +
		colon +
		separator +
		valuePart +
		t.aboutBody.Width(rightPadding).Render("")
}

func aboutButtonsLayout(t theme, width, activeButton int) buttonRowLayout {
	courseStyle := t.aboutButton
	okStyle := t.aboutButton

	if activeButton == 0 {
		courseStyle = t.aboutButtonActive
	} else {
		okStyle = t.aboutButtonActive
	}

	return centeredButtonRowLayout(t.aboutBody, width, aboutButtonGap, []styledButton{
		{label: "< Course >", style: courseStyle},
		{label: okButtonLabel, style: okStyle},
	})
}
