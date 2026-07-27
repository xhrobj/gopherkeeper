package grpcclient

import (
	"context"
	"errors"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/client/failure"
	"github.com/xhrobj/gopherkeeper/internal/model"
	gopherkeeperpb "github.com/xhrobj/gopherkeeper/internal/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestClient_CreateRecord(t *testing.T) {
	client := &Client{records: recordClientStub{create: func(
		ctx context.Context,
		request *gopherkeeperpb.CreateRecordRequest,
		_ ...grpc.CallOption,
	) (*gopherkeeperpb.Record, error) {
		assertBearerToken(t, ctx, "token")
		if request.GetTitle() != "Example" {
			t.Fatalf("CreateRecord() title = %q", request.GetTitle())
		}
		if request.GetPayload().GetText().GetText() != "secret" {
			t.Fatalf("CreateRecord() text = %q", request.GetPayload().GetText().GetText())
		}
		return protoTextRecord(), nil
	}}}

	record, err := client.CreateRecord(
		context.Background(),
		"token",
		"Example",
		&model.TextPayload{Text: "secret", Metadata: "notes"},
	)
	if err != nil {
		t.Fatalf("CreateRecord() error = %v", err)
	}
	if record.Metadata.ID != testRecordID || record.Payload.(*model.TextPayload).Text != "secret" {
		t.Fatalf("CreateRecord() record = %+v", record)
	}
}

func TestClient_ListRecords(t *testing.T) {
	client := &Client{records: recordClientStub{list: func(
		ctx context.Context,
		_ *gopherkeeperpb.ListRecordsRequest,
		_ ...grpc.CallOption,
	) (*gopherkeeperpb.ListRecordsResponse, error) {
		assertBearerToken(t, ctx, "token")
		response := &gopherkeeperpb.ListRecordsResponse{}
		response.SetRecords([]*gopherkeeperpb.RecordMetadata{
			protoRecordMetadata(gopherkeeperpb.RecordType_RECORD_TYPE_TEXT),
		})
		return response, nil
	}}}

	records, err := client.ListRecords(context.Background(), "token")
	if err != nil {
		t.Fatalf("ListRecords() error = %v", err)
	}
	if len(records) != 1 || records[0].ID != testRecordID {
		t.Fatalf("ListRecords() records = %+v", records)
	}
}

func TestClient_GetRecord(t *testing.T) {
	client := &Client{records: recordClientStub{get: func(
		ctx context.Context,
		request *gopherkeeperpb.GetRecordRequest,
		_ ...grpc.CallOption,
	) (*gopherkeeperpb.Record, error) {
		assertBearerToken(t, ctx, "token")
		if request.GetId() != testRecordID {
			t.Fatalf("GetRecord() id = %q", request.GetId())
		}
		return protoTextRecord(), nil
	}}}

	record, err := client.GetRecord(context.Background(), "token", testRecordID)
	if err != nil {
		t.Fatalf("GetRecord() error = %v", err)
	}
	if record.Metadata.ID != testRecordID {
		t.Fatalf("GetRecord() id = %q", record.Metadata.ID)
	}
}

func TestClient_UpdateRecord(t *testing.T) {
	client := &Client{records: recordClientStub{update: func(
		ctx context.Context,
		request *gopherkeeperpb.UpdateRecordRequest,
		_ ...grpc.CallOption,
	) (*gopherkeeperpb.Record, error) {
		assertBearerToken(t, ctx, "token")
		if request.GetId() != testRecordID || request.GetExpectedRevision() != 1 {
			t.Fatalf("UpdateRecord() id/revision = %q/%d", request.GetId(), request.GetExpectedRevision())
		}
		return protoTextRecord(), nil
	}}}

	_, err := client.UpdateRecord(
		context.Background(),
		"token",
		testRecordID,
		1,
		"Example",
		&model.TextPayload{Text: "secret"},
	)
	if err != nil {
		t.Fatalf("UpdateRecord() error = %v", err)
	}
}

func TestClient_DeleteRecord(t *testing.T) {
	client := &Client{records: recordClientStub{delete: func(
		ctx context.Context,
		request *gopherkeeperpb.DeleteRecordRequest,
		_ ...grpc.CallOption,
	) (*gopherkeeperpb.DeleteRecordResponse, error) {
		assertBearerToken(t, ctx, "token")
		if request.GetId() != testRecordID || request.GetExpectedRevision() != 1 {
			t.Fatalf("DeleteRecord() id/revision = %q/%d", request.GetId(), request.GetExpectedRevision())
		}
		return &gopherkeeperpb.DeleteRecordResponse{}, nil
	}}}

	if err := client.DeleteRecord(context.Background(), "token", testRecordID, 1); err != nil {
		t.Fatalf("DeleteRecord() error = %v", err)
	}
}

func TestClient_RecordMethodsMapErrors(t *testing.T) {
	tests := []struct {
		name    string
		code    codes.Code
		wantErr error
	}{
		{name: "unauthorized", code: codes.Unauthenticated, wantErr: model.ErrUnauthorized},
		{name: "not found", code: codes.NotFound, wantErr: model.ErrRecordNotFound},
		{name: "conflict", code: codes.Aborted, wantErr: model.ErrRecordRevisionConflict},
		{name: "precondition", code: codes.FailedPrecondition, wantErr: model.ErrRecordPreconditionRequired},
		{name: "too large", code: codes.ResourceExhausted, wantErr: model.ErrPayloadTooLarge},
		{name: "invalid", code: codes.InvalidArgument, wantErr: model.ErrInvalidRecordData},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := &Client{records: recordClientStub{get: func(
				context.Context,
				*gopherkeeperpb.GetRecordRequest,
				...grpc.CallOption,
			) (*gopherkeeperpb.Record, error) {
				return nil, status.Error(test.code, test.name)
			}}}

			_, err := client.GetRecord(context.Background(), "token", testRecordID)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("GetRecord() error = %v, want %v", err, test.wantErr)
			}
		})
	}
}

func TestClient_CreateRecordRejectsNilPayload(t *testing.T) {
	client := &Client{}
	_, err := client.CreateRecord(context.Background(), "token", "Example", nil)
	if !errors.Is(err, errRecordPayloadRequired) {
		t.Fatalf("CreateRecord() error = %v, want payload required", err)
	}
}

func TestClient_ListRecordsDoesNotReportLargeResponseAsPayloadError(t *testing.T) {
	client := &Client{records: recordClientStub{list: func(
		context.Context,
		*gopherkeeperpb.ListRecordsRequest,
		...grpc.CallOption,
	) (*gopherkeeperpb.ListRecordsResponse, error) {
		return nil, status.Error(codes.ResourceExhausted, "received message larger than max")
	}}}

	_, err := client.ListRecords(context.Background(), "token")
	if err == nil {
		t.Fatal("ListRecords() error = nil")
	}
	if errors.Is(err, model.ErrPayloadTooLarge) {
		t.Fatalf("ListRecords() error = %v, must not be payload-too-large", err)
	}
	if got := failure.Message(err); got != "Server response is too large" {
		t.Fatalf("failure.Message() = %q", got)
	}
}
