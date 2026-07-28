package grpcclient

import (
	"context"
	"fmt"
	"strings"

	"github.com/xhrobj/gopherkeeper/internal/apierror"
	"github.com/xhrobj/gopherkeeper/internal/client/failure"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RPCError представляет ошибку, возвращённую gRPC API Сервера.
type RPCError struct {
	// Code содержит канонический gRPC status code.
	Code codes.Code

	// APIErrorCode содержит transport-neutral код ошибки API, если Сервер его передал.
	APIErrorCode apierror.Code

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

func mapRPCError(operation string, err error) error {
	if err == nil {
		return nil
	}

	grpcStatus, ok := status.FromError(err)
	if !ok {
		return failure.Network("send "+operation+" gRPC request", err)
	}

	code := grpcStatus.Code()
	apiCode := apiErrorCodeFromStatus(grpcStatus)
	cause := error(nil)
	kind := failure.Unknown
	userMessage := failure.Reason(failure.Unknown)

	switch code {
	case codes.Canceled:
		cause = context.Canceled
		kind = failure.Canceled
		userMessage = failure.Reason(kind)
	case codes.DeadlineExceeded:
		cause = context.DeadlineExceeded
		kind = failure.Timeout
		userMessage = failure.Reason(kind)
	case codes.Unavailable:
		kind = unavailableRPCFailureKind(grpcStatus.Message())
		userMessage = failure.Reason(kind)
	default:
		if apiCode != "" {
			cause = failure.CauseFromAPIErrorCode(apiCode)
			kind = failure.KindFromAPIErrorCode(apiCode)
			userMessage = strings.TrimSpace(grpcStatus.Message())
			if userMessage == "" {
				userMessage = failure.Reason(kind)
			}
		}
	}

	return &RPCError{
		Code:         code,
		APIErrorCode: apiCode,
		Message:      grpcStatus.Message(),
		operation:    operation,
		cause:        cause,
		transport:    err,
		kind:         kind,
		userMessage:  userMessage,
	}
}

func mapListRecordsError(err error) error {
	grpcStatus := status.Convert(err)
	if grpcStatus.Code() != codes.ResourceExhausted || apiErrorCodeFromStatus(grpcStatus) != "" {
		return mapRPCError("list records", err)
	}

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

func apiErrorCodeFromStatus(grpcStatus *status.Status) apierror.Code {
	if grpcStatus == nil {
		return ""
	}

	for _, detail := range grpcStatus.Details() {
		info, ok := detail.(*errdetails.ErrorInfo)
		if !ok || info.GetDomain() != apierror.Domain {
			continue
		}

		if code, known := apierror.ParseGRPCReason(info.GetReason()); known {
			return code
		}
	}

	return ""
}

func unavailableRPCFailureKind(message string) failure.Kind {
	message = strings.ToLower(message)

	switch {
	case strings.Contains(message, "x509") || strings.Contains(message, "certificate"):
		return failure.TLSCertificate
	case strings.Contains(message, "tls"):
		return failure.TLSHandshake
	case strings.Contains(message, "no such host") || strings.Contains(message, "name or service not known"):
		return failure.HostNotFound
	case strings.Contains(message, "network is unreachable"):
		return failure.NetworkUnreachable
	case strings.Contains(message, "timeout") || strings.Contains(message, "deadline exceeded"):
		return failure.Timeout
	default:
		return failure.Unavailable
	}
}
