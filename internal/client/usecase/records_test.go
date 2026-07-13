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
	payload := &model.TextPayload{Text: "secret text", Metadata: "personal"}
	createdAt := time.Date(2026, time.July, 12, 12, 0, 0, 0, time.UTC)
	application := newTestApplicationWithRecords(
		userGatewayStub{},
		recordGatewayStub{
			create: func(
				_ context.Context,
				accessToken string,
				title string,
				gotPayload model.RecordPayload,
			) (model.Record, error) {
				if accessToken != "test.jwt.token" {
					t.Errorf("access token = %q, want test.jwt.token", accessToken)
				}
				if title != "Private note" {
					t.Errorf("title = %q, want Private note", title)
				}
				if !reflect.DeepEqual(gotPayload, payload) {
					t.Errorf("payload = %#v, want %#v", gotPayload, payload)
				}

				return model.Record{
					Metadata: model.RecordMetadata{
						ID:        testRecordID,
						Type:      model.RecordTypeText,
						Title:     title,
						Revision:  1,
						CreatedAt: createdAt,
						UpdatedAt: createdAt,
					},
					Payload: gotPayload,
				}, nil
			},
		},
		onlineSessionStorage(),
		"localhost:8080",
	)

	record, err := application.CreateRecord(context.Background(), CreateRecordRequest{
		Title:   "Private note",
		Payload: payload,
	})
	if err != nil {
		t.Fatalf("CreateRecord() error = %v", err)
	}
	if !reflect.DeepEqual(record.Payload, payload) {
		t.Errorf("payload = %#v, want %#v", record.Payload, payload)
	}
}

func TestApplication_CreateRecordRejectsInvalidRequest(t *testing.T) {
	tests := []struct {
		name    string
		request CreateRecordRequest
		wantErr error
	}{
		{
			name: "invalid title",
			request: CreateRecordRequest{
				Title:   " ",
				Payload: &model.TextPayload{Text: "secret"},
			},
			wantErr: model.ErrInvalidRecordTitle,
		},
		{
			name:    "missing payload",
			request: CreateRecordRequest{Title: "Private note"},
			wantErr: errUnexpectedRecordPayload,
		},
		{
			name: "invalid payload",
			request: CreateRecordRequest{
				Title:   "GitHub",
				Payload: &model.CredentialsPayload{Login: "alice"},
			},
			wantErr: model.ErrInvalidCredentialsPayload,
		},
		{
			name: "typed nil payload",
			request: CreateRecordRequest{
				Title:   "Encrypted backup",
				Payload: (*model.BinaryPayload)(nil),
			},
			wantErr: model.ErrInvalidBinaryPayload,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			application := newTestApplicationWithRecords(
				userGatewayStub{},
				recordGatewayStub{
					create: func(context.Context, string, string, model.RecordPayload) (model.Record, error) {
						t.Fatal("record gateway must not be called")
						return model.Record{}, nil
					},
				},
				sessionStorageStub{},
				"localhost:8080",
			)

			_, err := application.CreateRecord(context.Background(), tt.request)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("CreateRecord() error = %v, want %v", err, tt.wantErr)
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
		userGatewayStub{},
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
	want := model.Record{
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
	}
	application := newTestApplicationWithRecords(
		userGatewayStub{},
		recordGatewayStub{
			get: func(_ context.Context, accessToken, recordID string) (model.Record, error) {
				if accessToken != "test.jwt.token" {
					t.Errorf("access token = %q, want test.jwt.token", accessToken)
				}
				if recordID != testRecordID {
					t.Errorf("record ID = %q, want %q", recordID, testRecordID)
				}
				return want, nil
			},
		},
		onlineSessionStorage(),
		"localhost:8080",
	)

	got, err := application.GetRecord(context.Background(), testRecordID)
	if err != nil {
		t.Fatalf("GetRecord() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("GetRecord() = %#v, want %#v", got, want)
	}
}

func TestApplication_UpdateRecord(t *testing.T) {
	payload := &model.BinaryPayload{
		Filename:    "backup-v2.bin",
		Data:        []byte{0xff, 0x02, 0x01, 0x00},
		ContentType: "application/octet-stream",
		Metadata:    "updated private backup",
	}
	application := newTestApplicationWithRecords(
		userGatewayStub{},
		recordGatewayStub{
			update: func(
				_ context.Context,
				accessToken, recordID string,
				expectedRevision int64,
				title string,
				gotPayload model.RecordPayload,
			) (model.Record, error) {
				if accessToken != "test.jwt.token" || recordID != testRecordID || expectedRevision != 1 {
					t.Error("update request contains unexpected common values")
				}
				if title != "Updated backup" || !reflect.DeepEqual(gotPayload, payload) {
					t.Errorf("update title = %q, payload = %#v", title, gotPayload)
				}

				return model.Record{
					Metadata: model.RecordMetadata{
						ID:       testRecordID,
						Type:     model.RecordTypeBinary,
						Title:    title,
						Revision: 2,
					},
					Payload: gotPayload,
				}, nil
			},
		},
		onlineSessionStorage(),
		"localhost:8080",
	)

	record, err := application.UpdateRecord(context.Background(), UpdateRecordRequest{
		RecordID:         testRecordID,
		ExpectedRevision: 1,
		Title:            "Updated backup",
		Payload:          payload,
	})
	if err != nil {
		t.Fatalf("UpdateRecord() error = %v", err)
	}
	if record.Metadata.Revision != 2 {
		t.Errorf("revision = %d, want 2", record.Metadata.Revision)
	}
	if !reflect.DeepEqual(record.Payload, payload) {
		t.Errorf("payload = %#v, want %#v", record.Payload, payload)
	}
}

func TestApplication_UpdateRecordMapsGatewayError(t *testing.T) {
	password := "updated-correct-horse-battery-staple"
	application := newTestApplicationWithRecords(
		userGatewayStub{},
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
		userGatewayStub{},
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
