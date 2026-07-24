package usecase

import (
	"context"
	"errors"
	"testing"
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
	if err := (&Application{}).Logout(context.Background()); err == nil {
		t.Fatal("Logout() error = nil")
	}
}
