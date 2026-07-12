package usecase

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/client/session"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

const testRecordID = "550e8400-e29b-41d4-a716-446655440000"

type recordGatewayStub struct {
	create func(context.Context, string, string, model.RecordPayload) (model.Record, error)
	list   func(context.Context, string) ([]model.RecordMetadata, error)
	get    func(context.Context, string, string) (model.Record, error)
	update func(
		context.Context,
		string,
		string,
		int64,
		string,
		model.RecordPayload,
	) (model.Record, error)
	delete func(context.Context, string, string, int64) error
}

func (s recordGatewayStub) CreateRecord(
	ctx context.Context,
	accessToken string,
	title string,
	payload model.RecordPayload,
) (model.Record, error) {
	return s.create(ctx, accessToken, title, payload)
}

func (s recordGatewayStub) ListRecords(ctx context.Context, accessToken string) ([]model.RecordMetadata, error) {
	return s.list(ctx, accessToken)
}

func (s recordGatewayStub) GetRecord(
	ctx context.Context,
	accessToken, recordID string,
) (model.Record, error) {
	return s.get(ctx, accessToken, recordID)
}

func (s recordGatewayStub) UpdateRecord(
	ctx context.Context,
	accessToken, recordID string,
	expectedRevision int64,
	title string,
	payload model.RecordPayload,
) (model.Record, error) {
	return s.update(ctx, accessToken, recordID, expectedRevision, title, payload)
}

func (s recordGatewayStub) DeleteRecord(
	ctx context.Context,
	accessToken, recordID string,
	expectedRevision int64,
) error {
	return s.delete(ctx, accessToken, recordID, expectedRevision)
}

func TestApplication_CreateRecord(t *testing.T) {
	expiryMonth := 3
	expiryYear := 2038
	binaryData := []byte{0x00, 0x01, 0x02, 0xff}
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
		{
			name:  "binary",
			title: "Encrypted backup",
			payload: &model.BinaryPayload{
				Filename:    "backup.bin",
				Data:        binaryData,
				ContentType: "application/octet-stream",
				Metadata:    "private backup",
			},
		},
		{
			name:  "empty binary",
			title: "Empty backup",
			payload: &model.BinaryPayload{
				Filename: "empty.bin",
				Data:     []byte{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			createdAt := time.Date(2026, time.July, 11, 12, 0, 0, 0, time.UTC)
			application := newTestApplicationWithRecords(
				userClientStub{},
				recordGatewayStub{
					create: func(
						_ context.Context,
						accessToken string,
						title string,
						payload model.RecordPayload,
					) (model.Record, error) {
						if accessToken != "test.jwt.token" {
							t.Errorf("access token = %q, want test.jwt.token", accessToken)
						}
						if title != tt.title {
							t.Errorf("title = %q, want %q", title, tt.title)
						}
						if !reflect.DeepEqual(payload, tt.payload) {
							t.Errorf("payload = %#v, want %#v", payload, tt.payload)
						}

						return model.Record{
							Metadata: model.RecordMetadata{
								ID:        testRecordID,
								Type:      payload.RecordType(),
								Title:     title,
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
		{
			name: "typed nil binary payload",
			request: CreateRecordRequest{
				Title:   "Encrypted backup",
				Payload: (*model.BinaryPayload)(nil),
			},
			want: model.ErrInvalidBinaryPayload,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			application := newTestApplicationWithRecords(userClientStub{}, recordGatewayStub{
				create: func(context.Context, string, string, model.RecordPayload) (model.Record, error) {
					t.Fatal("record client must not be called")
					return model.Record{}, nil
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
	application := newTestApplicationWithRecords(
		userClientStub{},
		recordGatewayStub{
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
	application := newTestApplicationWithRecords(
		userClientStub{},
		recordGatewayStub{
			get: func(_ context.Context, accessToken, recordID string) (model.Record, error) {
				if accessToken != "test.jwt.token" {
					t.Errorf("access token = %q, want test.jwt.token", accessToken)
				}
				if recordID != testRecordID {
					t.Errorf("record ID = %q, want %q", recordID, testRecordID)
				}

				return model.Record{
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

func TestApplication_GetBinaryRecord(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{name: "binary", data: []byte{0x00, 0x01, 0x02, 0xff}},
		{name: "empty binary", data: []byte{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			application := newTestApplicationWithRecords(
				userClientStub{},
				recordGatewayStub{
					get: func(_ context.Context, accessToken, recordID string) (model.Record, error) {
						if accessToken != "test.jwt.token" {
							t.Errorf("access token = %q, want test.jwt.token", accessToken)
						}
						if recordID != testRecordID {
							t.Errorf("record ID = %q, want %q", recordID, testRecordID)
						}

						return model.Record{
							Metadata: model.RecordMetadata{
								ID:       testRecordID,
								Type:     model.RecordTypeBinary,
								Title:    "Encrypted backup",
								Revision: 1,
							},
							Payload: &model.BinaryPayload{
								Filename:    "backup.bin",
								Data:        tt.data,
								ContentType: "application/octet-stream",
								Metadata:    "private backup",
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
			payload, ok := record.Payload.(*model.BinaryPayload)
			if !ok {
				t.Fatalf("payload = %#v, want binary payload", record.Payload)
			}
			if payload.Filename != "backup.bin" || payload.ContentType != "application/octet-stream" ||
				payload.Metadata != "private backup" {
				t.Errorf("payload metadata = %#v, want binary metadata", payload)
			}
			if !reflect.DeepEqual(payload.Data, tt.data) {
				t.Errorf("payload data = %v, want %v", payload.Data, tt.data)
			}
		})
	}
}

func TestApplication_GetRecordRejectsTypedNilPayload(t *testing.T) {
	var payload *model.TextPayload
	application := newTestApplicationWithRecords(
		userClientStub{},
		recordGatewayStub{
			get: func(context.Context, string, string) (model.Record, error) {
				return model.Record{
					Metadata: model.RecordMetadata{Type: model.RecordTypeText},
					Payload:  payload,
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

func TestApplication_GetRecordRejectsMismatchedPayload(t *testing.T) {
	application := newTestApplicationWithRecords(
		userClientStub{},
		recordGatewayStub{
			get: func(context.Context, string, string) (model.Record, error) {
				return model.Record{
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
	binaryData := []byte{0xff, 0x02, 0x01, 0x00}
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
		{
			name:  "binary",
			title: "Updated backup",
			payload: &model.BinaryPayload{
				Filename:    "backup-v2.bin",
				Data:        binaryData,
				ContentType: "application/octet-stream",
				Metadata:    "updated private backup",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			application := newTestApplicationWithRecords(
				userClientStub{},
				recordGatewayStub{
					update: func(
						_ context.Context,
						accessToken, recordID string,
						expectedRevision int64,
						title string,
						payload model.RecordPayload,
					) (model.Record, error) {
						if accessToken != "test.jwt.token" || recordID != testRecordID || expectedRevision != 1 {
							t.Error("update request contains unexpected common values")
						}
						if title != tt.title || !reflect.DeepEqual(payload, tt.payload) {
							t.Errorf("update title = %q, payload = %#v, want %q and %#v", title, payload, tt.title, tt.payload)
						}

						return model.Record{
							Metadata: model.RecordMetadata{
								ID:       testRecordID,
								Type:     payload.RecordType(),
								Title:    title,
								Revision: 2,
							},
							Payload: payload,
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
	application := newTestApplicationWithRecords(
		userClientStub{},
		recordGatewayStub{
			update: func(context.Context, string, string, int64, string, model.RecordPayload) (model.Record, error) {
				return model.Record{}, fmt.Errorf("remote update: %w", model.ErrRecordRevisionConflict)
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
	application := newTestApplicationWithRecords(
		userClientStub{},
		recordGatewayStub{
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
