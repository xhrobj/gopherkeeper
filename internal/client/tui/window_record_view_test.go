package tui

import (
	"slices"
	"testing"

	recordmodel "github.com/xhrobj/gopherkeeper/internal/model"
)

func TestRecordViewButtonLabels_ReflectRecordState(t *testing.T) {
	tests := []struct {
		name  string
		state recordViewState
		want  []string
	}{
		{name: "idle", state: recordViewState{}, want: nil},
		{
			name: "binary",
			state: recordViewState{status: recordViewReady, record: recordmodel.Record{
				Metadata: recordmodel.RecordMetadata{Type: recordmodel.RecordTypeBinary},
				Payload:  &recordmodel.BinaryPayload{Filename: "backup.bin"},
			}},
			want: []string{"< Save As... >", closeButtonLabel},
		},
		{
			name: "credentials masked",
			state: recordViewState{status: recordViewReady, record: recordmodel.Record{
				Metadata: recordmodel.RecordMetadata{Type: recordmodel.RecordTypeCredentials},
				Payload:  &recordmodel.CredentialsPayload{Password: "secret"},
			}},
			want: []string{"< Reveal >", closeButtonLabel},
		},
		{
			name: "card revealed",
			state: recordViewState{status: recordViewReady, revealed: true, record: recordmodel.Record{
				Metadata: recordmodel.RecordMetadata{Type: recordmodel.RecordTypeCard},
				Payload:  &recordmodel.CardPayload{Number: "4111111111111111"},
			}},
			want: []string{"< Hide >", closeButtonLabel},
		},
		{
			name: "plain text",
			state: recordViewState{status: recordViewReady, record: recordmodel.Record{
				Metadata: recordmodel.RecordMetadata{Type: recordmodel.RecordTypeText},
				Payload:  &recordmodel.TextPayload{Text: "note"},
			}},
			want: []string{closeButtonLabel},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := recordViewButtonLabels(test.state); !slices.Equal(got, test.want) {
				t.Fatalf("recordViewButtonLabels() = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestWrapRecordViewText_NormalizesLineEndingsAndTabs(t *testing.T) {
	got := wrapRecordViewText("ab\r\n\tcd\ref", 4)
	want := []string{"ab", "    ", "cd", "ef"}
	if !slices.Equal(got, want) {
		t.Fatalf("wrapRecordViewText() = %#v, want %#v", got, want)
	}
}

func TestWrapRecordViewText_PreservesBlankParagraph(t *testing.T) {
	got := wrapRecordViewText("first\n\nlast", 8)
	want := []string{"first", "", "last"}
	if !slices.Equal(got, want) {
		t.Fatalf("wrapRecordViewText() = %#v, want %#v", got, want)
	}
}

func TestWrapRecordViewText_AccountsForWideRunes(t *testing.T) {
	got := wrapRecordViewText("界界a", 3)
	want := []string{"界", "界a"}
	if !slices.Equal(got, want) {
		t.Fatalf("wrapRecordViewText() = %#v, want %#v", got, want)
	}
}

func TestRecordViewTextAreaRange_FindsContiguousArea(t *testing.T) {
	lines := []recordViewLine{
		{label: "Title"},
		{textArea: true},
		{textArea: true},
		{label: "Notes"},
	}
	start, end, ok := recordViewTextAreaRange(lines)
	if start != 1 || end != 3 || !ok {
		t.Fatalf("recordViewTextAreaRange() = (%d, %d, %v), want (1, 3, true)", start, end, ok)
	}
}

func TestRecordViewTextAreaRange_ReportsMissingArea(t *testing.T) {
	start, end, ok := recordViewTextAreaRange([]recordViewLine{{label: "Title"}})
	if start != -1 || end != -1 || ok {
		t.Fatalf("recordViewTextAreaRange() = (%d, %d, %v), want (-1, -1, false)", start, end, ok)
	}
}

func TestRecordViewFormattingHelpers_HandleEmptyValues(t *testing.T) {
	if got := recordTypeTitle(""); got != "" {
		t.Fatalf("recordTypeTitle(empty) = %q", got)
	}
	if got := maskSecret(""); got != "" {
		t.Fatalf("maskSecret(empty) = %q", got)
	}
	if got := visibleCVV("", false); got != "" {
		t.Fatalf("visibleCVV(empty) = %q", got)
	}
	if got := formattedVisibleCardNumber("no digits", false); got != "" {
		t.Fatalf("formattedVisibleCardNumber(no digits) = %q", got)
	}
	if got := groupCardDigits(""); got != "" {
		t.Fatalf("groupCardDigits(empty) = %q", got)
	}
}

func TestRecordViewFormattingHelpers_FormatValues(t *testing.T) {
	if got := recordTypeTitle(recordmodel.RecordTypeCredentials); got != "Credentials" {
		t.Fatalf("recordTypeTitle() = %q, want Credentials", got)
	}
	if got := visibleSecret("secret", true); got != "secret" {
		t.Fatalf("visibleSecret(revealed) = %q", got)
	}
	if got := visibleSecret("x", false); got != "••••" {
		t.Fatalf("visibleSecret(masked) = %q", got)
	}
	if got := visibleCVV("014", false); got != "•••" {
		t.Fatalf("visibleCVV(masked) = %q", got)
	}
	if got := groupCardDigits("12345"); got != "1234 5" {
		t.Fatalf("groupCardDigits() = %q, want %q", got, "1234 5")
	}
}

func TestRecordViewButtonLabels_OfferRevealForSensitiveTypes(t *testing.T) {
	states := []recordViewState{
		{status: recordViewReady, record: recordmodel.Record{Metadata: recordmodel.RecordMetadata{Type: recordmodel.RecordTypeCredentials}, Payload: &recordmodel.CredentialsPayload{}}},
		{status: recordViewReady, record: recordmodel.Record{Metadata: recordmodel.RecordMetadata{Type: recordmodel.RecordTypeCard}, Payload: &recordmodel.CardPayload{}}},
	}
	for _, state := range states {
		if got := recordViewButtonLabels(state); !slices.Equal(got, []string{"< Reveal >", closeButtonLabel}) {
			t.Fatalf("recordViewButtonLabels() = %#v, want Reveal and Close", got)
		}
	}
}
