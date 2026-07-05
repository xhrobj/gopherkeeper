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

type userLoggerFunc func(context.Context, string, string) (model.User, error)

func (f userLoggerFunc) Login(
	ctx context.Context,
	login string,
	password string,
) (model.User, error) {
	return f(ctx, login, password)
}

func TestExecuteLogin_Interactive(t *testing.T) {
	createdAt := time.Date(2026, time.July, 4, 12, 0, 0, 0, time.UTC)
	passwords := &passwordReaderStub{
		hiddenValues: []string{testRegistrationPassword},
	}

	var output bytes.Buffer

	err := executeLogin(
		context.Background(),
		userLoggerFunc(func(_ context.Context, login, password string) (model.User, error) {
			if login != " Alice " {
				t.Errorf("login = %q, want %q", login, " Alice ")
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
		loginStreams{
			input:        strings.NewReader(""),
			output:       &output,
			promptOutput: io.Discard,
		},
		" Alice ",
		false,
	)
	if err != nil {
		t.Fatalf("executeLogin() error = %v", err)
	}

	if passwords.hiddenCalls != 1 {
		t.Errorf("hidden password reads = %d, want 1", passwords.hiddenCalls)
	}
	if passwords.lineCalls != 0 {
		t.Errorf("stdin password reads = %d, want 0", passwords.lineCalls)
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

func TestExecuteLogin_PasswordStdin(t *testing.T) {
	passwords := &passwordReaderStub{lineValue: testRegistrationPassword}

	err := executeLogin(
		context.Background(),
		userLoggerFunc(func(_ context.Context, login, password string) (model.User, error) {
			if login != "bob" {
				t.Errorf("login = %q, want bob", login)
			}
			if password != testRegistrationPassword {
				t.Error("login application received unexpected password")
			}

			return model.User{
				ID:        42,
				Login:     "bob",
				CreatedAt: time.Date(2026, time.July, 4, 12, 0, 0, 0, time.UTC),
			}, nil
		}),
		passwords,
		loginStreams{
			input:        strings.NewReader(testRegistrationPassword + "\n"),
			output:       io.Discard,
			promptOutput: io.Discard,
		},
		"bob",
		true,
	)
	if err != nil {
		t.Fatalf("executeLogin() error = %v", err)
	}

	if passwords.lineCalls != 1 {
		t.Errorf("stdin password reads = %d, want 1", passwords.lineCalls)
	}
	if passwords.hiddenCalls != 0 {
		t.Errorf("hidden password reads = %d, want 0", passwords.hiddenCalls)
	}
}

func TestExecuteLogin_ReturnsApplicationError(t *testing.T) {
	applicationError := errors.New("invalid login or password")

	err := executeLogin(
		context.Background(),
		userLoggerFunc(func(context.Context, string, string) (model.User, error) {
			return model.User{}, applicationError
		}),
		&passwordReaderStub{lineValue: testRegistrationPassword},
		loginStreams{
			input:        strings.NewReader(testRegistrationPassword + "\n"),
			output:       io.Discard,
			promptOutput: io.Discard,
		},
		"eve",
		true,
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
