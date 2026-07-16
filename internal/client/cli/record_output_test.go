package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

func TestWriteRecord_DisplaysPayload(t *testing.T) {
	expiryMonth := 3
	expiryYear := 2038
	recordedAt := time.Date(2026, time.July, 12, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		title   string
		payload model.RecordPayload
		wants   []string
	}{
		{
			name:    "text",
			title:   "Note",
			payload: &model.TextPayload{Text: "private note", Metadata: "personal"},
			wants:   []string{"Type: text", "Text:\nprivate note", "Metadata:\npersonal"},
		},
		{
			name:  "credentials",
			title: "GitHub",
			payload: &model.CredentialsPayload{
				Login:    "alice",
				Password: testCredentialsPassword,
				URL:      "https://github.com",
				Metadata: "recovery codes",
			},
			wants: []string{
				"Type: credentials",
				"Login: alice",
				"Password: " + testCredentialsPassword,
				"URL: https://github.com",
				"Metadata:\nrecovery codes",
			},
		},
		{
			name:  "card",
			title: "Joel's card",
			payload: &model.CardPayload{
				Number:      testCardNumber,
				Cardholder:  "Joel Miller",
				ExpiryMonth: &expiryMonth,
				ExpiryYear:  &expiryYear,
				CVV:         testCardCVV,
				Metadata:    "test card",
			},
			wants: []string{
				"Type: card",
				"Number: " + testCardNumber,
				"Cardholder: Joel Miller",
				"Expiry: 03/2038",
				"CVV: " + testCardCVV,
				"Metadata:\ntest card",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := model.Record{
				Metadata: model.RecordMetadata{
					ID:        testRecordID,
					Type:      tt.payload.RecordType(),
					Title:     tt.title,
					Revision:  1,
					CreatedAt: recordedAt,
					UpdatedAt: recordedAt,
				},
				Payload: tt.payload,
			}
			var output bytes.Buffer

			if err := writeRecord(&output, record, ""); err != nil {
				t.Fatalf("writeRecord() error = %v", err)
			}
			for _, want := range tt.wants {
				if !strings.Contains(output.String(), want) {
					t.Errorf("output = %q, want %q", output.String(), want)
				}
			}
		})
	}
}

func TestWriteRecordList(t *testing.T) {
	updatedAt := time.Date(2026, time.July, 15, 12, 0, 0, 0, time.UTC)
	records := []model.RecordMetadata{
		{
			ID:        testRecordID,
			Type:      model.RecordTypeText,
			Title:     "Note",
			Revision:  2,
			UpdatedAt: updatedAt,
		},
	}

	var output bytes.Buffer
	if err := writeRecordList(&output, records); err != nil {
		t.Fatalf("writeRecordList() error = %v", err)
	}

	for _, want := range []string{
		"TYPE",
		"TITLE",
		"REVISION",
		"UPDATED AT",
		testRecordID,
		"text",
		"Note",
		"2",
		"2026-07-15T12:00:00Z",
	} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("output = %q, want %q", output.String(), want)
		}
	}
}

func TestWriteRecordList_Empty(t *testing.T) {
	var output bytes.Buffer
	if err := writeRecordList(&output, nil); err != nil {
		t.Fatalf("writeRecordList() error = %v", err)
	}
	if output.String() != "No records found.\n" {
		t.Errorf("output = %q, want empty-list message", output.String())
	}
}
