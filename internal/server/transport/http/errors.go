package httpserver

import "github.com/xhrobj/gopherkeeper/internal/apierror"

const (
	errorCodeInvalidRequest       = string(apierror.InvalidRequest)
	errorCodeInvalidCredentials   = string(apierror.InvalidCredentials)
	errorCodeLoginAlreadyExists   = string(apierror.LoginAlreadyExists)
	errorCodeRequestTooLarge      = string(apierror.RequestTooLarge)
	errorCodePayloadTooLarge      = string(apierror.PayloadTooLarge)
	errorCodeInvalidRecordData    = string(apierror.InvalidRecordData)
	errorCodeUnsupportedMediaType = string(apierror.UnsupportedMediaType)
	errorCodeInternal             = string(apierror.Internal)
	errorCodeRecordDecryption     = string(apierror.RecordDecryptionFailed)
)

const (
	errorMessageRequestTooLarge      = "request is too large"
	errorMessageInvalidCredentials   = "invalid login or password"
	errorMessagePayloadTooLarge      = "payload is too large"
	errorMessageUnsupportedMediaType = "content type must be application/json"
	errorMessageInternal             = "internal server error"
	errorMessageRecordDecryption     = "record data could not be decrypted"
)
