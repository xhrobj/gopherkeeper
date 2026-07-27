package grpcserver

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/model"
	"github.com/xhrobj/gopherkeeper/internal/server/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestTransportError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code codes.Code
	}{
		{name: "nil", code: codes.OK},
		{name: "canceled", err: context.Canceled, code: codes.Canceled},
		{name: "deadline", err: context.DeadlineExceeded, code: codes.DeadlineExceeded},
		{name: "invalid credentials", err: service.ErrInvalidCredentials, code: codes.Unauthenticated},
		{name: "login exists", err: model.ErrLoginAlreadyExists, code: codes.AlreadyExists},
		{name: "payload too large", err: model.ErrPayloadTooLarge, code: codes.ResourceExhausted},
		{name: "not found", err: model.ErrRecordNotFound, code: codes.NotFound},
		{name: "revision conflict", err: model.ErrRecordRevisionConflict, code: codes.Aborted},
		{name: "precondition", err: model.ErrRecordPreconditionRequired, code: codes.FailedPrecondition},
		{name: "invalid request", err: model.ErrInvalidRecordTitle, code: codes.InvalidArgument},
		{name: "wrapped invalid request", err: fmt.Errorf("wrapped: %w", service.ErrInvalidLogin), code: codes.InvalidArgument},
		{name: "internal", err: errors.New("database unavailable"), code: codes.Internal},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := status.Code(transportError(test.err)); got != test.code {
				t.Fatalf("status.Code() = %s, want %s", got, test.code)
			}
		})
	}
}
