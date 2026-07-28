// Package errorcode преобразует доменные ошибки Сервера в коды API.
package errorcode

import (
	"errors"

	"github.com/xhrobj/gopherkeeper/internal/apierror"
	"github.com/xhrobj/gopherkeeper/internal/model"
	"github.com/xhrobj/gopherkeeper/internal/server/service"
)

// FromError возвращает стабильный код API для серверной ошибки.
// Для nil возвращается пустой код.
func FromError(err error) apierror.Code {
	if err == nil {
		return ""
	}

	switch {
	case errors.Is(err, service.ErrInvalidCredentials):
		return apierror.InvalidCredentials
	case errors.Is(err, model.ErrLoginAlreadyExists):
		return apierror.LoginAlreadyExists
	case errors.Is(err, model.ErrUserNotFound), errors.Is(err, model.ErrUnauthorized):
		return apierror.Unauthorized
	case errors.Is(err, model.ErrPayloadTooLarge):
		return apierror.PayloadTooLarge
	case errors.Is(err, model.ErrRecordNotFound):
		return apierror.RecordNotFound
	case errors.Is(err, model.ErrRecordRevisionConflict):
		return apierror.RecordRevisionConflict
	case errors.Is(err, model.ErrRecordDecryptionFailed):
		return apierror.RecordDecryptionFailed
	case errors.Is(err, model.ErrRecordPreconditionRequired):
		return apierror.PreconditionRequired
	case errors.Is(err, service.ErrInvalidLogin),
		errors.Is(err, service.ErrInvalidPassword),
		errors.Is(err, service.ErrPasswordTooShort),
		errors.Is(err, service.ErrPasswordTooLong):
		return apierror.InvalidRequest
	case errors.Is(err, model.ErrInvalidRecordID),
		errors.Is(err, model.ErrInvalidRecordRevision),
		errors.Is(err, model.ErrInvalidRecordTitle),
		errors.Is(err, model.ErrInvalidTextPayload),
		errors.Is(err, model.ErrInvalidCredentialsPayload),
		errors.Is(err, model.ErrInvalidCardPayload),
		errors.Is(err, model.ErrInvalidBinaryPayload),
		errors.Is(err, model.ErrRecordTypeUnsupported),
		errors.Is(err, model.ErrInvalidRecordData):
		return apierror.InvalidRecordData
	default:
		return apierror.Internal
	}
}
