package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/client/failure"
)

func TestApplication_Health(t *testing.T) {
	application := &Application{health: healthGatewayStub{health: func(context.Context) (string, error) {
		return "ok", nil
	}}}

	status, err := application.Health(context.Background())
	if err != nil {
		t.Fatalf("Health() error = %v", err)
	}
	if status != "ok" {
		t.Fatalf("Health() status = %q, want ok", status)
	}
}

func TestApplication_HealthWithoutGateway(t *testing.T) {
	_, err := (&Application{}).Health(context.Background())
	if err == nil {
		t.Fatal("Health() error = nil")
	}
	if got := failure.KindOf(err); got != failure.Unknown {
		t.Errorf("failure.KindOf(Health() error) = %d, want %d", got, failure.Unknown)
	}
	if got := failure.Message(err); got != "Server health check is unavailable" {
		t.Errorf("failure.Message(Health() error) = %q", got)
	}
}

func TestApplication_Logout(t *testing.T) {
	deleted := false
	application := &Application{sessions: func() (SessionStorage, error) {
		return sessionStorageStub{delete: func() error {
			deleted = true
			return nil
		}}, nil
	}}

	if err := application.Logout(context.Background()); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	if !deleted {
		t.Fatal("Logout() did not delete session")
	}
}

func TestApplication_LogoutReturnsDeleteError(t *testing.T) {
	want := errors.New("permission denied")
	application := &Application{sessions: func() (SessionStorage, error) {
		return sessionStorageStub{delete: func() error { return want }}, nil
	}}

	err := application.Logout(context.Background())
	if !errors.Is(err, want) {
		t.Fatalf("Logout() error = %v, want %v", err, want)
	}
}

func TestApplication_LogoutWithoutSessionStorage(t *testing.T) {
	err := (&Application{}).Logout(context.Background())
	if err == nil {
		t.Fatal("Logout() error = nil")
	}
	if got := failure.KindOf(err); got != failure.Unknown {
		t.Errorf("failure.KindOf(Logout() error) = %d, want %d", got, failure.Unknown)
	}
	if got := failure.Message(err); got != "Unable to log out" {
		t.Errorf("failure.Message(Logout() error) = %q", got)
	}
}
