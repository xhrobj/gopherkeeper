package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/client/httpclient"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

func TestApplication_CreateCardRecord(t *testing.T) {
	expiryMonth := 3
	expiryYear := 2038
	createdAt := time.Date(2026, time.July, 11, 12, 0, 0, 0, time.UTC)
	application := newApplicationWithRecords(
		nil,
		recordClientStub{
			create: func(
				_ context.Context,
				accessToken string,
				request httpclient.CreateRecordRequest,
			) (httpclient.Record, error) {
				if accessToken != "test.jwt.token" {
					t.Errorf("access token = %q, want test.jwt.token", accessToken)
				}
				payload, ok := request.Payload.(*model.CardPayload)
				if !ok {
					t.Fatalf("payload type = %T, want *model.CardPayload", request.Payload)
				}
				if payload.Number != "2013 0614 2020 0619" || payload.Cardholder != "Joel Miller" ||
					payload.ExpiryMonth == nil || *payload.ExpiryMonth != expiryMonth ||
					payload.ExpiryYear == nil || *payload.ExpiryYear != expiryYear ||
					payload.CVV != "014" || payload.Metadata != "test card" {
					t.Errorf("payload = %#v, want original card", payload)
				}

				return httpclient.Record{
					Metadata: model.RecordMetadata{
						ID:        testRecordID,
						Type:      model.RecordTypeCard,
						Title:     request.Title,
						Revision:  1,
						CreatedAt: createdAt,
						UpdatedAt: createdAt,
					},
					Payload: payload,
				}, nil
			},
		},
		onlineSessionStorage(),
		"localhost:8080",
	)

	record, err := application.CreateCardRecord(context.Background(), CreateCardRecordRequest{
		Title:       "Joel's card",
		Number:      "2013 0614 2020 0619",
		Cardholder:  "Joel Miller",
		ExpiryMonth: &expiryMonth,
		ExpiryYear:  &expiryYear,
		CVV:         "014",
		Metadata:    "test card",
	})
	if err != nil {
		t.Fatalf("CreateCardRecord() error = %v", err)
	}
	if record.Metadata.Type != model.RecordTypeCard || record.Payload.CVV != "014" {
		t.Errorf("record = %#v, want card with unchanged CVV", record)
	}
}

func TestApplication_UpdateCardRecord(t *testing.T) {
	expiryMonth := 3
	expiryYear := 2038
	application := newApplicationWithRecords(
		nil,
		recordClientStub{
			update: func(
				_ context.Context,
				accessToken, recordID string,
				expectedRevision int64,
				request httpclient.UpdateRecordRequest,
			) (httpclient.Record, error) {
				if accessToken != "test.jwt.token" || recordID != testRecordID || expectedRevision != 1 {
					t.Error("update request contains unexpected common values")
				}
				payload, ok := request.Payload.(*model.CardPayload)
				if !ok || payload.Number != "2013 0614 2020 0619" || payload.Cardholder != "Joel Miller" ||
					payload.CVV != "014" ||
					payload.ExpiryMonth == nil || *payload.ExpiryMonth != expiryMonth ||
					payload.ExpiryYear == nil || *payload.ExpiryYear != expiryYear ||
					payload.Metadata != "test card updated" {
					t.Fatalf("payload = %#v, want updated card", request.Payload)
				}

				return httpclient.Record{
					Metadata: model.RecordMetadata{
						ID:       testRecordID,
						Type:     model.RecordTypeCard,
						Title:    request.Title,
						Revision: 2,
					},
					Payload: payload,
				}, nil
			},
		},
		onlineSessionStorage(),
		"localhost:8080",
	)

	record, err := application.UpdateCardRecord(context.Background(), UpdateCardRecordRequest{
		RecordID:         testRecordID,
		ExpectedRevision: 1,
		Title:            "Joel's card updated",
		Number:           "2013 0614 2020 0619",
		Cardholder:       "Joel Miller",
		ExpiryMonth:      &expiryMonth,
		ExpiryYear:       &expiryYear,
		CVV:              "014",
		Metadata:         "test card updated",
	})
	if err != nil {
		t.Fatalf("UpdateCardRecord() error = %v", err)
	}
	if record.Metadata.Revision != 2 || record.Payload.CVV != "014" {
		t.Errorf("record = %#v, want revision 2 with unchanged CVV", record)
	}
}
