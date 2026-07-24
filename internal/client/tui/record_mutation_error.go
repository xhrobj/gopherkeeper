package tui

import (
	"context"
	"errors"

	recordmodel "github.com/xhrobj/gopherkeeper/internal/model"
)

type recordMutationKind int

const (
	recordMutationCreate recordMutationKind = iota
	recordMutationUpdate
)

func cleanRecordMutationError(err error, kind recordMutationKind) string {
	if err == nil {
		if kind == recordMutationUpdate {
			return "Unknown record update error"
		}
		return "Unknown record creation error"
	}

	if errors.Is(err, context.Canceled) {
		if kind == recordMutationUpdate {
			return "Record update canceled"
		}
		return "Record creation canceled"
	}

	if kind == recordMutationUpdate && errors.Is(err, recordmodel.ErrRecordRevisionConflict) {
		return "The record was changed on another device. Reopen it and try again."
	}

	if errors.Is(err, recordmodel.ErrPayloadTooLarge) {
		return "Payload exceeds the allowed size"
	}

	if message := cleanRecordValidationError(err); message != "" {
		return message
	}

	fallback := "Unable to create record"
	if kind == recordMutationUpdate {
		fallback = "Unable to update record"
	}

	return cleanFailureMessage(err, fallback)
}

func cleanRecordValidationError(err error) string {
	switch {
	case errors.Is(err, recordmodel.ErrInvalidRecordTitle):
		return "Invalid record title"
	case errors.Is(err, recordmodel.ErrInvalidTextPayload):
		return "Text or metadata is invalid; metadata must not exceed 255 characters"
	case errors.Is(err, recordmodel.ErrInvalidCredentialsPayload):
		return "Login and password are required; credentials fields must not exceed 255 characters"
	case errors.Is(err, recordmodel.ErrInvalidCardPayload):
		return "Card fields are invalid: use 12–20 digits, MM/YY, and an optional 3-digit CVV"
	case errors.Is(err, recordmodel.ErrInvalidBinaryPayload):
		return "Invalid filename, metadata, or binary data"
	default:
		return ""
	}
}
