package apierror

import (
	"regexp"
	"testing"
)

func TestParse(t *testing.T) {
	t.Parallel()

	known := []Code{
		InvalidRequest,
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
		UnsupportedMediaType,
	}

	for _, want := range known {
		want := want
		t.Run(string(want), func(t *testing.T) {
			t.Parallel()

			got, ok := Parse(string(want))
			if !ok || got != want {
				t.Fatalf("Parse(%q) = %q, %t; want %q, true", want, got, ok, want)
			}
		})
	}
}

func TestParseRejectsUnknownCode(t *testing.T) {
	t.Parallel()

	if got, ok := Parse("future_error"); ok || got != "" {
		t.Fatalf("Parse() = %q, %t; want empty code, false", got, ok)
	}
}

func TestGRPCReasonRoundTrip(t *testing.T) {
	t.Parallel()

	reasonPattern := regexp.MustCompile(`^[A-Z][A-Z0-9_]*[A-Z0-9]$`)
	known := []Code{
		InvalidRequest,
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
		UnsupportedMediaType,
	}

	for _, want := range known {
		want := want
		t.Run(string(want), func(t *testing.T) {
			t.Parallel()

			reason, ok := GRPCReason(want)
			if !ok || reason == "" {
				t.Fatalf("GRPCReason(%q) = %q, %t; want non-empty reason, true", want, reason, ok)
			}
			if len(reason) > 63 || !reasonPattern.MatchString(reason) {
				t.Fatalf("GRPCReason(%q) = %q; want valid ErrorInfo reason", want, reason)
			}

			got, ok := ParseGRPCReason(reason)
			if !ok || got != want {
				t.Fatalf("ParseGRPCReason(%q) = %q, %t; want %q, true", reason, got, ok, want)
			}
		})
	}
}

func TestGRPCReasonRejectsUnknownCode(t *testing.T) {
	t.Parallel()

	if got, ok := GRPCReason(Code("future_error")); ok || got != "" {
		t.Fatalf("GRPCReason() = %q, %t; want empty reason, false", got, ok)
	}
}

func TestParseGRPCReasonRejectsInvalidReason(t *testing.T) {
	t.Parallel()

	tests := []string{
		"future_error",
		"invalid_request",
		"FUTURE_ERROR",
		"",
	}

	for _, reason := range tests {
		reason := reason
		t.Run(reason, func(t *testing.T) {
			t.Parallel()

			if got, ok := ParseGRPCReason(reason); ok || got != "" {
				t.Fatalf("ParseGRPCReason(%q) = %q, %t; want empty code, false", reason, got, ok)
			}
		})
	}
}
