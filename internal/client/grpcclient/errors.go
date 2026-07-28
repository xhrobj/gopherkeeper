package grpcclient

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/xhrobj/gopherkeeper/internal/client/failure"
	"github.com/xhrobj/gopherkeeper/internal/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RPCError представляет ошибку, возвращённую gRPC API Сервера.
type RPCError struct {
	// Code содержит канонический gRPC status code.
	Code codes.Code

	// Message содержит безопасное сообщение gRPC API.
	Message string

	operation   string
	cause       error
	transport   error
	kind        failure.Kind
	userMessage string
}

var _ interface{ Unwrap() []error } = (*RPCError)(nil)

// Error возвращает диагностическое описание gRPC-ошибки.
func (e *RPCError) Error() string {
	message := strings.TrimSpace(e.Message)
	if message == "" {
		message = e.Code.String()
	}
	if e.operation == "" {
		return fmt.Sprintf("gRPC request failed: %s: %s", e.Code, message)
	}
	return fmt.Sprintf("%s gRPC request failed: %s: %s", e.operation, e.Code, message)
}

// Unwrap возвращает transport-neutral и исходную gRPC-причины ошибки.
func (e *RPCError) Unwrap() []error {
	result := make([]error, 0, 2)
	if e.cause != nil {
		result = append(result, e.cause)
	}
	if e.transport != nil {
		result = append(result, e.transport)
	}
	return result
}

// FailureKind возвращает категорию ошибки для пользовательского интерфейса.
func (e *RPCError) FailureKind() failure.Kind { return e.kind }

// UserMessage возвращает безопасное пользовательское сообщение.
func (e *RPCError) UserMessage() string {
	if e.userMessage != "" {
		return e.userMessage
	}
	return failure.Reason(e.kind)
}

func mapRPCError(operation string, err error, cause error) error {
	if err == nil {
		return nil
	}

	grpcStatus, ok := status.FromError(err)
	if !ok {
		return failure.Network("send "+operation+" gRPC request", err)
	}

	code := grpcStatus.Code()
	kind := failure.Unknown
	userMessage := grpcStatus.Message()

	switch code {
	case codes.Canceled:
		kind = failure.Canceled
		userMessage = failure.Reason(kind)
		if cause == nil {
			cause = context.Canceled
		}
	case codes.DeadlineExceeded:
		kind = failure.Timeout
		userMessage = failure.Reason(kind)
		if cause == nil {
			cause = context.DeadlineExceeded
		}
	case codes.Unavailable:
		kind = failure.KindOf(err)
		if kind == failure.Unknown {
			kind = failure.Unavailable
		}
		userMessage = failure.Reason(kind)
	case codes.Unauthenticated:
		kind = failure.Unauthorized
	case codes.AlreadyExists, codes.Aborted:
		kind = failure.Conflict
	case codes.NotFound:
		kind = failure.NotFound
	case codes.InvalidArgument, codes.FailedPrecondition:
		kind = failure.Validation
	case codes.ResourceExhausted:
		kind = failure.TooLarge
	case codes.DataLoss:
		if errors.Is(cause, model.ErrRecordDecryptionFailed) {
			userMessage = "Record data could not be decrypted"
		} else {
			userMessage = failure.Reason(failure.Unknown)
		}
	case codes.Internal:
		userMessage = "Internal server error"
	default:
		userMessage = failure.Reason(failure.Unknown)
	}

	return &RPCError{
		Code:        code,
		Message:     grpcStatus.Message(),
		operation:   operation,
		cause:       cause,
		transport:   err,
		kind:        kind,
		userMessage: userMessage,
	}
}

func mapListRecordsError(err error) error {
	if status.Code(err) != codes.ResourceExhausted {
		return mapRPCError("list records", err, recordErrorCause(err))
	}

	grpcStatus := status.Convert(err)
	return &RPCError{
		Code:        grpcStatus.Code(),
		Message:     grpcStatus.Message(),
		operation:   "list records",
		transport:   err,
		kind:        failure.TooLarge,
		userMessage: "Server response is too large",
	}
}

func invalidResponseError(operation string, err error) error {
	return failure.Wrap(
		failure.Unknown,
		"decode "+operation+" gRPC response",
		"Invalid server response",
		err,
	)
}

func registrationErrorCause(err error) error {
	if status.Code(err) == codes.AlreadyExists {
		return model.ErrLoginAlreadyExists
	}
	return nil
}

func loginErrorCause(err error) error {
	if status.Code(err) == codes.Unauthenticated {
		return model.ErrInvalidCredentials
	}
	return nil
}

func currentUserErrorCause(err error) error {
	if status.Code(err) == codes.Unauthenticated {
		return model.ErrUnauthorized
	}
	return nil
}

func recordErrorCause(err error) error {
	switch status.Code(err) {
	case codes.Unauthenticated:
		return model.ErrUnauthorized
	case codes.NotFound:
		return model.ErrRecordNotFound
	case codes.Aborted:
		return model.ErrRecordRevisionConflict
	case codes.DataLoss:
		return model.ErrRecordDecryptionFailed
	case codes.FailedPrecondition:
		return model.ErrRecordPreconditionRequired
	case codes.ResourceExhausted:
		return model.ErrPayloadTooLarge
	case codes.InvalidArgument:
		return model.ErrInvalidRecordData
	default:
		return nil
	}
}
