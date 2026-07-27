package grpcclient

import (
	"context"
	"testing"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/model"
	gopherkeeperpb "github.com/xhrobj/gopherkeeper/internal/proto"
	"google.golang.org/grpc"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const testRecordID = "00000000-0000-4000-8000-000000000042"

var testTime = time.Date(2026, time.July, 26, 12, 0, 0, 0, time.UTC)

type closerStub struct {
	called bool
	err    error
}

func (stub *closerStub) Close() error {
	stub.called = true
	return stub.err
}

type healthClientStub struct {
	check func(context.Context, *healthpb.HealthCheckRequest, ...grpc.CallOption) (*healthpb.HealthCheckResponse, error)
}

func (stub healthClientStub) Check(
	ctx context.Context,
	request *healthpb.HealthCheckRequest,
	options ...grpc.CallOption,
) (*healthpb.HealthCheckResponse, error) {
	return stub.check(ctx, request, options...)
}

type authClientStub struct {
	register    func(context.Context, *gopherkeeperpb.RegisterRequest, ...grpc.CallOption) (*gopherkeeperpb.RegisterResponse, error)
	login       func(context.Context, *gopherkeeperpb.LoginRequest, ...grpc.CallOption) (*gopherkeeperpb.LoginResponse, error)
	currentUser func(context.Context, *gopherkeeperpb.CurrentUserRequest, ...grpc.CallOption) (*gopherkeeperpb.CurrentUserResponse, error)
}

func (stub authClientStub) Register(
	ctx context.Context,
	request *gopherkeeperpb.RegisterRequest,
	options ...grpc.CallOption,
) (*gopherkeeperpb.RegisterResponse, error) {
	return stub.register(ctx, request, options...)
}

func (stub authClientStub) Login(
	ctx context.Context,
	request *gopherkeeperpb.LoginRequest,
	options ...grpc.CallOption,
) (*gopherkeeperpb.LoginResponse, error) {
	return stub.login(ctx, request, options...)
}

func (stub authClientStub) CurrentUser(
	ctx context.Context,
	request *gopherkeeperpb.CurrentUserRequest,
	options ...grpc.CallOption,
) (*gopherkeeperpb.CurrentUserResponse, error) {
	return stub.currentUser(ctx, request, options...)
}

type recordClientStub struct {
	create func(context.Context, *gopherkeeperpb.CreateRecordRequest, ...grpc.CallOption) (*gopherkeeperpb.Record, error)
	list   func(context.Context, *gopherkeeperpb.ListRecordsRequest, ...grpc.CallOption) (*gopherkeeperpb.ListRecordsResponse, error)
	get    func(context.Context, *gopherkeeperpb.GetRecordRequest, ...grpc.CallOption) (*gopherkeeperpb.Record, error)
	update func(context.Context, *gopherkeeperpb.UpdateRecordRequest, ...grpc.CallOption) (*gopherkeeperpb.Record, error)
	delete func(context.Context, *gopherkeeperpb.DeleteRecordRequest, ...grpc.CallOption) (*gopherkeeperpb.DeleteRecordResponse, error)
}

func (stub recordClientStub) CreateRecord(
	ctx context.Context,
	request *gopherkeeperpb.CreateRecordRequest,
	options ...grpc.CallOption,
) (*gopherkeeperpb.Record, error) {
	return stub.create(ctx, request, options...)
}

func (stub recordClientStub) ListRecords(
	ctx context.Context,
	request *gopherkeeperpb.ListRecordsRequest,
	options ...grpc.CallOption,
) (*gopherkeeperpb.ListRecordsResponse, error) {
	return stub.list(ctx, request, options...)
}

func (stub recordClientStub) GetRecord(
	ctx context.Context,
	request *gopherkeeperpb.GetRecordRequest,
	options ...grpc.CallOption,
) (*gopherkeeperpb.Record, error) {
	return stub.get(ctx, request, options...)
}

func (stub recordClientStub) UpdateRecord(
	ctx context.Context,
	request *gopherkeeperpb.UpdateRecordRequest,
	options ...grpc.CallOption,
) (*gopherkeeperpb.Record, error) {
	return stub.update(ctx, request, options...)
}

func (stub recordClientStub) DeleteRecord(
	ctx context.Context,
	request *gopherkeeperpb.DeleteRecordRequest,
	options ...grpc.CallOption,
) (*gopherkeeperpb.DeleteRecordResponse, error) {
	return stub.delete(ctx, request, options...)
}

func protoUser() *gopherkeeperpb.User {
	user := &gopherkeeperpb.User{}
	user.SetId(42)
	user.SetLogin("alice")
	user.SetCreatedAt(timestamppb.New(testTime))
	return user
}

func protoRecordMetadata(recordType gopherkeeperpb.RecordType) *gopherkeeperpb.RecordMetadata {
	metadata := &gopherkeeperpb.RecordMetadata{}
	metadata.SetId(testRecordID)
	metadata.SetType(recordType)
	metadata.SetTitle("Example")
	metadata.SetRevision(1)
	metadata.SetCreatedAt(timestamppb.New(testTime))
	metadata.SetUpdatedAt(timestamppb.New(testTime))
	return metadata
}

func protoTextRecord() *gopherkeeperpb.Record {
	payload := &gopherkeeperpb.TextPayload{}
	payload.SetText("secret")
	payload.SetMetadata("notes")
	wrapped := &gopherkeeperpb.RecordPayload{}
	wrapped.SetText(payload)
	record := &gopherkeeperpb.Record{}
	record.SetMetadata(protoRecordMetadata(gopherkeeperpb.RecordType_RECORD_TYPE_TEXT))
	record.SetPayload(wrapped)
	return record
}

func assertBearerToken(t *testing.T, ctx context.Context, want string) {
	t.Helper()
	values := outgoingMetadataValues(ctx, authorizationMetadataKey)
	if len(values) != 1 || values[0] != "Bearer "+want {
		t.Fatalf("authorization metadata = %v, want Bearer %s", values, want)
	}
}

func modelTextRecord() model.Record {
	return model.Record{
		Metadata: model.RecordMetadata{
			ID:        testRecordID,
			Type:      model.RecordTypeText,
			Title:     "Example",
			Revision:  1,
			CreatedAt: testTime,
			UpdatedAt: testTime,
		},
		Payload: &model.TextPayload{Text: "secret", Metadata: "notes"},
	}
}

func outgoingMetadataValues(ctx context.Context, key string) []string {
	outgoing, _ := metadata.FromOutgoingContext(ctx)
	return outgoing.Get(key)
}
