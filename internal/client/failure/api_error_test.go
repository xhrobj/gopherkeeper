package failure

import (
	"errors"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/apierror"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

func TestAPIErrorCodeMapping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		code      apierror.Code
		wantCause error
		wantKind  Kind
	}{
		{name: "invalid request", code: apierror.InvalidRequest, wantKind: Validation},
		{name: "invalid credentials", code: apierror.InvalidCredentials, wantCause: model.ErrInvalidCredentials, wantKind: Unauthorized},
		{name: "login already exists", code: apierror.LoginAlreadyExists, wantCause: model.ErrLoginAlreadyExists, wantKind: Conflict},
		{name: "unauthorized", code: apierror.Unauthorized, wantCause: model.ErrUnauthorized, wantKind: Unauthorized},
		{name: "request too large", code: apierror.RequestTooLarge, wantKind: TooLarge},
		{name: "payload too large", code: apierror.PayloadTooLarge, wantCause: model.ErrPayloadTooLarge, wantKind: TooLarge},
		{name: "invalid record data", code: apierror.InvalidRecordData, wantCause: model.ErrInvalidRecordData, wantKind: Validation},
		{name: "record not found", code: apierror.RecordNotFound, wantCause: model.ErrRecordNotFound, wantKind: NotFound},
		{name: "revision conflict", code: apierror.RecordRevisionConflict, wantCause: model.ErrRecordRevisionConflict, wantKind: Conflict},
		{name: "decryption failed", code: apierror.RecordDecryptionFailed, wantCause: model.ErrRecordDecryptionFailed, wantKind: Unknown},
		{name: "precondition required", code: apierror.PreconditionRequired, wantCause: model.ErrRecordPreconditionRequired, wantKind: Validation},
		{name: "internal", code: apierror.Internal, wantKind: Unknown},
		{name: "unsupported media type", code: apierror.UnsupportedMediaType, wantKind: Validation},
		{name: "empty", wantKind: Unknown},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			cause := CauseFromAPIErrorCode(test.code)
			if test.wantCause == nil {
				if cause != nil {
					t.Fatalf("CauseFromAPIErrorCode() = %v, want nil", cause)
				}
			} else if !errors.Is(cause, test.wantCause) {
				t.Fatalf("CauseFromAPIErrorCode() = %v, want %v", cause, test.wantCause)
			}

			if got := KindFromAPIErrorCode(test.code); got != test.wantKind {
				t.Fatalf("KindFromAPIErrorCode() = %d, want %d", got, test.wantKind)
			}
		})
	}
}
