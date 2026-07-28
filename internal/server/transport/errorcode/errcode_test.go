package errorcode

import (
	"errors"
	"fmt"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/apierror"
	"github.com/xhrobj/gopherkeeper/internal/model"
	"github.com/xhrobj/gopherkeeper/internal/server/service"
)

func TestFromError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want apierror.Code
	}{
		{name: "invalid credentials", err: service.ErrInvalidCredentials, want: apierror.InvalidCredentials},
		{name: "login already exists", err: model.ErrLoginAlreadyExists, want: apierror.LoginAlreadyExists},
		{name: "user not found", err: model.ErrUserNotFound, want: apierror.Unauthorized},
		{name: "unauthorized", err: model.ErrUnauthorized, want: apierror.Unauthorized},
		{name: "payload too large", err: model.ErrPayloadTooLarge, want: apierror.PayloadTooLarge},
		{name: "record not found", err: model.ErrRecordNotFound, want: apierror.RecordNotFound},
		{name: "revision conflict", err: model.ErrRecordRevisionConflict, want: apierror.RecordRevisionConflict},
		{name: "decryption failed", err: model.ErrRecordDecryptionFailed, want: apierror.RecordDecryptionFailed},
		{name: "precondition required", err: model.ErrRecordPreconditionRequired, want: apierror.PreconditionRequired},
		{name: "invalid login", err: service.ErrInvalidLogin, want: apierror.InvalidRequest},
		{name: "invalid password", err: service.ErrInvalidPassword, want: apierror.InvalidRequest},
		{name: "password too short", err: service.ErrPasswordTooShort, want: apierror.InvalidRequest},
		{name: "password too long", err: service.ErrPasswordTooLong, want: apierror.InvalidRequest},
		{name: "invalid record id", err: model.ErrInvalidRecordID, want: apierror.InvalidRecordData},
		{name: "invalid record revision", err: model.ErrInvalidRecordRevision, want: apierror.InvalidRecordData},
		{name: "invalid record title", err: model.ErrInvalidRecordTitle, want: apierror.InvalidRecordData},
		{name: "invalid text payload", err: model.ErrInvalidTextPayload, want: apierror.InvalidRecordData},
		{name: "invalid credentials payload", err: model.ErrInvalidCredentialsPayload, want: apierror.InvalidRecordData},
		{name: "invalid card payload", err: model.ErrInvalidCardPayload, want: apierror.InvalidRecordData},
		{name: "invalid binary payload", err: model.ErrInvalidBinaryPayload, want: apierror.InvalidRecordData},
		{name: "unsupported record type", err: model.ErrRecordTypeUnsupported, want: apierror.InvalidRecordData},
		{name: "invalid record data", err: model.ErrInvalidRecordData, want: apierror.InvalidRecordData},
		{name: "wrapped", err: fmt.Errorf("register user: %w", model.ErrLoginAlreadyExists), want: apierror.LoginAlreadyExists},
		{name: "unknown", err: errors.New("database failed"), want: apierror.Internal},
		{name: "nil", err: nil, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := FromError(tt.err); got != tt.want {
				t.Fatalf("FromError() = %q, want %q", got, tt.want)
			}
		})
	}
}
