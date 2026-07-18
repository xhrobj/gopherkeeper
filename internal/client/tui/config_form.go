package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
)

type configFocus int

const (
	configAddress configFocus = iota
	configCACertFile
	configSessionFile
	configCacheDir
	configSave
	configCancel
	configFocusCount
)

const (
	configLabelWidth    = 14
	configButtonGap     = 3
	configFirstFieldRow = 2
	configFieldRowStep  = 2
	configButtonRow     = 12
	configFileNotPassed = "not provided (in-memory only; use --config for persistence)"
)

type configForm struct {
	fields       [4]textField
	focus        configFocus
	errorMessage string
}

func newConfigForm(cfg config.Config) configForm {
	return configForm{
		fields: [4]textField{
			newUnicodeTextField(cfg.Address),
			newUnicodeTextField(cfg.CACertFile),
			newUnicodeTextField(cfg.SessionFile),
			newUnicodeTextField(cfg.CacheDir),
		},
		focus: configCancel,
	}
}

func (form configForm) config() config.Config {
	return config.Config{
		Address:     form.fields[configAddress].value,
		CACertFile:  form.fields[configCACertFile].value,
		SessionFile: form.fields[configSessionFile].value,
		CacheDir:    form.fields[configCacheDir].value,
	}
}

func (form *configForm) move(step int) {
	count := int(configFocusCount)
	form.focus = configFocus((int(form.focus) + step + count) % count)
	if field := form.activeField(); field != nil {
		field.cursor = min(field.cursor, len([]rune(field.value)))
	}
}

func (form *configForm) setFocus(focus configFocus) {
	if focus < 0 || focus >= configFocusCount {
		return
	}

	form.focus = focus
	if field := form.activeField(); field != nil {
		field.moveCursorToEnd()
	}
}

func (form *configForm) activeField() *textField {
	if form.focus > configCacheDir {
		return nil
	}
	return &form.fields[int(form.focus)]
}

func (form *configForm) moveCursor(step int) {
	if field := form.activeField(); field != nil {
		field.moveCursor(step)
	}
}

func (form *configForm) moveCursorToStart() {
	if field := form.activeField(); field != nil {
		field.moveCursorToStart()
	}
}

func (form *configForm) moveCursorToEnd() {
	if field := form.activeField(); field != nil {
		field.moveCursorToEnd()
	}
}

func (form *configForm) insert(value string) {
	if field := form.activeField(); field != nil && field.insert(value) {
		form.errorMessage = ""
	}
}

func (form *configForm) insertKey(key string) bool {
	field := form.activeField()
	if field == nil || !field.insertKey(key) {
		return false
	}
	form.errorMessage = ""
	return true
}

func (form *configForm) backspace() {
	if field := form.activeField(); field != nil && field.backspace() {
		form.errorMessage = ""
	}
}

func (form *configForm) delete() {
	if field := form.activeField(); field != nil && field.delete() {
		form.errorMessage = ""
	}
}

func configWindowWidth(screenWidth int) int {
	return clamp(screenWidth-12, 58, 82)
}

func renderConfigWindow(t theme, width int, form configForm, configFile string) string {
	contentWidth := max(1, width-4)
	inputWidth := max(18, contentWidth-configLabelWidth-2)

	rows := []string{
		renderConfigField(t, "Address", true, t.configRequiredYellow, form.fields[configAddress], inputWidth, form.focus == configAddress),
		t.windowBody.Width(contentWidth).Render(""),
		renderConfigField(t, "CA cert file", true, t.configRequiredYellow, form.fields[configCACertFile], inputWidth, form.focus == configCACertFile),
		t.windowBody.Width(contentWidth).Render(""),
		renderConfigField(t, "Session file", false, lipgloss.Style{}, form.fields[configSessionFile], inputWidth, form.focus == configSessionFile),
		t.windowBody.Width(contentWidth).Render(""),
		renderConfigField(t, "Cache dir", false, lipgloss.Style{}, form.fields[configCacheDir], inputWidth, form.focus == configCacheDir),
		t.windowBody.Width(contentWidth).Render(""),
		renderConfigFileField(t, configFile, inputWidth),
		renderConfigStatus(t, contentWidth, form.errorMessage),
		renderConfigButtons(t, contentWidth, form.focus),
	}

	title := t.windowTitle.Width(width).Render("Config")
	body := t.windowBody.
		Width(width).
		Padding(1, 2).
		Render(strings.Join(rows, "\n"))

	return lipgloss.JoinVertical(lipgloss.Left, title, body)
}

func renderConfigField(
	t theme,
	label string,
	required bool,
	requiredStyle lipgloss.Style,
	field textField,
	inputWidth int,
	active bool,
) string {
	labelPart := renderConfigLabel(t, label, required, requiredStyle)
	gap := t.windowBody.Width(2).Render("")

	return labelPart + gap + renderTextField(t, field, inputWidth, active)
}

func renderConfigLabel(t theme, label string, required bool, requiredStyle lipgloss.Style) string {
	labelPart := t.label.Render(label)
	requiredPart := ""

	if required {
		requiredPart = t.label.Render(" ") + requiredStyle.Render("*")
	}

	padding := max(0, configLabelWidth-lipgloss.Width(labelPart)-lipgloss.Width(requiredPart))

	return labelPart + requiredPart + t.windowBody.Width(padding).Render("")
}

func renderConfigFileField(t theme, configFile string, inputWidth int) string {
	value := configFile
	showTail := true
	valueStyle := t.readOnly

	if strings.TrimSpace(value) == "" {
		value = configFileNotPassed
		showTail = false
		valueStyle = t.configMissing
	}

	labelPart := t.label.Width(configLabelWidth).Render("Config file")
	gap := t.windowBody.Width(2).Render("")

	return labelPart + gap + renderConfigReadOnlyInput(t, valueStyle, value, inputWidth, showTail)
}

func renderConfigReadOnlyInput(t theme, style lipgloss.Style, value string, width int, showTail bool) string {
	width = max(1, width)
	runes := []rune(value)

	if showTail {
		runes = runes[visibleSuffixStart(runes, width):]
	} else {
		runes = runes[:visiblePrefixEnd(runes, width)]
	}

	text := string(runes)
	padding := max(0, width-lipgloss.Width(text))

	return style.Render(text) + t.windowBody.Width(padding).Render("")
}

func renderConfigStatus(t theme, width int, message string) string {
	if message == "" {
		return t.windowBody.Width(width).Render("")
	}

	return t.controlsKey.Width(width).AlignHorizontal(lipgloss.Center).Render(message)
}

func renderConfigButtons(t theme, width int, focus configFocus) string {
	save := t.button.Render("< Save >")
	cancel := t.button.Render("< Cancel >")

	if focus == configSave {
		save = t.buttonActive.Render("< Save >")
	}

	if focus == configCancel {
		cancel = t.buttonActive.Render("< Cancel >")
	}

	buttonsWidth := lipgloss.Width(save) + configButtonGap + lipgloss.Width(cancel)
	left := max(0, (width-buttonsWidth)/2)
	right := max(0, width-buttonsWidth-left)

	return t.windowBody.Width(left).Render("") +
		save +
		t.windowBody.Width(configButtonGap).Render("") +
		cancel +
		t.windowBody.Width(right).Render("")
}
