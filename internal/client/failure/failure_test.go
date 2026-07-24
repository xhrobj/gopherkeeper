package failure

import (
	"errors"
	"syscall"
	"testing"
)

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
