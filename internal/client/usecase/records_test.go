package usecase

import (
	"context"
	"errors"
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

func TestApplication_CreateCredentialsRecord(t *testing.T) {
	createdAt := time.Date(2026, time.July, 10, 12, 0, 0, 0, time.UTC)
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
				payload, ok := request.Payload.(*model.CredentialsPayload)
				if !ok {
					t.Fatalf("payload type = %T, want *model.CredentialsPayload", request.Payload)
				}
				if payload.Login != "alice" || payload.Password != "correct-horse-battery-staple" {
					t.Errorf("payload = %#v, want original credentials", payload)
				}

				return httpclient.Record{
					Metadata: model.RecordMetadata{
						ID:        testRecordID,
						Type:      model.RecordTypeCredentials,
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

	record, err := application.CreateCredentialsRecord(context.Background(), CreateCredentialsRecordRequest{
		Title:    "GitHub",
		Login:    "alice",
		Password: "correct-horse-battery-staple",
		URL:      "https://github.com",
		Metadata: "personal account",
	})
	if err != nil {
		t.Fatalf("CreateCredentialsRecord() error = %v", err)
	}
	if record.Metadata.Type != model.RecordTypeCredentials {
		t.Errorf("type = %q, want credentials", record.Metadata.Type)
	}
	if record.Payload.Password != "correct-horse-battery-staple" {
		t.Error("credentials password was not returned unchanged")
	}
}

func TestApplication_CreateCredentialsRecordValidationError(t *testing.T) {
	tests := []struct {
		name    string
		request CreateCredentialsRecordRequest
		want    error
	}{
		{
			name: "invalid title",
			request: CreateCredentialsRecordRequest{
				Title:    " ",
				Login:    "alice",
				Password: "correct-horse-battery-staple",
			},
			want: model.ErrInvalidRecordTitle,
		},
		{
			name: "empty login",
			request: CreateCredentialsRecordRequest{
				Title:    "GitHub",
				Login:    " ",
				Password: "correct-horse-battery-staple",
			},
			want: model.ErrInvalidCredentialsPayload,
		},
		{
			name: "empty password",
			request: CreateCredentialsRecordRequest{
				Title:    "GitHub",
				Login:    "alice",
				Password: "",
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

			_, err := application.CreateCredentialsRecord(context.Background(), tt.request)
			if !errors.Is(err, tt.want) {
				t.Fatalf("CreateCredentialsRecord() error = %v, want %v", err, tt.want)
			}
		})
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

func TestApplication_GetTextRecord(t *testing.T) {
	application := newApplicationWithRecords(
		nil,
		recordClientStub{
			get: func(context.Context, string, string) (httpclient.Record, error) {
				return httpclient.Record{
					Metadata: model.RecordMetadata{
						ID:   testRecordID,
						Type: model.RecordTypeText,
					},
					Payload: &model.TextPayload{Text: "secret note"},
				}, nil
			},
		},
		onlineSessionStorage(),
		"localhost:8080",
	)

	record, err := application.GetTextRecord(context.Background(), testRecordID)
	if err != nil {
		t.Fatalf("GetTextRecord() error = %v", err)
	}
	if record.Payload.Text != "secret note" {
		t.Errorf("text = %q, want secret note", record.Payload.Text)
	}
}

func TestApplication_UpdateCredentialsRecord(t *testing.T) {
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
				payload, ok := request.Payload.(*model.CredentialsPayload)
				if !ok || payload.Password != "updated-correct-horse-battery-staple" {
					t.Fatalf("payload = %#v, want updated credentials", request.Payload)
				}

				return httpclient.Record{
					Metadata: model.RecordMetadata{
						ID:       testRecordID,
						Type:     model.RecordTypeCredentials,
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

	record, err := application.UpdateCredentialsRecord(context.Background(), UpdateCredentialsRecordRequest{
		RecordID:         testRecordID,
		ExpectedRevision: 1,
		Title:            "Updated GitHub",
		Login:            "alice",
		Password:         "updated-correct-horse-battery-staple",
	})
	if err != nil {
		t.Fatalf("UpdateCredentialsRecord() error = %v", err)
	}
	if record.Metadata.Revision != 2 {
		t.Errorf("revision = %d, want 2", record.Metadata.Revision)
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

	_, err := application.UpdateCredentialsRecord(context.Background(), UpdateCredentialsRecordRequest{
		RecordID:         testRecordID,
		ExpectedRevision: 1,
		Title:            "GitHub",
		Login:            "alice",
		Password:         password,
	})
	if err == nil {
		t.Fatal("UpdateCredentialsRecord() error = nil, want conflict")
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
