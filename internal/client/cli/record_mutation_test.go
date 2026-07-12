package cli

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/client/usecase"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

func TestExecuteCreateRecord(t *testing.T) {
	payload := &model.TextPayload{Text: "private note", Metadata: "personal"}
	app := newApplicationStub(t)
	app.createRecord = func(_ context.Context, request usecase.CreateRecordRequest) (model.Record, error) {
		if request.Title != "Note" || !reflect.DeepEqual(request.Payload, payload) {
			t.Errorf("request = %#v, want title and payload", request)
		}
		return model.Record{Metadata: model.RecordMetadata{ID: testRecordID, Revision: 1}}, nil
	}
	var output bytes.Buffer

	if err := executeCreateRecord(context.Background(), app, &output, "Note", payload); err != nil {
		t.Fatalf("executeCreateRecord() error = %v", err)
	}
	want := "Created text record " + testRecordID + " with revision 1.\n"
	if output.String() != want {
		t.Errorf("output = %q, want %q", output.String(), want)
	}
}

func TestExecuteUpdateRecord(t *testing.T) {
	payload := &model.BinaryPayload{Filename: "backup.bin", Data: []byte("updated")}
	app := newApplicationStub(t)
	app.updateRecord = func(_ context.Context, request usecase.UpdateRecordRequest) (model.Record, error) {
		if request.RecordID != testRecordID || request.ExpectedRevision != 1 ||
			request.Title != "Updated backup" || !reflect.DeepEqual(request.Payload, payload) {
			t.Errorf("request = %#v, want update values", request)
		}
		return model.Record{Metadata: model.RecordMetadata{ID: testRecordID, Revision: 2}}, nil
	}
	var output bytes.Buffer

	err := executeUpdateRecord(context.Background(), app, &output, recordUpdateCommandRequest{
		recordID:         testRecordID,
		expectedRevision: 1,
		title:            "Updated backup",
		payload:          payload,
	})
	if err != nil {
		t.Fatalf("executeUpdateRecord() error = %v", err)
	}
	want := "Updated binary record " + testRecordID + " to revision 2.\n"
	if output.String() != want {
		t.Errorf("output = %q, want %q", output.String(), want)
	}
}

func TestExecuteRecordMutation_ReturnsApplicationError(t *testing.T) {
	wantErr := errors.New("application error")
	tests := []struct {
		name string
		run  func(application) error
	}{
		{
			name: "create",
			run: func(app application) error {
				return executeCreateRecord(
					context.Background(),
					app,
					&bytes.Buffer{},
					"Note",
					&model.TextPayload{Text: "private note"},
				)
			},
		},
		{
			name: "update",
			run: func(app application) error {
				return executeUpdateRecord(
					context.Background(),
					app,
					&bytes.Buffer{},
					recordUpdateCommandRequest{
						recordID:         testRecordID,
						expectedRevision: 1,
						title:            "Note",
						payload:          &model.TextPayload{Text: "private note"},
					},
				)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newApplicationStub(t)
			app.createRecord = func(context.Context, usecase.CreateRecordRequest) (model.Record, error) {
				return model.Record{}, wantErr
			}
			app.updateRecord = func(context.Context, usecase.UpdateRecordRequest) (model.Record, error) {
				return model.Record{}, wantErr
			}

			if err := tt.run(app); !errors.Is(err, wantErr) {
				t.Fatalf("mutation error = %v, want %v", err, wantErr)
			}
		})
	}
}
