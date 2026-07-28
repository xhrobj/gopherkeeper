package httpserver

const (
	errorCodeInvalidRequest       = "invalid_request"
	errorCodeInvalidCredentials   = "invalid_credentials"
	errorCodeLoginAlreadyExists   = "login_already_exists"
	errorCodePayloadTooLarge      = "payload_too_large"
	errorCodeUnsupportedMediaType = "unsupported_media_type"
	errorCodeInternal             = "internal_error"
	errorCodeRecordDecryption     = "record_decryption_failed"
)

const (
	errorMessageInvalidCredentials   = "invalid login or password"
	errorMessagePayloadTooLarge      = "payload is too large"
	errorMessageUnsupportedMediaType = "content type must be application/json"
	errorMessageInternal             = "internal server error"
	errorMessageRecordDecryption     = "record data could not be decrypted"
)
