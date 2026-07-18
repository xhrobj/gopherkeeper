package tui

import "charm.land/lipgloss/v2"

type theme struct {
	desktop                  lipgloss.Style
	menuBar                  lipgloss.Style
	menuItem                 lipgloss.Style
	menuMnemonic             lipgloss.Style
	menuActive               lipgloss.Style
	menuDisabled             lipgloss.Style
	menuHint                 lipgloss.Style
	windowTitle              lipgloss.Style
	windowBody               lipgloss.Style
	shadow                   lipgloss.Style
	button                   lipgloss.Style
	buttonActive             lipgloss.Style
	buttonDisabled           lipgloss.Style
	buttonDisabledActive     lipgloss.Style
	label                    lipgloss.Style
	input                    lipgloss.Style
	inputCursor              lipgloss.Style
	readOnly                 lipgloss.Style
	configRequiredYellow     lipgloss.Style
	configMissing            lipgloss.Style
	warning                  lipgloss.Style
	dropdown                 lipgloss.Style
	dropdownFrame            lipgloss.Style
	dropdownRow              lipgloss.Style
	dropdownMnemonic         lipgloss.Style
	dropdownPick             lipgloss.Style
	dropdownPickMnemonic     lipgloss.Style
	dropdownDisabled         lipgloss.Style
	dropdownDisabledMnemonic lipgloss.Style
	dropdownCurrent          lipgloss.Style
	dropdownCurrentSelected  lipgloss.Style
	aboutBody                lipgloss.Style
	aboutTitle               lipgloss.Style
	aboutLabel               lipgloss.Style
	aboutValue               lipgloss.Style
	aboutText                lipgloss.Style
	aboutInspiredYellow      lipgloss.Style
	aboutButton              lipgloss.Style
	aboutButtonActive        lipgloss.Style
	controlsText             lipgloss.Style
	controlsKey              lipgloss.Style
	errorBody                lipgloss.Style
	errorTitle               lipgloss.Style
	errorText                lipgloss.Style
	errorButton              lipgloss.Style
}

func newTheme() theme {
	var (
		black       = lipgloss.Color("#000000")
		blue        = lipgloss.Color("#0000AA")
		cyan        = lipgloss.Color("#00AAAA")
		lightGray   = lipgloss.Color("#AAAAAA")
		darkGray    = lipgloss.Color("#555555")
		magenta     = lipgloss.Color("#AA00AA")
		red         = lipgloss.Color("#AA0000")
		brightWhite = lipgloss.Color("#FFFFFF")
		yellow      = lipgloss.Color("#FFFF55")
	)

	return theme{
		desktop: lipgloss.NewStyle().
			Foreground(brightWhite).
			Background(blue),

		menuBar: lipgloss.NewStyle().
			Foreground(black).
			Background(lightGray),

		menuItem: lipgloss.NewStyle().
			Foreground(black).
			Background(lightGray),

		menuMnemonic: lipgloss.NewStyle().
			Foreground(brightWhite).
			Background(lightGray),

		menuActive: lipgloss.NewStyle().
			Foreground(black).
			Background(cyan),

		menuDisabled: lipgloss.NewStyle().
			Foreground(cyan).
			Background(lightGray),

		menuHint: lipgloss.NewStyle().
			Foreground(brightWhite).
			Background(lightGray),

		windowTitle: lipgloss.NewStyle().
			Foreground(yellow).
			Background(lightGray).
			Bold(true).
			AlignHorizontal(lipgloss.Center),

		windowBody: lipgloss.NewStyle().
			Foreground(brightWhite).
			Background(cyan),

		shadow: lipgloss.NewStyle().
			Background(black),

		button: lipgloss.NewStyle().
			Foreground(brightWhite).
			Background(cyan).
			Padding(0, 1),

		buttonActive: lipgloss.NewStyle().
			Foreground(cyan).
			Background(brightWhite).
			Padding(0, 1),

		buttonDisabled: lipgloss.NewStyle().
			Foreground(lightGray).
			Background(cyan).
			Padding(0, 1),

		buttonDisabledActive: lipgloss.NewStyle().
			Foreground(cyan).
			Background(lightGray).
			Padding(0, 1),

		label: lipgloss.NewStyle().
			Foreground(brightWhite).
			Background(cyan),

		input: lipgloss.NewStyle().
			Foreground(brightWhite).
			Background(blue),

		inputCursor: lipgloss.NewStyle().
			Foreground(black).
			Background(brightWhite),

		readOnly: lipgloss.NewStyle().
			Foreground(yellow).
			Background(cyan),

		configRequiredYellow: lipgloss.NewStyle().
			Foreground(yellow).
			Background(cyan),

		configMissing: lipgloss.NewStyle().
			Foreground(black).
			Background(cyan),

		warning: lipgloss.NewStyle().
			Foreground(yellow).
			Background(blue).
			Bold(true),

		dropdown: lipgloss.NewStyle().
			Foreground(black).
			Background(lightGray),

		dropdownFrame: lipgloss.NewStyle().
			Foreground(darkGray).
			Background(lightGray),

		dropdownRow: lipgloss.NewStyle().
			Foreground(black).
			Background(lightGray),

		dropdownMnemonic: lipgloss.NewStyle().
			Foreground(brightWhite).
			Background(lightGray),

		dropdownPick: lipgloss.NewStyle().
			Foreground(black).
			Background(cyan),

		dropdownPickMnemonic: lipgloss.NewStyle().
			Foreground(brightWhite).
			Background(cyan),

		dropdownDisabled: lipgloss.NewStyle().
			Foreground(cyan).
			Background(lightGray),

		dropdownDisabledMnemonic: lipgloss.NewStyle().
			Foreground(brightWhite).
			Background(lightGray),

		dropdownCurrent: lipgloss.NewStyle().
			Foreground(brightWhite).
			Background(lightGray),

		dropdownCurrentSelected: lipgloss.NewStyle().
			Foreground(brightWhite).
			Background(cyan),

		aboutBody: lipgloss.NewStyle().
			Foreground(brightWhite).
			Background(magenta),

		aboutTitle: lipgloss.NewStyle().
			Foreground(yellow).
			Background(magenta).
			Bold(true),

		aboutLabel: lipgloss.NewStyle().
			Foreground(lightGray).
			Background(magenta),

		aboutValue: lipgloss.NewStyle().
			Foreground(brightWhite).
			Background(magenta),

		aboutText: lipgloss.NewStyle().
			Foreground(brightWhite).
			Background(magenta),

		aboutInspiredYellow: lipgloss.NewStyle().
			Foreground(yellow).
			Background(magenta),

		aboutButton: lipgloss.NewStyle().
			Foreground(brightWhite).
			Background(magenta).
			Padding(0, 1),

		aboutButtonActive: lipgloss.NewStyle().
			Foreground(magenta).
			Background(brightWhite).
			Padding(0, 1),

		controlsText: lipgloss.NewStyle().
			Foreground(brightWhite).
			Background(cyan),

		controlsKey: lipgloss.NewStyle().
			Foreground(yellow).
			Background(cyan),

		errorBody: lipgloss.NewStyle().
			Foreground(brightWhite).
			Background(red),

		errorTitle: lipgloss.NewStyle().
			Foreground(yellow).
			Background(red).
			Bold(true),

		errorText: lipgloss.NewStyle().
			Foreground(brightWhite).
			Background(red),

		errorButton: lipgloss.NewStyle().
			Foreground(red).
			Background(brightWhite).
			Padding(0, 1),
	}
}
