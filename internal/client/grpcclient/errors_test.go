package grpcclient

import (
	"context"
	"errors"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/client/failure"
	"github.com/xhrobj/gopherkeeper/internal/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestMapRPCError(t *testing.T) {
	tests := []struct {
		name        string
		code        codes.Code
		cause       error
		wantKind    failure.Kind
		wantMessage string
	}{
		{name: "canceled", code: codes.Canceled, wantKind: failure.Canceled, wantMessage: "Operation canceled"},
		{name: "deadline", code: codes.DeadlineExceeded, wantKind: failure.Timeout, wantMessage: "Connection timed out"},
		{name: "unavailable", code: codes.Unavailable, wantKind: failure.Unavailable, wantMessage: "Connection refused"},
		{name: "unauthorized", code: codes.Unauthenticated, cause: model.ErrUnauthorized, wantKind: failure.Unauthorized, wantMessage: "message"},
		{name: "conflict", code: codes.Aborted, cause: model.ErrRecordRevisionConflict, wantKind: failure.Conflict, wantMessage: "message"},
		{name: "not found", code: codes.NotFound, cause: model.ErrRecordNotFound, wantKind: failure.NotFound, wantMessage: "message"},
		{name: "record data loss", code: codes.DataLoss, cause: model.ErrRecordDecryptionFailed, wantKind: failure.Unknown, wantMessage: "Record data could not be decrypted"},
		{name: "unclassified data loss", code: codes.DataLoss, wantKind: failure.Unknown, wantMessage: "Operation failed"},
		{name: "validation", code: codes.InvalidArgument, cause: model.ErrInvalidRecordData, wantKind: failure.Validation, wantMessage: "message"},
		{name: "too large", code: codes.ResourceExhausted, cause: model.ErrPayloadTooLarge, wantKind: failure.TooLarge, wantMessage: "message"},
		{name: "internal", code: codes.Internal, wantKind: failure.Unknown, wantMessage: "Internal server error"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertMappedRPCError(t, test.code, test.cause, test.wantKind, test.wantMessage)
		})
	}
}

func TestMapRPCErrorPreservesContextErrors(t *testing.T) {
	err := mapRPCError("test", status.FromContextError(context.Canceled).Err(), nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("mapRPCError() error = %v, want context canceled", err)
	}
}

func TestInvalidResponseError(t *testing.T) {
	cause := errors.New("malformed")
	err := invalidResponseError("login", cause)
	if !errors.Is(err, cause) {
		t.Fatalf("invalidResponseError() does not preserve cause: %v", err)
	}
	if got := failure.Message(err); got != "Invalid server response" {
		t.Fatalf("failure.Message() = %q", got)
	}
}

func TestRecordErrorCauseMapsDecryptionFailure(t *testing.T) {
	err := status.Error(codes.DataLoss, "record data could not be decrypted")
	if !errors.Is(recordErrorCause(err), model.ErrRecordDecryptionFailed) {
		t.Fatal("recordErrorCause() does not map DataLoss")
	}
}

func assertMappedRPCError(
	t *testing.T,
	code codes.Code,
	cause error,
	wantKind failure.Kind,
	wantMessage string,
) {
	t.Helper()

	err := mapRPCError("test", status.Error(code, "message"), cause)
	if err == nil {
		t.Fatal("mapRPCError() error = nil")
	}
	if got := failure.KindOf(err); got != wantKind {
		t.Fatalf("failure.KindOf() = %d, want %d", got, wantKind)
	}
	if got := failure.Message(err); got != wantMessage {
		t.Fatalf("failure.Message() = %q, want %q", got, wantMessage)
	}
	if cause != nil && !errors.Is(err, cause) {
		t.Fatalf("mapRPCError() does not preserve cause %v: %v", cause, err)
	}

	var rpcError *RPCError
	if !errors.As(err, &rpcError) || rpcError.Code != code {
		t.Fatalf("mapRPCError() RPCError = %#v, want code %s", rpcError, code)
	}
}
