package tui

import (
	"context"
	"errors"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/client/failure"
	recordmodel "github.com/xhrobj/gopherkeeper/internal/model"
)

type recordsBackendStub struct {
	backendStub
	listRecords func(context.Context) ([]recordmodel.RecordMetadata, error)
	getRecord   func(context.Context, string) (recordmodel.Record, error)
}

func (stub recordsBackendStub) ListRecords(ctx context.Context) ([]recordmodel.RecordMetadata, error) {
	if stub.listRecords == nil {
		return nil, errors.New("unexpected ListRecords call")
	}
	return stub.listRecords(ctx)
}

func (stub recordsBackendStub) GetRecord(ctx context.Context, id string) (recordmodel.Record, error) {
	if stub.getRecord == nil {
		return recordmodel.Record{}, errors.New("unexpected GetRecord call")
	}
	return stub.getRecord(ctx, id)
}

func TestOnlineRecordListCommand(t *testing.T) {
	want := []recordmodel.RecordMetadata{{ID: "42", Title: "Alice record"}}
	backend := recordsBackendStub{
		listRecords: func(context.Context) ([]recordmodel.RecordMetadata, error) {
			return want, nil
		},
	}

	message := onlineRecordListCommand(context.Background(), backend, 42)().(recordListResultMsg)
	if message.requestID != 42 || message.err != nil {
		t.Fatalf("message = %#v", message)
	}
	if len(message.records) != 1 || message.records[0].Title != "Alice record" {
		t.Fatalf("records = %#v", message.records)
	}
}

func TestCleanRecordListError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "unknown", want: "Unknown record list error"},
		{name: "canceled", err: context.Canceled, want: "Record loading canceled"},
		{name: "server refused", err: errors.New("list records: connection refused"), want: "Connection refused"},
		{name: "server prefix", err: failure.Context("list records", errors.New("access token is missing")), want: "Unable to load records from Server"},
		{name: "application prefix", err: failure.Context("create client application", errors.New("invalid config")), want: "Unable to load records from Server"},
		{name: "empty server", err: errors.New(""), want: "Unable to load records from Server"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := cleanRecordListError(test.err); got != test.want {
				t.Fatalf("error = %q, want %q", got, test.want)
			}
		})
	}
}
