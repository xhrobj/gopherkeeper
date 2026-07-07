package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

type userLogoutterFunc func(context.Context) error

func (f userLogoutterFunc) Logout(ctx context.Context) error {
	return f(ctx)
}

func TestExecuteLogout(t *testing.T) {
	var output bytes.Buffer
	var called bool

	err := executeLogout(
		context.Background(),
		userLogoutterFunc(func(context.Context) error {
			called = true
			return nil
		}),
		&output,
	)
	if err != nil {
		t.Fatalf("executeLogout() error = %v", err)
	}

	if !called {
		t.Error("logoutter was not called")
	}
	if got := output.String(); got != "logged out\n" {
		t.Errorf("output = %q, want logged out", got)
	}
}

func TestExecuteLogout_ReturnsApplicationError(t *testing.T) {
	applicationError := errors.New("permission denied")

	err := executeLogout(
		context.Background(),
		userLogoutterFunc(func(context.Context) error {
			return applicationError
		}),
		&bytes.Buffer{},
	)
	if err == nil {
		t.Fatal("executeLogout() error = nil, want application error")
	}
	if !errors.Is(err, applicationError) {
		t.Error("logout error does not preserve application error")
	}
	if strings.Contains(err.Error(), "test.jwt.token") {
		t.Error("logout error contains access token")
	}
}
