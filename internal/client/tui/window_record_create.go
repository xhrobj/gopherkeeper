package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
	recordmodel "github.com/xhrobj/gopherkeeper/internal/model"
)

const (
	recordCreateLabelWidth  = 12
	recordCreateFieldGap    = 2
	recordBinaryBrowseLabel = "< Browse >"
	recordBinaryBrowseGap   = 2
)

type recordFormControlLayout struct {
	control recordFormControl
	row     int
	bounds  layoutBounds
}

type recordFormWindowLayout struct {
	controls  []recordFormControlLayout
	buttonRow int
	rowCount  int
}

func (layout recordFormWindowLayout) control(control recordFormControl) (recordFormControlLayout, bool) {
	for _, candidate := range layout.controls {
		if candidate.control == control {
			return candidate, true
		}
	}
	return recordFormControlLayout{}, false
}

type recordFormWindowMode struct {
	titlePrefix string
	submitLabel string
}

var (
	recordCreateWindowMode = recordFormWindowMode{
		titlePrefix: "New ",
		submitLabel: "< Create >",
	}
	recordEditWindowMode = recordFormWindowMode{
		titlePrefix: "Edit ",
		submitLabel: "< Save >",
	}
)

func recordCreateWindowWidth(screenWidth int) int {
	return clamp(screenWidth-12, 62, 88)
}

func recordFormLayout(form *recordForm, contentWidth int) recordFormWindowLayout {
	inputWidth := max(20, contentWidth-recordCreateLabelWidth-recordCreateFieldGap)
	inputX := recordCreateLabelWidth + recordCreateFieldGap
	row := 0

	if form.editing {
		row += len(recordViewMetadataLines(form.recordMetadata, contentWidth)) + 1
	}

	controls := form.controls()
	layout := recordFormWindowLayout{
		controls: make([]recordFormControlLayout, 0, len(controls)),
	}

	for _, control := range controls {
		switch control {
		case recordFormSubmit, recordFormReveal, recordFormCancel:
			continue
		}

		height := 1
		if control == recordFormText {
			height = recordTextAreaHeight
		}
		width := inputWidth
		if field := form.field(control); field != nil {
			width = recordFormFieldWidth(*field, inputWidth)
		}
		layout.controls = append(layout.controls, recordFormControlLayout{
			control: control,
			row:     row,
			bounds: layoutBounds{
				x:      inputX,
				y:      row,
				width:  width,
				height: height,
			},
		})
		row += height + 1
	}

	layout.buttonRow = row
	layout.rowCount = row + 1

	return layout
}

func renderRecordCreateWindow(t theme, width int, form recordForm, pending, blocked bool, spinnerFrame string) string {
	return renderRecordFormWindow(t, width, form, pending, blocked, spinnerFrame, recordCreateWindowMode)
}

func renderRecordEditWindow(t theme, width int, form recordForm, pending, blocked bool, spinnerFrame string) string {
	return renderRecordFormWindow(t, width, form, pending, blocked, spinnerFrame, recordEditWindowMode)
}

func renderRecordEditLoadingWindow(t theme, width int, form recordForm, spinnerFrame string) string {
	return renderRecordFormWindow(t, width, form, true, true, spinnerFrame, recordEditWindowMode)
}

func renderRecordFormWindow(
	t theme,
	width int,
	form recordForm,
	pending bool,
	blocked bool,
	spinnerFrame string,
	mode recordFormWindowMode,
) string {
	contentWidth := max(1, width-4)
	inputWidth := max(20, contentWidth-recordCreateLabelWidth-recordCreateFieldGap)
	layout := recordFormLayout(&form, contentWidth)
	rows := make([]string, layout.rowCount)

	for index := range rows {
		rows[index] = t.recordFormBody.Width(contentWidth).Render("")
	}

	if form.editing {
		for index, line := range recordViewMetadataLines(form.recordMetadata, contentWidth) {
			rows[index] = renderRecordViewLineWithStyles(t.recordFormBody, t.recordFormInfo, contentWidth, line)
		}
	}

	setField := func(control recordFormControl, label string, field textField) {
		controlLayout, ok := layout.control(control)
		if !ok {
			return
		}
		rows[controlLayout.row] = renderRecordFormField(
			t,
			label,
			field,
			controlLayout.bounds.width,
			contentWidth,
			form.activeControl() == control && !blocked,
			recordFormControlRequired(form, control),
		)
	}

	setField(recordFormTitle, "Title", form.title)

	switch form.recordType {
	case recordmodel.RecordTypeText:
		textLayout, _ := layout.control(recordFormText)
		row := textLayout.row
		areaRows := renderRecordFormTextAreaRows(
			t,
			form.text,
			inputWidth,
			form.textAreaActive() && !blocked,
		)
		for offset, areaRow := range areaRows {
			rows[row+offset] = areaRow
		}
		setField(recordFormMetadata, "Notes", form.metadata)
	case recordmodel.RecordTypeCredentials:
		setField(recordFormLogin, "Login", form.login)
		password := form.password
		password.masked = !form.revealed
		setField(recordFormPassword, "Password", password)
		setField(recordFormURL, "URL", form.url)
		setField(recordFormMetadata, "Notes", form.metadata)
	case recordmodel.RecordTypeCard:
		number := form.number
		number.masked = !form.revealed
		cvv := form.cvv
		cvv.masked = !form.revealed
		setField(recordFormNumber, "Number", number)
		setField(recordFormCardholder, "Cardholder", form.cardholder)
		setField(recordFormExpiryMonth, "Expiry month", form.expiryMonth)
		setField(recordFormExpiryYear, "Expiry year", form.expiryYear)
		setField(recordFormCVV, "CVV", cvv)
		setField(recordFormMetadata, "Notes", form.metadata)
	case recordmodel.RecordTypeBinary:
		filePathLabel := "File path"
		if form.editing {
			filePathLabel = "Replace file"
		}
		filePathLayout, _ := layout.control(recordFormFilePath)
		filePathRow := filePathLayout.row
		rows[filePathRow] = renderRecordBinaryFilePath(
			t,
			filePathLabel,
			form.filePath.value,
			contentWidth,
			form.activeControl() == recordFormFilePath && !blocked,
			recordFormControlRequired(form, recordFormFilePath),
		)
		setField(recordFormMetadata, "Notes", form.metadata)
	}

	rows[layout.buttonRow] = renderRecordFormButtons(t, contentWidth, form, blocked, mode.submitLabel)

	title := mode.titlePrefix + recordCreateTypeTitle(form.recordType) + " Record"
	titleLine := renderWindowTitle(t.windowTitle, width, title, spinnerFrame, pending)
	body := t.recordFormBody.
		Width(width).
		Padding(1, 2).
		Render(strings.Join(rows, "\n"))

	return lipgloss.JoinVertical(lipgloss.Left, titleLine, body)
}

func recordFormControlRequired(form recordForm, control recordFormControl) bool {
	switch control {
	case recordFormTitle, recordFormText, recordFormLogin, recordFormPassword, recordFormNumber:
		return true
	case recordFormFilePath:
		return !form.editing
	default:
		return false
	}
}

func renderRecordFormLabel(t theme, label string, width int, required bool) string {
	if !required {
		return t.recordFormLabel.Width(width).Render(label)
	}

	labelPart := t.recordFormLabel.Render(label + " ")
	star := t.recordFormRequired.Render("*")
	used := lipgloss.Width(labelPart) + lipgloss.Width(star)

	return labelPart + star + t.recordFormLabel.Width(max(0, width-used)).Render("")
}

func recordFormFieldWidth(field textField, available int) int {
	available = max(1, available)

	if field.maxLength <= 0 || field.maxLength >= available {
		return available
	}

	return clamp(field.maxLength, 1, available)
}

func renderRecordFormField(
	t theme,
	label string,
	field textField,
	fieldWidth int,
	rowWidth int,
	active bool,
	required bool,
) string {
	labelPart := renderRecordFormLabel(t, label, recordCreateLabelWidth, required)
	gap := t.recordFormBody.Width(recordCreateFieldGap).Render("")
	input := renderTextField(t, field, fieldWidth, active)
	used := recordCreateLabelWidth + recordCreateFieldGap + fieldWidth

	return labelPart + gap + input + t.recordFormBody.Width(max(0, rowWidth-used)).Render("")
}

func renderRecordBinaryFilePath(
	t theme,
	label string,
	value string,
	rowWidth int,
	active bool,
	required bool,
) string {
	labelPart := renderRecordFormLabel(t, label, recordCreateLabelWidth, required)
	gap := t.recordFormBody.Width(recordCreateFieldGap).Render("")

	browseStyle := t.recordFormButton
	if active {
		browseStyle = t.recordFormButtonActive
	}
	browse := browseStyle.Render(recordBinaryBrowseLabel)
	browseWidth := lipgloss.Width(browse)
	pathWidth := max(1, rowWidth-recordCreateLabelWidth-recordCreateFieldGap-recordBinaryBrowseGap-browseWidth)

	pathStyle := t.recordFormReadOnly
	showTail := true

	if strings.TrimSpace(value) == "" {
		value = "No file selected"
		pathStyle = t.recordFormMissing
		showTail = false
	}

	formTheme := t
	formTheme.windowBody = t.recordFormBody
	path := renderConfigReadOnlyInput(formTheme, pathStyle, value, pathWidth, showTail)

	return labelPart + gap + path + t.recordFormBody.Width(recordBinaryBrowseGap).Render("") + browse
}

func renderRecordFormTextAreaRows(t theme, area textArea, width int, active bool) []string {
	inputRows := strings.Split(renderTextArea(t, area, width, recordTextAreaHeight, active), "\n")
	rows := make([]string, 0, recordTextAreaHeight)

	for index := 0; index < recordTextAreaHeight; index++ {
		label := ""
		required := false

		if index == 0 {
			label = "Text"
			required = true
		}

		inputRow := t.input.Width(width).Render("")
		if index < len(inputRows) {
			inputRow = inputRows[index]
		}

		rows = append(rows,
			renderRecordFormLabel(t, label, recordCreateLabelWidth, required)+
				t.recordFormBody.Width(recordCreateFieldGap).Render("")+
				inputRow,
		)
	}

	return rows
}

type recordFormButtonDefinition struct {
	control recordFormControl
	label   string
}

func recordFormButtonDefinitions(form recordForm, submitLabel string) []recordFormButtonDefinition {
	definitions := []recordFormButtonDefinition{{control: recordFormSubmit, label: submitLabel}}

	if form.recordType == recordmodel.RecordTypeCredentials || form.recordType == recordmodel.RecordTypeCard {
		label := "< Reveal >"
		if form.revealed {
			label = "< Hide >"
		}
		definitions = append(definitions, recordFormButtonDefinition{control: recordFormReveal, label: label})
	}

	return append(definitions, recordFormButtonDefinition{control: recordFormCancel, label: "< Cancel >"})
}

func recordFormButtonsLayout(
	t theme,
	width int,
	form recordForm,
	blocked bool,
	submitLabel string,
) buttonRowLayout {
	definitions := recordFormButtonDefinitions(form, submitLabel)
	buttons := make([]formButton, 0, len(definitions))

	for _, definition := range definitions {
		buttons = append(buttons, formButton{
			label:    definition.label,
			active:   form.activeControl() == definition.control,
			disabled: blocked || (definition.control == recordFormSubmit && !form.canSubmit()),
		})
	}

	formTheme := t
	formTheme.windowBody = t.recordFormBody
	formTheme.button = t.recordFormButton
	formTheme.buttonActive = t.recordFormButtonActive
	formTheme.buttonDisabled = t.recordFormButtonDisabled
	formTheme.buttonDisabledActive = t.recordFormButtonDisabledActive

	return formButtonsLayout(formTheme, width, 3, buttons)
}

func renderRecordFormButtons(
	t theme,
	width int,
	form recordForm,
	blocked bool,
	submitLabel string,
) string {
	return recordFormButtonsLayout(t, width, form, blocked, submitLabel).content
}

func recordCreateTypeTitle(recordType recordmodel.RecordType) string {
	value := string(recordType)
	if value == "" {
		return ""
	}

	return strings.ToUpper(value[:1]) + value[1:]
}
