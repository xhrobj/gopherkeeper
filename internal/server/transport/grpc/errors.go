package grpcserver

import (
	"context"
	"errors"

	"github.com/xhrobj/gopherkeeper/internal/model"
	"github.com/xhrobj/gopherkeeper/internal/server/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	errInvalidRequest      = errors.New("invalid gRPC request")
	errInvalidDependencies = errors.New("invalid gRPC dependencies")
)

func transportError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, "request canceled")
	case errors.Is(err, context.DeadlineExceeded):
		return status.Error(codes.DeadlineExceeded, "request deadline exceeded")
	case errors.Is(err, service.ErrInvalidCredentials):
		return status.Error(codes.Unauthenticated, "invalid login or password")
	case errors.Is(err, model.ErrLoginAlreadyExists):
		return status.Error(codes.AlreadyExists, "login is already registered")
	case errors.Is(err, model.ErrUserNotFound), errors.Is(err, model.ErrUnauthorized):
		return status.Error(codes.Unauthenticated, "authentication required")
	case errors.Is(err, model.ErrPayloadTooLarge):
		return status.Error(codes.ResourceExhausted, "payload is too large")
	case errors.Is(err, model.ErrRecordNotFound):
		return status.Error(codes.NotFound, "record not found")
	case errors.Is(err, model.ErrRecordRevisionConflict):
		return status.Error(codes.Aborted, "record revision conflict")
	case errors.Is(err, model.ErrRecordDecryptionFailed):
		return status.Error(codes.DataLoss, "record data could not be decrypted")
	case errors.Is(err, model.ErrRecordPreconditionRequired):
		return status.Error(codes.FailedPrecondition, "record revision is required")
	case errors.Is(err, errInvalidRequest),
		errors.Is(err, service.ErrInvalidLogin),
		errors.Is(err, service.ErrInvalidPassword),
		errors.Is(err, service.ErrPasswordTooShort),
		errors.Is(err, service.ErrPasswordTooLong),
		errors.Is(err, model.ErrInvalidRecordID),
		errors.Is(err, model.ErrInvalidRecordRevision),
		errors.Is(err, model.ErrInvalidRecordTitle),
		errors.Is(err, model.ErrInvalidTextPayload),
		errors.Is(err, model.ErrInvalidCredentialsPayload),
		errors.Is(err, model.ErrInvalidCardPayload),
		errors.Is(err, model.ErrInvalidBinaryPayload),
		errors.Is(err, model.ErrRecordTypeUnsupported):
		return status.Error(codes.InvalidArgument, "invalid request")
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}

func unauthenticatedError() error {
	return status.Error(codes.Unauthenticated, "authentication required")
}
