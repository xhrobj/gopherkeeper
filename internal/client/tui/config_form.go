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
	configTransport
	configGRPCAddress
	configCACertBrowse
	configSessionBrowse
	configCacheBrowse
	configSave
	configCancel
	configFocusCount
)

var configFocusOrder = [...]configFocus{
	configTransport,
	configAddress,
	configGRPCAddress,
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
	configLabelWidth      = 15
	configBrowseGap       = 1
	configBrowseLabel     = "<...>"
	configButtonGap       = 3
	configFirstControlRow = 2
	configControlRowStep  = 2
	configButtonRow       = 16
	configTransportGap    = 3
	configFileNotPassed   = "not provided (in-memory only; use --config for persistence)"
)

type configForm struct {
	fields       [4]textField
	grpcAddress  textField
	transport    config.Transport
	focus        configFocus
	errorMessage string
}

func newConfigForm(cfg config.Config) configForm {
	transport := cfg.Transport
	if transport != config.TransportGRPC {
		transport = config.TransportHTTPS
	}

	return configForm{
		fields: [4]textField{
			newUnicodeTextField(cfg.Address),
			newUnicodeTextField(cfg.CACertFile),
			newUnicodeTextField(cfg.SessionDir),
			newUnicodeTextField(cfg.CacheDir),
		},
		grpcAddress: newUnicodeTextField(cfg.GRPCAddress),
		transport:   transport,
		focus:       configCancel,
	}
}

func (form configForm) config() config.Config {
	return config.Config{
		Transport:   form.transport,
		Address:     form.fields[configAddress].value,
		GRPCAddress: form.grpcAddress.value,
		CACertFile:  form.fields[configCACertFile].value,
		SessionDir:  form.fields[configSessionDir].value,
		CacheDir:    form.fields[configCacheDir].value,
	}
}

func (form configForm) canSave() bool {
	if form.transport == config.TransportGRPC {
		return strings.TrimSpace(form.grpcAddress.value) != ""
	}

	return strings.TrimSpace(form.fields[configAddress].value) != ""
}

func (form *configForm) selectTransport(transport config.Transport) {
	if transport != config.TransportHTTPS && transport != config.TransportGRPC {
		return
	}

	form.transport = transport
	form.errorMessage = ""
}

func (form *configForm) moveTransport(step int) {
	if step < 0 {
		form.selectTransport(config.TransportHTTPS)
		return
	}

	form.selectTransport(config.TransportGRPC)
}

func (form *configForm) toggleTransport() {
	if form.transport == config.TransportGRPC {
		form.selectTransport(config.TransportHTTPS)
		return
	}

	form.selectTransport(config.TransportGRPC)
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
	switch form.focus {
	case configAddress, configCACertFile, configSessionDir, configCacheDir:
		return &form.fields[int(form.focus)]
	case configGRPCAddress:
		return &form.grpcAddress
	default:
		return nil
	}
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
	contentWidth    int
	inputWidth      int
	pathInputWidth  int
	transportBounds []layoutBounds
	fieldBounds     []layoutBounds
	fieldFocus      []configFocus
	browseBounds    []layoutBounds
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

	fieldFocus := []configFocus{
		configAddress,
		configGRPCAddress,
		configCACertFile,
		configSessionDir,
		configCacheDir,
	}
	fieldWidths := []int{inputWidth, inputWidth, pathInputWidth, pathInputWidth, pathInputWidth}
	fields := make([]layoutBounds, len(fieldFocus))
	for index, width := range fieldWidths {
		fields[index] = layoutBounds{
			x:      inputX,
			y:      configFirstControlRow + (index+1)*configControlRowStep,
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
			y:      configFirstControlRow + (index+3)*configControlRowStep,
			width:  browseWidth,
			height: 1,
		}
	}

	httpsLabel := "[ ] HTTPS"
	grpcLabel := "[ ] gRPC"
	transportBounds := []layoutBounds{
		{x: inputX, y: configFirstControlRow, width: lipgloss.Width(httpsLabel), height: 1},
		{x: inputX + lipgloss.Width(httpsLabel) + configTransportGap, y: configFirstControlRow, width: lipgloss.Width(grpcLabel), height: 1},
	}

	return configWindowLayout{
		contentWidth:    contentWidth,
		inputWidth:      inputWidth,
		pathInputWidth:  pathInputWidth,
		transportBounds: transportBounds,
		fieldBounds:     fields,
		fieldFocus:      fieldFocus,
		browseBounds:    browse,
	}
}

func renderConfigWindow(t theme, width int, form configForm, configFile string) string {
	layout := newConfigWindowLayout(t, width)
	contentWidth := layout.contentWidth
	inputWidth := layout.inputWidth
	pathInputWidth := layout.pathInputWidth

	rows := []string{
		renderConfigTransport(t, form.transport, form.focus == configTransport),
		t.windowBody.Width(contentWidth).Render(""),
		renderConfigField(
			t,
			"HTTPS address",
			form.transport == config.TransportHTTPS,
			t.configRequiredYellow,
			form.fields[configAddress],
			inputWidth,
			form.focus == configAddress,
		),
		t.windowBody.Width(contentWidth).Render(""),
		renderConfigField(
			t,
			"gRPC address",
			form.transport == config.TransportGRPC,
			t.configRequiredYellow,
			form.grpcAddress,
			inputWidth,
			form.focus == configGRPCAddress,
		),
		t.windowBody.Width(contentWidth).Render(""),
		renderConfigPathField(t, configPathFieldOptions{
			label:         "CA cert file",
			required:      true,
			requiredStyle: t.configRequiredYellow,
			field:         form.fields[configCACertFile],
			inputWidth:    pathInputWidth,
			fieldActive:   form.focus == configCACertFile,
			browseActive:  form.focus == configCACertBrowse,
		}),
		t.windowBody.Width(contentWidth).Render(""),
		renderConfigPathField(t, configPathFieldOptions{
			label:        "Session dir",
			field:        form.fields[configSessionDir],
			inputWidth:   pathInputWidth,
			fieldActive:  form.focus == configSessionDir,
			browseActive: form.focus == configSessionBrowse,
		}),
		t.windowBody.Width(contentWidth).Render(""),
		renderConfigPathField(t, configPathFieldOptions{
			label:        "Cache dir",
			field:        form.fields[configCacheDir],
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

func renderConfigTransport(t theme, transport config.Transport, active bool) string {
	labelPart := t.label.Width(configLabelWidth).Render("Transport")
	gap := t.windowBody.Width(2).Render("")

	https := "[ ] HTTPS"
	grpc := "[ ] gRPC"

	if transport == config.TransportGRPC {
		grpc = "[X] gRPC"
	} else {
		https = "[X] HTTPS"
	}

	httpsStyle := t.windowBody
	grpcStyle := t.windowBody

	if active {
		if transport == config.TransportGRPC {
			grpcStyle = t.buttonActive.Padding(0, 0)
		} else {
			httpsStyle = t.buttonActive.Padding(0, 0)
		}
	}

	optionsGap := t.windowBody.Width(configTransportGap).Render("")

	return labelPart + gap + httpsStyle.Render(https) + optionsGap + grpcStyle.Render(grpc)
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
