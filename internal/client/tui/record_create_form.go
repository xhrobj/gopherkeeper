package tui

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	recordmodel "github.com/xhrobj/gopherkeeper/internal/model"
)

type recordTypePicker struct {
	selected int
}

var recordCreateTypes = []recordmodel.RecordType{
	recordmodel.RecordTypeCredentials,
	recordmodel.RecordTypeCard,
	recordmodel.RecordTypeText,
	recordmodel.RecordTypeBinary,
}

func (picker *recordTypePicker) move(step int) {
	count := len(recordCreateTypes) + 1
	picker.selected = (picker.selected + step + count) % count
}

func (picker recordTypePicker) selectedType() (recordmodel.RecordType, bool) {
	if picker.selected < 0 || picker.selected >= len(recordCreateTypes) {
		return "", false
	}

	return recordCreateTypes[picker.selected], true
}

type recordFormControl int

const (
	recordFormTitle recordFormControl = iota
	recordFormText
	recordFormLogin
	recordFormPassword
	recordFormURL
	recordFormNumber
	recordFormCardholder
	recordFormExpiryMonth
	recordFormExpiryYear
	recordFormCVV
	recordFormFilePath
	recordFormMetadata
	recordFormSubmit
	recordFormReveal
	recordFormCancel
)

type recordForm struct {
	editing        bool
	recordType     recordmodel.RecordType
	recordMetadata recordmodel.RecordMetadata
	title          textField
	text           textArea
	textOriginal   string
	textDirty      bool
	login          textField
	password       textField
	url            textField
	number         textField
	cardholder     textField
	expiryMonth    textField
	expiryYear     textField
	cvv            textField
	filename       string
	filePath       textField
	metadata       textField
	binaryExisting bool
	focus          int
	revealed       bool
}

func newRecordCreateForm(recordType recordmodel.RecordType) recordForm {
	return recordForm{
		recordType:  recordType,
		title:       newUnicodeTextField(""),
		text:        newTextArea(""),
		login:       newUnicodeTextField(""),
		password:    newUnicodeTextField(""),
		url:         newUnicodeTextField(""),
		number:      newDigitsTextFieldWithLimit("", recordmodel.CardNumberMaxSize),
		cardholder:  newUnicodeTextFieldWithLimit("", recordmodel.CardholderMaxSize),
		expiryMonth: newDigitsTextFieldWithLimit("", 2),
		expiryYear:  newDigitsTextFieldWithLimit("", 2),
		cvv:         newDigitsTextFieldWithLimit("", recordmodel.CardCVVSize),
		filePath:    newUnicodeTextField(""),
		metadata:    newUnicodeTextField(""),
	}
}

func newRecordEditForm(record recordmodel.Record) recordForm {
	form := newRecordCreateForm(record.Metadata.Type)
	form.editing = true
	form.recordMetadata = record.Metadata
	form.title.setValue(record.Metadata.Title)
	populateRecordEditForm(&form, record.Payload)
	return form
}

func populateRecordEditForm(form *recordForm, recordPayload recordmodel.RecordPayload) {
	switch payload := recordPayload.(type) {
	case *recordmodel.TextPayload:
		populateTextRecordEditForm(form, payload)
	case *recordmodel.CredentialsPayload:
		populateCredentialsRecordEditForm(form, payload)
	case *recordmodel.CardPayload:
		populateCardRecordEditForm(form, payload)
	case *recordmodel.BinaryPayload:
		populateBinaryRecordEditForm(form, payload)
	}
}

func populateTextRecordEditForm(form *recordForm, payload *recordmodel.TextPayload) {
	if payload == nil {
		return
	}
	form.text.setValue(payload.Text)
	form.textOriginal = payload.Text
	form.metadata.setValue(payload.Metadata)
}

func populateCredentialsRecordEditForm(form *recordForm, payload *recordmodel.CredentialsPayload) {
	if payload == nil {
		return
	}
	form.login.setValue(payload.Login)
	form.password.setValue(payload.Password)
	form.url.setValue(payload.URL)
	form.metadata.setValue(payload.Metadata)
}

func populateCardRecordEditForm(form *recordForm, payload *recordmodel.CardPayload) {
	if payload == nil {
		return
	}
	form.number.setValue(payload.Number)
	form.cardholder.setValue(payload.Cardholder)
	populateCardExpiry(form, payload.ExpiryMonth, payload.ExpiryYear)
	form.cvv.setValue(payload.CVV)
	form.metadata.setValue(payload.Metadata)
}

func populateCardExpiry(form *recordForm, month, year *int) {
	if month == nil || year == nil {
		return
	}
	form.expiryMonth.setValue(fmt.Sprintf("%02d", *month))
	form.expiryYear.setValue(fmt.Sprintf("%02d", *year))
}

func populateBinaryRecordEditForm(form *recordForm, payload *recordmodel.BinaryPayload) {
	if payload == nil {
		return
	}
	form.filename = payload.Filename
	form.metadata.setValue(payload.Metadata)
	form.binaryExisting = payload.Data != nil
}

func (form recordForm) controls() []recordFormControl {
	switch form.recordType {
	case recordmodel.RecordTypeText:
		return []recordFormControl{
			recordFormTitle,
			recordFormText,
			recordFormMetadata,
			recordFormSubmit,
			recordFormCancel,
		}
	case recordmodel.RecordTypeCredentials:
		return []recordFormControl{
			recordFormTitle,
			recordFormLogin,
			recordFormPassword,
			recordFormURL,
			recordFormMetadata,
			recordFormSubmit,
			recordFormReveal,
			recordFormCancel,
		}
	case recordmodel.RecordTypeCard:
		return []recordFormControl{
			recordFormTitle,
			recordFormNumber,
			recordFormCardholder,
			recordFormExpiryMonth,
			recordFormExpiryYear,
			recordFormCVV,
			recordFormMetadata,
			recordFormSubmit,
			recordFormReveal,
			recordFormCancel,
		}
	case recordmodel.RecordTypeBinary:
		return []recordFormControl{
			recordFormTitle,
			recordFormFilePath,
			recordFormMetadata,
			recordFormSubmit,
			recordFormCancel,
		}
	default:
		return []recordFormControl{recordFormCancel}
	}
}

func (form recordForm) activeControl() recordFormControl {
	controls := form.controls()
	if len(controls) == 0 {
		return recordFormCancel
	}

	return controls[clamp(form.focus, 0, len(controls)-1)]
}

func (form *recordForm) move(step int) {
	controls := form.controls()
	if len(controls) == 0 {
		form.focus = 0
		return
	}
	form.focus = (form.focus + step + len(controls)) % len(controls)
	form.moveCursorToEnd()
}

func (form *recordForm) setFocus(control recordFormControl) {
	for index, candidate := range form.controls() {
		if candidate == control {
			form.focus = index
			form.moveCursorToEnd()
			return
		}
	}
}

func (form *recordForm) field(control recordFormControl) *textField {
	switch control {
	case recordFormTitle:
		return &form.title
	case recordFormLogin:
		return &form.login
	case recordFormPassword:
		return &form.password
	case recordFormURL:
		return &form.url
	case recordFormNumber:
		return &form.number
	case recordFormCardholder:
		return &form.cardholder
	case recordFormExpiryMonth:
		return &form.expiryMonth
	case recordFormExpiryYear:
		return &form.expiryYear
	case recordFormCVV:
		return &form.cvv
	case recordFormMetadata:
		return &form.metadata
	default:
		return nil
	}
}

func (form *recordForm) activeField() *textField {
	return form.field(form.activeControl())
}

func (form recordForm) textAreaActive() bool {
	return form.activeControl() == recordFormText
}

func (form *recordForm) insert(value string) {
	if form.textAreaActive() {
		form.mutateText(func() { form.text.insert(value) })
		return
	}

	if field := form.activeField(); field != nil {
		field.insert(value)
	}
}

func (form *recordForm) insertKey(key string) bool {
	if form.textAreaActive() {
		changed := false
		form.mutateText(func() { changed = form.text.insertKey(key) })
		return changed
	}

	if field := form.activeField(); field != nil {
		return field.insertKey(key)
	}

	return false
}

func (form *recordForm) backspace() {
	if form.textAreaActive() {
		form.mutateText(func() { form.text.backspace() })
		return
	}

	if field := form.activeField(); field != nil {
		field.backspace()
	}
}

func (form *recordForm) delete() {
	if form.textAreaActive() {
		form.mutateText(func() { form.text.delete() })
		return
	}

	if field := form.activeField(); field != nil {
		field.delete()
	}
}

func (form *recordForm) insertNewline() {
	if !form.textAreaActive() {
		return
	}

	form.mutateText(func() { form.text.insertNewline() })
	form.text.ensureVisible(recordTextAreaHeight)
}

func (form *recordForm) mutateText(change func()) {
	before := form.text.value
	change()

	if form.editing && form.text.value != before {
		form.textDirty = true
	}
}

func recordFormControlIsButton(control recordFormControl) bool {
	switch control {
	case recordFormFilePath, recordFormSubmit, recordFormReveal, recordFormCancel:
		return true
	default:
		return false
	}
}

func (form *recordForm) moveCursor(step int) {
	if form.textAreaActive() {
		form.text.moveCursor(step)
		form.text.ensureVisible(recordTextAreaHeight)
		return
	}

	if field := form.activeField(); field != nil {
		field.moveCursor(step)
	}
}

func (form *recordForm) moveLine(step int) {
	if form.textAreaActive() {
		form.text.moveLine(step)
		form.text.ensureVisible(recordTextAreaHeight)
		return
	}

	form.move(step)
}

func (form *recordForm) movePage(step int) {
	if form.textAreaActive() {
		form.text.movePage(step, recordTextAreaHeight)
		form.text.ensureVisible(recordTextAreaHeight)
	}
}

func (form *recordForm) moveCursorToStart() {
	if form.textAreaActive() {
		form.text.moveCursorToLineStart()
		return
	}

	if field := form.activeField(); field != nil {
		field.moveCursorToStart()
	}
}

func (form *recordForm) moveCursorToEnd() {
	if form.textAreaActive() {
		form.text.moveCursorToLineEnd()
		form.text.ensureVisible(recordTextAreaHeight)
		return
	}

	if field := form.activeField(); field != nil {
		field.moveCursorToEnd()
	}
}

func (form recordForm) canSubmit() bool {
	if strings.TrimSpace(form.title.value) == "" {
		return false
	}

	switch form.recordType {
	case recordmodel.RecordTypeText:
		return form.text.value != ""
	case recordmodel.RecordTypeCredentials:
		return strings.TrimSpace(form.login.value) != "" && strings.TrimSpace(form.password.value) != ""
	case recordmodel.RecordTypeCard:
		if len(form.number.value) < recordmodel.CardNumberMinSize ||
			len(form.number.value) > recordmodel.CardNumberMaxSize ||
			!recordCreateASCIIDigits(form.number.value) {
			return false
		}
		if form.cvv.value != "" && len(form.cvv.value) != recordmodel.CardCVVSize {
			return false
		}
		_, _, err := parseRecordCardExpiry(
			strings.TrimSpace(form.expiryMonth.value),
			strings.TrimSpace(form.expiryYear.value),
		)
		return err == nil
	case recordmodel.RecordTypeBinary:
		if form.editing {
			return strings.TrimSpace(form.filePath.value) != "" || form.binaryExisting
		}
		return strings.TrimSpace(form.filePath.value) != ""
	default:
		return false
	}
}

func (form *recordForm) toggleReveal() {
	form.revealed = !form.revealed
}

type recordFormInput struct {
	recordType     recordmodel.RecordType
	title          string
	text           string
	login          string
	password       string
	url            string
	number         string
	cardholder     string
	expiryMonth    string
	expiryYear     string
	cvv            string
	filename       string
	filePath       string
	metadata       string
	binaryExisting bool
	editing        bool
}

func recordFormInputFrom(form recordForm) recordFormInput {
	text := form.text.value
	if form.editing && !form.textDirty {
		text = form.textOriginal
	}

	return recordFormInput{
		recordType:     form.recordType,
		title:          strings.TrimSpace(form.title.value),
		text:           text,
		login:          form.login.value,
		password:       form.password.value,
		url:            form.url.value,
		number:         form.number.value,
		cardholder:     form.cardholder.value,
		expiryMonth:    normalizeRecordCardExpiryPart(form.expiryMonth.value),
		expiryYear:     normalizeRecordCardExpiryPart(form.expiryYear.value),
		cvv:            form.cvv.value,
		filename:       form.filename,
		filePath:       strings.TrimSpace(form.filePath.value),
		metadata:       form.metadata.value,
		binaryExisting: form.binaryExisting,
		editing:        form.editing,
	}
}

func (input recordFormInput) buildPayload(
	readBinary binaryFileReader,
	existing recordmodel.RecordPayload,
) (recordmodel.RecordPayload, error) {
	if err := recordmodel.ValidateRecordTitle(input.title); err != nil {
		return nil, err
	}

	payload, err := input.payloadForType(readBinary, existing)
	if err != nil {
		return nil, err
	}
	if err := payload.Validate(); err != nil {
		return nil, err
	}

	return payload, nil
}

func (input recordFormInput) payloadForType(
	readBinary binaryFileReader,
	existing recordmodel.RecordPayload,
) (recordmodel.RecordPayload, error) {
	switch input.recordType {
	case recordmodel.RecordTypeText:
		return &recordmodel.TextPayload{Text: input.text, Metadata: input.metadata}, nil
	case recordmodel.RecordTypeCredentials:
		return &recordmodel.CredentialsPayload{
			Login: input.login, Password: input.password, URL: input.url, Metadata: input.metadata,
		}, nil
	case recordmodel.RecordTypeCard:
		return input.cardPayload()
	case recordmodel.RecordTypeBinary:
		return input.binaryPayload(readBinary, existing)
	default:
		return nil, recordmodel.ErrRecordTypeUnsupported
	}
}

func (input recordFormInput) cardPayload() (recordmodel.RecordPayload, error) {
	month, year, err := parseRecordCardExpiry(input.expiryMonth, input.expiryYear)
	if err != nil {
		return nil, err
	}

	return &recordmodel.CardPayload{
		Number: input.number, Cardholder: input.cardholder,
		ExpiryMonth: month, ExpiryYear: year, CVV: input.cvv, Metadata: input.metadata,
	}, nil
}

func (input recordFormInput) binaryPayload(
	readBinary binaryFileReader,
	existing recordmodel.RecordPayload,
) (recordmodel.RecordPayload, error) {
	if err := input.validateBinarySelection(); err != nil {
		return nil, err
	}

	filename, data, err := input.binaryContent(readBinary, existing)
	if err != nil {
		return nil, err
	}

	return &recordmodel.BinaryPayload{
		Filename: filename, Data: data, Metadata: input.metadata,
	}, nil
}

func (input recordFormInput) validateBinarySelection() error {
	if input.editing {
		if input.filePath == "" && !input.binaryExisting {
			return errors.New("binary payload is required")
		}
		return nil
	}

	if input.filePath == "" {
		return errors.New("binary file path is required")
	}

	return nil
}

func (input recordFormInput) binaryContent(
	readBinary binaryFileReader,
	existing recordmodel.RecordPayload,
) (string, []byte, error) {
	if input.filePath != "" {
		return readBinary(input.filePath)
	}

	filename := input.filename
	if existingPayload, ok := existing.(*recordmodel.BinaryPayload); ok && existingPayload != nil {
		return filename, cloneBinaryData(existingPayload.Data), nil
	}

	return filename, nil, nil
}

func cloneBinaryData(data []byte) []byte {
	if data == nil {
		return nil
	}

	cloned := make([]byte, len(data))
	copy(cloned, data)

	return cloned
}

func normalizeRecordCardExpiryPart(value string) string {
	value = strings.TrimSpace(value)
	if len(value) == 1 {
		return "0" + value
	}

	return value
}

func parseRecordCardExpiry(monthValue, yearValue string) (*int, *int, error) {
	if monthValue == "" && yearValue == "" {
		return nil, nil, nil
	}

	if monthValue == "" || yearValue == "" {
		return nil, nil, errors.New("expiry month and year must be provided together")
	}

	if len(monthValue) < 1 || len(monthValue) > 2 || !recordCreateASCIIDigits(monthValue) {
		return nil, nil, errors.New("expiry month must use M or MM format")
	}

	if len(yearValue) < 1 || len(yearValue) > 2 || !recordCreateASCIIDigits(yearValue) {
		return nil, nil, errors.New("expiry year must use Y or YY format")
	}

	month, err := strconv.Atoi(monthValue)
	if err != nil || month < 1 || month > 12 {
		return nil, nil, errors.New("expiry month must be between 01 and 12")
	}

	year, err := strconv.Atoi(yearValue)
	if err != nil || year < 0 || year > recordmodel.CardExpiryYearMax {
		return nil, nil, errors.New("expiry year must be between 00 and 99")
	}

	return &month, &year, nil
}

func recordCreateASCIIDigits(value string) bool {
	for _, symbol := range value {
		if symbol < '0' || symbol > '9' {
			return false
		}
	}

	return true
}
