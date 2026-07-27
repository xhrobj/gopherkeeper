package tui

import (
	"fmt"

	recordmodel "github.com/xhrobj/gopherkeeper/internal/model"
)

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
