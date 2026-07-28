package tui

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/xhrobj/gopherkeeper/internal/client/usecase"
	recordmodel "github.com/xhrobj/gopherkeeper/internal/model"
)

func TestRecordViewLines_MasksAndRevealsSensitiveFields(t *testing.T) {
	record := recordmodel.Record{
		Metadata: recordmodel.RecordMetadata{
			ID:       "7a79b627-0473-48a0-a001-887e79419719",
			Type:     recordmodel.RecordTypeCredentials,
			Title:    "GitHub",
			Revision: 1,
		},
		Payload: &recordmodel.CredentialsPayload{Login: "alice", Password: "secret"},
	}

	masked := recordViewLinesText(recordViewLines(newTheme(), recordViewState{status: recordViewReady, record: record}, 60))
	if strings.Contains(masked, "Password: secret") {
		t.Fatal("password is visible in the masked record view")
	}
	if !strings.Contains(masked, "Password: ••••••") {
		t.Fatalf("masked password was not rendered:\n%s", masked)
	}

	revealed := recordViewLinesText(recordViewLines(newTheme(), recordViewState{status: recordViewReady, record: record, revealed: true}, 60))
	if !strings.Contains(revealed, "Password: secret") {
		t.Fatalf("revealed password was not rendered:\n%s", revealed)
	}
}

func TestRenderRecordViewWindow_RendersStructuredDetails(t *testing.T) {
	record := recordmodel.Record{
		Metadata: recordmodel.RecordMetadata{
			ID:        "7a79b627-0473-48a0-a001-887e79419719",
			Type:      recordmodel.RecordTypeText,
			Title:     "Recovery codes",
			Revision:  3,
			CreatedAt: time.Date(2026, time.July, 19, 11, 0, 0, 0, time.Local),
			UpdatedAt: time.Date(2026, time.July, 19, 12, 0, 0, 0, time.Local),
		},
		Payload: &recordmodel.TextPayload{Text: "first line\nsecond line", Metadata: "private note"},
	}
	state := recordViewState{status: recordViewReady, record: record}
	styled := renderRecordViewWindow(newTheme(), newRecordViewWindowOptions(72, 24, state, 0, false, false, ""))
	plain := ansi.Strip(styled)

	for _, want := range []string{
		"Record Text Online",
		"ID: 7a79b627-0473-48a0-a001-887e79419719",
		"Revision: 3",
		"Created at: 2026-07-19 11:00",
		"Updated at: 2026-07-19 12:00",
		"Title: Recovery codes",
		"Text: first line",
		"second line",
		"Notes: private note",
		"< Close >",
	} {
		if !strings.Contains(plain, want) {
			t.Fatalf("record view does not contain %q:\n%s", want, plain)
		}
	}
	if strings.Contains(plain, "Type:") {
		t.Fatalf("record view still repeats the record type in the body:\n%s", plain)
	}
	if strings.Contains(plain, "PgUp/PgDn") || strings.Contains(plain, "Esc Close") {
		t.Fatalf("record view still contains the removed navigation hint:\n%s", plain)
	}
}

func TestRenderRecordViewWindow_LoadingUsesTitleSpinnerAndDisabledButtons(t *testing.T) {
	theme := newTheme()
	state := recordViewState{
		status: recordViewLoading,
		record: recordmodel.Record{Metadata: recordmodel.RecordMetadata{Type: recordmodel.RecordTypeBinary}},
	}
	styled := renderRecordViewWindow(theme, newRecordViewWindowOptions(72, 20, state, 1, true, true, "⠋"))
	plain := ansi.Strip(styled)
	if !strings.Contains(plain, "Record Binary Online ⠋") {
		t.Fatalf("loading title has no stable type and spinner:\n%s", plain)
	}
	if strings.Contains(plain, "Loading record") {
		t.Fatalf("loading message remained in record body:\n%s", plain)
	}
	buttons := renderRecordViewButtons(theme, 68, state, 1, true)
	if !strings.Contains(buttons, theme.buttonDisabled.Render("< Save As... >")) ||
		!strings.Contains(buttons, theme.buttonDisabledActive.Render("< Close >")) {
		t.Fatal("loading record buttons are not visibly disabled")
	}
}

func TestRecordViewLines_MetadataUsesSeparateLabelsAndValues(t *testing.T) {
	record := recordmodel.Record{
		Metadata: recordmodel.RecordMetadata{
			ID:       "record-id",
			Type:     recordmodel.RecordTypeText,
			Revision: 7,
		},
		Payload: &recordmodel.TextPayload{Text: "value"},
	}

	lines := recordViewLines(newTheme(), recordViewState{status: recordViewReady, record: record}, 60)
	want := map[string]string{
		"ID: ":         "record-id",
		"Revision: ":   "7",
		"Created at: ": "",
		"Updated at: ": "",
	}
	for label, value := range want {
		found := false
		for _, line := range lines {
			if line.label == label && line.value == value && line.message == "" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("structured line %q = %q was not found: %#v", label, value, lines)
		}
	}
}

func TestFormattedVisibleCardNumber_MasksUnicodeDigits(t *testing.T) {
	got := formattedVisibleCardNumber("１２３４５６７８９０１２３４５６", false)
	want := "•••• •••• •••• ３４５６"
	if got != want {
		t.Fatalf("masked card number = %q, want %q", got, want)
	}
}

func TestRecordViewLines_WrapsLongValues(t *testing.T) {
	record := recordmodel.Record{
		Metadata: recordmodel.RecordMetadata{Type: recordmodel.RecordTypeText, Title: "Text", Revision: 1},
		Payload:  &recordmodel.TextPayload{Text: strings.Repeat("x", 40)},
	}
	lines := recordViewLines(newTheme(), recordViewState{status: recordViewReady, record: record}, 20)
	plain := recordViewLinesText(lines)
	if strings.Count(plain, "xxxxxxxx") < 2 {
		t.Fatalf("long text was not wrapped into multiple rows:\n%s", plain)
	}
}

func TestRecordViewLines_AlwaysShowsNotes(t *testing.T) {
	record := recordmodel.Record{
		Metadata: recordmodel.RecordMetadata{
			Type:     recordmodel.RecordTypeCredentials,
			Title:    "GitHub",
			Revision: 1,
		},
		Payload: &recordmodel.CredentialsPayload{Login: "alice", Password: "secret"},
	}

	plain := recordViewLinesText(recordViewLines(newTheme(), recordViewState{status: recordViewReady, record: record}, 60))
	if !strings.Contains(plain, "Notes: ") {
		t.Fatalf("empty Notes field is missing:\n%s", plain)
	}
	if strings.Contains(plain, "Metadata:") {
		t.Fatalf("record view still contains Metadata label:\n%s", plain)
	}
}

func TestRecordViewWindowHeightForState_ShrinksCredentialsWindowToContent(t *testing.T) {
	state := recordViewState{
		status: recordViewReady,
		record: recordmodel.Record{
			Metadata: recordmodel.RecordMetadata{Type: recordmodel.RecordTypeCredentials, Title: "GitHub", Revision: 1},
			Payload:  &recordmodel.CredentialsPayload{Login: "alice", Password: "secret"},
		},
	}

	got := recordViewWindowHeightForState(newTheme(), 100, 42, state)
	full := recordViewWindowHeight(42)
	if got >= full {
		t.Fatalf("credentials view height = %d, want less than full height %d", got, full)
	}
	want := clamp(len(recordViewLines(newTheme(), state, recordViewContentWidth(100)))+5, 16, full)
	if got != want {
		t.Fatalf("credentials view height = %d, want %d", got, want)
	}
}

func TestRenderRecordViewWindow_BinaryOffersSaveAs(t *testing.T) {
	state := recordViewState{
		status: recordViewReady,
		record: recordmodel.Record{
			Metadata: recordmodel.RecordMetadata{Type: recordmodel.RecordTypeBinary, Title: "Backup", Revision: 1},
			Payload:  &recordmodel.BinaryPayload{Filename: "backup.bin", Data: []byte{0x01}},
		},
	}
	plain := ansi.Strip(renderRecordViewWindow(newTheme(), newRecordViewWindowOptions(72, 20, state, 0, false, false, "")))
	if !strings.Contains(plain, "< Save As... >") || !strings.Contains(plain, "< Close >") {
		t.Fatalf("binary record actions are missing:\n%s", plain)
	}
}

func TestRecordViewLines_CardholderIsRenderedUppercase(t *testing.T) {
	state := recordViewState{
		status: recordViewReady,
		record: recordmodel.Record{
			Metadata: recordmodel.RecordMetadata{Type: recordmodel.RecordTypeCard, Title: "Primary", Revision: 2},
			Payload: &recordmodel.CardPayload{
				Number:     "4111111111111111",
				Cardholder: "Joel Miller",
			},
		},
	}
	plain := ansi.Strip(renderRecordViewWindow(newTheme(), newRecordViewWindowOptions(72, 24, state, 0, false, false, "")))
	if !strings.Contains(plain, "JOEL MILLER") {
		t.Fatalf("cardholder is not rendered uppercase:\n%s", plain)
	}
	if strings.Contains(plain, "Joel Miller") {
		t.Fatalf("cardholder keeps mixed case:\n%s", plain)
	}
}

func TestRecordViewLines_CardPreviewLooksLikeCard(t *testing.T) {
	month := 12
	year := 30
	state := recordViewState{
		status: recordViewReady,
		record: recordmodel.Record{
			Metadata: recordmodel.RecordMetadata{Type: recordmodel.RecordTypeCard, Title: "Primary", Revision: 2},
			Payload: &recordmodel.CardPayload{
				Number:      "4111111111111111",
				Cardholder:  "JOEL MILLER",
				ExpiryMonth: &month,
				ExpiryYear:  &year,
				CVV:         "014",
			},
		},
	}
	plain := ansi.Strip(renderRecordViewWindow(newTheme(), newRecordViewWindowOptions(72, 24, state, 0, false, false, "")))
	for _, want := range []string{"Record Card Online", "•••• •••• •••• 1111", "JOEL MILLER", "▒▒▒▒", "exp 12/30", "[cvv •••]"} {
		if !strings.Contains(plain, want) {
			t.Fatalf("card preview does not contain %q:\n%s", want, plain)
		}
	}
	if strings.Contains(plain, "┌") || strings.Contains(plain, "└") {
		t.Fatalf("card preview still contains the removed frame:\n%s", plain)
	}
}

func TestRenderRecordViewAndDeleteForms_KeepOneBlankRowAroundButtons(t *testing.T) {
	theme := newTheme()
	records := []recordmodel.Record{
		{Metadata: recordmodel.RecordMetadata{ID: "credentials-id", Type: recordmodel.RecordTypeCredentials, Title: "Account", Revision: 1}, Payload: &recordmodel.CredentialsPayload{Login: "alice", Password: "secret"}},
		{Metadata: recordmodel.RecordMetadata{ID: "card-id", Type: recordmodel.RecordTypeCard, Title: "Card", Revision: 1}, Payload: &recordmodel.CardPayload{Number: "4111111111111111"}},
		{Metadata: recordmodel.RecordMetadata{ID: "text-id", Type: recordmodel.RecordTypeText, Title: "Note", Revision: 1}, Payload: &recordmodel.TextPayload{Text: "secret"}},
		{Metadata: recordmodel.RecordMetadata{ID: "binary-id", Type: recordmodel.RecordTypeBinary, Title: "Backup", Revision: 1}, Payload: &recordmodel.BinaryPayload{Filename: "backup.bin", Data: []byte("data")}},
	}

	for _, record := range records {
		t.Run(string(record.Metadata.Type), func(t *testing.T) {
			viewState := recordViewState{status: recordViewReady, record: record}
			viewHeight := recordViewWindowHeightForState(theme, 100, 40, viewState)
			assertOneBlankRowAroundRecordButtons(t, ansi.Strip(renderRecordViewWindow(theme, newRecordViewWindowOptions(recordViewWindowWidth(100), viewHeight, viewState, 0, false, false, ""))), "< Close >")

			deleteState := recordDeleteState{metadata: record.Metadata}
			assertOneBlankRowAroundRecordButtons(t, ansi.Strip(renderRecordDeleteWindow(theme, recordDeleteWindowWidth(100), deleteState, false, false, "", 1)), "< Cancel >")
		})
	}
}

func TestCleanCachedRecordViewError_SuggestsSyncForUnreadableRecord(t *testing.T) {
	err := errors.Join(
		usecase.ErrLocalCacheRecordsUnreadable,
		errors.New("corrupted encrypted payload"),
	)

	got := cleanCachedRecordViewError(err)
	want := "This cached record is damaged or incompatible.\nRun Cache/Sync... to restore it from the Server."
	if got != want {
		t.Fatalf("cleanCachedRecordViewError() = %q, want %q", got, want)
	}
}

func recordViewLinesText(lines []recordViewLine) string {
	plain := make([]string, 0, len(lines))
	for _, line := range lines {
		plain = append(plain, line.message+line.label+line.value)
	}
	return strings.Join(plain, "\n")
}

func assertOneBlankRowAroundRecordButtons(t *testing.T, rendered, buttonLabel string) {
	t.Helper()
	lines := strings.Split(rendered, "\n")
	buttonRow := lineIndexContaining(lines, buttonLabel)
	if buttonRow < 2 || buttonRow+1 >= len(lines) {
		t.Fatalf("button row %q = %d:\n%s", buttonLabel, buttonRow, rendered)
	}
	if strings.TrimSpace(lines[buttonRow-1]) != "" || strings.TrimSpace(lines[buttonRow-2]) == "" {
		t.Fatalf("form does not have exactly one blank row above %q:\n%s", buttonLabel, rendered)
	}
	if strings.TrimSpace(lines[buttonRow+1]) != "" || buttonRow+2 != len(lines) {
		t.Fatalf("form does not have exactly one blank row below %q:\n%s", buttonLabel, rendered)
	}
}
