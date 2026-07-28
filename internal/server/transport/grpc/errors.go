package grpcserver

import (
	"context"
	"errors"

	"github.com/xhrobj/gopherkeeper/internal/apierror"
	"github.com/xhrobj/gopherkeeper/internal/server/transport/errorcode"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type grpcErrorSpec struct {
	code    codes.Code
	message string
}

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
	}

	apiCode := errorcode.FromError(err)
	if errors.Is(err, errInvalidRequest) {
		apiCode = apierror.InvalidRequest
	}

	spec := grpcErrorSpecForCode(apiCode)

	return statusErrorWithAPIErrorCode(spec.code, spec.message, apiCode)
}

func unauthenticatedError() error {
	return statusErrorWithAPIErrorCode(
		codes.Unauthenticated,
		"authentication required",
		apierror.Unauthorized,
	)
}

func grpcErrorSpecForCode(apiCode apierror.Code) grpcErrorSpec {
	switch apiCode {
	case apierror.InvalidRequest:
		return grpcErrorSpec{code: codes.InvalidArgument, message: "invalid request"}
	case apierror.InvalidRecordData:
		return grpcErrorSpec{code: codes.InvalidArgument, message: "invalid record data"}
	case apierror.InvalidCredentials:
		return grpcErrorSpec{code: codes.Unauthenticated, message: "invalid login or password"}
	case apierror.LoginAlreadyExists:
		return grpcErrorSpec{code: codes.AlreadyExists, message: "login is already registered"}
	case apierror.Unauthorized:
		return grpcErrorSpec{code: codes.Unauthenticated, message: "authentication required"}
	case apierror.PayloadTooLarge:
		return grpcErrorSpec{code: codes.ResourceExhausted, message: "payload is too large"}
	case apierror.RecordNotFound:
		return grpcErrorSpec{code: codes.NotFound, message: "record not found"}
	case apierror.RecordRevisionConflict:
		return grpcErrorSpec{code: codes.Aborted, message: "record revision conflict"}
	case apierror.RecordDecryptionFailed:
		return grpcErrorSpec{code: codes.DataLoss, message: "record data could not be decrypted"}
	case apierror.PreconditionRequired:
		return grpcErrorSpec{code: codes.FailedPrecondition, message: "record revision is required"}
	default:
		return grpcErrorSpec{code: codes.Internal, message: "internal server error"}
	}
}

func statusErrorWithAPIErrorCode(code codes.Code, message string, apiCode apierror.Code) error {
	grpcStatus := status.New(code, message)

	reason, ok := apierror.GRPCReason(apiCode)
	if !ok {
		return grpcStatus.Err()
	}

	withDetails, err := grpcStatus.WithDetails(&errdetails.ErrorInfo{
		Reason: reason,
		Domain: apierror.Domain,
	})
	if err != nil {
		return grpcStatus.Err()
	}

	return withDetails.Err()
}
