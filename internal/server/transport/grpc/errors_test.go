package grpcserver

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/apierror"
	"github.com/xhrobj/gopherkeeper/internal/model"
	"github.com/xhrobj/gopherkeeper/internal/server/service"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestTransportError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		grpcCode codes.Code
		apiCode  apierror.Code
	}{
		{name: "nil", grpcCode: codes.OK},
		{name: "canceled", err: context.Canceled, grpcCode: codes.Canceled},
		{name: "deadline", err: context.DeadlineExceeded, grpcCode: codes.DeadlineExceeded},
		{name: "invalid credentials", err: service.ErrInvalidCredentials, grpcCode: codes.Unauthenticated, apiCode: apierror.InvalidCredentials},
		{name: "login exists", err: model.ErrLoginAlreadyExists, grpcCode: codes.AlreadyExists, apiCode: apierror.LoginAlreadyExists},
		{name: "unauthorized", err: model.ErrUnauthorized, grpcCode: codes.Unauthenticated, apiCode: apierror.Unauthorized},
		{name: "payload too large", err: model.ErrPayloadTooLarge, grpcCode: codes.ResourceExhausted, apiCode: apierror.PayloadTooLarge},
		{name: "not found", err: model.ErrRecordNotFound, grpcCode: codes.NotFound, apiCode: apierror.RecordNotFound},
		{name: "revision conflict", err: model.ErrRecordRevisionConflict, grpcCode: codes.Aborted, apiCode: apierror.RecordRevisionConflict},
		{name: "record decryption", err: model.ErrRecordDecryptionFailed, grpcCode: codes.DataLoss, apiCode: apierror.RecordDecryptionFailed},
		{name: "precondition", err: model.ErrRecordPreconditionRequired, grpcCode: codes.FailedPrecondition, apiCode: apierror.PreconditionRequired},
		{name: "invalid record data", err: model.ErrInvalidRecordTitle, grpcCode: codes.InvalidArgument, apiCode: apierror.InvalidRecordData},
		{name: "wrapped invalid request", err: fmt.Errorf("wrapped: %w", service.ErrInvalidLogin), grpcCode: codes.InvalidArgument, apiCode: apierror.InvalidRequest},
		{name: "internal", err: errors.New("database unavailable"), grpcCode: codes.Internal, apiCode: apierror.Internal},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			err := transportError(test.err)
			if got := status.Code(err); got != test.grpcCode {
				t.Fatalf("status.Code() = %s, want %s", got, test.grpcCode)
			}
			if got := apiErrorCodeFromGRPCError(err); got != test.apiCode {
				t.Fatalf("ErrorInfo reason = %q, want %q", got, test.apiCode)
			}
		})
	}
}

func TestUnauthenticatedErrorIncludesAPIErrorCode(t *testing.T) {
	t.Parallel()

	err := unauthenticatedError()
	if got := status.Code(err); got != codes.Unauthenticated {
		t.Fatalf("status.Code() = %s, want %s", got, codes.Unauthenticated)
	}
	if got := apiErrorCodeFromGRPCError(err); got != apierror.Unauthorized {
		t.Fatalf("ErrorInfo reason = %q, want %q", got, apierror.Unauthorized)
	}
}

func apiErrorCodeFromGRPCError(err error) apierror.Code {
	if err == nil {
		return ""
	}

	for _, detail := range status.Convert(err).Details() {
		info, ok := detail.(*errdetails.ErrorInfo)
		if !ok || info.GetDomain() != apierror.Domain {
			continue
		}

		code, _ := apierror.ParseGRPCReason(info.GetReason())
		return code
	}

	return ""
}
