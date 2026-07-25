package tui

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	recordmodel "github.com/xhrobj/gopherkeeper/internal/model"
)

func TestParseRecordCardExpiry(t *testing.T) {
	tests := []struct {
		name      string
		month     string
		year      string
		wantMonth int
		wantYear  int
		wantEmpty bool
		wantErr   bool
	}{
		{name: "empty", wantEmpty: true},
		{name: "valid", month: "12", year: "30", wantMonth: 12, wantYear: 30},
		{name: "zero year", month: "01", year: "00", wantMonth: 1, wantYear: 0},
		{name: "month only", month: "12", wantErr: true},
		{name: "year only", year: "30", wantErr: true},
		{name: "short month", month: "1", year: "30", wantMonth: 1, wantYear: 30},
		{name: "short year", month: "12", year: "3", wantMonth: 12, wantYear: 3},
		{name: "month zero", month: "00", year: "30", wantErr: true},
		{name: "month too large", month: "13", year: "30", wantErr: true},
		{name: "non digits", month: "AA", year: "BB", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertParsedRecordCardExpiry(t, test.month, test.year, test.wantMonth, test.wantYear, test.wantEmpty, test.wantErr)
		})
	}
}

func assertParsedRecordCardExpiry(
	t *testing.T,
	monthValue,
	yearValue string,
	wantMonth,
	wantYear int,
	wantEmpty,
	wantErr bool,
) {
	t.Helper()
	month, year, err := parseRecordCardExpiry(monthValue, yearValue)
	if (err != nil) != wantErr {
		t.Fatalf("parseRecordCardExpiry() error = %v, wantErr %t", err, wantErr)
	}
	if wantErr {
		return
	}
	if wantEmpty {
		if month != nil || year != nil {
			t.Fatalf("expiry = %v/%v, want empty", month, year)
		}
		return
	}
	if month == nil || year == nil || *month != wantMonth || *year != wantYear {
		t.Fatalf("expiry = %v/%v, want %d/%d", month, year, wantMonth, wantYear)
	}
}

func TestRecordFormInput_BuildPayload(t *testing.T) {
	binaryData := []byte{0x01, 0x02, 0xff}
	readBinary := func(path string) (string, []byte, error) {
		if path != "/tmp/backup.bin" {
			t.Fatalf("binary path = %q", path)
		}
		return "backup.bin", binaryData, nil
	}

	tests := []struct {
		name  string
		input recordFormInput
		want  recordmodel.RecordPayload
	}{
		{
			name: "text",
			input: recordFormInput{
				recordType: recordmodel.RecordTypeText,
				title:      "Private note",
				text:       "first\nsecond",
				metadata:   "personal",
			},
			want: &recordmodel.TextPayload{Text: "first\nsecond", Metadata: "personal"},
		},
		{
			name: "credentials",
			input: recordFormInput{
				recordType: recordmodel.RecordTypeCredentials,
				title:      "GitHub",
				login:      "alice",
				password:   "secret",
				url:        "https://github.com",
			},
			want: &recordmodel.CredentialsPayload{
				Login: "alice", Password: "secret", URL: "https://github.com",
			},
		},
		{
			name: "card",
			input: recordFormInput{
				recordType:  recordmodel.RecordTypeCard,
				title:       "Primary card",
				number:      "4111111111111111",
				cardholder:  "JOEL MILLER",
				expiryMonth: "12",
				expiryYear:  "30",
				cvv:         "014",
			},
			want: &recordmodel.CardPayload{
				Number: "4111111111111111", Cardholder: "JOEL MILLER",
				ExpiryMonth: intPointer(12), ExpiryYear: intPointer(30), CVV: "014",
			},
		},
		{
			name: "binary",
			input: recordFormInput{
				recordType: recordmodel.RecordTypeBinary,
				title:      "Backup",
				filePath:   "/tmp/backup.bin",
			},
			want: &recordmodel.BinaryPayload{Filename: "backup.bin", Data: binaryData},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			payload, err := test.input.buildPayload(readBinary, nil)
			if err != nil {
				t.Fatalf("buildPayload() error = %v", err)
			}
			if !reflect.DeepEqual(payload, test.want) {
				t.Fatalf("payload = %#v, want %#v", payload, test.want)
			}
		})
	}
}

func intPointer(value int) *int {
	return &value
}

func TestRecordCreateForm_ExpiryFieldsAcceptAtMostTwoDigits(t *testing.T) {
	form := newRecordCreateForm(recordmodel.RecordTypeCard)
	form.setFocus(recordFormExpiryMonth)
	form.insert("1A299")
	if form.expiryMonth.value != "12" {
		t.Fatalf("expiry month = %q, want 12", form.expiryMonth.value)
	}

	form.setFocus(recordFormExpiryYear)
	form.insert("3B099")
	if form.expiryYear.value != "30" {
		t.Fatalf("expiry year = %q, want 30", form.expiryYear.value)
	}
}

func TestRecordFormInputFrom_PadsSingleDigitCardExpiry(t *testing.T) {
	form := newRecordCreateForm(recordmodel.RecordTypeCard)
	form.expiryMonth.setValue("7")
	form.expiryYear.setValue("5")

	input := recordFormInputFrom(form)
	if input.expiryMonth != "07" || input.expiryYear != "05" {
		t.Fatalf("normalized expiry = %q/%q, want 07/05", input.expiryMonth, input.expiryYear)
	}
	month, year, err := parseRecordCardExpiry(input.expiryMonth, input.expiryYear)
	if err != nil {
		t.Fatalf("parseRecordCardExpiry() error = %v", err)
	}
	if month == nil || year == nil || *month != 7 || *year != 5 {
		t.Fatalf("parsed expiry = %v/%v, want 7/5", month, year)
	}
}

func TestRecordCreateForm_ExpiryFieldsRequireCompletePair(t *testing.T) {
	form := newRecordCreateForm(recordmodel.RecordTypeCard)
	form.title.setValue("Primary card")
	form.number.setValue("4111111111111111")
	form.expiryMonth.setValue("12")
	if form.canSubmit() {
		t.Fatal("form accepted expiry month without year")
	}

	form.expiryYear.setValue("30")
	if !form.canSubmit() {
		t.Fatal("form rejected complete expiry pair")
	}
}

func TestRenderRecordCardForm_UsesRegularCompactFields(t *testing.T) {
	form := newRecordCreateForm(recordmodel.RecordTypeCard)
	form.number.setValue("4111111111111111")
	form.cardholder.setValue("JOEL MILLER")
	form.expiryMonth.setValue("12")
	form.expiryYear.setValue("30")
	form.cvv.setValue("014")
	form.revealed = true

	got := ansi.Strip(renderRecordCreateWindow(newTheme(), 72, form, false, false, ""))
	for _, wanted := range []string{"Number *", "4111111111111111", "Cardholder", "JOEL MILLER", "Expiry month", "12", "Expiry year", "30", "CVV", "014"} {
		if !strings.Contains(got, wanted) {
			t.Fatalf("card form does not contain %q:\n%s", wanted, got)
		}
	}
	for _, unwanted := range []string{"▒▒▒▒", "exp 12/30", "cvv 014"} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("card form still contains card-preview fragment %q:\n%s", unwanted, got)
		}
	}
	for control, want := range map[recordFormControl]int{
		recordFormNumber:      recordmodel.CardNumberMaxSize,
		recordFormCardholder:  recordmodel.CardholderMaxSize,
		recordFormExpiryMonth: 2,
		recordFormExpiryYear:  2,
		recordFormCVV:         recordmodel.CardCVVSize,
	} {
		field := form.field(control)
		if field == nil {
			t.Fatalf("field for control %d is missing", control)
		}
		if width := recordFormFieldWidth(*field, 40); width != want {
			t.Fatalf("field %d width = %d, want %d", control, width, want)
		}
	}
}

func TestTextArea_PreservesLinesAndLimit(t *testing.T) {
	area := newTextArea("first\r\nsecond")
	if area.value != "first\nsecond" {
		t.Fatalf("value = %q", area.value)
	}
	area.insert("\nthird")
	if !strings.HasSuffix(area.value, "\nthird") {
		t.Fatalf("value = %q", area.value)
	}

	area = newTextArea(strings.Repeat("a", recordmodel.TextPayloadMaxSize))
	if area.insert("b") {
		t.Fatal("text area accepted data above the domain limit")
	}
	if len(area.value) != recordmodel.TextPayloadMaxSize {
		t.Fatalf("text size = %d", len(area.value))
	}
}

func TestRecordFormInput_BuildPayloadRejectsInvalidExpiry(t *testing.T) {
	input := recordFormInput{
		recordType:  recordmodel.RecordTypeCard,
		title:       "Primary card",
		number:      "4111111111111111",
		expiryMonth: "13",
		expiryYear:  "30",
	}
	_, err := input.buildPayload(func(string) (string, []byte, error) {
		return "", nil, errors.New("must not be called")
	}, nil)
	if err == nil {
		t.Fatal("buildPayload() accepted invalid expiry")
	}
}

func TestRecordCreateForm_CardFieldsUseDomainLimits(t *testing.T) {
	form := newRecordCreateForm(recordmodel.RecordTypeCard)

	form.setFocus(recordFormNumber)
	form.insert("4111 1111-1111A1111")
	if form.number.value != "4111111111111111" {
		t.Fatalf("number = %q, want digits only", form.number.value)
	}
	form.insert("1234567890")
	if len(form.number.value) != recordmodel.CardNumberMaxSize {
		t.Fatalf("number length = %d, want %d", len(form.number.value), recordmodel.CardNumberMaxSize)
	}

	form.setFocus(recordFormCVV)
	form.insert("01A4")
	if form.cvv.value != "014" {
		t.Fatalf("CVV = %q, want 014", form.cvv.value)
	}

	form.setFocus(recordFormCardholder)
	form.insert(strings.Repeat("Я", recordmodel.CardholderMaxSize+5))
	if got := len([]rune(form.cardholder.value)); got != recordmodel.CardholderMaxSize {
		t.Fatalf("cardholder length = %d, want %d", got, recordmodel.CardholderMaxSize)
	}
}

func TestRenderRecordForms_MarkOnlyRequiredFieldsAndOmitOptionalHints(t *testing.T) {
	theme := newTheme()

	tests := []struct {
		name        string
		form        recordForm
		render      func(recordForm) string
		required    []string
		notRequired []string
	}{
		{
			name: "text create",
			form: newRecordCreateForm(recordmodel.RecordTypeText),
			render: func(form recordForm) string {
				return renderRecordCreateWindow(theme, 72, form, false, false, "")
			},
			required:    []string{"Title *", "Text *"},
			notRequired: []string{"Notes *"},
		},
		{
			name: "credentials create",
			form: newRecordCreateForm(recordmodel.RecordTypeCredentials),
			render: func(form recordForm) string {
				return renderRecordCreateWindow(theme, 72, form, false, false, "")
			},
			required:    []string{"Title *", "Login *", "Password *"},
			notRequired: []string{"URL *", "Notes *"},
		},
		{
			name: "card create",
			form: newRecordCreateForm(recordmodel.RecordTypeCard),
			render: func(form recordForm) string {
				return renderRecordCreateWindow(theme, 72, form, false, false, "")
			},
			required:    []string{"Title *", "Number *"},
			notRequired: []string{"Cardholder *", "Expiry month *", "Expiry year *", "CVV *", "Notes *"},
		},
		{
			name: "binary create",
			form: newRecordCreateForm(recordmodel.RecordTypeBinary),
			render: func(form recordForm) string {
				return renderRecordCreateWindow(theme, 72, form, false, false, "")
			},
			required:    []string{"Title *", "File path *"},
			notRequired: []string{"Notes *"},
		},
		{
			name: "binary edit",
			form: newRecordEditForm(recordmodel.Record{
				Metadata: recordmodel.RecordMetadata{Type: recordmodel.RecordTypeBinary, Title: "Backup"},
				Payload:  &recordmodel.BinaryPayload{Filename: "backup.bin"},
			}),
			render: func(form recordForm) string {
				return renderRecordEditWindow(theme, 72, form, false, false, "")
			},
			required:    []string{"Title *", "Replace file"},
			notRequired: []string{"Filename", "Replace file *", "Notes *"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plain := ansi.Strip(test.render(test.form))
			for _, wanted := range test.required {
				if !strings.Contains(plain, wanted) {
					t.Fatalf("form does not contain required label %q:\n%s", wanted, plain)
				}
			}
			for _, unwanted := range test.notRequired {
				if strings.Contains(plain, unwanted) {
					t.Fatalf("optional label is marked as required %q:\n%s", unwanted, plain)
				}
			}
			if strings.Contains(strings.ToLower(plain), "optional") || strings.Contains(plain, "Leave Replace file") {
				t.Fatalf("form still contains an optional-field hint:\n%s", plain)
			}
		})
	}
}

func TestRecordCreateForm_BinaryPathIsReadOnlyPickerValue(t *testing.T) {
	form := newRecordCreateForm(recordmodel.RecordTypeBinary)
	form.filePath.setValue("fixtures/backup.bin")
	form.setFocus(recordFormFilePath)

	form.insert("changed")
	form.backspace()
	form.delete()
	if form.filePath.value != "fixtures/backup.bin" {
		t.Fatalf("read-only path = %q", form.filePath.value)
	}

	plain := ansi.Strip(renderRecordCreateWindow(newTheme(), 72, form, false, false, ""))
	for _, wanted := range []string{"File path *", "fixtures/backup.bin", "< Browse >"} {
		if !strings.Contains(plain, wanted) {
			t.Fatalf("binary form does not contain %q:\n%s", wanted, plain)
		}
	}
}

func TestRenderRecordTypePickerWindow_UsesPurpleWizardLayout(t *testing.T) {
	theme := newTheme()
	picker := recordTypePicker{}
	rendered := renderRecordTypePickerWindow(theme, picker)
	plain := ansi.Strip(rendered)

	for _, want := range []string{
		"New Record Wizard",
		"Select record type:",
		"CREDENTIALS",
		"Login, password and URL",
		"CARD",
		"Payment card details",
		"TEXT",
		"Arbitrary private text",
		"BINARY",
		"File up to 2 MiB",
		"< Cancel >",
	} {
		if !strings.Contains(plain, want) {
			t.Fatalf("wizard does not contain %q:\n%s", want, plain)
		}
	}
	if strings.Contains(plain, "< Next >") {
		t.Fatalf("wizard still contains a Next button:\n%s", plain)
	}

	previous := -1
	for _, value := range []string{"CREDENTIALS", "CARD", "TEXT", "BINARY"} {
		index := strings.Index(plain, value)
		if index <= previous {
			t.Fatalf("wizard type order is wrong near %q:\n%s", value, plain)
		}
		previous = index
	}

	selected := renderRecordTypePickerRow(theme, recordTypePickerWidth-4, recordmodel.RecordTypeCredentials, true)
	if !strings.Contains(selected, theme.wizardSelectedType.Width(12).Render("CREDENTIALS")) ||
		!strings.Contains(selected, theme.wizardSelectedDescription.Width(recordTypePickerWidth-4-12).Render("Login, password and URL")) {
		t.Fatal("default credentials row does not use the selected wizard styles")
	}
	unselected := renderRecordTypePickerRow(theme, recordTypePickerWidth-4, recordmodel.RecordTypeCard, false)
	if !strings.Contains(unselected, theme.wizardType.Width(12).Render("CARD")) ||
		!strings.Contains(unselected, theme.wizardDescription.Width(recordTypePickerWidth-4-12).Render("Payment card details")) {
		t.Fatal("unselected card row does not use yellow type and white description styles")
	}
}

func TestRenderRecordForms_UseNotesAsMetadataFieldLabel(t *testing.T) {
	theme := newTheme()
	for _, recordType := range recordCreateTypes {
		plain := ansi.Strip(renderRecordCreateWindow(
			theme,
			72,
			newRecordCreateForm(recordType),
			false,
			false,
			"",
		))
		if !strings.Contains(plain, "Notes") {
			t.Fatalf("%s form does not contain Notes label:\n%s", recordType, plain)
		}
		if strings.Contains(plain, "Metadata") {
			t.Fatalf("%s form still contains Metadata label:\n%s", recordType, plain)
		}
	}
}

func TestRenderRecordEditForms_UseNotesAsMetadataFieldLabel(t *testing.T) {
	theme := newTheme()
	records := []recordmodel.Record{
		{
			Metadata: recordmodel.RecordMetadata{Type: recordmodel.RecordTypeCredentials, Title: "Account"},
			Payload:  &recordmodel.CredentialsPayload{Login: "alice", Password: "secret"},
		},
		{
			Metadata: recordmodel.RecordMetadata{Type: recordmodel.RecordTypeCard, Title: "Card"},
			Payload:  &recordmodel.CardPayload{Number: "4111111111111111"},
		},
		{
			Metadata: recordmodel.RecordMetadata{Type: recordmodel.RecordTypeText, Title: "Note"},
			Payload:  &recordmodel.TextPayload{Text: "secret"},
		},
		{
			Metadata: recordmodel.RecordMetadata{Type: recordmodel.RecordTypeBinary, Title: "Backup"},
			Payload:  &recordmodel.BinaryPayload{Filename: "backup.bin"},
		},
	}

	for _, record := range records {
		plain := ansi.Strip(renderRecordEditWindow(
			theme,
			72,
			newRecordEditForm(record),
			false,
			false,
			"",
		))
		if !strings.Contains(plain, "Notes") {
			t.Fatalf("%s edit form does not contain Notes label:\n%s", record.Metadata.Type, plain)
		}
		if strings.Contains(plain, "Metadata") {
			t.Fatalf("%s edit form still contains Metadata label:\n%s", record.Metadata.Type, plain)
		}
	}
}

func TestRenderRecordForms_UsePurpleBodyForCreateAndEdit(t *testing.T) {
	theme := newTheme()
	records := []recordmodel.Record{
		{
			Metadata: recordmodel.RecordMetadata{ID: "credentials-id", Type: recordmodel.RecordTypeCredentials, Title: "Account", Revision: 1},
			Payload:  &recordmodel.CredentialsPayload{Login: "alice", Password: "secret"},
		},
		{
			Metadata: recordmodel.RecordMetadata{ID: "card-id", Type: recordmodel.RecordTypeCard, Title: "Card", Revision: 1},
			Payload:  &recordmodel.CardPayload{Number: "4111111111111111"},
		},
		{
			Metadata: recordmodel.RecordMetadata{ID: "text-id", Type: recordmodel.RecordTypeText, Title: "Note", Revision: 1},
			Payload:  &recordmodel.TextPayload{Text: "secret"},
		},
		{
			Metadata: recordmodel.RecordMetadata{ID: "binary-id", Type: recordmodel.RecordTypeBinary, Title: "Backup", Revision: 1},
			Payload:  &recordmodel.BinaryPayload{Filename: "backup.bin"},
		},
	}

	for _, record := range records {
		t.Run(string(record.Metadata.Type), func(t *testing.T) {
			create := renderRecordCreateWindow(theme, 72, newRecordCreateForm(record.Metadata.Type), false, false, "")
			edit := renderRecordEditWindow(theme, 72, newRecordEditForm(record), false, false, "")
			for name, rendered := range map[string]string{"create": create, "edit": edit} {
				if !strings.Contains(rendered, theme.recordFormLabel.Render("Title ")) {
					t.Fatalf("%s %s form does not use the purple record label style", record.Metadata.Type, name)
				}
				if strings.Contains(rendered, theme.label.Render("Title ")) {
					t.Fatalf("%s %s form still uses the cyan form label style", record.Metadata.Type, name)
				}
			}
		})
	}
}

func TestRenderRecordEditForms_ShowServiceFieldsBeforeTitle(t *testing.T) {
	theme := newTheme()
	records := []recordmodel.Record{
		{
			Metadata: recordmodel.RecordMetadata{ID: "credentials-id", Type: recordmodel.RecordTypeCredentials, Title: "Account", Revision: 7},
			Payload:  &recordmodel.CredentialsPayload{Login: "alice", Password: "secret"},
		},
		{
			Metadata: recordmodel.RecordMetadata{ID: "card-id", Type: recordmodel.RecordTypeCard, Title: "Card", Revision: 7},
			Payload:  &recordmodel.CardPayload{Number: "4111111111111111"},
		},
		{
			Metadata: recordmodel.RecordMetadata{ID: "text-id", Type: recordmodel.RecordTypeText, Title: "Note", Revision: 7},
			Payload:  &recordmodel.TextPayload{Text: "secret"},
		},
		{
			Metadata: recordmodel.RecordMetadata{ID: "binary-id", Type: recordmodel.RecordTypeBinary, Title: "Backup", Revision: 7},
			Payload:  &recordmodel.BinaryPayload{Filename: "backup.bin"},
		},
	}

	for _, record := range records {
		t.Run(string(record.Metadata.Type), func(t *testing.T) {
			plain := ansi.Strip(renderRecordEditWindow(theme, 72, newRecordEditForm(record), false, false, ""))
			for _, want := range []string{
				"ID: " + record.Metadata.ID,
				"Revision: 7",
				"Created at:",
				"Updated at:",
				"Title *",
			} {
				if !strings.Contains(plain, want) {
					t.Fatalf("%s edit form does not contain %q:\n%s", record.Metadata.Type, want, plain)
				}
			}
			lines := strings.Split(plain, "\n")
			updatedRow := lineIndexContaining(lines, "Updated at:")
			titleRow := lineIndexContaining(lines, "Title *")
			if updatedRow < 0 || titleRow != updatedRow+2 || strings.TrimSpace(lines[updatedRow+1]) != "" {
				t.Fatalf("%s edit form does not separate service fields from Title with one blank row:\n%s", record.Metadata.Type, plain)
			}
		})
	}
}

func TestRecordFormLayout_LeavesOneBlankRowBeforeButtons(t *testing.T) {
	form := newRecordCreateForm(recordmodel.RecordTypeCredentials)
	layout := recordFormLayout(&form, 68)
	notesLayout, _ := layout.control(recordFormMetadata)
	notesRow := notesLayout.row
	if layout.buttonRow != notesRow+2 {
		t.Fatalf("create button row = %d, want %d", layout.buttonRow, notesRow+2)
	}
}

func TestRenderRecordForms_KeepOneBlankRowAroundButtonsAndHidePendingStatus(t *testing.T) {
	theme := newTheme()
	records := []recordmodel.Record{
		{
			Metadata: recordmodel.RecordMetadata{ID: "credentials-id", Type: recordmodel.RecordTypeCredentials, Title: "Account", Revision: 1},
			Payload:  &recordmodel.CredentialsPayload{Login: "alice", Password: "secret"},
		},
		{
			Metadata: recordmodel.RecordMetadata{ID: "card-id", Type: recordmodel.RecordTypeCard, Title: "Card", Revision: 1},
			Payload:  &recordmodel.CardPayload{Number: "4111111111111111"},
		},
		{
			Metadata: recordmodel.RecordMetadata{ID: "text-id", Type: recordmodel.RecordTypeText, Title: "Note", Revision: 1},
			Payload:  &recordmodel.TextPayload{Text: "secret"},
		},
		{
			Metadata: recordmodel.RecordMetadata{ID: "binary-id", Type: recordmodel.RecordTypeBinary, Title: "Backup", Revision: 1},
			Payload:  &recordmodel.BinaryPayload{Filename: "backup.bin"},
		},
	}

	for _, record := range records {
		t.Run(string(record.Metadata.Type), func(t *testing.T) {
			windows := map[string]string{
				"create": renderRecordCreateWindow(theme, 72, newRecordCreateForm(record.Metadata.Type), true, true, "⠋"),
				"edit":   renderRecordEditWindow(theme, 72, newRecordEditForm(record), true, true, "⠋"),
			}
			for mode, window := range windows {
				plain := ansi.Strip(window)
				if strings.Contains(plain, "Creating record") || strings.Contains(plain, "Saving record") {
					t.Fatalf("%s %s form still contains pending status:\n%s", record.Metadata.Type, mode, plain)
				}
				buttonLabel := "< Create >"
				if mode == "edit" {
					buttonLabel = "< Save >"
				}
				lines := strings.Split(plain, "\n")
				buttonRow := lineIndexContaining(lines, buttonLabel)
				if buttonRow <= 0 || buttonRow+1 >= len(lines) {
					t.Fatalf("%s %s button row = %d:\n%s", record.Metadata.Type, mode, buttonRow, plain)
				}
				if strings.TrimSpace(lines[buttonRow-1]) != "" {
					t.Fatalf("%s %s form has no blank row above buttons:\n%s", record.Metadata.Type, mode, plain)
				}
				if buttonRow < 2 || strings.TrimSpace(lines[buttonRow-2]) == "" {
					t.Fatalf("%s %s form has more than one blank row above buttons:\n%s", record.Metadata.Type, mode, plain)
				}
				if strings.TrimSpace(lines[buttonRow+1]) != "" {
					t.Fatalf("%s %s form has no blank row below buttons:\n%s", record.Metadata.Type, mode, plain)
				}
				if buttonRow+2 != len(lines) {
					t.Fatalf("%s %s form has more than one row below buttons:\n%s", record.Metadata.Type, mode, plain)
				}
			}
		})
	}
}

func TestRecordFormInput_BuildPayloadUsesSelectedBinaryFileName(t *testing.T) {
	input := recordFormInput{
		recordType:     recordmodel.RecordTypeBinary,
		title:          "Backup",
		filename:       "old.bin",
		filePath:       "/tmp/new-name.bin",
		binaryExisting: true,
		editing:        true,
	}

	payload, err := input.buildPayload(func(path string) (string, []byte, error) {
		return "new-name.bin", []byte("new data"), nil
	}, &recordmodel.BinaryPayload{Filename: "old.bin", Data: []byte("old data")})
	if err != nil {
		t.Fatalf("buildPayload() error = %v", err)
	}
	binaryPayload := payload.(*recordmodel.BinaryPayload)
	if binaryPayload.Filename != "new-name.bin" {
		t.Fatalf("filename = %q, want %q", binaryPayload.Filename, "new-name.bin")
	}
}
