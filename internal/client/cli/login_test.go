package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

func TestExecuteLogin(t *testing.T) {
	createdAt := time.Date(2026, time.July, 4, 12, 0, 0, 0, time.UTC)
	passwords := &passwordReaderStub{
		hiddenValues: []string{testRegistrationPassword},
	}

	var output bytes.Buffer

	err := executeLogin(
		context.Background(),
		userLoggerFunc(func(_ context.Context, login, password string) (model.User, error) {
			if login != "alice" {
				t.Errorf("login = %q, want alice", login)
			}
			if password != testRegistrationPassword {
				t.Error("login application received unexpected password")
			}

			return model.User{
				ID:        42,
				Login:     "alice",
				CreatedAt: createdAt,
			}, nil
		}),
		passwords,
		passwordStreams{
			input:        strings.NewReader(""),
			output:       &output,
			promptOutput: io.Discard,
		},
		" Alice ",
	)
	if err != nil {
		t.Fatalf("executeLogin() error = %v", err)
	}

	if passwords.hiddenCalls != 1 {
		t.Errorf("hidden password reads = %d, want 1", passwords.hiddenCalls)
	}
	if passwords.lineCalls != 0 {
		t.Errorf("line reads = %d, want 0", passwords.lineCalls)
	}
	if got := output.String(); got != "User alice logged in successfully.\n" {
		t.Errorf("output = %q, want success message", got)
	}
	if strings.Contains(output.String(), testRegistrationPassword) {
		t.Error("login output contains password")
	}
	if strings.Contains(output.String(), "test.jwt.token") {
		t.Error("login output contains access token")
	}
}

func TestExecuteLogin_ReturnsApplicationError(t *testing.T) {
	applicationError := errors.New("invalid login or password")

	err := executeLogin(
		context.Background(),
		userLoggerFunc(func(context.Context, string, string) (model.User, error) {
			return model.User{}, applicationError
		}),
		&passwordReaderStub{hiddenValues: []string{testRegistrationPassword}},
		passwordStreams{
			input:        strings.NewReader(""),
			output:       io.Discard,
			promptOutput: io.Discard,
		},
		"eve",
	)
	if err == nil {
		t.Fatal("executeLogin() error = nil, want application error")
	}
	if !errors.Is(err, applicationError) {
		t.Error("login error does not preserve application error")
	}
	if strings.Contains(err.Error(), testRegistrationPassword) {
		t.Error("application error contains password")
	}
	if strings.Contains(err.Error(), "test.jwt.token") {
		t.Error("application error contains access token")
	}
}

func TestExecuteLogin_RejectsInvalidLoginBeforePasswordInput(t *testing.T) {
	loggerCalled := false
	passwords := &passwordReaderStub{
		hiddenValues: []string{testRegistrationPassword},
	}

	err := executeLogin(
		context.Background(),
		userLoggerFunc(func(context.Context, string, string) (model.User, error) {
			loggerCalled = true
			return model.User{}, nil
		}),
		passwords,
		passwordStreams{
			input:        strings.NewReader(""),
			output:       io.Discard,
			promptOutput: io.Discard,
		},
		"-a",
	)
	if !errors.Is(err, errInvalidLoginArgument) {
		t.Fatalf("executeLogin() error = %v, want %v", err, errInvalidLoginArgument)
	}
	if loggerCalled {
		t.Error("login application was called for invalid login")
	}
	if passwords.hiddenCalls != 0 {
		t.Errorf("hidden password reads = %d, want 0", passwords.hiddenCalls)
	}
	if passwords.lineCalls != 0 {
		t.Errorf("line reads = %d, want 0", passwords.lineCalls)
	}
}
