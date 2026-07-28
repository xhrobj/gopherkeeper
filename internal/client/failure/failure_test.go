package failure

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"net/http"
	"net/url"
	"syscall"
	"testing"
)

type classifiedError struct {
	kind    Kind
	message string
	cause   error
}

type timeoutError struct{}

func (e classifiedError) Error() string       { return "classified error" }
func (e classifiedError) Unwrap() error       { return e.cause }
func (e classifiedError) FailureKind() Kind   { return e.kind }
func (e classifiedError) UserMessage() string { return e.message }

func (timeoutError) Error() string   { return "request timed out" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return true }

func TestError(t *testing.T) {
	cause := errors.New("low-level failure")
	tests := []struct {
		name string
		err  *Error
		want string
	}{
		{
			name: "detail and cause",
			err:  &Error{detail: "load configuration", cause: cause},
			want: "load configuration: low-level failure",
		},
		{
			name: "detail only",
			err:  &Error{detail: "load configuration"},
			want: "load configuration",
		},
		{
			name: "cause only",
			err:  &Error{cause: cause},
			want: "low-level failure",
		},
		{
			name: "empty",
			err:  &Error{},
			want: "client error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Fatalf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWrap(t *testing.T) {
	cause := errors.New("disk path /private/cache.db")
	err := Wrap(Validation, "  read encrypted cache  ", "  Unable to read local cache  ", cause)

	if got := KindOf(err); got != Validation {
		t.Errorf("KindOf() = %d, want %d", got, Validation)
	}
	if got := Message(err); got != "Unable to read local cache" {
		t.Errorf("Message() = %q, want %q", got, "Unable to read local cache")
	}
	if got := err.Error(); got != "read encrypted cache: disk path /private/cache.db" {
		t.Errorf("Error() = %q", got)
	}
	if !errors.Is(err, cause) {
		t.Fatal("Wrap() did not preserve cause")
	}
}

func TestKindOf(t *testing.T) {
	certificate := &x509.Certificate{DNSNames: []string{"example.com"}}
	tests := []struct {
		name string
		err  error
		want Kind
	}{
		{name: "nil", err: nil, want: Unknown},
		{
			name: "typed provider has priority",
			err:  classifiedError{kind: NotFound, cause: context.Canceled},
			want: NotFound,
		},
		{name: "canceled", err: context.Canceled, want: Canceled},
		{name: "deadline", err: context.DeadlineExceeded, want: Timeout},
		{name: "connection refused", err: syscall.ECONNREFUSED, want: Unavailable},
		{name: "network unreachable", err: syscall.ENETUNREACH, want: NetworkUnreachable},
		{name: "host unreachable", err: syscall.EHOSTUNREACH, want: NetworkUnreachable},
		{name: "DNS timeout", err: &net.DNSError{IsTimeout: true}, want: Timeout},
		{name: "DNS lookup", err: &net.DNSError{Name: "missing.example"}, want: HostNotFound},
		{name: "certificate verification", err: &tls.CertificateVerificationError{Err: x509.UnknownAuthorityError{Cert: certificate}}, want: TLSCertificate},
		{name: "unknown CA", err: x509.UnknownAuthorityError{Cert: certificate}, want: TLSCertificate},
		{name: "hostname", err: x509.HostnameError{Certificate: certificate, Host: "other.example"}, want: TLSCertificate},
		{name: "invalid certificate", err: x509.CertificateInvalidError{Cert: certificate, Reason: x509.Expired}, want: TLSCertificate},
		{name: "TLS alert", err: tls.AlertError(40), want: TLSHandshake},
		{name: "TLS record", err: tls.RecordHeaderError{Msg: "invalid record"}, want: TLSHandshake},
		{name: "network timeout", err: timeoutError{}, want: Timeout},
		{
			name: "URL wrapped HTTPS mismatch",
			err: &url.Error{
				Op:  "Get",
				URL: "https://localhost/health",
				Err: http.ErrSchemeMismatch,
			},
			want: HTTPSRequired,
		},
		{
			name: "URL wrapped HTTPS mismatch text is not classified",
			err: &url.Error{
				Op:  "Get",
				URL: "https://localhost/health",
				Err: errors.New("server gave HTTP response to HTTPS client"),
			},
			want: Unknown,
		},
		{
			name: "URL wrapped untyped TLS alert",
			err: &url.Error{
				Op:  "Get",
				URL: "https://localhost/health",
				Err: errors.New("remote error: tls: handshake failure"),
			},
			want: TLSHandshake,
		},
		{name: "certificate text is not classified", err: errors.New("x509: certificate signed by unknown authority"), want: Unknown},
		{name: "TLS text is not classified", err: errors.New("remote error: tls: handshake failure"), want: Unknown},
		{name: "host text is not classified", err: errors.New("dial tcp: no such host"), want: Unknown},
		{name: "network text is not classified", err: errors.New("dial tcp: network is unreachable"), want: Unknown},
		{name: "refused text is not classified", err: errors.New("dial tcp: connection refused"), want: Unknown},
		{name: "timeout text is not classified", err: errors.New("request timeout"), want: Unknown},
		{name: "unknown", err: errors.New("socket closed"), want: Unknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := KindOf(tt.err); got != tt.want {
				t.Fatalf("KindOf() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestMessage(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "nil", err: nil, want: ""},
		{
			name: "provider",
			err:  classifiedError{kind: Validation, message: "  Invalid input  "},
			want: "Invalid input",
		},
		{
			name: "typed fallback",
			err:  Wrap(TooLarge, "payload rejected", "", nil),
			want: "Payload exceeds the allowed size",
		},
		{
			name: "untyped error is not exposed",
			err:  errors.New("open /Users/alice/.config/gopherkeeper/client.json: permission denied"),
			want: "Operation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Message(tt.err); got != tt.want {
				t.Fatalf("Message() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestContext(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		if err := Context("load config", nil); err != nil {
			t.Fatalf("Context() = %v, want nil", err)
		}
	})

	t.Run("empty operation", func(t *testing.T) {
		cause := errors.New("failure")
		if Context("  ", cause) != cause {
			t.Fatalf("Context() returned a new error for empty operation")
		}
	})

	t.Run("raw error", func(t *testing.T) {
		cause := errors.New("open /private/session.json: permission denied")
		err := Context("load online session", cause)

		if got := err.Error(); got != "load online session: open /private/session.json: permission denied" {
			t.Errorf("Error() = %q", got)
		}
		if got := Message(err); got != "Operation failed" {
			t.Errorf("Message() = %q, want Operation failed", got)
		}
		if !errors.Is(err, cause) {
			t.Fatal("Context() did not preserve cause")
		}
	})

	t.Run("typed error", func(t *testing.T) {
		cause := classifiedError{kind: Unauthorized, message: "Not logged in"}
		err := Context("load records", cause)

		if got := KindOf(err); got != Unauthorized {
			t.Errorf("KindOf() = %d, want %d", got, Unauthorized)
		}
		if got := Message(err); got != "Not logged in" {
			t.Errorf("Message() = %q, want Not logged in", got)
		}
	})
}

func TestNetworkPreservesDiagnosticContextAndUserMessage(t *testing.T) {
	err := Network("send login request", syscall.ECONNREFUSED)

	if got := KindOf(err); got != Unavailable {
		t.Fatalf("KindOf() = %d, want %d", got, Unavailable)
	}
	if got := Message(err); got != "Connection refused" {
		t.Fatalf("Message() = %q, want Connection refused", got)
	}
	if got := err.Error(); got != "send login request: connection refused" {
		t.Fatalf("Error() = %q", got)
	}
	if !errors.Is(err, syscall.ECONNREFUSED) {
		t.Fatal("Network() did not preserve cause")
	}
}

func TestNetworkKeepsUnknownFailureUnknown(t *testing.T) {
	cause := errors.New("socket closed")
	err := Network("send request", cause)

	if got := KindOf(err); got != Unknown {
		t.Fatalf("KindOf() = %d, want %d", got, Unknown)
	}
	if got := Message(err); got != "Connection failed" {
		t.Fatalf("Message() = %q, want Connection failed", got)
	}
}

func TestReason(t *testing.T) {
	tests := []struct {
		kind Kind
		want string
	}{
		{kind: Unknown, want: "Operation failed"},
		{kind: Canceled, want: "Operation canceled"},
		{kind: Unavailable, want: "Connection refused"},
		{kind: HostNotFound, want: "Host not found"},
		{kind: NetworkUnreachable, want: "Network unreachable"},
		{kind: Timeout, want: "Connection timed out"},
		{kind: TLSCertificate, want: "Certificate verification failed"},
		{kind: TLSHandshake, want: "TLS handshake failed"},
		{kind: HTTPSRequired, want: "Server does not support HTTPS"},
		{kind: Unauthorized, want: "Not authorized"},
		{kind: Conflict, want: "Conflict"},
		{kind: NotFound, want: "Not found"},
		{kind: Validation, want: "Invalid data"},
		{kind: TooLarge, want: "Payload exceeds the allowed size"},
	}

	for _, tt := range tests {
		if got := Reason(tt.kind); got != tt.want {
			t.Errorf("Reason(%d) = %q, want %q", tt.kind, got, tt.want)
		}
	}
}
