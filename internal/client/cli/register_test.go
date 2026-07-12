package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

const testRegistrationPassword = "correct-horse-battery-staple"

type passwordReaderStub struct {
	hiddenValues []string
	lineValue    string
	hiddenCalls  int
	lineCalls    int
}

func (r *passwordReaderStub) ReadHidden(
	_ io.Reader,
	_ io.Writer,
	_ string,
) (string, error) {
	if r.hiddenCalls >= len(r.hiddenValues) {
		return "", errors.New("unexpected hidden password read")
	}

	value := r.hiddenValues[r.hiddenCalls]
	r.hiddenCalls++

	return value, nil
}

func (r *passwordReaderStub) ReadLine(io.Reader) (string, error) {
	r.lineCalls++
	return r.lineValue, nil
}

func TestExecuteRegistration_Interactive(t *testing.T) {
	passwords := &passwordReaderStub{
		hiddenValues: []string{testRegistrationPassword, testRegistrationPassword},
	}
	var output bytes.Buffer

	err := executeRegistration(
		context.Background(),
		userRegistererFunc(func(_ context.Context, login, password string) (model.User, error) {
			if login != " Alice " {
				t.Errorf("login = %q, want %q", login, " Alice ")
			}
			if password != testRegistrationPassword {
				t.Error("registrar received unexpected password")
			}

			return model.User{Login: "alice"}, nil
		}),
		passwords,
		passwordStreams{
			input:        strings.NewReader(""),
			output:       &output,
			promptOutput: io.Discard,
		},
		" Alice ",
		false,
	)
	if err != nil {
		t.Fatalf("executeRegistration() error = %v", err)
	}

	if passwords.hiddenCalls != 2 {
		t.Errorf("hidden password reads = %d, want 2", passwords.hiddenCalls)
	}
	if passwords.lineCalls != 0 {
		t.Errorf("stdin password reads = %d, want 0", passwords.lineCalls)
	}
	if got := output.String(); got != "User alice registered successfully.\n" {
		t.Errorf("output = %q, want success message", got)
	}
	if strings.Contains(output.String(), testRegistrationPassword) {
		t.Error("registration output contains password")
	}
}

func TestExecuteRegistration_PasswordStdin(t *testing.T) {
	passwords := &passwordReaderStub{lineValue: testRegistrationPassword}

	err := executeRegistration(
		context.Background(),
		userRegistererFunc(func(_ context.Context, login, password string) (model.User, error) {
			if login != "bob" {
				t.Errorf("login = %q, want bob", login)
			}
			if password != testRegistrationPassword {
				t.Error("registrar received unexpected password")
			}

			return model.User{Login: "bob"}, nil
		}),
		passwords,
		passwordStreams{
			input:        strings.NewReader(testRegistrationPassword + "\n"),
			output:       io.Discard,
			promptOutput: io.Discard,
		},
		"bob",
		true,
	)
	if err != nil {
		t.Fatalf("executeRegistration() error = %v", err)
	}

	if passwords.lineCalls != 1 {
		t.Errorf("stdin password reads = %d, want 1", passwords.lineCalls)
	}
	if passwords.hiddenCalls != 0 {
		t.Errorf("hidden password reads = %d, want 0", passwords.hiddenCalls)
	}
}

func TestExecuteRegistration_RejectsMismatchedPasswords(t *testing.T) {
	registrarCalled := false
	passwords := &passwordReaderStub{
		hiddenValues: []string{testRegistrationPassword, "different-password"},
	}

	err := executeRegistration(
		context.Background(),
		userRegistererFunc(func(context.Context, string, string) (model.User, error) {
			registrarCalled = true
			return model.User{}, nil
		}),
		passwords,
		passwordStreams{
			input:        strings.NewReader(""),
			output:       io.Discard,
			promptOutput: io.Discard,
		},
		"eve",
		false,
	)
	if err == nil {
		t.Fatal("executeRegistration() error = nil, want password mismatch")
	}
	if err.Error() != "passwords do not match" {
		t.Errorf("error = %q, want password mismatch", err)
	}
	if registrarCalled {
		t.Error("registrar was called for mismatched passwords")
	}
	if strings.Contains(err.Error(), testRegistrationPassword) {
		t.Error("password mismatch error contains password")
	}
}

func TestExecuteRegistration_ReturnsApplicationError(t *testing.T) {
	applicationError := errors.New(`login "ALICE" is already registered`)

	err := executeRegistration(
		context.Background(),
		userRegistererFunc(func(context.Context, string, string) (model.User, error) {
			return model.User{}, applicationError
		}),
		&passwordReaderStub{lineValue: testRegistrationPassword},
		passwordStreams{
			input:        strings.NewReader(testRegistrationPassword + "\n"),
			output:       io.Discard,
			promptOutput: io.Discard,
		},
		"ALICE",
		true,
	)
	if err == nil {
		t.Fatal("executeRegistration() error = nil, want application error")
	}
	if !errors.Is(err, applicationError) {
		t.Error("registration error does not preserve application error")
	}
	if strings.Contains(err.Error(), testRegistrationPassword) {
		t.Error("application error contains password")
	}
}

func TestTerminalPasswordReader_ReadLine(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "newline",
			input: testRegistrationPassword + "\nignored",
			want:  testRegistrationPassword,
		},
		{
			name:  "Windows newline",
			input: testRegistrationPassword + "\r\n",
			want:  testRegistrationPassword,
		},
		{
			name:  "EOF after password",
			input: testRegistrationPassword,
			want:  testRegistrationPassword,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := (terminalPasswordReader{}).ReadLine(strings.NewReader(tt.input))
			if err != nil {
				t.Fatalf("ReadLine() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("ReadLine() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTerminalPasswordReader_ReadLineRejectsEmptyStdin(t *testing.T) {
	_, err := (terminalPasswordReader{}).ReadLine(strings.NewReader(""))
	if err == nil {
		t.Fatal("ReadLine() error = nil, want no data error")
	}
}

func TestTerminalPasswordReader_ReadHiddenRejectsNonTerminal(t *testing.T) {
	_, err := (terminalPasswordReader{}).ReadHidden(
		strings.NewReader(testRegistrationPassword),
		io.Discard,
		"Password: ",
	)
	if err == nil {
		t.Fatal("ReadHidden() error = nil, want non-terminal error")
	}
	if !strings.Contains(err.Error(), "--password-stdin") {
		t.Errorf("ReadHidden() error = %q, want password-stdin hint", err)
	}
}

func TestExecuteRegistration_RejectsInvalidLoginBeforePasswordInput(t *testing.T) {
	registrarCalled := false
	passwords := &passwordReaderStub{
		hiddenValues: []string{testRegistrationPassword, testRegistrationPassword},
		lineValue:    testRegistrationPassword,
	}

	err := executeRegistration(
		context.Background(),
		userRegistererFunc(func(context.Context, string, string) (model.User, error) {
			registrarCalled = true
			return model.User{}, nil
		}),
		passwords,
		passwordStreams{
			input:        strings.NewReader(testRegistrationPassword + "\n"),
			output:       io.Discard,
			promptOutput: io.Discard,
		},
		"-a",
		false,
	)
	if !errors.Is(err, errInvalidLoginArgument) {
		t.Fatalf("executeRegistration() error = %v, want %v", err, errInvalidLoginArgument)
	}
	if registrarCalled {
		t.Error("registrar was called for invalid login")
	}
	if passwords.hiddenCalls != 0 {
		t.Errorf("hidden password reads = %d, want 0", passwords.hiddenCalls)
	}
	if passwords.lineCalls != 0 {
		t.Errorf("stdin password reads = %d, want 0", passwords.lineCalls)
	}
}
