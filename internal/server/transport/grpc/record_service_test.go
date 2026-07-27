package grpcserver

import (
	"context"
	"testing"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/model"
	gopherkeeperpb "github.com/xhrobj/gopherkeeper/internal/proto"
	"github.com/xhrobj/gopherkeeper/internal/server/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const grpcTestRecordID = "00000000-0000-4000-8000-000000000042"

type recordServiceCRUDSpy struct {
	t       *testing.T
	created model.Record
	updated model.Record

	createCalled bool
	listCalled   bool
	getCalled    bool
	updateCalled bool
	deleteCalled bool
}

func (spy *recordServiceCRUDSpy) manager() recordManagerStub {
	return recordManagerStub{
		create: spy.create,
		list:   spy.list,
		get:    spy.get,
		update: spy.update,
		delete: spy.delete,
	}
}

func (spy *recordServiceCRUDSpy) create(
	_ context.Context,
	request service.CreateRecordRequest,
) (model.Record, error) {
	spy.t.Helper()
	spy.createCalled = true
	if request.UserID != 42 || request.Title != "Private note" {
		spy.t.Fatalf("Create() request = %#v", request)
	}
	payload, ok := request.Payload.(*model.TextPayload)
	if !ok || payload.Text != "secret-v1" || payload.Metadata != "notes" {
		spy.t.Fatalf("Create() payload = %#v", request.Payload)
	}
	return spy.created, nil
}

func (spy *recordServiceCRUDSpy) list(_ context.Context, userID int64) ([]model.RecordMetadata, error) {
	spy.t.Helper()
	spy.listCalled = true
	if userID != 42 {
		spy.t.Fatalf("List() userID = %d", userID)
	}
	return []model.RecordMetadata{spy.created.Metadata}, nil
}

func (spy *recordServiceCRUDSpy) get(
	_ context.Context,
	userID int64,
	recordID string,
) (model.Record, error) {
	spy.t.Helper()
	spy.getCalled = true
	if userID != 42 || recordID != grpcTestRecordID {
		spy.t.Fatalf("Get() arguments = %d / %q", userID, recordID)
	}
	return spy.created, nil
}

func (spy *recordServiceCRUDSpy) update(
	_ context.Context,
	request service.UpdateRecordRequest,
) (model.Record, error) {
	spy.t.Helper()
	spy.updateCalled = true
	if request.UserID != 42 || request.RecordID != grpcTestRecordID ||
		request.ExpectedRevision != 1 || request.Title != "Updated note" {
		spy.t.Fatalf("Update() request = %#v", request)
	}
	payload, ok := request.Payload.(*model.TextPayload)
	if !ok || payload.Text != "secret-v2" {
		spy.t.Fatalf("Update() payload = %#v", request.Payload)
	}
	return spy.updated, nil
}

func (spy *recordServiceCRUDSpy) delete(_ context.Context, request service.DeleteRecordRequest) error {
	spy.t.Helper()
	spy.deleteCalled = true
	if request.UserID != 42 || request.RecordID != grpcTestRecordID || request.ExpectedRevision != 2 {
		spy.t.Fatalf("Delete() request = %#v", request)
	}
	return nil
}

func (spy *recordServiceCRUDSpy) assertCalled() {
	spy.t.Helper()
	if !spy.createCalled || !spy.listCalled || !spy.getCalled || !spy.updateCalled || !spy.deleteCalled {
		spy.t.Fatalf("CRUD calls = create:%t list:%t get:%t update:%t delete:%t",
			spy.createCalled, spy.listCalled, spy.getCalled, spy.updateCalled, spy.deleteCalled)
	}
}

func TestRecordService_CRUD(t *testing.T) {
	createdAt := time.Date(2026, time.July, 24, 12, 0, 0, 0, time.UTC)
	spy := &recordServiceCRUDSpy{
		t:       t,
		created: grpcTestTextRecord(createdAt, model.RecordInitialRevision, "Private note", "secret-v1"),
		updated: grpcTestTextRecord(createdAt, 2, "Updated note", "secret-v2"),
	}
	server := newRecordService(spy.manager())
	ctx := context.WithValue(context.Background(), userIDContextKey{}, int64(42))

	createRequest := &gopherkeeperpb.CreateRecordRequest{}
	createRequest.SetTitle("Private note")
	createRequest.SetPayload(newProtoTextPayload("secret-v1", "notes"))
	createResponse, err := server.CreateRecord(ctx, createRequest)
	if err != nil {
		t.Fatalf("CreateRecord() error = %v", err)
	}
	if createResponse.GetMetadata().GetRevision() != 1 {
		t.Fatalf("CreateRecord() revision = %d", createResponse.GetMetadata().GetRevision())
	}

	listResponse, err := server.ListRecords(ctx, &gopherkeeperpb.ListRecordsRequest{})
	if err != nil {
		t.Fatalf("ListRecords() error = %v", err)
	}
	if len(listResponse.GetRecords()) != 1 || listResponse.GetRecords()[0].GetId() != grpcTestRecordID {
		t.Fatalf("ListRecords() response = %#v", listResponse)
	}

	getRequest := &gopherkeeperpb.GetRecordRequest{}
	getRequest.SetId(grpcTestRecordID)
	getResponse, err := server.GetRecord(ctx, getRequest)
	if err != nil {
		t.Fatalf("GetRecord() error = %v", err)
	}
	if getResponse.GetPayload().GetText().GetText() != "secret-v1" {
		t.Fatalf("GetRecord() payload = %#v", getResponse.GetPayload())
	}

	updateRequest := &gopherkeeperpb.UpdateRecordRequest{}
	updateRequest.SetId(grpcTestRecordID)
	updateRequest.SetExpectedRevision(1)
	updateRequest.SetTitle("Updated note")
	updateRequest.SetPayload(newProtoTextPayload("secret-v2", "notes"))
	updateResponse, err := server.UpdateRecord(ctx, updateRequest)
	if err != nil {
		t.Fatalf("UpdateRecord() error = %v", err)
	}
	if updateResponse.GetMetadata().GetRevision() != 2 {
		t.Fatalf("UpdateRecord() revision = %d", updateResponse.GetMetadata().GetRevision())
	}

	deleteRequest := &gopherkeeperpb.DeleteRecordRequest{}
	deleteRequest.SetId(grpcTestRecordID)
	deleteRequest.SetExpectedRevision(2)
	if _, err := server.DeleteRecord(ctx, deleteRequest); err != nil {
		t.Fatalf("DeleteRecord() error = %v", err)
	}

	spy.assertCalled()
}

func TestRecordService_RejectsInvalidCalls(t *testing.T) {
	server := newRecordService(nil)
	ctx := context.WithValue(context.Background(), userIDContextKey{}, int64(42))

	if _, err := server.CreateRecord(context.Background(), &gopherkeeperpb.CreateRecordRequest{}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("CreateRecord() code = %s, want %s", status.Code(err), codes.Unauthenticated)
	}
	if _, err := server.ListRecords(context.Background(), nil); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("ListRecords() code = %s, want %s", status.Code(err), codes.Unauthenticated)
	}
	if _, err := server.GetRecord(ctx, nil); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("GetRecord() code = %s, want %s", status.Code(err), codes.InvalidArgument)
	}
	if _, err := server.ListRecords(ctx, &gopherkeeperpb.ListRecordsRequest{}); status.Code(err) != codes.Internal {
		t.Fatalf("ListRecords() code = %s, want %s", status.Code(err), codes.Internal)
	}
	if _, err := server.ListRecords(ctx, nil); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("ListRecords(nil) code = %s, want %s", status.Code(err), codes.InvalidArgument)
	}

	manager := recordManagerStub{}
	server = newRecordService(manager)
	if _, err := server.UpdateRecord(ctx, &gopherkeeperpb.UpdateRecordRequest{}); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("UpdateRecord() code = %s, want %s", status.Code(err), codes.FailedPrecondition)
	}
	if _, err := server.DeleteRecord(ctx, &gopherkeeperpb.DeleteRecordRequest{}); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("DeleteRecord() code = %s, want %s", status.Code(err), codes.FailedPrecondition)
	}
}

func grpcTestTextRecord(createdAt time.Time, revision int64, title, text string) model.Record {
	return model.Record{
		Metadata: model.RecordMetadata{
			ID:        grpcTestRecordID,
			Type:      model.RecordTypeText,
			Title:     title,
			Revision:  revision,
			CreatedAt: createdAt,
			UpdatedAt: createdAt.Add(time.Duration(revision-1) * time.Minute),
		},
		Payload: &model.TextPayload{Text: text, Metadata: "notes"},
	}
}
