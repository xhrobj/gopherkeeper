package grpcclient

import (
	"context"
	"errors"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/apierror"
	"github.com/xhrobj/gopherkeeper/internal/client/failure"
	"github.com/xhrobj/gopherkeeper/internal/model"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestMapRPCErrorMapsTransportFailures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		err         error
		wantCause   error
		wantKind    failure.Kind
		wantMessage string
	}{
		{
			name:        "canceled",
			err:         status.FromContextError(context.Canceled).Err(),
			wantCause:   context.Canceled,
			wantKind:    failure.Canceled,
			wantMessage: "Operation canceled",
		},
		{
			name:        "deadline",
			err:         status.FromContextError(context.DeadlineExceeded).Err(),
			wantCause:   context.DeadlineExceeded,
			wantKind:    failure.Timeout,
			wantMessage: "Connection timed out",
		},
		{
			name:        "unavailable",
			err:         status.Error(codes.Unavailable, "dial tcp: connection refused"),
			wantKind:    failure.Unavailable,
			wantMessage: "Connection refused",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			err := mapRPCError("test", test.err)
			if test.wantCause != nil && !errors.Is(err, test.wantCause) {
				t.Fatalf("mapRPCError() error = %v, want cause %v", err, test.wantCause)
			}
			if got := failure.KindOf(err); got != test.wantKind {
				t.Fatalf("failure.KindOf() = %d, want %d", got, test.wantKind)
			}
			if got := failure.Message(err); got != test.wantMessage {
				t.Fatalf("failure.Message() = %q, want %q", got, test.wantMessage)
			}
		})
	}
}

func TestMapRPCErrorUsesAPIErrorDetails(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		grpcCode  codes.Code
		apiCode   apierror.Code
		message   string
		wantCause error
		wantKind  failure.Kind
	}{
		{name: "invalid request", grpcCode: codes.InvalidArgument, apiCode: apierror.InvalidRequest, message: "invalid request", wantKind: failure.Validation},
		{name: "invalid credentials", grpcCode: codes.Unauthenticated, apiCode: apierror.InvalidCredentials, message: "invalid login or password", wantCause: model.ErrInvalidCredentials, wantKind: failure.Unauthorized},
		{name: "login already exists", grpcCode: codes.AlreadyExists, apiCode: apierror.LoginAlreadyExists, message: "login is already registered", wantCause: model.ErrLoginAlreadyExists, wantKind: failure.Conflict},
		{name: "unauthorized", grpcCode: codes.Unauthenticated, apiCode: apierror.Unauthorized, message: "authentication required", wantCause: model.ErrUnauthorized, wantKind: failure.Unauthorized},
		{name: "payload too large", grpcCode: codes.ResourceExhausted, apiCode: apierror.PayloadTooLarge, message: "payload is too large", wantCause: model.ErrPayloadTooLarge, wantKind: failure.TooLarge},
		{name: "invalid record data", grpcCode: codes.InvalidArgument, apiCode: apierror.InvalidRecordData, message: "invalid record data", wantCause: model.ErrInvalidRecordData, wantKind: failure.Validation},
		{name: "record not found", grpcCode: codes.NotFound, apiCode: apierror.RecordNotFound, message: "record not found", wantCause: model.ErrRecordNotFound, wantKind: failure.NotFound},
		{name: "revision conflict", grpcCode: codes.Aborted, apiCode: apierror.RecordRevisionConflict, message: "record revision conflict", wantCause: model.ErrRecordRevisionConflict, wantKind: failure.Conflict},
		{name: "decryption failed", grpcCode: codes.DataLoss, apiCode: apierror.RecordDecryptionFailed, message: "record data could not be decrypted", wantCause: model.ErrRecordDecryptionFailed, wantKind: failure.Unknown},
		{name: "precondition required", grpcCode: codes.FailedPrecondition, apiCode: apierror.PreconditionRequired, message: "record revision is required", wantCause: model.ErrRecordPreconditionRequired, wantKind: failure.Validation},
		{name: "internal", grpcCode: codes.Internal, apiCode: apierror.Internal, message: "internal server error", wantKind: failure.Unknown},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			transportErr := grpcErrorWithAPIErrorCode(t, test.grpcCode, test.message, test.apiCode)
			err := mapRPCError("test", transportErr)

			if test.wantCause == nil {
				if got := failure.CauseFromAPIErrorCode(test.apiCode); got != nil {
					t.Fatalf("CauseFromAPIErrorCode() = %v, want nil", got)
				}
			} else if !errors.Is(err, test.wantCause) {
				t.Fatalf("mapRPCError() error = %v, want cause %v", err, test.wantCause)
			}
			if got := failure.KindOf(err); got != test.wantKind {
				t.Fatalf("failure.KindOf() = %d, want %d", got, test.wantKind)
			}
			if got := failure.Message(err); got != test.message {
				t.Fatalf("failure.Message() = %q, want %q", got, test.message)
			}

			var rpcError *RPCError
			if !errors.As(err, &rpcError) || rpcError.APIErrorCode != test.apiCode {
				t.Fatalf("mapRPCError() RPCError = %#v, want API code %q", rpcError, test.apiCode)
			}
		})
	}
}

func TestMapRPCErrorRequiresAPIErrorDetails(t *testing.T) {
	t.Parallel()

	err := mapRPCError("registration", status.Error(codes.AlreadyExists, "login is already registered"))
	if errors.Is(err, model.ErrLoginAlreadyExists) {
		t.Fatalf("mapRPCError() mapped status-only error to domain cause: %v", err)
	}
	if got := failure.KindOf(err); got != failure.Unknown {
		t.Fatalf("failure.KindOf() = %d, want %d", got, failure.Unknown)
	}
	if got := failure.Message(err); got != "Operation failed" {
		t.Fatalf("failure.Message() = %q, want generic message", got)
	}
}

func TestAPIErrorCodeFromStatusRejectsForeignDetails(t *testing.T) {
	t.Parallel()

	grpcStatus := status.New(codes.InvalidArgument, "invalid request")
	withDetails, err := grpcStatus.WithDetails(&errdetails.ErrorInfo{
		Reason: "INVALID_REQUEST",
		Domain: "other.api",
	})
	if err != nil {
		t.Fatalf("WithDetails() error = %v", err)
	}

	if got := apiErrorCodeFromStatus(withDetails); got != "" {
		t.Fatalf("apiErrorCodeFromStatus() = %q, want empty code", got)
	}
}

func TestMapListRecordsErrorDistinguishesAPILimit(t *testing.T) {
	t.Parallel()

	apiErr := grpcErrorWithAPIErrorCode(
		t,
		codes.ResourceExhausted,
		"payload is too large",
		apierror.PayloadTooLarge,
	)
	mappedAPIError := mapListRecordsError(apiErr)
	if !errors.Is(mappedAPIError, model.ErrPayloadTooLarge) {
		t.Fatalf("mapListRecordsError() error = %v, want payload-too-large cause", mappedAPIError)
	}
	if got := failure.Message(mappedAPIError); got != "payload is too large" {
		t.Fatalf("failure.Message() = %q, want API message", got)
	}

	responseLimitError := mapListRecordsError(status.Error(codes.ResourceExhausted, "received message larger than max"))
	if errors.Is(responseLimitError, model.ErrPayloadTooLarge) {
		t.Fatalf("mapListRecordsError() mapped local response limit to API cause: %v", responseLimitError)
	}
	if got := failure.Message(responseLimitError); got != "Server response is too large" {
		t.Fatalf("failure.Message() = %q, want response-limit message", got)
	}
}

func TestInvalidResponseError(t *testing.T) {
	t.Parallel()

	cause := errors.New("malformed")
	err := invalidResponseError("login", cause)
	if !errors.Is(err, cause) {
		t.Fatalf("invalidResponseError() does not preserve cause: %v", err)
	}
	if got := failure.Message(err); got != "Invalid server response" {
		t.Fatalf("failure.Message() = %q", got)
	}
}

func TestUnavailableRPCFailureKind(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		message string
		want    failure.Kind
	}{
		{name: "certificate", message: "x509: certificate signed by unknown authority", want: failure.TLSCertificate},
		{name: "TLS handshake", message: "transport: authentication handshake failed: tls: handshake failure", want: failure.TLSHandshake},
		{name: "host not found", message: "dial tcp: lookup missing.example: no such host", want: failure.HostNotFound},
		{name: "network unreachable", message: "dial tcp: network is unreachable", want: failure.NetworkUnreachable},
		{name: "timeout", message: "connection error: i/o timeout", want: failure.Timeout},
		{name: "connection refused", message: "dial tcp: connection refused", want: failure.Unavailable},
		{name: "generic unavailable", message: "transport is closing", want: failure.Unavailable},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := unavailableRPCFailureKind(test.message); got != test.want {
				t.Fatalf("unavailableRPCFailureKind() = %d, want %d", got, test.want)
			}
		})
	}
}

func grpcErrorWithAPIErrorCode(
	t *testing.T,
	code codes.Code,
	message string,
	apiCode apierror.Code,
) error {
	t.Helper()

	grpcStatus := status.New(code, message)
	reason, ok := apierror.GRPCReason(apiCode)
	if !ok {
		t.Fatalf("GRPCReason(%q) is unknown", apiCode)
	}

	withDetails, err := grpcStatus.WithDetails(&errdetails.ErrorInfo{
		Reason: reason,
		Domain: apierror.Domain,
	})
	if err != nil {
		t.Fatalf("WithDetails() error = %v", err)
	}

	return withDetails.Err()
}
