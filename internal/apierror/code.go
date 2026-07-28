// Package apierror определяет стабильные семантические коды ошибок API.
package apierror

// Domain идентифицирует ошибки GopherKeeper API в transport-specific metadata.
const Domain = "gopherkeeper.api"

const (
	// InvalidRequest обозначает некорректный запрос к API.
	InvalidRequest Code = "invalid_request"

	// InvalidCredentials обозначает неверные учётные данные.
	InvalidCredentials Code = "invalid_credentials"

	// LoginAlreadyExists обозначает занятый login.
	LoginAlreadyExists Code = "login_already_exists"

	// Unauthorized обозначает отсутствие действующей авторизации.
	Unauthorized Code = "unauthorized"

	// RequestTooLarge обозначает превышение допустимого размера transport-запроса.
	RequestTooLarge Code = "request_too_large"

	// PayloadTooLarge обозначает превышение допустимого размера данных записи.
	PayloadTooLarge Code = "payload_too_large"

	// InvalidRecordData обозначает некорректные данные приватной записи.
	InvalidRecordData Code = "invalid_record_data"

	// RecordNotFound обозначает отсутствие запрошенной записи.
	RecordNotFound Code = "record_not_found"

	// RecordRevisionConflict обозначает конфликт ревизий записи.
	RecordRevisionConflict Code = "record_revision_conflict"

	// RecordDecryptionFailed обозначает невозможность расшифровать запись.
	RecordDecryptionFailed Code = "record_decryption_failed"

	// PreconditionRequired обозначает отсутствие обязательной ревизии записи.
	PreconditionRequired Code = "precondition_required"

	// Internal обозначает внутреннюю ошибку Сервера.
	Internal Code = "internal_error"

	// UnsupportedMediaType обозначает неподдерживаемый формат HTTP-запроса.
	UnsupportedMediaType Code = "unsupported_media_type"
)

// Code идентифицирует ошибку API независимо от используемого транспорта.
type Code string

// Parse преобразует строковое представление в известный код ошибки API.
func Parse(value string) (Code, bool) {
	code := Code(value)

	switch code {
	case InvalidRequest,
		InvalidCredentials,
		LoginAlreadyExists,
		Unauthorized,
		RequestTooLarge,
		PayloadTooLarge,
		InvalidRecordData,
		RecordNotFound,
		RecordRevisionConflict,
		RecordDecryptionFailed,
		PreconditionRequired,
		Internal,
		UnsupportedMediaType:
		return code, true
	default:
		return "", false
	}
}

// GRPCReason возвращает стабильное значение google.rpc.ErrorInfo.Reason для кода API.
func GRPCReason(code Code) (string, bool) {
	switch code {
	case InvalidRequest:
		return "INVALID_REQUEST", true
	case InvalidCredentials:
		return "INVALID_CREDENTIALS", true
	case LoginAlreadyExists:
		return "LOGIN_ALREADY_EXISTS", true
	case Unauthorized:
		return "UNAUTHORIZED", true
	case RequestTooLarge:
		return "REQUEST_TOO_LARGE", true
	case PayloadTooLarge:
		return "PAYLOAD_TOO_LARGE", true
	case InvalidRecordData:
		return "INVALID_RECORD_DATA", true
	case RecordNotFound:
		return "RECORD_NOT_FOUND", true
	case RecordRevisionConflict:
		return "RECORD_REVISION_CONFLICT", true
	case RecordDecryptionFailed:
		return "RECORD_DECRYPTION_FAILED", true
	case PreconditionRequired:
		return "PRECONDITION_REQUIRED", true
	case Internal:
		return "INTERNAL_ERROR", true
	case UnsupportedMediaType:
		return "UNSUPPORTED_MEDIA_TYPE", true
	default:
		return "", false
	}
}

// ParseGRPCReason преобразует google.rpc.ErrorInfo.Reason в известный код ошибки API.
func ParseGRPCReason(reason string) (Code, bool) {
	switch reason {
	case "INVALID_REQUEST":
		return InvalidRequest, true
	case "INVALID_CREDENTIALS":
		return InvalidCredentials, true
	case "LOGIN_ALREADY_EXISTS":
		return LoginAlreadyExists, true
	case "UNAUTHORIZED":
		return Unauthorized, true
	case "REQUEST_TOO_LARGE":
		return RequestTooLarge, true
	case "PAYLOAD_TOO_LARGE":
		return PayloadTooLarge, true
	case "INVALID_RECORD_DATA":
		return InvalidRecordData, true
	case "RECORD_NOT_FOUND":
		return RecordNotFound, true
	case "RECORD_REVISION_CONFLICT":
		return RecordRevisionConflict, true
	case "RECORD_DECRYPTION_FAILED":
		return RecordDecryptionFailed, true
	case "PRECONDITION_REQUIRED":
		return PreconditionRequired, true
	case "INTERNAL_ERROR":
		return Internal, true
	case "UNSUPPORTED_MEDIA_TYPE":
		return UnsupportedMediaType, true
	default:
		return "", false
	}
}
