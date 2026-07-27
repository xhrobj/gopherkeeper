package tui

import (
	"errors"
	"strconv"
	"strings"

	recordmodel "github.com/xhrobj/gopherkeeper/internal/model"
)

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
