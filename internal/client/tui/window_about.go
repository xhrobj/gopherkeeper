package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

const (
	aboutButtonRow = 18
	aboutButtonGap = 3
)

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
	return renderAbout(t, width, version, date, commit, activeButton)
}

func renderAbout(
	t theme,
	width int,
	version string,
	date string,
	commit string,
	activeButton int,
) string {
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
		renderAboutButtons(t, width, activeButton),
		renderAboutEmptyRow(t, width),
	}

	return strings.Join(rows, "\n")
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
		{label: "< OK >", style: okStyle},
	})
}

func renderAboutButtons(t theme, width, activeButton int) string {
	return aboutButtonsLayout(t, width, activeButton).content
}
