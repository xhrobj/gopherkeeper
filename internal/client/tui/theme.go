package tui

import "charm.land/lipgloss/v2"

type theme struct {
	desktop                        lipgloss.Style
	menuBar                        lipgloss.Style
	menuItem                       lipgloss.Style
	menuMnemonic                   lipgloss.Style
	menuActive                     lipgloss.Style
	menuDisabled                   lipgloss.Style
	menuBlocked                    lipgloss.Style
	menuHint                       lipgloss.Style
	menuHintBlocked                lipgloss.Style
	windowTitle                    lipgloss.Style
	windowTitleInactive            lipgloss.Style
	windowBody                     lipgloss.Style
	recordFormBody                 lipgloss.Style
	recordFormLabel                lipgloss.Style
	recordFormInfo                 lipgloss.Style
	recordFormReadOnly             lipgloss.Style
	recordFormMissing              lipgloss.Style
	recordFormRequired             lipgloss.Style
	recordFormButton               lipgloss.Style
	recordFormButtonActive         lipgloss.Style
	recordFormButtonDisabled       lipgloss.Style
	recordFormButtonDisabledActive lipgloss.Style
	shadow                         lipgloss.Style
	button                         lipgloss.Style
	buttonActive                   lipgloss.Style
	buttonDisabled                 lipgloss.Style
	buttonDisabledActive           lipgloss.Style
	label                          lipgloss.Style
	input                          lipgloss.Style
	inputCursor                    lipgloss.Style
	readOnly                       lipgloss.Style
	recordHeader                   lipgloss.Style
	recordRow                      lipgloss.Style
	recordSelected                 lipgloss.Style
	recordGrid                     lipgloss.Style
	recordInfo                     lipgloss.Style
	recordMessage                  lipgloss.Style
	recordMemoBody                 lipgloss.Style
	recordMemoTrack                lipgloss.Style
	recordCard                     lipgloss.Style
	recordCardMuted                lipgloss.Style
	recordCardStripe               lipgloss.Style
	configRequiredYellow           lipgloss.Style
	configMissing                  lipgloss.Style
	configPickerArea               lipgloss.Style
	configPickerDirectory          lipgloss.Style
	configPickerFile               lipgloss.Style
	configPickerSelectedDirectory  lipgloss.Style
	configPickerSelectedFile       lipgloss.Style
	warning                        lipgloss.Style
	dropdown                       lipgloss.Style
	dropdownFrame                  lipgloss.Style
	dropdownRow                    lipgloss.Style
	dropdownMnemonic               lipgloss.Style
	dropdownPick                   lipgloss.Style
	dropdownPickMnemonic           lipgloss.Style
	dropdownDisabled               lipgloss.Style
	dropdownDisabledMnemonic       lipgloss.Style
	dropdownCurrent                lipgloss.Style
	dropdownCurrentSelected        lipgloss.Style
	aboutBody                      lipgloss.Style
	aboutTitle                     lipgloss.Style
	aboutLabel                     lipgloss.Style
	aboutValue                     lipgloss.Style
	aboutText                      lipgloss.Style
	aboutInspiredYellow            lipgloss.Style
	aboutButton                    lipgloss.Style
	aboutButtonActive              lipgloss.Style
	aboutButtonDisabled            lipgloss.Style
	aboutButtonDisabledActive      lipgloss.Style
	wizardBody                     lipgloss.Style
	wizardPrompt                   lipgloss.Style
	wizardType                     lipgloss.Style
	wizardDescription              lipgloss.Style
	wizardSelected                 lipgloss.Style
	wizardSelectedType             lipgloss.Style
	wizardSelectedDescription      lipgloss.Style
	controlsText                   lipgloss.Style
	controlsSeparator              lipgloss.Style
	controlsKey                    lipgloss.Style
	errorBody                      lipgloss.Style
	errorTitle                     lipgloss.Style
	errorLabel                     lipgloss.Style
	errorText                      lipgloss.Style
	errorButton                    lipgloss.Style
	errorButtonDisabled            lipgloss.Style
	errorButtonDisabledActive      lipgloss.Style
}

func newTheme() theme {
	var (
		black         = lipgloss.Color("#000000")
		blue          = lipgloss.Color("#0000AA")
		cyan          = lipgloss.Color("#00AAAA")
		interfaceGray = lipgloss.Color("#AAAAAA")
		detailGray    = lipgloss.Color("#E4E4E4")
		frameGray     = lipgloss.Color("#555555")
		magenta       = lipgloss.Color("#AA00AA")
		red           = lipgloss.Color("#AA0000")
		brightWhite   = lipgloss.Color("#FFFFFF")
		yellow        = lipgloss.Color("#FFFF55")
	)

	return theme{
		desktop: lipgloss.NewStyle().
			Foreground(brightWhite).
			Background(blue),

		menuBar: lipgloss.NewStyle().
			Foreground(black).
			Background(interfaceGray),

		menuItem: lipgloss.NewStyle().
			Foreground(black).
			Background(interfaceGray),

		menuMnemonic: lipgloss.NewStyle().
			Foreground(brightWhite).
			Background(interfaceGray),

		menuActive: lipgloss.NewStyle().
			Foreground(black).
			Background(cyan),

		menuDisabled: lipgloss.NewStyle().
			Foreground(cyan).
			Background(interfaceGray),

		menuBlocked: lipgloss.NewStyle().
			Foreground(frameGray).
			Background(interfaceGray),

		menuHint: lipgloss.NewStyle().
			Foreground(brightWhite).
			Background(interfaceGray),

		menuHintBlocked: lipgloss.NewStyle().
			Foreground(frameGray).
			Background(interfaceGray),

		windowTitle: lipgloss.NewStyle().
			Foreground(yellow).
			Background(interfaceGray).
			Bold(true).
			AlignHorizontal(lipgloss.Center),

		windowTitleInactive: lipgloss.NewStyle().
			Foreground(frameGray).
			Background(interfaceGray).
			Bold(true).
			AlignHorizontal(lipgloss.Center),

		windowBody: lipgloss.NewStyle().
			Foreground(brightWhite).
			Background(cyan),

		recordFormBody: lipgloss.NewStyle().
			Foreground(brightWhite).
			Background(magenta),

		recordFormLabel: lipgloss.NewStyle().
			Foreground(brightWhite).
			Background(magenta),

		recordFormInfo: lipgloss.NewStyle().
			Foreground(detailGray).
			Background(magenta),

		recordFormReadOnly: lipgloss.NewStyle().
			Foreground(yellow).
			Background(magenta),

		recordFormMissing: lipgloss.NewStyle().
			Foreground(detailGray).
			Background(magenta),

		recordFormRequired: lipgloss.NewStyle().
			Foreground(yellow).
			Background(magenta),

		recordFormButton: lipgloss.NewStyle().
			Foreground(brightWhite).
			Background(magenta).
			Padding(0, 1),

		recordFormButtonActive: lipgloss.NewStyle().
			Foreground(magenta).
			Background(brightWhite).
			Padding(0, 1),

		recordFormButtonDisabled: lipgloss.NewStyle().
			Foreground(interfaceGray).
			Background(magenta).
			Padding(0, 1),

		recordFormButtonDisabledActive: lipgloss.NewStyle().
			Foreground(magenta).
			Background(interfaceGray).
			Padding(0, 1),

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
			Foreground(interfaceGray).
			Background(cyan).
			Padding(0, 1),

		buttonDisabledActive: lipgloss.NewStyle().
			Foreground(cyan).
			Background(interfaceGray).
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

		recordHeader: lipgloss.NewStyle().
			Foreground(brightWhite).
			Background(cyan).
			Bold(true),

		recordRow: lipgloss.NewStyle().
			Foreground(brightWhite).
			Background(cyan),

		recordSelected: lipgloss.NewStyle().
			Foreground(brightWhite).
			Background(blue),

		recordGrid: lipgloss.NewStyle().
			Foreground(interfaceGray).
			Background(cyan),

		recordInfo: lipgloss.NewStyle().
			Foreground(detailGray).
			Background(cyan),

		recordMessage: lipgloss.NewStyle().
			Foreground(yellow).
			Background(cyan),

		recordMemoBody: lipgloss.NewStyle().
			Foreground(brightWhite).
			Background(blue),

		recordMemoTrack: lipgloss.NewStyle().
			Foreground(black).
			Background(interfaceGray),

		recordCard: lipgloss.NewStyle().
			Foreground(brightWhite).
			Background(black),

		recordCardMuted: lipgloss.NewStyle().
			Foreground(detailGray).
			Background(black),

		recordCardStripe: lipgloss.NewStyle().
			Foreground(interfaceGray).
			Background(black),

		configRequiredYellow: lipgloss.NewStyle().
			Foreground(yellow).
			Background(cyan),

		configMissing: lipgloss.NewStyle().
			Foreground(detailGray).
			Background(cyan),

		configPickerArea: lipgloss.NewStyle().
			Foreground(brightWhite).
			Background(blue),

		configPickerDirectory: lipgloss.NewStyle().
			Foreground(interfaceGray).
			Background(blue),

		configPickerFile: lipgloss.NewStyle().
			Foreground(brightWhite).
			Background(blue),

		configPickerSelectedDirectory: lipgloss.NewStyle().
			Foreground(interfaceGray).
			Background(brightWhite),

		configPickerSelectedFile: lipgloss.NewStyle().
			Foreground(blue).
			Background(brightWhite).
			Bold(true),

		warning: lipgloss.NewStyle().
			Foreground(yellow).
			Background(blue).
			Bold(true),

		dropdown: lipgloss.NewStyle().
			Foreground(black).
			Background(interfaceGray),

		dropdownFrame: lipgloss.NewStyle().
			Foreground(frameGray).
			Background(interfaceGray),

		dropdownRow: lipgloss.NewStyle().
			Foreground(black).
			Background(interfaceGray),

		dropdownMnemonic: lipgloss.NewStyle().
			Foreground(brightWhite).
			Background(interfaceGray),

		dropdownPick: lipgloss.NewStyle().
			Foreground(black).
			Background(cyan),

		dropdownPickMnemonic: lipgloss.NewStyle().
			Foreground(brightWhite).
			Background(cyan),

		dropdownDisabled: lipgloss.NewStyle().
			Foreground(cyan).
			Background(interfaceGray),

		dropdownDisabledMnemonic: lipgloss.NewStyle().
			Foreground(brightWhite).
			Background(interfaceGray),

		dropdownCurrent: lipgloss.NewStyle().
			Foreground(brightWhite).
			Background(interfaceGray),

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
			Foreground(interfaceGray).
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

		aboutButtonDisabled: lipgloss.NewStyle().
			Foreground(interfaceGray).
			Background(magenta).
			Padding(0, 1),

		aboutButtonDisabledActive: lipgloss.NewStyle().
			Foreground(magenta).
			Background(interfaceGray).
			Padding(0, 1),

		wizardBody: lipgloss.NewStyle().
			Foreground(brightWhite).
			Background(magenta),

		wizardPrompt: lipgloss.NewStyle().
			Foreground(brightWhite).
			Background(magenta),

		wizardType: lipgloss.NewStyle().
			Foreground(yellow).
			Background(magenta).
			Bold(true),

		wizardDescription: lipgloss.NewStyle().
			Foreground(brightWhite).
			Background(magenta),

		wizardSelected: lipgloss.NewStyle().
			Foreground(brightWhite).
			Background(blue),

		wizardSelectedType: lipgloss.NewStyle().
			Foreground(yellow).
			Background(blue).
			Bold(true),

		wizardSelectedDescription: lipgloss.NewStyle().
			Foreground(brightWhite).
			Background(blue),

		controlsText: lipgloss.NewStyle().
			Foreground(brightWhite).
			Background(cyan),

		controlsSeparator: lipgloss.NewStyle().
			Foreground(detailGray).
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
			Bold(true).
			AlignHorizontal(lipgloss.Center),

		errorLabel: lipgloss.NewStyle().
			Foreground(interfaceGray).
			Background(red),

		errorText: lipgloss.NewStyle().
			Foreground(brightWhite).
			Background(red),

		errorButton: lipgloss.NewStyle().
			Foreground(red).
			Background(brightWhite).
			Padding(0, 1),

		errorButtonDisabled: lipgloss.NewStyle().
			Foreground(interfaceGray).
			Background(red).
			Padding(0, 1),

		errorButtonDisabledActive: lipgloss.NewStyle().
			Foreground(red).
			Background(interfaceGray).
			Padding(0, 1),
	}
}
