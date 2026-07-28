package failure

import (
	"github.com/xhrobj/gopherkeeper/internal/apierror"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

// CauseFromAPIErrorCode возвращает transport-neutral причину для кода ошибки API.
func CauseFromAPIErrorCode(code apierror.Code) error {
	switch code {
	case apierror.InvalidCredentials:
		return model.ErrInvalidCredentials
	case apierror.LoginAlreadyExists:
		return model.ErrLoginAlreadyExists
	case apierror.Unauthorized:
		return model.ErrUnauthorized
	case apierror.PayloadTooLarge:
		return model.ErrPayloadTooLarge
	case apierror.InvalidRecordData:
		return model.ErrInvalidRecordData
	case apierror.RecordNotFound:
		return model.ErrRecordNotFound
	case apierror.RecordRevisionConflict:
		return model.ErrRecordRevisionConflict
	case apierror.RecordDecryptionFailed:
		return model.ErrRecordDecryptionFailed
	case apierror.PreconditionRequired:
		return model.ErrRecordPreconditionRequired
	default:
		return nil
	}
}

// KindFromAPIErrorCode возвращает пользовательскую категорию для кода ошибки API.
func KindFromAPIErrorCode(code apierror.Code) Kind {
	switch code {
	case apierror.InvalidRequest, apierror.InvalidRecordData, apierror.PreconditionRequired, apierror.UnsupportedMediaType:
		return Validation
	case apierror.InvalidCredentials, apierror.Unauthorized:
		return Unauthorized
	case apierror.LoginAlreadyExists, apierror.RecordRevisionConflict:
		return Conflict
	case apierror.RequestTooLarge, apierror.PayloadTooLarge:
		return TooLarge
	case apierror.RecordNotFound:
		return NotFound
	default:
		return Unknown
	}
}
