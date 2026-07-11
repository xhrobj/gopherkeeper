package usecase

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/client/httpclient"
	"github.com/xhrobj/gopherkeeper/internal/client/session"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

const testRecordID = "550e8400-e29b-41d4-a716-446655440000"

type recordClientStub struct {
	create func(context.Context, string, httpclient.CreateRecordRequest) (httpclient.Record, error)
	list   func(context.Context, string) ([]model.RecordMetadata, error)
	get    func(context.Context, string, string) (httpclient.Record, error)
	update func(
		context.Context,
		string,
		string,
		int64,
		httpclient.UpdateRecordRequest,
	) (httpclient.Record, error)
	delete func(context.Context, string, string, int64) error
}

func (s recordClientStub) CreateRecord(
	ctx context.Context,
	accessToken string,
	request httpclient.CreateRecordRequest,
) (httpclient.Record, error) {
	return s.create(ctx, accessToken, request)
}

func (s recordClientStub) ListRecords(ctx context.Context, accessToken string) ([]model.RecordMetadata, error) {
	return s.list(ctx, accessToken)
}

func (s recordClientStub) GetRecord(
	ctx context.Context,
	accessToken, recordID string,
) (httpclient.Record, error) {
	return s.get(ctx, accessToken, recordID)
}

func (s recordClientStub) UpdateRecord(
	ctx context.Context,
	accessToken, recordID string,
	expectedRevision int64,
	request httpclient.UpdateRecordRequest,
) (httpclient.Record, error) {
	return s.update(ctx, accessToken, recordID, expectedRevision, request)
}

func (s recordClientStub) DeleteRecord(
	ctx context.Context,
	accessToken, recordID string,
	expectedRevision int64,
) error {
	return s.delete(ctx, accessToken, recordID, expectedRevision)
}

func TestApplication_CreateRecord(t *testing.T) {
	expiryMonth := 3
	expiryYear := 2038
	tests := []struct {
		name    string
		title   string
		payload model.RecordPayload
	}{
		{
			name:  "text",
			title: "Private note",
			payload: &model.TextPayload{
				Text:     "secret text",
				Metadata: "personal",
			},
		},
		{
			name:  "credentials",
			title: "GitHub",
			payload: &model.CredentialsPayload{
				Login:    "alice",
				Password: "correct-horse-battery-staple",
				URL:      "https://github.com",
				Metadata: "personal account",
			},
		},
		{
			name:  "card",
			title: "Joel's card",
			payload: &model.CardPayload{
				Number:      "2013 0614 2020 0619",
				Cardholder:  "Joel Miller",
				ExpiryMonth: &expiryMonth,
				ExpiryYear:  &expiryYear,
				CVV:         "014",
				Metadata:    "test card",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
						if request.Title != tt.title {
							t.Errorf("title = %q, want %q", request.Title, tt.title)
						}
						if !reflect.DeepEqual(request.Payload, tt.payload) {
							t.Errorf("payload = %#v, want %#v", request.Payload, tt.payload)
						}

						return httpclient.Record{
							Metadata: model.RecordMetadata{
								ID:        testRecordID,
								Type:      request.Payload.RecordType(),
								Title:     request.Title,
								Revision:  1,
								CreatedAt: createdAt,
								UpdatedAt: createdAt,
							},
							Payload: request.Payload,
						}, nil
					},
				},
				onlineSessionStorage(),
				"localhost:8080",
			)

			record, err := application.CreateRecord(context.Background(), CreateRecordRequest{
				Title:   tt.title,
				Payload: tt.payload,
			})
			if err != nil {
				t.Fatalf("CreateRecord() error = %v", err)
			}
			if record.Metadata.Type != tt.payload.RecordType() {
				t.Errorf("type = %q, want %q", record.Metadata.Type, tt.payload.RecordType())
			}
			if !reflect.DeepEqual(record.Payload, tt.payload) {
				t.Errorf("payload = %#v, want %#v", record.Payload, tt.payload)
			}
		})
	}
}

func TestApplication_CreateRecordValidationError(t *testing.T) {
	tests := []struct {
		name    string
		request CreateRecordRequest
		want    error
	}{
		{
			name: "invalid title",
			request: CreateRecordRequest{
				Title:   " ",
				Payload: &model.TextPayload{Text: "secret"},
			},
			want: model.ErrInvalidRecordTitle,
		},
		{
			name: "missing payload",
			request: CreateRecordRequest{
				Title: "Private note",
			},
			want: errUnexpectedRecordPayload,
		},
		{
			name: "invalid payload",
			request: CreateRecordRequest{
				Title:   "GitHub",
				Payload: &model.CredentialsPayload{Login: "alice"},
			},
			want: model.ErrInvalidCredentialsPayload,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			application := newApplicationWithRecords(nil, recordClientStub{
				create: func(context.Context, string, httpclient.CreateRecordRequest) (httpclient.Record, error) {
					t.Fatal("record client must not be called")
					return httpclient.Record{}, nil
				},
			}, sessionStorageStub{}, "localhost:8080")

			_, err := application.CreateRecord(context.Background(), tt.request)
			if !errors.Is(err, tt.want) {
				t.Fatalf("CreateRecord() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestApplication_ListRecords(t *testing.T) {
	want := []model.RecordMetadata{{
		ID:       testRecordID,
		Type:     model.RecordTypeText,
		Title:    "Private note",
		Revision: 1,
	}}
	application := newApplicationWithRecords(
		nil,
		recordClientStub{
			list: func(_ context.Context, accessToken string) ([]model.RecordMetadata, error) {
				if accessToken != "test.jwt.token" {
					t.Errorf("access token = %q, want test.jwt.token", accessToken)
				}

				return want, nil
			},
		},
		onlineSessionStorage(),
		"localhost:8080",
	)

	got, err := application.ListRecords(context.Background())
	if err != nil {
		t.Fatalf("ListRecords() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ListRecords() = %#v, want %#v", got, want)
	}
}

func TestApplication_GetRecord(t *testing.T) {
	application := newApplicationWithRecords(
		nil,
		recordClientStub{
			get: func(_ context.Context, accessToken, recordID string) (httpclient.Record, error) {
				if accessToken != "test.jwt.token" {
					t.Errorf("access token = %q, want test.jwt.token", accessToken)
				}
				if recordID != testRecordID {
					t.Errorf("record ID = %q, want %q", recordID, testRecordID)
				}

				return httpclient.Record{
					Metadata: model.RecordMetadata{
						ID:       testRecordID,
						Type:     model.RecordTypeCredentials,
						Title:    "GitHub",
						Revision: 1,
					},
					Payload: &model.CredentialsPayload{
						Login:    "alice",
						Password: "correct-horse-battery-staple",
					},
				}, nil
			},
		},
		onlineSessionStorage(),
		"localhost:8080",
	)

	record, err := application.GetRecord(context.Background(), testRecordID)
	if err != nil {
		t.Fatalf("GetRecord() error = %v", err)
	}
	payload, ok := record.Payload.(*model.CredentialsPayload)
	if !ok || payload.Login != "alice" {
		t.Fatalf("payload = %#v, want credentials payload", record.Payload)
	}
}

func TestApplication_GetRecordRejectsMismatchedPayload(t *testing.T) {
	application := newApplicationWithRecords(
		nil,
		recordClientStub{
			get: func(context.Context, string, string) (httpclient.Record, error) {
				return httpclient.Record{
					Metadata: model.RecordMetadata{Type: model.RecordTypeText},
					Payload:  &model.CredentialsPayload{Login: "alice", Password: "secret"},
				}, nil
			},
		},
		onlineSessionStorage(),
		"localhost:8080",
	)

	_, err := application.GetRecord(context.Background(), testRecordID)
	if !errors.Is(err, errUnexpectedRecordPayload) {
		t.Fatalf("GetRecord() error = %v, want unexpected payload", err)
	}
}

func TestApplication_UpdateRecord(t *testing.T) {
	expiryMonth := 3
	expiryYear := 2038
	tests := []struct {
		name    string
		title   string
		payload model.RecordPayload
	}{
		{
			name:    "text",
			title:   "Updated note",
			payload: &model.TextPayload{Text: "updated secret", Metadata: "updated"},
		},
		{
			name:  "credentials",
			title: "Updated GitHub",
			payload: &model.CredentialsPayload{
				Login:    "alice",
				Password: "updated-correct-horse-battery-staple",
			},
		},
		{
			name:  "card",
			title: "Joel's card updated",
			payload: &model.CardPayload{
				Number:      "2013 0614 2020 0619",
				Cardholder:  "Joel Miller",
				ExpiryMonth: &expiryMonth,
				ExpiryYear:  &expiryYear,
				CVV:         "014",
				Metadata:    "test card updated",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
						if request.Title != tt.title || !reflect.DeepEqual(request.Payload, tt.payload) {
							t.Errorf("update request = %#v, want title and payload unchanged", request)
						}

						return httpclient.Record{
							Metadata: model.RecordMetadata{
								ID:       testRecordID,
								Type:     request.Payload.RecordType(),
								Title:    request.Title,
								Revision: 2,
							},
							Payload: request.Payload,
						}, nil
					},
				},
				onlineSessionStorage(),
				"localhost:8080",
			)

			record, err := application.UpdateRecord(context.Background(), UpdateRecordRequest{
				RecordID:         testRecordID,
				ExpectedRevision: 1,
				Title:            tt.title,
				Payload:          tt.payload,
			})
			if err != nil {
				t.Fatalf("UpdateRecord() error = %v", err)
			}
			if record.Metadata.Revision != 2 {
				t.Errorf("revision = %d, want 2", record.Metadata.Revision)
			}
			if !reflect.DeepEqual(record.Payload, tt.payload) {
				t.Errorf("payload = %#v, want %#v", record.Payload, tt.payload)
			}
		})
	}
}

func TestApplication_UpdateRecordMapsAPIError(t *testing.T) {
	password := "updated-correct-horse-battery-staple"
	application := newApplicationWithRecords(
		nil,
		recordClientStub{
			update: func(context.Context, string, string, int64, httpclient.UpdateRecordRequest) (httpclient.Record, error) {
				return httpclient.Record{}, &httpclient.APIError{
					StatusCode: 409,
					Code:       "record_revision_conflict",
					Message:    "record revision conflict",
				}
			},
		},
		onlineSessionStorage(),
		"localhost:8080",
	)

	_, err := application.UpdateRecord(context.Background(), UpdateRecordRequest{
		RecordID:         testRecordID,
		ExpectedRevision: 1,
		Title:            "GitHub",
		Payload: &model.CredentialsPayload{
			Login:    "alice",
			Password: password,
		},
	})
	if err == nil {
		t.Fatal("UpdateRecord() error = nil, want conflict")
	}
	if err.Error() != "record revision conflict" {
		t.Fatalf("error = %q, want record revision conflict", err)
	}
	if strings.Contains(err.Error(), password) {
		t.Error("error contains credentials password")
	}
}

func TestApplication_DeleteRecord(t *testing.T) {
	application := newApplicationWithRecords(
		nil,
		recordClientStub{
			delete: func(_ context.Context, accessToken, recordID string, expectedRevision int64) error {
				if accessToken != "test.jwt.token" || recordID != testRecordID || expectedRevision != 2 {
					t.Error("delete request contains unexpected values")
				}
				return nil
			},
		},
		onlineSessionStorage(),
		"localhost:8080",
	)

	if err := application.DeleteRecord(context.Background(), DeleteRecordRequest{
		RecordID:         testRecordID,
		ExpectedRevision: 2,
	}); err != nil {
		t.Fatalf("DeleteRecord() error = %v", err)
	}
}

func onlineSessionStorage() sessionStorageStub {
	return sessionStorageStub{
		load: func(expectedServerAddress string) (session.Session, error) {
			if expectedServerAddress != "localhost:8080" {
				return session.Session{}, errors.New("unexpected server address")
			}

			return testOnlineSession(), nil
		},
	}
}
