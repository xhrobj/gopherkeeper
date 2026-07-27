package tui

import (
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
