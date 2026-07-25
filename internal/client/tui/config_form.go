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
	configSessionDir
	configCacheDir
	configCACertBrowse
	configSessionBrowse
	configCacheBrowse
	configSave
	configCancel
	configFocusCount
)

var configFocusOrder = [...]configFocus{
	configAddress,
	configCACertFile,
	configCACertBrowse,
	configSessionDir,
	configSessionBrowse,
	configCacheDir,
	configCacheBrowse,
	configSave,
	configCancel,
}

const (
	configLabelWidth    = 14
	configBrowseGap     = 1
	configBrowseLabel   = "<...>"
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
			newUnicodeTextField(cfg.SessionDir),
			newUnicodeTextField(cfg.CacheDir),
		},
		focus: configCancel,
	}
}

func (form configForm) config() config.Config {
	return config.Config{
		Address:    form.fields[configAddress].value,
		CACertFile: form.fields[configCACertFile].value,
		SessionDir: form.fields[configSessionDir].value,
		CacheDir:   form.fields[configCacheDir].value,
	}
}

func (form configForm) canSave() bool {
	return strings.TrimSpace(form.fields[configAddress].value) != ""
}

func (form *configForm) move(step int) {
	current := 0
	for index, focus := range configFocusOrder {
		if focus == form.focus {
			current = index
			break
		}
	}

	count := len(configFocusOrder)
	for range count {
		current = (current + step + count) % count
		candidate := configFocusOrder[current]
		if candidate == configSave && !form.canSave() {
			continue
		}
		form.setFocus(candidate)
		return
	}
}

func (form *configForm) setFocus(focus configFocus) {
	if focus < 0 || focus >= configFocusCount || focus == configSave && !form.canSave() {
		return
	}

	form.focus = focus
	if field := form.activeField(); field != nil {
		field.moveCursorToEnd()
	}
}

func (form *configForm) activeField() *textField {
	index, ok := configFieldIndex(form.focus)
	if !ok {
		return nil
	}

	return &form.fields[index]
}

func configFieldIndex(focus configFocus) (int, bool) {
	if focus < configAddress || focus > configCacheDir {
		return 0, false
	}

	return int(focus), true
}

func configFieldFocus(index int) configFocus {
	if index < int(configAddress) || index > int(configCacheDir) {
		return configAddress
	}

	return configFocus(index)
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

func configInputWidths(t theme, contentWidth int) (int, int) {
	inputWidth := max(18, contentWidth-configLabelWidth-2)
	browseWidth := lipgloss.Width(t.button.Render(configBrowseLabel))
	pathInputWidth := max(10, inputWidth-configBrowseGap-browseWidth)

	return inputWidth, pathInputWidth
}

type configWindowLayout struct {
	contentWidth   int
	inputWidth     int
	pathInputWidth int
	fieldBounds    []layoutBounds
	browseBounds   []layoutBounds
}

type configPathFieldOptions struct {
	label         string
	required      bool
	requiredStyle lipgloss.Style
	field         textField
	inputWidth    int
	fieldActive   bool
	browseActive  bool
}

func newConfigWindowLayout(t theme, windowWidth int) configWindowLayout {
	contentWidth := max(1, windowWidth-4)
	inputWidth, pathInputWidth := configInputWidths(t, contentWidth)
	inputX := 2 + configLabelWidth + 2
	widths := []int{inputWidth, pathInputWidth, pathInputWidth, pathInputWidth}
	fields := make([]layoutBounds, len(widths))

	for index, width := range widths {
		fields[index] = layoutBounds{
			x:      inputX,
			y:      configFirstFieldRow + index*configFieldRowStep,
			width:  width,
			height: 1,
		}
	}

	browseWidth := lipgloss.Width(t.button.Render(configBrowseLabel))
	browseX := inputX + pathInputWidth + configBrowseGap
	browse := make([]layoutBounds, 3)

	for index := range browse {
		browse[index] = layoutBounds{
			x:      browseX,
			y:      configFirstFieldRow + (index+1)*configFieldRowStep,
			width:  browseWidth,
			height: 1,
		}
	}

	return configWindowLayout{
		contentWidth:   contentWidth,
		inputWidth:     inputWidth,
		pathInputWidth: pathInputWidth,
		fieldBounds:    fields,
		browseBounds:   browse,
	}
}

func renderConfigWindow(t theme, width int, form configForm, configFile string) string {
	layout := newConfigWindowLayout(t, width)
	contentWidth := layout.contentWidth
	inputWidth := layout.inputWidth
	pathInputWidth := layout.pathInputWidth

	rows := []string{
		renderConfigField(t, "Address", true, t.configRequiredYellow, form.fields[0], inputWidth, form.focus == configAddress),
		t.windowBody.Width(contentWidth).Render(""),
		renderConfigPathField(t, configPathFieldOptions{
			label:         "CA cert file",
			required:      true,
			requiredStyle: t.configRequiredYellow,
			field:         form.fields[1],
			inputWidth:    pathInputWidth,
			fieldActive:   form.focus == configCACertFile,
			browseActive:  form.focus == configCACertBrowse,
		}),
		t.windowBody.Width(contentWidth).Render(""),
		renderConfigPathField(t, configPathFieldOptions{
			label:        "Session dir",
			field:        form.fields[2],
			inputWidth:   pathInputWidth,
			fieldActive:  form.focus == configSessionDir,
			browseActive: form.focus == configSessionBrowse,
		}),
		t.windowBody.Width(contentWidth).Render(""),
		renderConfigPathField(t, configPathFieldOptions{
			label:        "Cache dir",
			field:        form.fields[3],
			inputWidth:   pathInputWidth,
			fieldActive:  form.focus == configCacheDir,
			browseActive: form.focus == configCacheBrowse,
		}),
		t.windowBody.Width(contentWidth).Render(""),
		renderConfigFileField(t, configFile, inputWidth),
		renderConfigStatus(t, contentWidth, form.errorMessage),
		renderConfigButtons(t, contentWidth, form.focus, form.canSave()),
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

func renderConfigPathField(t theme, options configPathFieldOptions) string {
	browse := t.button.Render(configBrowseLabel)
	if options.browseActive {
		browse = t.buttonActive.Render(configBrowseLabel)
	}

	return renderConfigField(
		t,
		options.label,
		options.required,
		options.requiredStyle,
		options.field,
		options.inputWidth,
		options.fieldActive,
	) +
		t.windowBody.Width(configBrowseGap).Render("") +
		browse
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

func configButtonsLayout(t theme, width int, focus configFocus, canSave bool) buttonRowLayout {
	saveStyle := t.button
	cancelStyle := t.button
	if !canSave {
		saveStyle = t.buttonDisabled
	} else if focus == configSave {
		saveStyle = t.buttonActive
	}
	if focus == configCancel {
		cancelStyle = t.buttonActive
	}

	return centeredButtonRowLayout(t.windowBody, width, configButtonGap, []styledButton{
		{label: "< Save >", style: saveStyle},
		{label: "< Cancel >", style: cancelStyle},
	})
}

func renderConfigButtons(t theme, width int, focus configFocus, canSave bool) string {
	return configButtonsLayout(t, width, focus, canSave).content
}
