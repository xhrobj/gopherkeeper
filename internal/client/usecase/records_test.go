package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/client/httpclient"
	"github.com/xhrobj/gopherkeeper/internal/client/session"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

const testRecordID = "550e8400-e29b-41d4-a716-446655440000"

type recordClientStub struct {
	create func(context.Context, string, httpclient.CreateTextRecordRequest) (httpclient.TextRecord, error)
	list   func(context.Context, string) ([]model.RecordMetadata, error)
	get    func(context.Context, string, string) (httpclient.TextRecord, error)
	update func(
		context.Context,
		string,
		string,
		int64,
		httpclient.UpdateTextRecordRequest,
	) (httpclient.TextRecord, error)
	delete func(context.Context, string, string, int64) error
}

func (s recordClientStub) CreateTextRecord(
	ctx context.Context,
	accessToken string,
	request httpclient.CreateTextRecordRequest,
) (httpclient.TextRecord, error) {
	return s.create(ctx, accessToken, request)
}

func (s recordClientStub) ListRecords(ctx context.Context, accessToken string) ([]model.RecordMetadata, error) {
	return s.list(ctx, accessToken)
}

func (s recordClientStub) GetTextRecord(
	ctx context.Context,
	accessToken string,
	recordID string,
) (httpclient.TextRecord, error) {
	return s.get(ctx, accessToken, recordID)
}

func (s recordClientStub) UpdateTextRecord(
	ctx context.Context,
	accessToken string,
	recordID string,
	expectedRevision int64,
	request httpclient.UpdateTextRecordRequest,
) (httpclient.TextRecord, error) {
	return s.update(ctx, accessToken, recordID, expectedRevision, request)
}

func (s recordClientStub) DeleteRecord(
	ctx context.Context,
	accessToken string,
	recordID string,
	expectedRevision int64,
) error {
	return s.delete(ctx, accessToken, recordID, expectedRevision)
}

func TestApplication_UpdateTextRecord(t *testing.T) {
	updatedAt := time.Date(2026, time.July, 9, 12, 5, 0, 0, time.UTC)
	application := newApplicationWithRecords(
		nil,
		recordClientStub{
			update: func(
				_ context.Context,
				accessToken string,
				recordID string,
				expectedRevision int64,
				request httpclient.UpdateTextRecordRequest,
			) (httpclient.TextRecord, error) {
				if accessToken != "test.jwt.token" {
					t.Errorf("access token = %q, want test.jwt.token", accessToken)
				}
				if recordID != testRecordID {
					t.Errorf("record ID = %q, want %q", recordID, testRecordID)
				}
				if expectedRevision != 1 {
					t.Errorf("expected revision = %d, want 1", expectedRevision)
				}
				if request.Title != "Updated note" {
					t.Errorf("title = %q, want Updated note", request.Title)
				}
				if request.Payload.Text != "updated secret" {
					t.Errorf("text = %q, want updated secret", request.Payload.Text)
				}
				if request.Payload.Metadata != "updated private metadata" {
					t.Errorf("metadata = %q, want updated private metadata", request.Payload.Metadata)
				}

				return httpclient.TextRecord{
					Metadata: model.RecordMetadata{
						ID:        testRecordID,
						Type:      model.RecordTypeText,
						Title:     request.Title,
						Revision:  2,
						UpdatedAt: updatedAt,
					},
					Payload: request.Payload,
				}, nil
			},
		},
		sessionStorageStub{
			load: func(expectedServerAddress string) (session.Session, error) {
				if expectedServerAddress != "localhost:8080" {
					t.Errorf("expected server address = %q, want localhost:8080", expectedServerAddress)
				}
				return testOnlineSession(), nil
			},
		},
		"localhost:8080",
	)

	record, err := application.UpdateTextRecord(context.Background(), UpdateTextRecordRequest{
		RecordID:         testRecordID,
		ExpectedRevision: 1,
		Title:            "Updated note",
		Text:             "updated secret",
		Metadata:         "updated private metadata",
	})
	if err != nil {
		t.Fatalf("UpdateTextRecord() error = %v", err)
	}
	if record.Metadata.Revision != 2 {
		t.Errorf("revision = %d, want 2", record.Metadata.Revision)
	}
	if record.Payload.Text != "updated secret" {
		t.Errorf("text = %q, want updated secret", record.Payload.Text)
	}
}

func TestApplication_UpdateTextRecordValidationError(t *testing.T) {
	tests := []struct {
		name    string
		request UpdateTextRecordRequest
		want    error
	}{
		{
			name: "invalid record ID",
			request: UpdateTextRecordRequest{
				RecordID:         "not-a-uuid",
				ExpectedRevision: 1,
				Title:            "Updated note",
				Text:             "updated secret",
			},
			want: model.ErrInvalidRecordID,
		},
		{
			name: "invalid revision",
			request: UpdateTextRecordRequest{
				RecordID:         testRecordID,
				ExpectedRevision: 0,
				Title:            "Updated note",
				Text:             "updated secret",
			},
			want: model.ErrInvalidRecordRevision,
		},
		{
			name: "invalid title",
			request: UpdateTextRecordRequest{
				RecordID:         testRecordID,
				ExpectedRevision: 1,
				Title:            " ",
				Text:             "updated secret",
			},
			want: model.ErrInvalidRecordTitle,
		},
		{
			name: "invalid payload",
			request: UpdateTextRecordRequest{
				RecordID:         testRecordID,
				ExpectedRevision: 1,
				Title:            "Updated note",
				Text:             "",
			},
			want: model.ErrInvalidTextPayload,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			application := newApplicationWithRecords(nil, recordClientStub{
				update: func(context.Context, string, string, int64, httpclient.UpdateTextRecordRequest) (httpclient.TextRecord, error) {
					t.Fatal("record client must not be called")
					return httpclient.TextRecord{}, nil
				},
			}, sessionStorageStub{}, "localhost:8080")

			_, err := application.UpdateTextRecord(context.Background(), tt.request)
			if !errors.Is(err, tt.want) {
				t.Fatalf("UpdateTextRecord() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestApplication_UpdateTextRecordMapsAPIError(t *testing.T) {
	application := newApplicationWithRecords(nil, recordClientStub{
		update: func(context.Context, string, string, int64, httpclient.UpdateTextRecordRequest) (httpclient.TextRecord, error) {
			return httpclient.TextRecord{}, &httpclient.APIError{
				StatusCode: 409,
				Code:       "record_revision_conflict",
				Message:    "record revision conflict",
			}
		},
	}, sessionStorageStub{
		load: func(string) (session.Session, error) {
			return testOnlineSession(), nil
		},
	}, "localhost:8080")

	_, err := application.UpdateTextRecord(context.Background(), UpdateTextRecordRequest{
		RecordID:         testRecordID,
		ExpectedRevision: 1,
		Title:            "Updated note",
		Text:             "updated secret",
	})
	if err == nil {
		t.Fatal("UpdateTextRecord() error = nil, want conflict")
	}
	if err.Error() != "record revision conflict" {
		t.Fatalf("UpdateTextRecord() error = %q, want record revision conflict", err)
	}
}

func TestApplication_DeleteRecord(t *testing.T) {
	application := newApplicationWithRecords(nil, recordClientStub{
		delete: func(_ context.Context, accessToken string, recordID string, expectedRevision int64) error {
			if accessToken != "test.jwt.token" {
				t.Errorf("access token = %q, want test.jwt.token", accessToken)
			}
			if recordID != testRecordID {
				t.Errorf("record ID = %q, want %q", recordID, testRecordID)
			}
			if expectedRevision != 2 {
				t.Errorf("expected revision = %d, want 2", expectedRevision)
			}
			return nil
		},
	}, sessionStorageStub{
		load: func(string) (session.Session, error) {
			return testOnlineSession(), nil
		},
	}, "localhost:8080")

	if err := application.DeleteRecord(context.Background(), DeleteRecordRequest{
		RecordID:         testRecordID,
		ExpectedRevision: 2,
	}); err != nil {
		t.Fatalf("DeleteRecord() error = %v", err)
	}
}

func TestApplication_DeleteRecordValidationError(t *testing.T) {
	tests := []struct {
		name    string
		request DeleteRecordRequest
		want    error
	}{
		{
			name: "invalid record ID",
			request: DeleteRecordRequest{
				RecordID:         "not-a-uuid",
				ExpectedRevision: 2,
			},
			want: model.ErrInvalidRecordID,
		},
		{
			name: "invalid revision",
			request: DeleteRecordRequest{
				RecordID:         testRecordID,
				ExpectedRevision: 0,
			},
			want: model.ErrInvalidRecordRevision,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			application := newApplicationWithRecords(nil, recordClientStub{
				delete: func(context.Context, string, string, int64) error {
					t.Fatal("record client must not be called")
					return nil
				},
			}, sessionStorageStub{}, "localhost:8080")

			err := application.DeleteRecord(context.Background(), tt.request)
			if !errors.Is(err, tt.want) {
				t.Fatalf("DeleteRecord() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestApplication_DeleteRecordMapsAPIError(t *testing.T) {
	application := newApplicationWithRecords(nil, recordClientStub{
		delete: func(context.Context, string, string, int64) error {
			return &httpclient.APIError{
				StatusCode: 404,
				Code:       "record_not_found",
				Message:    "record not found",
			}
		},
	}, sessionStorageStub{
		load: func(string) (session.Session, error) {
			return testOnlineSession(), nil
		},
	}, "localhost:8080")

	err := application.DeleteRecord(context.Background(), DeleteRecordRequest{
		RecordID:         testRecordID,
		ExpectedRevision: 2,
	})
	if err == nil {
		t.Fatal("DeleteRecord() error = nil, want not found")
	}
	if err.Error() != "record not found" {
		t.Fatalf("DeleteRecord() error = %q, want record not found", err)
	}
}
