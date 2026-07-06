package httpserver

const (
	errorCodeInvalidRequest       = "invalid_request"
	errorCodeInvalidCredentials   = "invalid_credentials"
	errorCodeLoginAlreadyExists   = "login_already_exists"
	errorCodePayloadTooLarge      = "payload_too_large"
	errorCodeUnsupportedMediaType = "unsupported_media_type"
	errorCodeInternal             = "internal_error"
)

const (
	errorMessageInvalidCredentials   = "invalid login or password"
	errorMessagePayloadTooLarge      = "request body is too large"
	errorMessageUnsupportedMediaType = "content type must be application/json"
	errorMessageInternal             = "internal server error"
)
